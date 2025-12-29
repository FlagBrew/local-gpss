package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/apex/log"
)

func ExecGpssConsole[T any](ctx context.Context, args models.GpssConsoleArgs) (*T, error) {
	logger := log.FromContext(ctx)
	var path string

	switch runtime.GOOS {
	case "darwin":
		path = "./bin/GpssConsole"
	case "linux":
		path = "./bin/GpssConsole"
	case "windows":
		path = "./bin/GpssConsole.exe"
	default:
		logger.WithField("platform", runtime.GOOS).Error("unsupported platform")
	}

	// Make sure it exists
	if _, err := os.Stat(path); err != nil {
		logger.WithField("path", path).Error("GPSS Console binary is missing from disk, please make sure you grab it from the latest release")
		return nil, fmt.Errorf("GPSS Console binary is missing from disk")
	}

	cmd := exec.CommandContext(ctx, path, "--mode", args.Mode, "--pokemon", args.Pokemon, "--generation", args.Generation, "--ver", args.Version)

	output, err := cmd.Output()
	if err != nil {
		logger.WithError(err).Error("GPSS Console command failed")
		return nil, err
	}

	if strings.Contains(string(output), "\"error\"") {
		return nil, fmt.Errorf("GPSS Console returned an error")
	}

	var t T
	if err := json.Unmarshal(output, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func PrepareCall(r *http.Request, mode string) (*models.GpssConsoleArgs, int, error) {
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

	err := r.ParseMultipartForm(2 * 1024 * 1024)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to parse form")
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
