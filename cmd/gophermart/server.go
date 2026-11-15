package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/config"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/handler"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
)

func newServer(cfg config.AppConfig, store repository.Storage, log *zap.SugaredLogger) *http.Server {
	return &http.Server{
		Addr:              cfg.Address,
		Handler:           handler.NewRouter(store, []byte(cfg.Secret), log),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func runServer(srv *http.Server) error {
	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func shutdown(srv *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
