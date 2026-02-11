package content

import (
	"blogengine/internal/storage"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/gofrs/uuid/v5"
)

type AssetManager struct {
	store      storage.Provider
	mu         sync.RWMutex
	uuidToPath map[uuid.UUID]string
	pathToUuid map[string]uuid.UUID
	namespace  uuid.UUID
}

func NewAssetManager(store storage.Provider, ns uuid.UUID) *AssetManager {

	return &AssetManager{
		store:      store,
		uuidToPath: make(map[uuid.UUID]string),
		pathToUuid: make(map[string]uuid.UUID),
		namespace:  ns,
	}
}

// Obfuscate takes a real path (from markdown) and returns a UUID, returning the existing UUID if present
func (am *AssetManager) Obfuscate(path string) (uuid.UUID, error) {
	if path == "" {
		return uuid.UUID{}, fmt.Errorf("invalid path")
	}

	cleanPath := filepath.Clean(path)

	am.mu.RLock()
	if existingUUID, ok := am.pathToUuid[cleanPath]; ok {
		defer am.mu.RUnlock()
		return existingUUID, nil
	}
	am.mu.RUnlock()

	newUuid := uuid.NewV5(am.namespace, cleanPath)

	am.mu.Lock()
	defer am.mu.Unlock()

	if existingUUID, ok := am.pathToUuid[cleanPath]; ok {
		return existingUUID, nil
	}

	am.pathToUuid[cleanPath] = newUuid
	am.uuidToPath[newUuid] = cleanPath

	return newUuid, nil
}

// Retrieve returns the file stream for a given UUID
func (am *AssetManager) Retrieve(uuid uuid.UUID) (io.ReadCloser, error) {
	if uuid.IsNil() {
		return nil, fmt.Errorf("uuid must not be nil")
	}

	am.mu.RLock()
	storedPath, ok := am.uuidToPath[uuid]
	am.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("asset not found")
	}

	return am.store.Open(storedPath)
}

func (am *AssetManager) GetRelativePath(uuid uuid.UUID) (string, error) {
	if uuid.IsNil() {
		return "", fmt.Errorf("uuid must not be nil")
	}
	am.mu.RLock()
	path, ok := am.uuidToPath[uuid]
	am.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("asset not found")
	}

	return path, nil
}
