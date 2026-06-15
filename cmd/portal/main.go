package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gfaia/free-thinker/internal/config"
	"github.com/gfaia/free-thinker/internal/db"
	"github.com/gfaia/free-thinker/internal/portal"
	"github.com/gfaia/free-thinker/internal/storage"
)

func main() {
	var (
		configPath = flag.String("config", "config.yaml", "path to config file")
		addr       = flag.String("addr", "", "portal listen address override")
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if *addr != "" {
		cfg.Portal.Addr = *addr
	}

	store, err := db.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	fstore := storage.NewFileStore(cfg.Storage.ContentRoot)
	server := portal.NewServer(store, fstore)
	httpServer := &http.Server{
		Addr:              cfg.Portal.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		log.Printf("[portal] listening on http://%s", cfg.Portal.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[portal] shutdown failed: %v", err)
	}
	log.Println("[portal] stopped")
}
