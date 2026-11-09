package settings

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/brennanMKE/WoodDust/internal/db"
)

var testKey = bytes.Repeat([]byte{7}, 32)

const goodToken = "slk_AbCdEfGh012345678901234567890123456789-_xyz"

func TestTokenShape(t *testing.T) {
	if len(goodToken) != 47 {
		t.Fatalf("fixture length %d", len(goodToken))
	}
	d := Defs[ShortLinksAPIKey]
	if _, err := d.Check(" " + goodToken + "\n"); err != nil {
		t.Fatalf("good token refused: %v", err)
	}
	for _, bad := range []string{"", "slk_short", "da_AbCdEfGh_" + strings.Repeat("x", 40), goodToken + "x", strings.Replace(goodToken, "A", "!", 1)} {
		if _, err := d.Check(bad); !errors.Is(err, ErrInvalid) {
			t.Errorf("%q accepted", bad)
		}
	}
	if h := d.Hint(goodToken); h != "slk_AbCdEfGh…" || strings.Contains(h, "0123") {
		t.Errorf("hint %q", h)
	}
}

func TestPlainDefs(t *testing.T) {
	u := Defs[ShortLinksAPIURL]
	if v, err := u.Check("http://127.0.0.1:9000/"); err != nil || v != "http://127.0.0.1:9000" {
		t.Errorf("api url: %q %v", v, err)
	}
	for _, bad := range []string{"ftp://x", "http://x/path", "x", "https://u:p@x"} {
		if _, err := u.Check(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	c := Defs[ShortLinksCampaign]
	if _, err := c.Check("Posters!"); err == nil {
		t.Error("bad campaign accepted")
	}
	if Defs[ShortLinksPublicBase].Default != "https://go.dustalmanac.com" || c.Default != "posters" {
		t.Error("defaults changed")
	}
}

func TestCipher(t *testing.T) {
	c, err := NewCipher(testKey)
	if err != nil {
		t.Fatal(err)
	}
	sealed := c.Seal(ShortLinksAPIKey, []byte(goodToken))
	if bytes.Contains(sealed, []byte("slk_")) {
		t.Fatal("plaintext visible")
	}
	got, err := c.Open(ShortLinksAPIKey, sealed)
	if err != nil || string(got) != goodToken {
		t.Fatalf("round trip: %q %v", got, err)
	}
	if _, err := c.Open(ShortLinksAPIURL, sealed); err == nil {
		t.Error("opened under another key name (AAD not bound)")
	}
	other, _ := NewCipher(bytes.Repeat([]byte{8}, 32))
	if _, err := other.Open(ShortLinksAPIKey, sealed); err == nil {
		t.Error("opened with another key")
	}
	if c, err := NewCipher(nil); c != nil || err != nil {
		t.Error("nil key should give nil cipher")
	}
	if _, err := NewCipher(make([]byte, 16)); err == nil {
		t.Error("short key accepted")
	}
}

func TestStore(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	if err := db.MigrateDown(url, 0); err != nil {
		t.Fatal(err)
	}
	if err := db.MigrateUp(url); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	c, _ := NewCipher(testKey)
	st := NewStore(pool, c)

	if _, ok, err := st.Value(ctx, ShortLinksAPIKey); ok || err != nil {
		t.Fatalf("unset secret: ok=%v err=%v", ok, err)
	}
	if v, ok, _ := st.Value(ctx, ShortLinksCampaign); !ok || v != "posters" {
		t.Fatalf("default campaign %q", v)
	}
	in, err := st.Set(ctx, ShortLinksAPIKey, goodToken+"\n", "cli:test")
	if err != nil {
		t.Fatal(err)
	}
	if !in.Set || in.Hint != "slk_AbCdEfGh…" || in.UpdatedBy != "cli:test" || in.CheckedAt != nil {
		t.Fatalf("info %+v", in)
	}
	var raw []byte
	_ = pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, ShortLinksAPIKey).Scan(&raw)
	if bytes.Contains(raw, []byte(goodToken[4:])) {
		t.Fatal("secret stored in clear")
	}
	if v, ok, err := st.Value(ctx, ShortLinksAPIKey); !ok || err != nil || v != goodToken {
		t.Fatalf("value %q %v", v, err)
	}
	if err := st.RecordCheck(ctx, ShortLinksAPIKey, false, "refused", "user:x"); err != nil {
		t.Fatal(err)
	}
	in, _ = st.Info(ctx, ShortLinksAPIKey)
	if in.CheckOK == nil || *in.CheckOK || in.CheckError != "refused" {
		t.Fatalf("check not recorded: %+v", in)
	}
	if _, err := st.Set(ctx, "nope", "x", "t"); !errors.Is(err, ErrUnknownKey) {
		t.Fatal("unknown key accepted")
	}
	if err := st.Clear(ctx, ShortLinksAPIKey, "cli:test"); err != nil {
		t.Fatal(err)
	}
	if in, _ := st.Info(ctx, ShortLinksAPIKey); in.Set {
		t.Fatal("not cleared")
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM settings_audit WHERE key = $1`, ShortLinksAPIKey).Scan(&n)
	if n != 3 { // set, test, clear
		t.Fatalf("audit rows %d", n)
	}
	if _, err := NewStore(pool, nil).Set(ctx, ShortLinksAPIKey, goodToken, "t"); !errors.Is(err, ErrNoCipher) {
		t.Fatalf("set without cipher: %v", err)
	}
}
