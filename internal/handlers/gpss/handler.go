package gpss

import "github.com/go-chi/chi/v5"

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Route(r chi.Router) {

}
