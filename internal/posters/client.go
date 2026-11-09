package posters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Token is a ShortLinks API token. It never prints: fmt and slog both see
// "[redacted]", so it cannot leak into logs or error messages by accident.
type Token string

// String implements fmt.Stringer.
func (Token) String() string { return "[redacted]" }

// GoString implements fmt.GoStringer.
func (Token) GoString() string { return `posters.Token("[redacted]")` }

// MarshalText keeps the token out of JSON and text encodings (slog's
// JSON handler marshals structs that contain one).
func (Token) MarshalText() ([]byte, error) { return []byte("[redacted]"), nil }

// LogValue implements slog.LogValuer.
func (Token) LogValue() slog.Value { return slog.StringValue("[redacted]") }

// Timeout bounds every ShortLinks call. There are no retries: a failure falls
// back to the stock link.
const Timeout = 5 * time.Second

// Client calls one ShortLinks instance over loopback (ShortLinks v0.4.0
// API tokens; ~/Developer/brennanMKE/ShortLinks/docs/using-api-tokens.md).
type Client struct {
	BaseURL string // e.g. http://127.0.0.1:8084
	Token   Token
	HTTP    *http.Client // nil: a client with Timeout
}

// APIError is a non-2xx answer from ShortLinks.
type APIError struct {
	Status     int
	Code       string // ShortLinks' {"error": "..."} string
	RetryAfter int    // seconds, for 429
}

func (e *APIError) Error() string { return fmt.Sprintf("shortlinks: %d %s", e.Status, e.Code) }

// Plain says what went wrong in words for the Settings card. It never
// includes the token.
func Plain(err error) string {
	var ae *APIError
	if errors.As(err, &ae) {
		switch {
		case ae.Status == http.StatusUnauthorized:
			return "ShortLinks refused the key: it is wrong, revoked or expired. Create a new token and replace it."
		case ae.Code == "api_token_read_only":
			return "The key is read-only, so it can't create links. Create a token without “Read-only” ticked."
		case ae.Code == "api_token_not_permitted":
			return "ShortLinks doesn't allow API tokens for this request (api_token_not_permitted)."
		case ae.Status == http.StatusTooManyRequests:
			if ae.RetryAfter > 0 {
				return fmt.Sprintf("ShortLinks is rate-limiting us. Try again in %d s.", ae.RetryAfter)
			}
			return "ShortLinks is rate-limiting us. Try again shortly."
		case ae.Status == http.StatusConflict:
			return "ShortLinks says that link key is already taken."
		}
		if ae.Code != "" {
			return fmt.Sprintf("ShortLinks answered %d: %s.", ae.Status, ae.Code)
		}
		return fmt.Sprintf("ShortLinks answered %d.", ae.Status)
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout") {
		return "ShortLinks didn't answer within 5 seconds."
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		return "Couldn't reach ShortLinks: " + ue.Err.Error()
	}
	return "ShortLinks call failed: " + err.Error()
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: Timeout}
}

func (c *Client) do(ctx context.Context, method, path string, body any, accept string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+string(c.Token))
	req.Header.Set("Accept", accept)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 512<<10))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		ae := &APIError{Status: res.StatusCode}
		var eb struct {
			Error any `json:"error"`
		}
		if json.Unmarshal(b, &eb) == nil {
			switch v := eb.Error.(type) {
			case string:
				ae.Code = v
			case map[string]any:
				if s, ok := v["code"].(string); ok {
					ae.Code = s
				}
			}
		}
		if len(ae.Code) > 80 {
			ae.Code = ae.Code[:80]
		}
		ae.RetryAfter, _ = strconv.Atoi(res.Header.Get("Retry-After"))
		return nil, ae
	}
	return b, nil
}

// Me is GET /api/me with a token.
type Me struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	BaseURL  string `json:"base_url"`
	APIToken *struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		ReadOnly bool   `json:"read_only"`
	} `json:"api_token"`
}

// Me checks the token.
func (c *Client) Me(ctx context.Context) (Me, error) {
	var m Me
	b, err := c.do(ctx, http.MethodGet, "/api/me", nil, "application/json")
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("unexpected /api/me answer: %w", err)
	}
	return m, nil
}

// LinkRequest is POST /api/links.
type LinkRequest struct {
	DestinationURL string `json:"destination_url"`
	Key            string `json:"key,omitempty"` // custom alias; never deduplicated
	Title          string `json:"title,omitempty"`
	CampaignSlug   string `json:"campaign_slug,omitempty"`
	UTMSource      string `json:"utm_source"`
	UTMMedium      string `json:"utm_medium"`
	UTMCampaign    string `json:"utm_campaign"`
	UTMContent     string `json:"utm_content"`
	Placement      string `json:"placement,omitempty"`
}

// CreateLink creates a link and returns its key.
func (c *Client) CreateLink(ctx context.Context, lr LinkRequest) (string, error) {
	b, err := c.do(ctx, http.MethodPost, "/api/links", lr, "application/json")
	if err != nil {
		return "", err
	}
	var out struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(b, &out); err != nil || out.Key == "" {
		return "", errors.New("unexpected /api/links answer (no key)")
	}
	return out.Key, nil
}

// QR fetches GET /api/links/{key}/qr.svg.
func (c *Client) QR(ctx context.Context, key string) (string, error) {
	b, err := c.do(ctx, http.MethodGet, "/api/links/"+url.PathEscape(key)+"/qr.svg", nil, "image/svg+xml")
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if !(strings.HasPrefix(s, "<svg") || strings.HasPrefix(s, "<?xml")) || !strings.Contains(s, "</svg>") {
		return "", errors.New("unexpected qr.svg answer")
	}
	return s, nil
}

// Campaign checks that GET /api/campaigns/{slug} exists for the token's owner.
func (c *Client) Campaign(ctx context.Context, slug string) error {
	_, err := c.do(ctx, http.MethodGet, "/api/campaigns/"+url.PathEscape(slug), nil, "application/json")
	return err
}
