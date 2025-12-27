package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/FlagBrew/local-gpss/internal/handlers/gpss"
	"github.com/FlagBrew/local-gpss/internal/handlers/legality"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lrstanley/chix"
)

func httpServer(ctx context.Context) *http.Server {
	chix.DefaultAPIPrefix = "/api/"

	r := chi.NewRouter()

	r.Use(
		chix.UseContextIP,
		middleware.RequestID,
		chix.UseStructuredLogger(logger),
		chix.UseDebug(cli.Debug),
		chix.UseRecoverer,
		middleware.StripSlashes,
		middleware.Compress(5),
		middleware.Maybe(middleware.StripSlashes, func(r *http.Request) bool {
			return !strings.HasPrefix(r.URL.Path, "/debug/")
		}),
		chix.UseNextURL,
	)

	if cli.Debug {
		r.Mount("/debug", middleware.Profiler())
	}

	r.Route("/api/v2/gpss", gpss.NewHandler().Route)
	r.Route("/api/v2/pksm", legality.NewHandler().Route)

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.HTTP.ListeningAddr, cfg.HTTP.Port),
		Handler: r,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		// Some sane defaults.
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}
