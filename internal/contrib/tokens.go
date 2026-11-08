// Package contrib holds anonymous submissions (contract §6), the admin API
// and API keys (contract §7): validation, abuse controls, the Postgres store
// for the contribution tables, and the HTTP handlers mounted under /api/v1.
package contrib

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"
)

// ---- form tokens -----------------------------------------------------------

// FormTokens issues and checks the signed timestamp carried by the submit
// form, so the minimum fill time needs no cookie (PLAN §6).
type FormTokens struct {
	key    []byte
	MinAge time.Duration
	MaxAge time.Duration
	now    func() time.Time
}

// NewFormTokens signs with key (≥32 bytes).
func NewFormTokens(key []byte, minAge, maxAge time.Duration) *FormTokens {
	return &FormTokens{key: key, MinAge: minAge, MaxAge: maxAge, now: time.Now}
}

// Token check results.
var (
	ErrTokenInvalid = errors.New("form token invalid")
	ErrTokenExpired = errors.New("form token expired")
	ErrTokenTooFast = errors.New("form submitted too fast")
)

func (f *FormTokens) sign(payload []byte) []byte {
	m := hmac.New(sha256.New, f.key)
	m.Write([]byte("wooddust form token v1\x00"))
	m.Write(payload)
	return m.Sum(nil)
}

// Issue returns a fresh token: base64url(issued unix seconds ‖ 8 random
// bytes) "." base64url(HMAC).
func (f *FormTokens) Issue() string {
	payload := make([]byte, 16)
	binary.BigEndian.PutUint64(payload, uint64(f.now().Unix()))
	_, _ = rand.Read(payload[8:])
	enc := base64.RawURLEncoding
	return enc.EncodeToString(payload) + "." + enc.EncodeToString(f.sign(payload))
}

// Check verifies the signature and age.
func (f *FormTokens) Check(tok string) error {
	p, s, ok := strings.Cut(tok, ".")
	if !ok {
		return ErrTokenInvalid
	}
	enc := base64.RawURLEncoding
	payload, err1 := enc.DecodeString(p)
	sig, err2 := enc.DecodeString(s)
	if err1 != nil || err2 != nil || len(payload) != 16 || !hmac.Equal(sig, f.sign(payload)) {
		return ErrTokenInvalid
	}
	issued := time.Unix(int64(binary.BigEndian.Uint64(payload)), 0)
	age := f.now().Sub(issued)
	switch {
	case age < -time.Minute:
		return ErrTokenInvalid
	case age < f.MinAge:
		return ErrTokenTooFast
	case age > f.MaxAge:
		return ErrTokenExpired
	}
	return nil
}

// ---- IP hashing --------------------------------------------------------------

// IPHasher turns a client IP into a keyed hash under a salt that rotates
// every UTC day. The raw IP is never stored.
type IPHasher struct {
	key []byte
	now func() time.Time
}

// NewIPHasher hashes with key (≥32 bytes).
func NewIPHasher(key []byte) *IPHasher { return &IPHasher{key: key, now: time.Now} }

// Hash returns 32 hex characters.
func (h *IPHasher) Hash(ip string) string {
	day := h.now().UTC().Format("2006-01-02")
	salt := hmac.New(sha256.New, h.key)
	salt.Write([]byte("wooddust ip salt\x00" + day))
	m := hmac.New(sha256.New, salt.Sum(nil))
	m.Write([]byte(ip))
	return hex.EncodeToString(m.Sum(nil))[:32]
}

// DeriveKey derives an independent key from a master key for one purpose.
func DeriveKey(master []byte, purpose string) []byte {
	m := hmac.New(sha256.New, master)
	m.Write([]byte("wooddust derive\x00" + purpose))
	return m.Sum(nil)
}

// ---- status tokens -----------------------------------------------------------

// NewStatusToken returns 32 random bytes, base64url, and the SHA-256 to store.
func NewStatusToken() (token string, hash []byte) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token)
}

// HashToken is SHA-256 of the token string.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// ---- API keys ------------------------------------------------------------------

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Scopes an API key may carry (contract §7).
var Scopes = []string{ScopeSubmissionsRead, ScopeSubmissionsWrite, ScopeStatsRead}

// Scope names.
const (
	ScopeSubmissionsRead  = "submissions:read"
	ScopeSubmissionsWrite = "submissions:write"
	ScopeStatsRead        = "stats:read"
)

// ValidScope reports whether s is a known scope.
func ValidScope(s string) bool {
	for _, v := range Scopes {
		if v == s {
			return true
		}
	}
	return false
}

func randBase62(n int) string {
	var sb strings.Builder
	max := big.NewInt(int64(len(base62)))
	for range n {
		i, _ := rand.Int(rand.Reader, max)
		sb.WriteByte(base62[i.Int64()])
	}
	return sb.String()
}

// encodeBase62 renders bytes as a base62 number.
func encodeBase62(b []byte) string {
	n := new(big.Int).SetBytes(b)
	if n.Sign() == 0 {
		return "0"
	}
	base := big.NewInt(62)
	mod := new(big.Int)
	var out []byte
	for n.Sign() > 0 {
		n.DivMod(n, base, mod)
		out = append(out, base62[mod.Int64()])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

// NewAPIKey returns "da_<8-char prefix>_<32 random bytes base62>", its
// prefix, and the SHA-256 of the full key (the only thing stored).
func NewAPIKey() (key, prefix string, hash []byte) {
	prefix = randBase62(8)
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	key = "da_" + prefix + "_" + encodeBase62(secret)
	return key, prefix, HashToken(key)
}

// ParseAPIKey extracts the prefix from a key of the right shape.
func ParseAPIKey(key string) (prefix string, ok bool) {
	rest, ok := strings.CutPrefix(key, "da_")
	if !ok {
		return "", false
	}
	prefix, secret, ok := strings.Cut(rest, "_")
	if !ok || len(prefix) != 8 || len(secret) < 20 || len(secret) > 64 {
		return "", false
	}
	for _, c := range prefix + secret {
		if !strings.ContainsRune(base62, c) {
			return "", false
		}
	}
	return prefix, true
}

// KeyMatches compares a presented key with a stored hash in constant time.
func KeyMatches(key string, stored []byte) bool {
	return subtle.ConstantTimeCompare(HashToken(key), stored) == 1
}
