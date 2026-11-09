// Package config reads the server configuration from the environment.
package config

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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

	// DevInsecure allows http:// sign-in over localhost (Portiko DEV_INSECURE
	// and insecure rp cookies). Refused for an https BASE_URL.
	DevInsecure bool

	Identity    Identity
	Submissions Submissions

	// SettingsKey (SETTINGS_KEY, exactly 32 bytes hex or base64) encrypts
	// secret operator settings at rest (internal/settings). Nil: secret
	// settings cannot be set or used.
	SettingsKey []byte
}

// Identity configures the embedded Portiko provider and the site's own
// relying party (PLAN §5). Everything is read from IDENTITY_* variables so
// nothing clashes with the site's DATABASE_URL and PORT.
type Identity struct {
	DatabaseURL   string // IDENTITY_DATABASE_URL; empty disables sign-in
	CryptoKey     string // IDENTITY_CRYPTO_KEY, 32 bytes hex/base64
	BootstrapCode string // IDENTITY_BOOTSTRAP_CODE: opens passkey enrollment
	SMTPHost      string // IDENTITY_SMTP_HOST (empty: codes are logged)
	SMTPPort      int    // IDENTITY_SMTP_PORT
	SMTPUsername  string // IDENTITY_SMTP_USERNAME
	SMTPPassword  string // IDENTITY_SMTP_PASSWORD
	EmailFrom     string // IDENTITY_EMAIL_FROM
	TemplateDir   string // IDENTITY_TEMPLATE_DIR (optional page overrides)
	ClientID      string // IDENTITY_CLIENT_ID, default "wooddust"
	ClientSecret  string // IDENTITY_CLIENT_SECRET (from register-client)
	CookieKey     []byte // COOKIE_KEY, ≥32 bytes: rp flow cookies
}

// Enabled reports whether sign-in is configured at all.
func (i Identity) Enabled() bool { return i.DatabaseURL != "" }

// Submissions configures anonymous submissions (contract §6, PLAN §6).
type Submissions struct {
	FormTokenKey  []byte        // FORM_TOKEN_KEY (32 bytes hex); signs form tokens
	IPHashKey     []byte        // IP_HASH_KEY (optional; derived from FORM_TOKEN_KEY)
	EphemeralKeys bool          // true when FORM_TOKEN_KEY was unset and a random key is used
	PerHour       int           // SUBMIT_PER_IP_HOUR (5)
	PerDay        int           // SUBMIT_PER_IP_DAY (20)
	GlobalPerDay  int           // SUBMIT_GLOBAL_DAY (200)
	MinFill       time.Duration // SUBMIT_MIN_FILL_SECONDS (4)
	MaxAge        time.Duration // SUBMIT_FORM_MAX_AGE_HOURS (24)
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
		DevInsecure:    boolEnv("DEV_INSECURE"),
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
	if c.DevInsecure && u.Scheme == "https" {
		return c, fmt.Errorf("DEV_INSECURE must not be set for an https BASE_URL")
	}

	id := Identity{
		DatabaseURL:   strings.TrimSpace(os.Getenv("IDENTITY_DATABASE_URL")),
		CryptoKey:     strings.TrimSpace(os.Getenv("IDENTITY_CRYPTO_KEY")),
		BootstrapCode: strings.TrimSpace(os.Getenv("IDENTITY_BOOTSTRAP_CODE")),
		SMTPHost:      os.Getenv("IDENTITY_SMTP_HOST"),
		SMTPUsername:  os.Getenv("IDENTITY_SMTP_USERNAME"),
		SMTPPassword:  os.Getenv("IDENTITY_SMTP_PASSWORD"),
		EmailFrom:     os.Getenv("IDENTITY_EMAIL_FROM"),
		TemplateDir:   os.Getenv("IDENTITY_TEMPLATE_DIR"),
		ClientID:      env("IDENTITY_CLIENT_ID", "wooddust"),
		ClientSecret:  os.Getenv("IDENTITY_CLIENT_SECRET"),
	}
	if id.SMTPPort, err = intEnv("IDENTITY_SMTP_PORT", 587); err != nil {
		return c, err
	}
	if v := strings.TrimSpace(os.Getenv("COOKIE_KEY")); v != "" {
		if id.CookieKey, err = decodeKey(v); err != nil {
			return c, fmt.Errorf("COOKIE_KEY: %w", err)
		}
	}
	if id.Enabled() {
		var problems []string
		if id.CryptoKey == "" {
			problems = append(problems, "IDENTITY_CRYPTO_KEY")
		}
		if id.ClientSecret == "" {
			problems = append(problems, "IDENTITY_CLIENT_SECRET")
		}
		if len(id.CookieKey) == 0 {
			problems = append(problems, "COOKIE_KEY")
		}
		if len(problems) > 0 {
			return c, fmt.Errorf("IDENTITY_DATABASE_URL is set, so these are required too: %s (generate keys with: openssl rand -hex 32)",
				strings.Join(problems, ", "))
		}
	}
	c.Identity = id

	s := Submissions{PerHour: 5, PerDay: 20, GlobalPerDay: 200}
	if v := strings.TrimSpace(os.Getenv("FORM_TOKEN_KEY")); v != "" {
		if s.FormTokenKey, err = decodeKey(v); err != nil {
			return c, fmt.Errorf("FORM_TOKEN_KEY: %w", err)
		}
	}
	if v := strings.TrimSpace(os.Getenv("IP_HASH_KEY")); v != "" {
		if s.IPHashKey, err = decodeKey(v); err != nil {
			return c, fmt.Errorf("IP_HASH_KEY: %w", err)
		}
	}
	if s.PerHour, err = intEnv("SUBMIT_PER_IP_HOUR", s.PerHour); err != nil {
		return c, err
	}
	if s.PerDay, err = intEnv("SUBMIT_PER_IP_DAY", s.PerDay); err != nil {
		return c, err
	}
	if s.GlobalPerDay, err = intEnv("SUBMIT_GLOBAL_DAY", s.GlobalPerDay); err != nil {
		return c, err
	}
	minFill, err := intEnv("SUBMIT_MIN_FILL_SECONDS", 4)
	if err != nil {
		return c, err
	}
	maxAge, err := intEnv("SUBMIT_FORM_MAX_AGE_HOURS", 24)
	if err != nil {
		return c, err
	}
	s.MinFill, s.MaxAge = time.Duration(minFill)*time.Second, time.Duration(maxAge)*time.Hour
	c.Submissions = s

	if v := strings.TrimSpace(os.Getenv("SETTINGS_KEY")); v != "" {
		if c.SettingsKey, err = decodeKey(v); err != nil {
			return c, fmt.Errorf("SETTINGS_KEY: %w", err)
		}
		if len(c.SettingsKey) != 32 {
			return c, fmt.Errorf("SETTINGS_KEY must be exactly 32 bytes (openssl rand -hex 32), got %d", len(c.SettingsKey))
		}
	}
	return c, nil
}

// Addr is the listen address.
func (c Config) Addr() string { return fmt.Sprintf("%s:%d", BindHost, c.Port) }

// Host is BASE_URL's host without a port: the WebAuthn RP ID.
func (c Config) Host() string {
	u, _ := url.Parse(c.BaseURL)
	return u.Hostname()
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func boolEnv(k string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(k))) {
	case "1", "true", "yes":
		return true
	}
	return false
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

// decodeKey reads a key of at least 32 bytes written as hex or base64.
func decodeKey(s string) ([]byte, error) {
	var raw []byte
	if b, err := hex.DecodeString(s); err == nil {
		raw = b
	} else if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		raw = b
	} else if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		raw = b
	} else {
		return nil, fmt.Errorf("must be hex or base64 (generate with: openssl rand -hex 32)")
	}
	if len(raw) < 32 {
		return nil, fmt.Errorf("must be at least 32 bytes, got %d", len(raw))
	}
	return raw, nil
}
