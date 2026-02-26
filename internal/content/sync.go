package content

import (
	"blogengine/internal/storage"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// SyncAssets walks the local sources directory and uploads any missing images to the Store
func SyncAssets(ctx context.Context, store storage.Provider, sourceDir string, logger *slog.Logger) error {
	logger.Info("starting asset sync", "dir", sourceDir)

	rootDir, err := os.OpenRoot(sourceDir)
	if err != nil {
		return fmt.Errorf("could not open directory: %s", sourceDir)
	}

	wantedExtensions := []string{".jpg", ".jpeg", ".png", ".gif"}

	return filepath.WalkDir(rootDir.Name(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !slices.Contains(wantedExtensions, ext) {
			return nil
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		// absCwd, err := filepath.Abs(".")
		// if err != nil {
		// 	return err
		// }

		relPath, err := filepath.Rel(sourceDir, absPath)
		if err != nil {
			return err
		}
		objectKey := filepath.ToSlash(relPath)

		// check if already exists
		if store.Exists(ctx, objectKey) {
			return nil
		}

		logger.Info("syncing missing asset to bucket", "key", objectKey)

		file, err := os.Open(path)
		if err != nil {
			logger.Error("failed to open local file", "path", path, "err", err)
			return nil
		}
		defer file.Close()

		if err := store.Save(ctx, objectKey, file); err != nil {
			logger.Error("failed to upload to bucket", "key", objectKey, "err", err)
		}

		return nil
	})
}
