package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/config"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/integration"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/logger"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/repository"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.NewAppConfig()

	log, err := logger.New()
	if err != nil {
		return err
	}
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, closeStore, err := repository.NewStorage(cfg)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	defer closeStore(ctx)

	srv := newServer(cfg, store, log)

	errCh := make(chan error, 1)
	go func() {
		errCh <- runServer(srv)
	}()
	log.Infow("starting server", "address", cfg.Address)

	worker := integration.NewWorker(integration.NewClient(), cfg.AccrualAddress)
	jobs := make(chan []string)

	var wg sync.WaitGroup

	wg.Go(func() {
		worker.RunCollector(ctx, store, log, jobs)
	})

	wg.Go(func() {
		worker.RunWorker(ctx, store, log, jobs)
	})
	defer func() {
		stop()
		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		log.Infow("shutting down server")
		return shutdown(srv)
	case err = <-errCh:
		return fmt.Errorf("server error: %w", err)
	}
}
