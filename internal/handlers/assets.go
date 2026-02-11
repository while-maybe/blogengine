package handlers

import (
	"blogengine/internal/content"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
)

type AssetHandler struct {
	Assets    content.MediaService
	Processor content.ImageProcessorService
}

const cacheForAYear = 31536000

func (h *AssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// expected format: /assets/<uuid>-<width>
	path := strings.TrimPrefix(r.URL.Path, "/assets/")
	parts := strings.Split(path, "_")

	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	idStr := parts[0]
	widthStr := parts[1]

	requestedWidth, err := strconv.Atoi(widthStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	supportedWidths := []int{800, 1200, 1920}
	if !slices.Contains(supportedWidths, requestedWidth) {
		http.NotFound(w, r)
		return
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cachePath := filepath.Join("data", "cache", path+".webp")

	// webp already exists
	if _, err := os.Stat(cachePath); err == nil {
		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("Content-Type", "image/webp")
		// attempt to cache in the browser for a long time
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", cacheForAYear))
		http.ServeFile(w, r, cachePath)
		return
	}

	// webp does not exist
	w.Header().Set("X-Cache", "MISS")

	// access the source file saved on disk
	relPath, err := h.Assets.GetRelativePath(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// context that doesn't die when the user leaves the page
	bgCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	wantedWidths := []int{800, 1200, 1920}
	for _, w := range wantedWidths {
		h.Processor.Enqueue(bgCtx, content.ImageJob{
			SourcePath: relPath,
			ID:         id.String(),
			Width:      w,
		})
	}

	originalPath := filepath.Join("sources", relPath)
	http.ServeFile(w, r, originalPath)
}
