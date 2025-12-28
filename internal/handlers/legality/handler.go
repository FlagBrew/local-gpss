package legality

import (
	"net/http"

	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/FlagBrew/local-gpss/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/lrstanley/chix"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Route(r chi.Router) {
	r.Post("/legality", h.legalityCheck)
	r.Post("/legalize", h.legalize)
}

func (h *Handler) legalityCheck(w http.ResponseWriter, r *http.Request) {
	args, statusCode, err := utils.PrepareCall(r, "legality")
	if err != nil {
		chix.JSON(w, r, statusCode, chix.M{"error": err.Error()})
		return
	}

	result, err := utils.ExecGpssConsole[models.GpssLegalityCheckReply](r.Context(), *args)

	if err != nil {
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": err.Error()})
		return
	}

	chix.JSON(w, r, http.StatusOK, result)
}

func (h *Handler) legalize(w http.ResponseWriter, r *http.Request) {
	args, statusCode, err := utils.PrepareCall(r, "legalize")
	if err != nil {
		chix.JSON(w, r, statusCode, chix.M{"error": err.Error()})
		return
	}

	result, err := utils.ExecGpssConsole[models.GpssAutoLegalityReply](r.Context(), *args)

	if err != nil {
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": err.Error()})
		return
	}

	chix.JSON(w, r, http.StatusOK, result)
}
