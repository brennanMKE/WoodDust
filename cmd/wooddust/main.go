// Command wooddust is the Dust Almanac server.
//
//	wooddust serve            run the HTTP server (migrates up first)
//	wooddust seed [-check]    validate data/ and replace the published tables
//	wooddust migrate up|down [N]|version
//	wooddust apikey create|list|revoke
//	wooddust users list|admin|unadmin
//	wooddust identity <portiko command>
//
// Configuration comes from the environment; see .env.example.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/brennanMKE/WoodDust/internal/config"
	"github.com/brennanMKE/WoodDust/internal/content"
	"github.com/brennanMKE/WoodDust/internal/contrib"
	"github.com/brennanMKE/WoodDust/internal/data"
	"github.com/brennanMKE/WoodDust/internal/db"
	"github.com/brennanMKE/WoodDust/internal/identity"
	"github.com/brennanMKE/WoodDust/internal/memstore"
	"github.com/brennanMKE/WoodDust/internal/middleware"
	"github.com/brennanMKE/WoodDust/internal/site"
	"github.com/brennanMKE/WoodDust/internal/store"
	"github.com/brennanMKE/WoodDust/web"

	"github.com/Kolonnade/Portiko/rp/session"
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

  apikey create -name NAME -scopes a,b [-expires YYYY-MM-DD]
                        create an API key (printed once); scopes:
                        submissions:read, submissions:write, stats:read
  apikey list           list keys (prefix, scopes, last use; never the key)
  apikey revoke PREFIX  revoke a key

  users list            local users (Portiko subjects) and who is admin
  users admin SUBJECT   make a signed-in subject an admin
  users unadmin SUBJECT remove admin

  identity <cmd>        Portiko operator commands against IDENTITY_DATABASE_URL:
                        register-site (this site as the first-party client),
                        register-client, migrate, keys, rotate-key, activate-key,
                        retire-key, seal-keys, unseal-keys, set-logout-uri, ...
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
	case "apikey":
		err = apikeyCmd(ctx, cfg, os.Args[2:])
	case "users":
		err = usersCmd(ctx, cfg, os.Args[2:])
	case "identity":
		os.Exit(identity.ServerMain(cfg, os.Args[2:]))
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
	var cstore *contrib.Store
	if cfg.DatabaseURL == "" {
		log.Warn("DATABASE_URL is unset: serving data/ from memory (development only); submissions and admin are unavailable", "data_dir", cfg.DataDir)
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
		cstore = contrib.NewStore(pool)
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

	// Submissions (contract §6).
	formKey := cfg.Submissions.FormTokenKey
	if formKey == nil {
		formKey = make([]byte, 32)
		_, _ = rand.Read(formKey)
		log.Warn("FORM_TOKEN_KEY is unset: using a random key, so open forms break on restart and IP hashes change (set it in production)")
	}
	ipKey := cfg.Submissions.IPHashKey
	if ipKey == nil {
		ipKey = contrib.DeriveKey(formKey, "ip-hash")
	}
	submitLimiter := contrib.NewSubmitLimiter(cfg.Submissions.PerHour, cfg.Submissions.PerDay, cfg.Submissions.GlobalPerDay)
	auth := &contrib.Auth{Origin: cfg.BaseURL}
	if cstore != nil {
		auth.Users = cstore
		times, err := cstore.RecentTimes(ctx)
		if err != nil {
			return fmt.Errorf("submissions: %w", err)
		}
		submitLimiter.Seed(times)
	}
	api := &contrib.API{
		Store: cstore, Auth: auth, Tokens: contrib.NewFormTokens(formKey, cfg.Submissions.MinFill, cfg.Submissions.MaxAge),
		IPs: contrib.NewIPHasher(ipKey), Limiter: submitLimiter, Species: cat.GetSpecies, ClientIP: middleware.IP, Log: log,
	}

	srv := site.New(site.Options{
		Catalog: cat, Pages: pages, SiteName: cfg.SiteName, BaseURL: cfg.BaseURL,
		ImagesDir: cfg.ImagesDir, Dist: web.DistFS(), Logger: log, Contrib: api,
	})
	mux := srv.Routes()

	// Sign-in: the embedded Portiko provider (PLAN §5).
	switch {
	case !cfg.Identity.Enabled():
		log.Warn("IDENTITY_DATABASE_URL is unset: sign-in is disabled; /admin answers 503")
		identity.MountDisabled(mux, "Sign-in is disabled on this server (IDENTITY_DATABASE_URL is not set).")
	case cstore == nil:
		log.Warn("IDENTITY_DATABASE_URL is set but DATABASE_URL is not: sign-in needs the users table, so it is disabled")
		identity.MountDisabled(mux, "Sign-in needs DATABASE_URL (the local users table).")
	default:
		sessions := cstore.Sessions()
		id, err := identity.New(ctx, cfg, identity.Options{
			Sessions: sessions,
			OnSignIn: func(ctx context.Context, s *session.Session) error {
				u, err := cstore.SignedIn(ctx, s.Subject, s.Name)
				if err == nil {
					log.Info("signed in", "subject", s.Subject, "is_admin", u.IsAdmin)
				}
				return err
			},
			Logger: log,
		})
		if err != nil {
			return err
		}
		defer id.Close()
		go id.Sweep(ctx)
		auth.Resolve = id.Resolve
		id.Mount(mux)
		go func() {
			t := time.NewTicker(time.Hour)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					if _, err := sessions.Sweep(ctx); err != nil {
						log.Warn("sweeping sessions", "err", err)
					}
				}
			}
		}()
		log.Info("sign-in enabled", "issuer", cfg.BaseURL, "rp_id", cfg.Host(), "client_id", cfg.Identity.ClientID)
	}

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
				submitLimiter.Sweep()
			}
		}
	}()

	handler := middleware.Chain(mux,
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

// openContrib migrates and opens the contribution store for CLI commands.
func openContrib(ctx context.Context, cfg config.Config) (*contrib.Store, func(), error) {
	if cfg.DatabaseURL == "" {
		return nil, nil, errors.New("DATABASE_URL is required")
	}
	if err := db.MigrateUp(cfg.DatabaseURL); err != nil {
		return nil, nil, fmt.Errorf("migrate: %w", err)
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	return contrib.NewStore(pool), pool.Close, nil
}

func apikeyCmd(ctx context.Context, cfg config.Config, args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("apikey needs create, list or revoke")
	}
	st, done, err := openContrib(ctx, cfg)
	if err != nil {
		return err
	}
	defer done()
	actor := "cli:" + os.Getenv("USER")
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("apikey create", flag.ContinueOnError)
		name := fs.String("name", "", "a name for the key, e.g. home-assistant")
		scopes := fs.String("scopes", "", "comma-separated: submissions:read, submissions:write, stats:read")
		expires := fs.String("expires", "", "optional expiry date, YYYY-MM-DD (UTC)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		req := contrib.KeyRequest{Name: *name}
		for _, sc := range strings.Split(*scopes, ",") {
			if sc = strings.TrimSpace(sc); sc != "" {
				req.Scopes = append(req.Scopes, sc)
			}
		}
		if *expires != "" {
			t, err := time.Parse("2006-01-02", *expires)
			if err != nil {
				return fmt.Errorf("-expires: %w", err)
			}
			req.ExpiresAt = &t
		}
		if fields := contrib.CheckKeyRequest(&req); len(fields) > 0 {
			for f, m := range fields {
				fmt.Fprintf(os.Stderr, "%s: %s\n", f, m)
			}
			return errors.New("invalid key request")
		}
		k, key, err := st.CreateAPIKey(ctx, req.Name, req.Scopes, req.ExpiresAt, nil, actor)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "created key %q (prefix %s, scopes %s). It is shown once; store it now:\n", k.Name, k.Prefix, strings.Join(k.Scopes, ","))
		fmt.Println(key)
		return nil
	case "list":
		keys, err := st.APIKeys(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("%-10s %-20s %-45s %-20s %-20s %s\n", "PREFIX", "NAME", "SCOPES", "LAST USED", "EXPIRES", "REVOKED")
		ts := func(t *time.Time) string {
			if t == nil {
				return "-"
			}
			return t.UTC().Format(time.RFC3339)
		}
		for _, k := range keys {
			fmt.Printf("%-10s %-20s %-45s %-20s %-20s %s\n", k.Prefix, k.Name, strings.Join(k.Scopes, ","), ts(k.LastUsedAt), ts(k.ExpiresAt), ts(k.RevokedAt))
		}
		return nil
	case "revoke":
		if len(args) != 2 {
			return errors.New("usage: apikey revoke PREFIX")
		}
		if err := st.RevokeAPIKey(ctx, args[1], actor); err != nil {
			return fmt.Errorf("revoke %s: %w", args[1], err)
		}
		fmt.Println("revoked", args[1])
		return nil
	}
	return fmt.Errorf("unknown apikey command %q", args[0])
}

func usersCmd(ctx context.Context, cfg config.Config, args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("users needs list, admin or unadmin")
	}
	st, done, err := openContrib(ctx, cfg)
	if err != nil {
		return err
	}
	defer done()
	switch args[0] {
	case "list":
		us, err := st.Users(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("%-40s %-20s %-6s %s\n", "SUBJECT", "NAME", "ADMIN", "LAST SEEN")
		for _, u := range us {
			fmt.Printf("%-40s %-20s %-6v %s\n", u.Subject, u.DisplayName, u.IsAdmin, u.LastSeenAt.UTC().Format(time.RFC3339))
		}
		return nil
	case "admin", "unadmin":
		if len(args) != 2 {
			return fmt.Errorf("usage: users %s SUBJECT", args[0])
		}
		return st.SetAdmin(ctx, args[1], args[0] == "admin", "cli:"+os.Getenv("USER"))
	}
	return fmt.Errorf("unknown users command %q", args[0])
}
