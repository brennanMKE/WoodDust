package posters

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/brennanMKE/WoodDust/internal/contrib"
	"github.com/brennanMKE/WoodDust/internal/settings"
)

// SettingsStore is what this package needs from internal/settings (a fake
// in tests). Nil means no database: every link falls back to stock.
type SettingsStore interface {
	Value(ctx context.Context, key string) (string, bool, error)
	Info(ctx context.Context, key string) (settings.Info, error)
	Set(ctx context.Context, key, value, actor string) (settings.Info, error)
	Clear(ctx context.Context, key, actor string) error
	RecordCheck(ctx context.Context, key string, ok bool, msg, auditAs string) error
	CanSeal() bool
}

// Abuse ceilings for minted links (docs/posters.md "Tracking"): per hashed
// client IP per hour and per day, and site-wide per day.
const (
	LinksPerIPHour   = 3
	LinksPerIPDay    = 10
	LinksGlobalDay   = 100
	maxShop          = 40
	maxLinkBodyBytes = 4 << 10
)

// Service serves POST /api/v1/posters/links and the admin Settings card.
type Service struct {
	Settings SettingsStore // nil: no database
	SiteBase string        // BASE_URL: destinations are SiteBase + poster path
	Auth     *contrib.Auth
	Tokens   *contrib.FormTokens
	IPs      *contrib.IPHasher
	Limiter  *contrib.SubmitLimiter // contrib.NewSubmitLimiter(LinksPerIPHour, LinksPerIPDay, LinksGlobalDay)
	ClientIP func(*http.Request) string
	HTTP     *http.Client // nil: default with Timeout (tests point it at a fake)
	Stock    fs.FS        // posterfiles.FS: qr/<id>.svg
	Log      *slog.Logger
}

// Register mounts the endpoints.
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/posters/links", s.createLink)
	mux.HandleFunc("GET /api/v1/admin/settings", s.adminGet)
	mux.HandleFunc("PUT /api/v1/admin/settings/{key}", s.adminSet)
	mux.HandleFunc("DELETE /api/v1/admin/settings/{key}", s.adminClear)
	mux.HandleFunc("POST /api/v1/admin/settings/shortlinks/test", s.adminTest)
}

// Link is the endpoint's answer.
type Link struct {
	ShortURL string `json:"short_url"`
	QRSVG    string `json:"qr_svg"`
	Fallback bool   `json:"fallback,omitempty"`
}

// StockLink returns the stock link and its embedded QR code (rendered on
// the fly if the build did not generate it).
func (s *Service) StockLink(p Poster) Link {
	return StockLink(s.Stock, p)
}

// StockLink is the stock link for p, its QR from fsys (qr/<id>.svg) when present.
func StockLink(fsys fs.FS, p Poster) Link {
	l := Link{ShortURL: p.StockURL()}
	if fsys != nil {
		if b, err := fs.ReadFile(fsys, "qr/"+p.ID+".svg"); err == nil {
			l.QRSVG = string(b)
		}
	}
	if l.QRSVG == "" {
		l.QRSVG, _ = QRSVG(l.ShortURL)
	}
	return l
}

// CleanShop trims a shop name to one line of at most 40 characters with no
// control characters. Empty means "not sent".
func CleanShop(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > maxShop {
		s = strings.TrimSpace(string([]rune(s)[:maxShop]))
	}
	return s
}

const keyAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// newKey is a random 10-character ShortLinks alias. A custom alias is never
// deduplicated by ShortLinks (generated keys are, per destination, which
// would give every plaque the same link).
func newKey() string {
	var b strings.Builder
	max := big.NewInt(int64(len(keyAlphabet)))
	for range 10 {
		i, _ := rand.Int(rand.Reader, max)
		b.WriteByte(keyAlphabet[i.Int64()])
	}
	return b.String()
}

type linkRequest struct {
	Poster    string `json:"poster"`
	Shop      string `json:"shop"`
	FormToken string `json:"form_token"`
	Website   string `json:"website"` // honeypot
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, msg string, fields map[string]string) {
	writeJSON(w, status, map[string]errBody{"error": {Code: code, Message: msg, Fields: fields}})
}

func (s *Service) createLink(w http.ResponseWriter, r *http.Request) {
	ipKey := s.IPs.Hash(s.ClientIP(r))
	if ok, wait := s.Limiter.Check(ipKey); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(max(int(math.Ceil(wait.Seconds())), 1)))
		writeError(w, http.StatusTooManyRequests, "rate_limited",
			"Lots of posters made today. Please try again later, or download the ready-made poster.", nil)
		return
	}
	var req linkRequest
	body := http.MaxBytesReader(w, r.Body, maxLinkBodyBytes)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "The request body must be JSON.", nil)
		return
	}
	_, _ = io.Copy(io.Discard, body)

	spam := ""
	switch err := s.Tokens.Check(req.FormToken); {
	case strings.TrimSpace(req.Website) != "":
		spam = "honeypot"
	case errors.Is(err, contrib.ErrTokenTooFast):
		spam = "too_fast"
	case err != nil:
		writeError(w, http.StatusUnprocessableEntity, "invalid", "This page has expired. Reload it and try again.",
			map[string]string{"form_token": "Expired or invalid. Reload the page."})
		return
	}
	p, ok := Get(req.Poster)
	if !ok || !p.Published {
		writeError(w, http.StatusUnprocessableEntity, "invalid", "Unknown poster.", map[string]string{"poster": "Unknown poster."})
		return
	}
	s.Limiter.Record(ipKey)
	if spam != "" {
		// A normal-looking answer, so bots learn nothing; nothing is minted.
		s.Log.Info("poster link refused as spam", "reason", spam, "poster", p.ID)
		writeJSON(w, http.StatusOK, Link{ShortURL: p.StockURL(), QRSVG: s.StockLink(p).QRSVG})
		return
	}
	writeJSON(w, http.StatusOK, s.mint(r.Context(), p, CleanShop(req.Shop)))
}

// shortLinks reads the ShortLinks settings. ok=false: not configured.
func (s *Service) shortLinks(ctx context.Context) (c *Client, publicBase, campaign string, ok bool, err error) {
	if s.Settings == nil {
		return nil, "", "", false, nil
	}
	key, set, err := s.Settings.Value(ctx, settings.ShortLinksAPIKey)
	if err != nil || !set {
		return nil, "", "", false, err
	}
	apiURL, _, err := s.Settings.Value(ctx, settings.ShortLinksAPIURL)
	if err != nil {
		return nil, "", "", false, err
	}
	if publicBase, _, err = s.Settings.Value(ctx, settings.ShortLinksPublicBase); err != nil {
		return nil, "", "", false, err
	}
	if campaign, _, err = s.Settings.Value(ctx, settings.ShortLinksCampaign); err != nil {
		return nil, "", "", false, err
	}
	return &Client{BaseURL: apiURL, Token: Token(key), HTTP: s.HTTP}, publicBase, campaign, true, nil
}

// mint creates the plaque's own link, or falls back to the stock link. It
// never fails.
func (s *Service) mint(ctx context.Context, p Poster, shop string) Link {
	fallback := s.StockLink(p)
	fallback.Fallback = true
	c, publicBase, campaign, ok, err := s.shortLinks(ctx)
	if err != nil {
		s.Log.Error("poster link: reading settings", "err", err)
		return fallback
	}
	if !ok {
		return fallback
	}
	// ShortLinks redirects to destination_url as stored: the utm_* fields
	// are only labels for its reports. Its own UI bakes the same values into
	// the URL, so do the same here or the visit arrives untagged.
	utm := url.Values{
		"utm_source":   {"flyer"},
		"utm_medium":   {"print"},
		"utm_campaign": {"poster-" + p.ID},
		"utm_content":  {"custom"},
	}
	key, err := c.CreateLink(ctx, LinkRequest{
		DestinationURL: strings.TrimRight(s.SiteBase, "/") + p.Destination + "?" + utm.Encode(),
		Key:            newKey(),
		Title:          "Poster: " + p.Title + " (plaque)",
		CampaignSlug:   campaign,
		UTMSource:      utm.Get("utm_source"),
		UTMMedium:      utm.Get("utm_medium"),
		UTMCampaign:    utm.Get("utm_campaign"),
		UTMContent:     utm.Get("utm_content"),
		Placement:      shop,
	})
	if err != nil {
		s.failed(ctx, "creating link", err)
		return fallback
	}
	link := Link{ShortURL: strings.TrimRight(publicBase, "/") + "/u/" + key}
	if link.QRSVG, err = c.QR(ctx, key); err != nil {
		// The link exists; draw its code here rather than lose it.
		s.Log.Warn("poster link: qr.svg failed; rendering locally", "err", err, "key", key)
		if link.QRSVG, err = QRSVG(link.ShortURL); err != nil {
			s.failed(ctx, "rendering QR", err)
			return fallback
		}
	}
	_ = s.Settings.RecordCheck(ctx, settings.ShortLinksAPIKey, true, "", "")
	s.Log.Info("poster link minted", "poster", p.ID, "key", key, "placement", shop != "")
	return link
}

func (s *Service) failed(ctx context.Context, what string, err error) {
	msg := Plain(err)
	s.Log.Warn("poster link: falling back to stock", "step", what, "err", err)
	if rerr := s.Settings.RecordCheck(ctx, settings.ShortLinksAPIKey, false, msg, ""); rerr != nil {
		s.Log.Warn("poster link: recording failure", "err", rerr)
	}
}
