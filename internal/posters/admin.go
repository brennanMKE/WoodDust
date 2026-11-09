package posters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/brennanMKE/WoodDust/internal/contrib"
	"github.com/brennanMKE/WoodDust/internal/settings"
)

// Settings card statuses (contract §7).
const (
	StatusNotConfigured = "not_configured"
	StatusUntested      = "untested"
	StatusOK            = "ok"
	StatusFailing       = "failing"
)

// ShortLinksView is the admin Settings card. It never carries the key.
type ShortLinksView struct {
	Status     string        `json:"status"`
	Encryption bool          `json:"encryption_available"` // SETTINGS_KEY is set
	Key        settings.Info `json:"key"`
	APIURL     settings.Info `json:"api_url"`
	PublicBase settings.Info `json:"public_base"`
	Campaign   settings.Info `json:"campaign"`
}

func (s *Service) view(r *http.Request) (ShortLinksView, error) {
	ctx := r.Context()
	v := ShortLinksView{Encryption: s.Settings.CanSeal()}
	var err error
	for _, f := range []struct {
		key string
		dst *settings.Info
	}{{settings.ShortLinksAPIKey, &v.Key}, {settings.ShortLinksAPIURL, &v.APIURL},
		{settings.ShortLinksPublicBase, &v.PublicBase}, {settings.ShortLinksCampaign, &v.Campaign}} {
		if *f.dst, err = s.Settings.Info(ctx, f.key); err != nil {
			return v, err
		}
	}
	switch {
	case !v.Encryption || !v.Key.Set:
		v.Status = StatusNotConfigured
	case v.Key.CheckedAt == nil || v.Key.CheckOK == nil:
		v.Status = StatusUntested
	case !*v.Key.CheckOK:
		v.Status = StatusFailing
	default:
		v.Status = StatusOK
	}
	return v, nil
}

// admin authorizes a session-only admin request.
func (s *Service) admin(w http.ResponseWriter, r *http.Request) (actor string, ok bool) {
	if s.Settings == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable",
			"This server runs without a database (DATABASE_URL is not set), so settings are unavailable.", nil)
		return "", false
	}
	p, err := s.Auth.Authorize(r, "")
	if err != nil {
		contrib.WriteAuthError(w, err)
		return "", false
	}
	return p.Actor(), true
}

func (s *Service) writeView(w http.ResponseWriter, r *http.Request) {
	v, err := s.view(r)
	if err != nil {
		s.Log.Error("settings view", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]ShortLinksView{"shortlinks": v})
}

func (s *Service) adminGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.admin(w, r); ok {
		s.writeView(w, r)
	}
}

func (s *Service) adminSet(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.admin(w, r)
	if !ok {
		return
	}
	key := r.PathValue("key")
	if _, ok := settings.Lookup(key); !ok {
		writeError(w, http.StatusNotFound, "not_found", "No such setting.", nil)
		return
	}
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<10)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "The request body must be JSON.", nil)
		return
	}
	_, err := s.Settings.Set(r.Context(), key, body.Value, actor)
	switch {
	case errors.Is(err, settings.ErrInvalid):
		msg := strings.TrimPrefix(err.Error(), settings.ErrInvalid.Error()+": ")
		writeError(w, http.StatusUnprocessableEntity, "invalid", msg, map[string]string{"value": msg})
		return
	case errors.Is(err, settings.ErrNoCipher):
		writeError(w, http.StatusConflict, "no_settings_key", err.Error(), nil)
		return
	case err != nil:
		s.Log.Error("settings set", "key", key, "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.", nil)
		return
	}
	s.Log.Info("setting changed", "key", key, "actor", actor)
	s.writeView(w, r)
}

func (s *Service) adminClear(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.admin(w, r)
	if !ok {
		return
	}
	key := r.PathValue("key")
	err := s.Settings.Clear(r.Context(), key, actor)
	if errors.Is(err, settings.ErrUnknownKey) {
		writeError(w, http.StatusNotFound, "not_found", "No such setting.", nil)
		return
	}
	if err != nil {
		s.Log.Error("settings clear", "key", key, "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.", nil)
		return
	}
	s.Log.Info("setting cleared", "key", key, "actor", actor)
	s.writeView(w, r)
}

// adminTest runs Test connection: the token works and is not read-only, its
// base_url matches shortlinks_public_base, and the campaign exists. The
// result is stored, so the card and the dashboard banner show it.
func (s *Service) adminTest(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.admin(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	msg, configured, err := s.Check(ctx)
	if err != nil {
		s.Log.Error("settings test", "err", err)
		writeError(w, http.StatusConflict, "unreadable", "The stored key can't be read: "+err.Error(), nil)
		return
	}
	if !configured {
		writeError(w, http.StatusConflict, "not_configured", "Set the ShortLinks API key first.", nil)
		return
	}
	if err := s.Settings.RecordCheck(ctx, settings.ShortLinksAPIKey, msg == "", msg, actor); err != nil {
		s.Log.Error("settings test: recording", "err", err)
	}
	s.Log.Info("shortlinks connection test", "ok", msg == "", "actor", actor)
	s.writeView(w, r)
}

// Check runs the connection test and returns "" when everything is in
// order, or what is wrong in plain words.
func (s *Service) Check(ctx context.Context) (msg string, configured bool, err error) {
	c, publicBase, campaign, configured, err := s.shortLinks(ctx)
	if err != nil || !configured {
		return "", configured, err
	}
	me, err := c.Me(ctx)
	switch {
	case err != nil:
		return Plain(err), true, nil
	case me.APIToken == nil:
		return "ShortLinks didn't recognize this as an API token (no api_token in /api/me).", true, nil
	case me.APIToken.ReadOnly:
		return fmt.Sprintf("The token “%s” is read-only, so it can't create links. Create one without “Read-only” ticked.", me.APIToken.Name), true, nil
	case strings.TrimRight(me.BaseURL, "/") != strings.TrimRight(publicBase, "/"):
		return fmt.Sprintf("ShortLinks says its links are under %s, but shortlinks_public_base is %s. The printed link and the QR code would differ.", me.BaseURL, publicBase), true, nil
	}
	if err := c.Campaign(ctx, campaign); err != nil {
		var ae *APIError
		if errors.As(err, &ae) && ae.Status == http.StatusNotFound {
			return fmt.Sprintf("The token works, but there is no campaign “%s” on that account. Create it in ShortLinks.", campaign), true, nil
		}
		return Plain(err), true, nil
	}
	return "", true, nil
}
