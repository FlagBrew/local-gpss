package utils

import (
	"context"
	"encoding/json"
	"fmt"
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
