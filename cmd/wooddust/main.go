// Command wooddust is the Dust Almanac server.
//
//	wooddust serve            run the HTTP server (migrates up first)
//	wooddust seed [-check]    validate data/ and replace the published tables
//	wooddust migrate up|down [N]|version
//
// Configuration comes from the environment; see .env.example.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/brennanMKE/WoodDust/internal/config"
	"github.com/brennanMKE/WoodDust/internal/content"
	"github.com/brennanMKE/WoodDust/internal/data"
	"github.com/brennanMKE/WoodDust/internal/db"
	"github.com/brennanMKE/WoodDust/internal/memstore"
	"github.com/brennanMKE/WoodDust/internal/middleware"
	"github.com/brennanMKE/WoodDust/internal/site"
	"github.com/brennanMKE/WoodDust/internal/store"
	"github.com/brennanMKE/WoodDust/web"
)

func usage() {
	fmt.Fprint(os.Stderr, `usage: wooddust <command> [flags]

commands:
  serve                 run the HTTP server on 127.0.0.1:$PORT (applies migrations first)
  seed [-check]         check data/ integrity, then replace the published tables
                        -check: validate only (no database needed)
  migrate up            apply all migrations
  migrate down [N]      roll back N migrations (default: all)
  migrate version       print the current migration version
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}
	log := newLogger(cfg.LogFormat)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch os.Args[1] {
	case "serve":
		err = serve(ctx, cfg, log)
	case "seed":
		err = seed(ctx, cfg, os.Args[2:])
	case "migrate":
		err = migrateCmd(cfg, os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newLogger(format string) *slog.Logger {
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func pagesDir(cfg config.Config) string { return filepath.Join(cfg.ContentDir, "pages") }

func serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	var cat site.Catalog
	if cfg.DatabaseURL == "" {
		log.Warn("DATABASE_URL is unset: serving data/ from memory (development only)", "data_dir", cfg.DataDir)
		ds, err := data.Load(cfg.DataDir)
		if err != nil {
			return err
		}
		cat = memstore.New(ds)
	} else {
		if err := db.MigrateUp(cfg.DatabaseURL); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		pool, err := db.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		cat = store.New(pool)
	}

	pages, err := content.Load(pagesDir(cfg))
	if err != nil {
		return fmt.Errorf("content: %w", err)
	}
	log.Info("content loaded", "pages", len(pages.All()))
	for _, msg := range pages.BrokenAnchors() {
		log.Warn("content", "problem", msg)
	}

	trusted, err := middleware.ParseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		return fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	srv := site.New(site.Options{
		Catalog: cat, Pages: pages, SiteName: cfg.SiteName, BaseURL: cfg.BaseURL,
		ImagesDir: cfg.ImagesDir, Dist: web.DistFS(), Logger: log,
	})
	limiter := middleware.NewRateLimiter(cfg.APIRatePerMin, cfg.APIRateBurst)
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				limiter.Sweep()
			}
		}
	}()

	handler := middleware.Chain(srv.Routes(),
		middleware.ClientIP(trusted),
		middleware.Logging(log),
		middleware.SecurityHeaders,
		limiter.Middleware("/api/"),
		middleware.Gzip,
	)
	hs := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", hs.Addr, "base_url", cfg.BaseURL, "site", cfg.SiteName)
		errc <- hs.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return hs.Shutdown(sctx)
	}
}

// integrity loads data/ and content/ and runs every cross-file check.
func integrity(cfg config.Config) (*data.Dataset, data.Report, error) {
	ds, err := data.Load(cfg.DataDir)
	if err != nil {
		return nil, data.Report{}, err
	}
	rep := data.Check(ds)
	pages, err := content.Load(pagesDir(cfg))
	if err != nil {
		rep.Errors = append(rep.Errors, "content: "+err.Error())
		return ds, rep, nil
	}
	for file, keys := range pages.CitedKeys() {
		rep.Merge(data.CheckKeys(ds, file, keys))
	}
	for _, msg := range pages.BrokenAnchors() {
		rep.Errors = append(rep.Errors, msg)
	}
	return ds, rep, nil
}

func seed(ctx context.Context, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	check := fs.Bool("check", false, "validate data/ and content/ only; do not touch the database")
	strict := fs.Bool("strict", false, "treat warnings as errors")
	_ = fs.Parse(args)

	ds, rep, err := integrity(cfg)
	if err != nil {
		return err
	}
	for _, w := range rep.Warnings {
		fmt.Println("warning:", w)
	}
	for _, e := range rep.Errors {
		fmt.Println("error:", e)
	}
	fmt.Printf("integrity: %d species, %d sources, %d errors, %d warnings\n",
		len(ds.Species), len(ds.Sources), len(rep.Errors), len(rep.Warnings))
	if !rep.OK() || (*strict && len(rep.Warnings) > 0) {
		return errors.New("integrity check failed")
	}
	if *check {
		return nil
	}
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required to seed (use -check to validate only)")
	}
	if err := db.MigrateUp(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	res, err := store.New(pool).Seed(ctx, ds, rep.Warnings)
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	fmt.Printf("seeded: %d species, %d names, %d hazards, %d sources (data sha256 %s)\n",
		res.Species, res.Names, res.Hazards, res.Sources, ds.SHA256[:12])
	return nil
}

func migrateCmd(cfg config.Config, args []string) error {
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if len(args) == 0 {
		usage()
		return errors.New("migrate needs up, down or version")
	}
	switch args[0] {
	case "up":
		return db.MigrateUp(cfg.DatabaseURL)
	case "down":
		steps := 0
		if len(args) > 1 {
			n, err := strconv.Atoi(args[1])
			if err != nil || n < 1 {
				return fmt.Errorf("down: N must be a positive integer")
			}
			steps = n
		}
		return db.MigrateDown(cfg.DatabaseURL, steps)
	case "version":
		v, dirty, err := db.Version(cfg.DatabaseURL)
		if err != nil {
			return err
		}
		fmt.Printf("version %d dirty=%v\n", v, dirty)
		return nil
	}
	return fmt.Errorf("unknown migrate command %q", args[0])
}
