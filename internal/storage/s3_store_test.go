//go:build integration

package storage

import (
	"blogengine/internal/config"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTest(ctx context.Context) (testcontainers.Container, *S3Store, func(), error) {
	tomlConfigFile, err := os.Open("../../infra/garage/garage.toml")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not read garage.toml: %w", err)
	}

	metaDir, err := os.MkdirTemp("", "garage-meta-*")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create meta dir: %w", err)
	}

	dataDir, err := os.MkdirTemp("", "garage-data-*")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "dxflrs/garage:v1.0.0",
		ExposedPorts: []string{"3900/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      tomlConfigFile.Name(),
				ContainerFilePath: "/etc/garage.toml",
				FileMode:          0644,
			},
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Binds = []string{
				metaDir + ":/tmp/garage/meta",
				dataDir + ":/tmp/garage/data",
			}
		},
		Env: map[string]string{
			"GARAGE_RPC_SECRET": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		WaitingFor: wait.ForListeningPort("3900/tcp"),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start container: %w", err)
	}

	cleanup := func() {
		if c != nil {
			c.Terminate(ctx)
		}
		os.RemoveAll(metaDir)
		os.RemoveAll(dataDir)
	}

	success := false
	defer func() {
		if !success {
			cleanup()
		}
	}()

	// garage needs a little time to fully initialise
	time.Sleep(2 * time.Second)

	// Get node ID from status
	nodeID, err := getNodeID(ctx, c)
	if err != nil {
		return nil, nil, nil, err
	}

	// Configure layout
	if err := execGarage(ctx, c, "layout assign", []string{"/garage", "layout", "assign", "-z", "dc1", "-c", "1G", nodeID}); err != nil {
		return nil, nil, nil, err
	}

	if err := execGarage(ctx, c, "layout apply", []string{"/garage", "layout", "apply", "--version", "1"}); err != nil {
		return nil, nil, nil, err
	}

	// Create access key
	accessKey, secretKey, err := createKey(ctx, c)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create bucket and allow key
	if err := execGarage(ctx, c, "bucket create", []string{"/garage", "bucket", "create", "test-bucket"}); err != nil {
		return nil, nil, nil, err
	}

	if err := execGarage(ctx, c, "bucket allow", []string{"/garage", "bucket", "allow", "--read", "--write", "--owner", "--key", "test-key", "test-bucket"}); err != nil {
		return nil, nil, nil, err
	}

	host, _ := c.Host(ctx)
	port, _ := c.MappedPort(ctx, "3900")

	cfg := config.S3Config{
		Endpoint:  fmt.Sprintf("http://%s:%s", host, port.Port()),
		Region:    "garage",
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    "test-bucket",
	}

	store, err := NewS3Store(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create store: %w", err)
	}

	success = true
	return c, store, cleanup, nil
}

func execGarage(ctx context.Context, c testcontainers.Container, name string, cmd []string) error {
	_, reader, err := c.Exec(ctx, cmd)
	if err != nil {
		return fmt.Errorf("%s exec error: %w", name, err)
	}
	output, _ := io.ReadAll(reader)
	fmt.Printf("=== %s ===\n%s\n", name, string(output))
	return nil
}

func getNodeID(ctx context.Context, c testcontainers.Container) (string, error) {
	_, reader, err := c.Exec(ctx, []string{"/garage", "status"})
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}
	output, _ := io.ReadAll(reader)

	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 1 && len(fields[0]) == 16 {
			if _, err := hex.DecodeString(fields[0]); err == nil {
				return fields[0], nil
			}
		}
	}
	return "", fmt.Errorf("could not parse node ID from status output")
}

func createKey(ctx context.Context, c testcontainers.Container) (accessKey, secretKey string, err error) {
	_, reader, err := c.Exec(ctx, []string{"/garage", "key", "create", "test-key"})
	if err != nil {
		return "", "", fmt.Errorf("failed to create key: %w", err)
	}
	output, _ := io.ReadAll(reader)

	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "Key ID:") {
			accessKey = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
		if strings.Contains(line, "Secret key:") {
			secretKey = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}

	if accessKey == "" || secretKey == "" {
		return "", "", fmt.Errorf("failed to parse key output:\n%s", string(output))
	}

	return accessKey, secretKey, nil
}

var testStore *S3Store

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, store, cleanup, err := setupTest(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		container.Terminate(ctx)
		cleanup()
	}()

	testStore = store
	os.Exit(m.Run())
}

func TestObjectStorageCRUD(t *testing.T) {
	ctx := context.Background()
	key := "test/hello.txt"

	content := "hello world"

	// save
	err := testStore.Save(ctx, key, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// retrieve
	rc, err := testStore.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer rc.Close()

	// compare with original content
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("could not read object: %v", err)
	}
	if string(got) != content {
		t.Errorf("expected %q, got %q", content, string(got))
	}
}
