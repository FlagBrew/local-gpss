package legality

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
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

func (h *Handler) prepareCall(r *http.Request, mode string) (*models.GpssConsoleArgs, int, error) {
	generation := r.Header.Get("generation")
	if generation == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("version header is required")
	}

	version := r.Header.Get("version")
	if version == "" && mode == "legalize" {
		return nil, http.StatusBadRequest, fmt.Errorf("version header is required")
	}

	args := models.GpssConsoleArgs{
		Version:    version,
		Generation: generation,
		Mode:       mode,
	}

	pkmn, _, err := r.FormFile("pkmn")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error reading pkmn file: %w", err)
	}

	defer pkmn.Close()

	// Base64 encode the pokemon
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, pkmn); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error reading pkmn file from body: %w", err)
	}

	pkmn.Close()
	b64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	args.Pokemon = b64Str

	return &args, http.StatusOK, nil
}

func (h *Handler) legalityCheck(w http.ResponseWriter, r *http.Request) {
	args, statusCode, err := h.prepareCall(r, "legality")
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
	args, statusCode, err := h.prepareCall(r, "legalize")
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
