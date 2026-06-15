package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gfaia/free-thinker/internal/config"
	"github.com/gfaia/free-thinker/internal/db"
	"github.com/gfaia/free-thinker/internal/engine"
	"github.com/gfaia/free-thinker/internal/fetcher"
	"github.com/gfaia/free-thinker/internal/fetcher/sources/zhihu"
	"github.com/gfaia/free-thinker/internal/storage"

	"github.com/robfig/cron/v3"
)

func main() {
	var (
		configPath = flag.String("config", "config.yaml", "path to config file")
		once       = flag.Bool("once", false, "run once instead of as a daemon")
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store, err := db.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	fstore := storage.NewFileStore(cfg.Storage.ContentRoot)

	reg := fetcher.NewRegistry()
	if err := reg.Register(zhihu.New()); err != nil {
		log.Fatalf("register zhihu: %v", err)
	}

	eng := engine.New(reg, store, fstore, cfg.Fetchers)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *once {
		if err := eng.RunOnce(ctx); err != nil {
			log.Fatalf("run once: %v", err)
		}
		return
	}

	c := cron.New()
	_, err = c.AddFunc(cfg.Schedule.Cron, func() {
		log.Printf("[cron] tick starting")
		if err := eng.RunOnce(context.Background()); err != nil {
			log.Printf("[cron] tick failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("add cron: %v", err)
	}
	c.Start()
	log.Printf("[cron] started with schedule: %s", cfg.Schedule.Cron)

	<-ctx.Done()
	stopCtx := c.Stop()
	<-stopCtx.Done()
	log.Println("[cron] stopped")
}
