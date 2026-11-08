// Package config reads the server configuration from the environment.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// BindHost is fixed: the app only ever listens on loopback; Apache is the
// public endpoint (PLAN §1).
const BindHost = "127.0.0.1"

// Config is the process configuration.
type Config struct {
	DatabaseURL    string // empty: serve from DATA_DIR in memory (dev only)
	Port           int
	SiteName       string
	BaseURL        string // origin without trailing slash, e.g. https://wooddust.sstools.co
	ImagesDir      string
	DataDir        string
	ContentDir     string // contains pages/*.md
	TrustedProxies string // comma-separated IPs/CIDRs allowed to set X-Forwarded-For
	APIRatePerMin  int
	APIRateBurst   int
	LogFormat      string // "text" or "json"
}

// Load reads the environment and applies defaults.
func Load() (Config, error) {
	c := Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		Port:           8080,
		SiteName:       env("SITE_NAME", "Dust Almanac"),
		ImagesDir:      env("IMAGES_DIR", "images"),
		DataDir:        env("DATA_DIR", "data"),
		ContentDir:     env("CONTENT_DIR", "content"),
		TrustedProxies: os.Getenv("TRUSTED_PROXIES"),
		APIRatePerMin:  120,
		APIRateBurst:   60,
		LogFormat:      env("LOG_FORMAT", "text"),
	}
	var err error
	if c.Port, err = intEnv("PORT", c.Port); err != nil {
		return c, err
	}
	if c.APIRatePerMin, err = intEnv("API_RATE_PER_MIN", c.APIRatePerMin); err != nil {
		return c, err
	}
	if c.APIRateBurst, err = intEnv("API_RATE_BURST", c.APIRateBurst); err != nil {
		return c, err
	}
	c.BaseURL = strings.TrimRight(env("BASE_URL", fmt.Sprintf("http://localhost:%d", c.Port)), "/")
	u, err := url.Parse(c.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || (u.Path != "" && u.Path != "/") {
		return c, fmt.Errorf("BASE_URL must be an origin like https://example.com, got %q", c.BaseURL)
	}
	return c, nil
}

// Addr is the listen address.
func (c Config) Addr() string { return fmt.Sprintf("%s:%d", BindHost, c.Port) }

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func intEnv(k string, def int) (int, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 || n > 65535*1000 {
		return def, fmt.Errorf("%s must be a positive integer, got %q", k, v)
	}
	return n, nil
}
