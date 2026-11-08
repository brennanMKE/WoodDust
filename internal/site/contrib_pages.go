package site

import (
	"net/http"
	"net/url"

	"github.com/brennanMKE/WoodDust/internal/contrib"
)

// submitData is the submit route's initial data (contract §2).
type submitData struct {
	FormToken   string          `json:"form_token"`
	Kinds       []string        `json:"kinds"`
	Target      *speciesSummary `json:"target"`
	PrefillName string          `json:"prefill_name"`
	Available   bool            `json:"available"`
}

type submitBody struct {
	Target      *speciesSummary
	PrefillName string
	Available   bool
}

// pageSubmit renders /submit and /submit/{slug} with a fresh form token.
func (s *Server) pageSubmit(w http.ResponseWriter, r *http.Request) {
	data := submitData{Kinds: contrib.Kinds, PrefillName: truncate(r.URL.Query().Get("name"), 200)}
	if slug := r.PathValue("slug"); slug != "" {
		sp, ok, err := s.cat.GetSpecies(r.Context(), slug)
		if err != nil {
			s.renderError(w, r, err)
			return
		}
		if !ok {
			s.renderNotFound(w, r, "", nil)
			return
		}
		sum := s.summary(sp)
		data.Target = &sum
	}
	if s.contrib != nil && s.contrib.Store != nil {
		data.FormToken = s.contrib.Tokens.Issue()
		data.Available = true
	}
	canonical := "/submit"
	if data.Target != nil {
		canonical += "/" + data.Target.Slug
	}
	head := headData{
		Title:       "Suggest a wood or a correction | " + s.siteName,
		Description: "Suggest a missing wood, a correction, or a better source for " + s.siteName + ".",
		Canonical:   s.absURL(canonical),
		NoIndex:     true,
	}
	w.Header().Set("Cache-Control", "no-store")
	s.render(w, r, view{Route: "submit", Head: head, Template: "body_submit",
		Body: submitBody{Target: data.Target, PrefillName: data.PrefillName, Available: data.Available}, Data: data})
}

// pageSubmissionStatus renders /submit/status/{token}; unknown tokens are 404.
func (s *Server) pageSubmissionStatus(w http.ResponseWriter, r *http.Request) {
	if s.contrib == nil {
		s.renderNotFound(w, r, "", nil)
		return
	}
	v, ok, err := s.contrib.Status(r.Context(), r.PathValue("token"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	if !ok {
		s.renderNotFound(w, r, "", nil)
		return
	}
	head := headData{
		Title:       "Suggestion " + v.Reference + " | " + s.siteName,
		Description: "The status of your suggestion.",
		NoIndex:     true,
	}
	// The token is a bearer secret: keep it out of Referer headers.
	w.Header().Set("Referrer-Policy", "no-referrer")
	s.render(w, r, view{Route: "submission_status", Head: head, Template: "body_submission_status", Body: v, Data: v})
}

// pageAdmin renders the admin console shell. Without a session it sends the
// browser to sign in; with a non-admin session it is 403; with sign-in
// disabled it is 503 (contract §2, §7).
func (s *Server) pageAdmin(w http.ResponseWriter, r *http.Request) {
	if s.contrib == nil {
		s.renderNotFound(w, r, "", nil)
		return
	}
	u, res, msg := s.contrib.Auth.PageUser(r)
	head := headData{Title: "Admin | " + s.siteName, Description: "Admin console.", NoIndex: true}
	switch res {
	case contrib.PageSignIn:
		http.Redirect(w, r, "/auth/login?return="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
		return
	case contrib.PageForbidden, contrib.PageUnavailable:
		status := http.StatusForbidden
		if res == contrib.PageUnavailable {
			status = http.StatusServiceUnavailable
		}
		w.Header().Set("Cache-Control", "no-store")
		s.render(w, r, view{Status: status, Route: "admin", Head: head, Template: "body_admin",
			Body: struct{ Message string }{msg}, Data: map[string]any{"user": nil, "error": msg}})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	s.render(w, r, view{Route: "admin", Head: head, Template: "body_admin",
		Body: struct{ Message string }{""}, Data: map[string]any{"user": contrib.ViewOf(u)}})
}
