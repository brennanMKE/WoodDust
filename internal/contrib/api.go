package contrib

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/brennanMKE/WoodDust/internal/model"
)

// SpeciesLookup finds a published species by slug.
type SpeciesLookup func(ctx context.Context, slug string) (model.Species, bool, error)

// API serves the submission, admin and API-key endpoints under /api/v1.
type API struct {
	Store    *Store // nil: no database (every endpoint here answers 503)
	Auth     *Auth
	Tokens   *FormTokens
	IPs      *IPHasher
	Limiter  *SubmitLimiter
	Species  SpeciesLookup
	ClientIP func(*http.Request) string
	Log      *slog.Logger
}

// Register mounts the endpoints on mux.
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/submissions", a.create)
	mux.HandleFunc("GET /api/v1/submissions/form-token", a.formToken)
	mux.HandleFunc("GET /api/v1/submissions/check-url", a.checkURL)
	mux.HandleFunc("GET /api/v1/submissions", a.list)
	mux.HandleFunc("GET /api/v1/submissions/{id}", a.get)
	mux.HandleFunc("PATCH /api/v1/submissions/{id}", a.patch)
	// One pattern for both, because /status/{token} and /{id}/export overlap.
	mux.HandleFunc("GET /api/v1/submissions/{a}/{b}", a.twoSegments)
	mux.HandleFunc("GET /api/v1/stats", a.stats)
	mux.HandleFunc("GET /api/v1/me", a.me)
	mux.HandleFunc("GET /api/v1/admin/api-keys", a.listKeys)
	mux.HandleFunc("POST /api/v1/admin/api-keys", a.createKey)
	mux.HandleFunc("DELETE /api/v1/admin/api-keys/{prefix}", a.revokeKey)
}

// ---- helpers ----------------------------------------------------------------

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

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]errBody{"error": {Code: code, Message: msg}})
}

func (a *API) internal(w http.ResponseWriter, r *http.Request, err error) {
	a.Log.Error("contrib handler", "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
}

func (a *API) needStore(w http.ResponseWriter) bool {
	if a.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", msgNoDatabase)
		return false
	}
	return true
}

func (a *API) authorize(w http.ResponseWriter, r *http.Request, scope string) (Principal, bool) {
	p, err := a.Auth.Authorize(r, scope)
	if err != nil {
		writeAuthError(w, err)
		return p, false
	}
	return p, true
}

func decode(w http.ResponseWriter, r *http.Request, v any, max int64) bool {
	body := http.MaxBytesReader(w, r.Body, max)
	if err := json.NewDecoder(body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "The request body must be JSON.")
		return false
	}
	_, _ = io.Copy(io.Discard, body)
	return true
}

func (a *API) speciesExists(ctx context.Context) func(string) bool {
	return func(slug string) bool {
		_, ok, err := a.Species(ctx, slug)
		return err == nil && ok
	}
}

// ---- public: submissions -------------------------------------------------------

// CreateResponse is the 201 body.
type CreateResponse struct {
	ID        int64    `json:"id"`
	Reference string   `json:"reference"`
	StatusURL string   `json:"status_url"`
	Warnings  []string `json:"warnings"`
}

func (a *API) create(w http.ResponseWriter, r *http.Request) {
	if !a.needStore(w) {
		return
	}
	ip := a.ClientIP(r)
	if ok, wait := a.Limiter.Check(ip); !ok {
		secs := int(math.Ceil(wait.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(max(secs, 1)))
		writeError(w, http.StatusTooManyRequests, "rate_limited",
			"We have received a lot of suggestions. Please try again later.")
		return
	}
	var req Request
	if !decode(w, r, &req, 64<<10) {
		return
	}

	spam := ""
	switch err := a.Tokens.Check(req.FormToken); {
	case strings.TrimSpace(req.Website) != "":
		spam = "honeypot"
	case errors.Is(err, ErrTokenTooFast):
		spam = "too_fast"
	case err != nil:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]errBody{"error": {
			Code: "invalid", Message: "This form has expired. Reload the page and try again.",
			Fields: map[string]string{"form_token": "Expired or invalid. Reload the page."}}})
		return
	}

	fields, warnings := Validate(&req, a.speciesExists(r.Context()))
	if spam == "" && len(fields) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]errBody{"error": {
			Code: "invalid", Message: "Please fix the highlighted fields.", Fields: fields}})
		return
	}
	if spam != "" {
		coerceSpam(&req, fields)
	}

	token, hash := NewStatusToken()
	sub, err := a.Store.CreateSubmission(r.Context(), NewSubmission{
		Request: req, Warnings: warnings, StatusHash: hash, IPHash: a.IPs.Hash(ip), Spam: spam,
	})
	if err != nil {
		a.internal(w, r, err)
		return
	}
	a.Limiter.Record(ip)
	if spam != "" {
		a.Log.Info("submission stored as spam", "reference", sub.Reference, "reason", spam)
	}
	// Spam gets the same answer as a real submission (its status page shows
	// "new"), so bots learn nothing.
	writeJSON(w, http.StatusCreated, CreateResponse{
		ID: sub.ID, Reference: sub.Reference, StatusURL: "/submit/status/" + token, Warnings: warnings,
	})
}

// coerceSpam makes a spam request storable whatever it contained.
func coerceSpam(req *Request, fields map[string]string) {
	if !validKind(req.Kind) {
		req.Kind = "new_species"
	}
	if _, bad := fields["target_slug"]; bad {
		req.TargetSlug = ""
	}
	if _, bad := fields["hazard_type"]; bad {
		req.HazardType = ""
	}
	clip := func(s string, n int) string {
		if utf8.RuneCountInString(s) > n {
			return string([]rune(s)[:n])
		}
		return s
	}
	req.Details = clip(req.Details, maxDetails)
	req.SpeciesName = clip(req.SpeciesName, maxName)
	req.BotanicalName = clip(req.BotanicalName, maxName)
	if len(req.Citations) > maxCitations {
		req.Citations = req.Citations[:maxCitations]
	}
	req.Email = ""
}

func (a *API) formToken(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"form_token": a.Tokens.Issue()})
}

func (a *API) checkURL(w http.ResponseWriter, r *http.Request) {
	_, warnings := CheckURL(r.URL.Query().Get("url"))
	if warnings == nil {
		warnings = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"warnings": warnings, "leads_only": len(warnings) > 0})
}

// StatusView is GET /api/v1/submissions/status/{token}.
type StatusView struct {
	Reference string    `json:"reference"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Notes     []Note    `json:"notes"`
}

// Status looks up the submitter's view by status token. ok is false for an
// unknown token.
func (a *API) Status(ctx context.Context, token string) (StatusView, bool, error) {
	if a.Store == nil || len(token) < 40 || len(token) > 64 {
		return StatusView{}, false, nil
	}
	s, notes, err := a.Store.SubmissionByToken(ctx, HashToken(token))
	if errors.Is(err, ErrNotFound) {
		return StatusView{}, false, nil
	}
	if err != nil {
		return StatusView{}, false, err
	}
	status := s.Status
	if status == "spam" {
		status = "new"
	}
	return StatusView{Reference: s.Reference, Kind: s.Kind, Status: status,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt, Notes: notes}, true, nil
}

func (a *API) twoSegments(w http.ResponseWriter, r *http.Request) {
	first, second := r.PathValue("a"), r.PathValue("b")
	switch {
	case first == "status":
		if !a.needStore(w) {
			return
		}
		v, ok, err := a.Status(r.Context(), second)
		if err != nil {
			a.internal(w, r, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "No submission with that status link.")
			return
		}
		writeJSON(w, http.StatusOK, v)
	case second == "export":
		a.export(w, r, first)
	default:
		writeError(w, http.StatusNotFound, "not_found", "No such API endpoint.")
	}
}

// ---- admin: submissions ---------------------------------------------------------

func (a *API) list(w http.ResponseWriter, r *http.Request) {
	if !a.needStore(w) {
		return
	}
	if _, ok := a.authorize(w, r, ScopeSubmissionsRead); !ok {
		return
	}
	q := r.URL.Query()
	f := ListFilter{Status: q.Get("status"), Kind: q.Get("kind"), Limit: 50}
	if f.Status != "" && !validStatus(f.Status) {
		writeError(w, http.StatusBadRequest, "bad_request", "Unknown status.")
		return
	}
	if f.Kind != "" && !validKind(f.Kind) {
		writeError(w, http.StatusBadRequest, "bad_request", "Unknown kind.")
		return
	}
	if v := q.Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			if t, err = time.Parse("2006-01-02", v); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "since must be RFC 3339 or YYYY-MM-DD.")
				return
			}
		}
		f.Since = &t
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "bad_request", "limit must be a positive integer.")
			return
		}
		f.Limit = min(n, 200)
	}
	if v := q.Get("cursor"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "bad_request", "Bad cursor.")
			return
		}
		f.Cursor = n
	}
	subs, next, err := a.Store.ListSubmissions(r.Context(), f)
	if err != nil {
		a.internal(w, r, err)
		return
	}
	var cursor *string
	if next > 0 {
		c := strconv.FormatInt(next, 10)
		cursor = &c
	}
	writeJSON(w, http.StatusOK, map[string]any{"submissions": subs, "next_cursor": cursor})
}

func parseID(w http.ResponseWriter, raw string) (int64, bool) {
	raw = strings.TrimPrefix(strings.ToUpper(raw), "DA-")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusNotFound, "not_found", "No such submission.")
		return 0, false
	}
	return id, true
}

func (a *API) get(w http.ResponseWriter, r *http.Request) {
	if !a.needStore(w) {
		return
	}
	if _, ok := a.authorize(w, r, ScopeSubmissionsRead); !ok {
		return
	}
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	s, evs, err := a.Store.GetSubmission(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "No such submission.")
		return
	}
	if err != nil {
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Submission
		Events []Event `json:"events"`
	}{s, evs})
}

func (a *API) patch(w http.ResponseWriter, r *http.Request) {
	if !a.needStore(w) {
		return
	}
	p, ok := a.authorize(w, r, ScopeSubmissionsWrite)
	if !ok {
		return
	}
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var u Update
	if !decode(w, r, &u, 16<<10) {
		return
	}
	fields := map[string]string{}
	if u.Status != nil && !validStatus(*u.Status) {
		fields["status"] = "Unknown status."
	}
	if u.Note != nil {
		u.Note.Text = strings.TrimSpace(u.Note.Text)
		if n := utf8.RuneCountInString(u.Note.Text); n == 0 || n > maxDetails {
			fields["note"] = "A note needs 1 to 4000 characters."
		}
	}
	if u.Status == nil && u.Note == nil {
		fields["status"] = "Send a status, a note, or both."
	}
	if len(fields) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]errBody{"error": {Code: "invalid", Message: "Invalid update.", Fields: fields}})
		return
	}
	s, err := a.Store.UpdateSubmission(r.Context(), id, p.Actor(), u)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "No such submission.")
		return
	}
	if err != nil {
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (a *API) stats(w http.ResponseWriter, r *http.Request) {
	if !a.needStore(w) {
		return
	}
	if _, ok := a.authorize(w, r, ScopeStatsRead); !ok {
		return
	}
	s, err := a.Store.Stats(r.Context())
	if err != nil {
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// ---- session-only ---------------------------------------------------------------------

// MeView is the user object in GET /api/v1/me and the admin pages' data.
type MeView struct {
	DisplayName string `json:"display_name"`
	IsAdmin     bool   `json:"is_admin"`
}

// ViewOf returns the public view of a user.
func ViewOf(u *User) MeView {
	name := u.DisplayName
	if name == "" {
		name = "Owner"
	}
	return MeView{DisplayName: name, IsAdmin: u.IsAdmin}
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	u, err := a.Auth.Me(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]MeView{"user": ViewOf(u)})
}

func (a *API) listKeys(w http.ResponseWriter, r *http.Request) {
	if !a.needStore(w) {
		return
	}
	if _, ok := a.authorize(w, r, ""); !ok {
		return
	}
	keys, err := a.Store.APIKeys(r.Context())
	if err != nil {
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

// KeyRequest is POST /api/v1/admin/api-keys.
type KeyRequest struct {
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// CheckKeyRequest validates a key request.
func CheckKeyRequest(k *KeyRequest) map[string]string {
	fields := map[string]string{}
	k.Name = strings.TrimSpace(k.Name)
	if k.Name == "" || utf8.RuneCountInString(k.Name) > 100 {
		fields["name"] = "Give the key a name (up to 100 characters)."
	}
	if len(k.Scopes) == 0 {
		fields["scopes"] = "Choose at least one scope."
	}
	for _, s := range k.Scopes {
		if !ValidScope(s) {
			fields["scopes"] = "Unknown scope " + s + "."
		}
	}
	if k.ExpiresAt != nil && !k.ExpiresAt.After(time.Now()) {
		fields["expires_at"] = "Must be in the future."
	}
	return fields
}

func (a *API) createKey(w http.ResponseWriter, r *http.Request) {
	if !a.needStore(w) {
		return
	}
	p, ok := a.authorize(w, r, "")
	if !ok {
		return
	}
	var req KeyRequest
	if !decode(w, r, &req, 4<<10) {
		return
	}
	if fields := CheckKeyRequest(&req); len(fields) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]errBody{"error": {Code: "invalid", Message: "Invalid key request.", Fields: fields}})
		return
	}
	k, key, err := a.Store.CreateAPIKey(r.Context(), req.Name, req.Scopes, req.ExpiresAt, &p.User.ID, p.Actor())
	if err != nil {
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		APIKey
		Key string `json:"key"`
	}{k, key})
}

func (a *API) revokeKey(w http.ResponseWriter, r *http.Request) {
	if !a.needStore(w) {
		return
	}
	p, ok := a.authorize(w, r, "")
	if !ok {
		return
	}
	err := a.Store.RevokeAPIKey(r.Context(), r.PathValue("prefix"), p.Actor())
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "No active key with that prefix.")
		return
	}
	if err != nil {
		a.internal(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
