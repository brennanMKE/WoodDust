package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientIP(t *testing.T) {
	trusted, err := ParseTrustedProxies("127.0.0.1, ::1, 10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ remote, xff, want string }{
		{"203.0.113.9:1234", "1.2.3.4", "203.0.113.9"},              // untrusted peer: header ignored
		{"127.0.0.1:5555", "198.51.100.7", "198.51.100.7"},          // Apache
		{"127.0.0.1:5555", "6.6.6.6, 198.51.100.7", "198.51.100.7"}, // spoofed left entry ignored
		{"127.0.0.1:5555", "198.51.100.7, 10.1.2.3", "198.51.100.7"},
		{"127.0.0.1:5555", "", "127.0.0.1"},
		{"[::1]:5555", "2001:db8::1", "2001:db8::1"},
	}
	for _, c := range cases {
		var got string
		h := ClientIP(trusted)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = IP(r) }))
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		h.ServeHTTP(httptest.NewRecorder(), r)
		if got != c.want {
			t.Errorf("remote %s xff %q: got %s want %s", c.remote, c.xff, got, c.want)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	l := NewRateLimiter(60, 2)
	now := time.Unix(0, 0)
	l.now = func() time.Time { return now }
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), l.Middleware("/api/"))
	do := func(path string) int {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		r.RemoteAddr = "192.0.2.1:1"
		h.ServeHTTP(rec, r)
		return rec.Code
	}
	if do("/api/x") != 200 || do("/api/x") != 200 {
		t.Fatal("burst should pass")
	}
	if c := do("/api/x"); c != 429 {
		t.Fatalf("third request: %d", c)
	}
	if do("/wood/oak") != 200 {
		t.Fatal("non-API paths are not limited")
	}
	now = now.Add(time.Second)
	if do("/api/x") != 200 {
		t.Fatal("token should refill after 1s")
	}
}

func TestGzipAndHeaders(t *testing.T) {
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/img" {
			w.Header().Set("Content-Type", "image/jpeg")
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		_, _ = io.WriteString(w, strings.Repeat("hello ", 100))
	}), SecurityHeaders, Gzip)

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip, deflate")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("html should be gzipped")
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(zr)
	if !strings.HasPrefix(string(b), "hello") {
		t.Error("bad gzip body")
	}
	for _, k := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy"} {
		if rec.Header().Get(k) == "" {
			t.Errorf("missing %s", k)
		}
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Error("CSP must set frame-ancestors 'none'")
	}

	r = httptest.NewRequest("GET", "/img", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("jpeg should not be gzipped")
	}
	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip;q=0")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("q=0 should disable gzip")
	}
}
