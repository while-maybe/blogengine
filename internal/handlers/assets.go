package handlers

import (
	"blogengine/internal/content"
	"blogengine/internal/telemetry"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type AssetHandler struct {
	Assets    content.MediaService
	Processor content.ImageProcessorService
	Tracer    trace.Tracer
	Metrics   *telemetry.Metrics
	Logger    *slog.Logger
}

const cacheForAYear = 31536000

func (h *AssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.Tracer.Start(r.Context(), "AssetHandler.ServeHTTP")
	defer span.End()

	// expected format: /assets/{key} where key = <uuid>_<width>
	key := r.PathValue("key")
	parts := strings.Split(key, "_")

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

	variantKey := fmt.Sprintf("%s_%d.webp", id.String(), requestedWidth)
	if h.Assets.Exists(ctx, variantKey) {
		span.SetAttributes(attribute.String("cache.status", "hit"))
		h.Metrics.CacheHitsTotal.Add(ctx, 1)

		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("Content-Type", "image/webp")
		// attempt to cache in the browser for a long time
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", cacheForAYear))

		reader, err := h.Assets.RetrieveKey(ctx, variantKey) // Need to update Retrieve to accept string key?
		if err != nil {
			// Handle error
			http.NotFound(w, r)
			return
		}
		defer reader.Close()

		io.Copy(w, reader)
		return
	}

	span.SetAttributes(attribute.String("cache.status", "miss"))
	h.Metrics.CacheMissesTotal.Add(ctx, 1)

	// webp does not exist
	w.Header().Set("X-Cache", "MISS")

	// access the source file saved on disk
	relPath, err := h.Assets.GetRelativePath(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// context that doesn't die when the user leaves the page
	bgCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	currentSpan := trace.SpanFromContext(r.Context())

	wantedWidths := []int{800, 1200, 1920}
	for _, w := range wantedWidths {
		h.Processor.Enqueue(bgCtx, content.ImageJob{
			SourcePath: relPath,
			ID:         id.String(),
			Width:      w,
			ParentSpan: currentSpan.SpanContext(),
		})
	}

	reader, err := h.Assets.Retrieve(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to retrieve asset from S3", "id", id, "err", err)
		http.NotFound(w, r)
		return
	}
	defer reader.Close()

	ext := filepath.Ext(relPath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream" // fallback
	}
	w.Header().Set("Content-Type", mimeType)

	if _, err := io.Copy(w, reader); err != nil {
		h.Logger.Warn("stream interrupted", "err", err)
	}
}
