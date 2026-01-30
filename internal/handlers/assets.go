package handlers

import (
	"blogengine/internal/content"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofrs/uuid/v5"
)

type AssetHandler struct {
	Assets content.MediaService
}

const cacheAge = 31536000

func (h *AssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/assets/")

	id, err := uuid.FromString(idStr)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	stream, err := h.Assets.Retrieve(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	defer stream.Close()

	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", cacheAge))

	_, _ = io.Copy(w, stream)
}
