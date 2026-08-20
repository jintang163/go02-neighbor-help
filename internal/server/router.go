package server

import (
	"net/http"

	"go02-neighbor-help/internal/handler"
)

func NewMux(h *handler.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.Routes(mux)
	return mux
}
