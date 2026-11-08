// Package middleware holds the HTTP wrappers: security headers, gzip,
// request logging, client-IP resolution behind Apache, and a per-IP rate
// limit for the API.
package middleware

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Chain applies wrappers so the first one listed is outermost.
func Chain(h http.Handler, wrappers ...func(http.Handler) http.Handler) http.Handler {
	for i := len(wrappers) - 1; i >= 0; i-- {
		h = wrappers[i](h)
	}
	return h
}

// CSP allows same-origin resources only. Inline <script> blocks on our pages
// are JSON (type="application/json" and "application/ld+json"), which
// browsers never execute, so script-src needs no 'unsafe-inline' or hash.
const CSP = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
	"font-src 'self'; connect-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'none'; " +
	"form-action 'self'; frame-ancestors 'none'"

// SecurityHeaders sets the site-wide security headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", CSP)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// ---- client IP ------------------------------------------------------------

type ctxKey int

const clientIPKey ctxKey = 0

// TrustedProxies is a set of addresses/prefixes allowed to set
// X-Forwarded-For (Apache on 127.0.0.1 in production).
type TrustedProxies []netip.Prefix

// ParseTrustedProxies parses "127.0.0.1, ::1, 10.0.0.0/8".
func ParseTrustedProxies(v string) (TrustedProxies, error) {
	var out TrustedProxies
	for _, f := range strings.Split(v, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if strings.Contains(f, "/") {
			p, err := netip.ParsePrefix(f)
			if err != nil {
				return nil, err
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(f)
		if err != nil {
			return nil, err
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

func (t TrustedProxies) contains(a netip.Addr) bool {
	a = a.Unmap()
	for _, p := range t {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// ClientIP stores the real client address in the request context. When the
// peer is a trusted proxy, it walks X-Forwarded-For from the right and takes
// the first address that is not itself a trusted proxy; otherwise the header
// is ignored, so clients cannot spoof it.
func ClientIP(trusted TrustedProxies) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolveIP(r, trusted)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey, ip)))
		})
	}
}

func resolveIP(r *http.Request, trusted TrustedProxies) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	if !trusted.contains(peer) {
		return peer.String()
	}
	var hops []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(v, ",")...)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break
		}
		a = a.Unmap()
		if !trusted.contains(a) {
			return a.String()
		}
	}
	return peer.String()
}

// IP returns the client IP stored by ClientIP (or RemoteAddr's host).
func IP(r *http.Request) string {
	if v, ok := r.Context().Value(clientIPKey).(string); ok && v != "" {
		return v
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---- logging ----------------------------------------------------------------

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Logging writes one structured line per request.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic", "path", r.URL.Path, "panic", rec)
					if sw.status == 0 {
						http.Error(sw, "internal server error", http.StatusInternalServerError)
					}
				}
				log.Info("request",
					"method", r.Method, "path", r.URL.Path, "status", sw.status,
					"bytes", sw.bytes, "dur_ms", time.Since(start).Milliseconds(), "ip", IP(r))
			}()
			next.ServeHTTP(sw, r)
		})
	}
}

// ---- gzip -------------------------------------------------------------------

var gzPool = sync.Pool{New: func() any { w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression); return w }}

func compressible(ct string) bool {
	ct = strings.ToLower(ct)
	for _, p := range []string{"text/", "application/json", "application/ld+json", "application/xml", "application/javascript", "text/javascript", "image/svg+xml", "application/manifest+json"} {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

type gzipWriter struct {
	http.ResponseWriter
	gz      *gzip.Writer
	decided bool
}

func (w *gzipWriter) decide(status int) {
	if w.decided {
		return
	}
	w.decided = true
	h := w.Header()
	if h.Get("Content-Encoding") != "" || h.Get("Content-Range") != "" || status == http.StatusPartialContent ||
		status == http.StatusNoContent || status == http.StatusNotModified || !compressible(h.Get("Content-Type")) {
		return
	}
	h.Del("Content-Length")
	h.Set("Content-Encoding", "gzip")
	g := gzPool.Get().(*gzip.Writer)
	g.Reset(w.ResponseWriter)
	w.gz = g
}

func (w *gzipWriter) WriteHeader(code int) {
	w.decide(code)
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	if !w.decided {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}
		w.WriteHeader(http.StatusOK)
	}
	if w.gz != nil {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *gzipWriter) close() {
	if w.gz != nil {
		_ = w.gz.Close()
		gzPool.Put(w.gz)
		w.gz = nil
	}
}

func (w *gzipWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *gzipWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijack not supported")
}

// Gzip compresses text-like responses for clients that accept it.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")
		if r.Method == http.MethodHead || !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.close()
		next.ServeHTTP(gw, r)
	})
}

func acceptsGzip(v string) bool {
	for _, part := range strings.Split(v, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(name), "gzip") {
			q := 1.0
			if p := strings.TrimSpace(params); strings.HasPrefix(p, "q=") {
				if v, err := strconv.ParseFloat(p[2:], 64); err == nil {
					q = v
				}
			}
			return q > 0
		}
	}
	return false
}
