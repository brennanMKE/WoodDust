// Package settings holds operator settings that change at run time without a
// deploy (docs/posters.md "The key in admin Settings"): the ShortLinks API key
// and where to reach ShortLinks.
//
// Keys are allowlisted here. A secret value is sealed with AES-256-GCM under
// SETTINGS_KEY, with the key name as additional data so a sealed value cannot
// be moved to another key. Only a short display hint is kept in clear. There
// is deliberately no way to print a secret back out: the server decrypts it
// only to use it.
package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Setting names.
const (
	ShortLinksAPIKey     = "shortlinks_api_key"
	ShortLinksAPIURL     = "shortlinks_api_url"
	ShortLinksPublicBase = "shortlinks_public_base"
	ShortLinksCampaign   = "shortlinks_campaign"
)

// DefaultShortLinksAPIURL is the go.dustalmanac.com instance on photon's
// loopback. The port is a placeholder until that instance is registered in
// /etc/photon/ports.conf; set shortlinks_api_url once it is.
const DefaultShortLinksAPIURL = "http://127.0.0.1:8084"

// Def describes one allowlisted setting.
type Def struct {
	Name     string
	Secret   bool
	Default  string
	Validate func(string) error
}

var (
	reShortLinksKey = regexp.MustCompile(`^slk_[A-Za-z0-9_-]{43}$`)
	reCampaign      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
)

// ErrInvalid wraps a validation failure; its message is safe to show.
var ErrInvalid = errors.New("invalid value")

func invalid(msg string) error { return fmt.Errorf("%w: %s", ErrInvalid, msg) }

func validOrigin(v string) error {
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return invalid("must be an origin such as https://go.example.com (no path)")
	}
	return nil
}

// Defs is the allowlist.
var Defs = map[string]Def{
	ShortLinksAPIKey: {Name: ShortLinksAPIKey, Secret: true, Validate: func(v string) error {
		if !reShortLinksKey.MatchString(v) {
			return invalid("a ShortLinks API token looks like slk_ followed by 43 letters, digits, - or _")
		}
		return nil
	}},
	ShortLinksAPIURL:     {Name: ShortLinksAPIURL, Default: DefaultShortLinksAPIURL, Validate: validOrigin},
	ShortLinksPublicBase: {Name: ShortLinksPublicBase, Default: "https://go.dustalmanac.com", Validate: validOrigin},
	ShortLinksCampaign: {Name: ShortLinksCampaign, Default: "posters", Validate: func(v string) error {
		if !reCampaign.MatchString(v) {
			return invalid("a campaign slug is lowercase letters, digits and hyphens")
		}
		return nil
	}},
}

// Lookup returns the definition of an allowlisted key.
func Lookup(key string) (Def, bool) {
	d, ok := Defs[key]
	return d, ok
}

// Names returns the allowlisted keys in a fixed order.
func Names() []string {
	return []string{ShortLinksAPIKey, ShortLinksAPIURL, ShortLinksPublicBase, ShortLinksCampaign}
}

// Normalize trims the value and drops a trailing slash from origins.
func (d Def) Normalize(v string) string {
	v = strings.TrimSpace(v)
	if d.Name == ShortLinksAPIURL || d.Name == ShortLinksPublicBase {
		v = strings.TrimRight(v, "/")
	}
	return v
}

// Check normalizes and validates a value for d.
func (d Def) Check(v string) (string, error) {
	v = d.Normalize(v)
	if v == "" {
		return "", invalid("a value is required (use clear to remove a setting)")
	}
	if !utf8.ValidString(v) || len(v) > 512 {
		return "", invalid("too long or not text")
	}
	if d.Validate != nil {
		if err := d.Validate(v); err != nil {
			return "", err
		}
	}
	return v, nil
}

// Hint is what may be shown of a value: the first 12 characters of a secret
// (slk_AbCdEfGh, which ShortLinks itself shows to identify a token), or
// the whole of a plain value.
func (d Def) Hint(v string) string {
	if !d.Secret {
		return v
	}
	if len(v) <= 12 {
		return ""
	}
	return v[:12] + "…"
}

// ---- encryption ---------------------------------------------------------------

// Cipher seals secret values (AES-256-GCM, AAD = key name).
type Cipher struct{ aead cipher.AEAD }

// ErrNoCipher means SETTINGS_KEY is not configured.
var ErrNoCipher = errors.New("SETTINGS_KEY is not set, so secret settings are unavailable (generate one with: openssl rand -hex 32)")

// NewCipher builds a Cipher from a 32-byte key. A nil key returns nil, nil.
func NewCipher(key []byte) (*Cipher, error) {
	if key == nil {
		return nil, nil
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("settings key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Seal returns nonce || ciphertext.
func (c *Cipher) Seal(name string, plain []byte) []byte {
	nonce := make([]byte, c.aead.NonceSize(), c.aead.NonceSize()+len(plain)+c.aead.Overhead())
	_, _ = rand.Read(nonce)
	return c.aead.Seal(nonce, nonce, plain, []byte(name))
}

// Open reverses Seal; it fails if the value was sealed under another name or key.
func (c *Cipher) Open(name string, sealed []byte) ([]byte, error) {
	n := c.aead.NonceSize()
	if len(sealed) < n+c.aead.Overhead() {
		return nil, errors.New("sealed value too short")
	}
	plain, err := c.aead.Open(nil, sealed[:n], sealed[n:], []byte(name))
	if err != nil {
		return nil, errors.New("cannot decrypt setting (wrong SETTINGS_KEY, or the value was tampered with)")
	}
	return plain, nil
}
