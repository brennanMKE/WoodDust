package contrib

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/brennanMKE/WoodDust/internal/model"
)

// Kinds of submission (contract §6).
var Kinds = []string{"new_species", "correction", "new_source", "reaction_report"}

// Statuses in workflow order, plus spam (PLAN §6).
var Statuses = []string{"new", "triaging", "needs_info", "accepted", "rejected", "exported", "merged", "spam"}

// closedStatuses delete the submitter's email.
var closedStatuses = map[string]bool{"rejected": true, "merged": true, "spam": true}

func validKind(k string) bool   { return contains(Kinds, k) }
func validStatus(s string) bool { return contains(Statuses, s) }

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// LeadsOnlyDomains are sources we use only to find leads, never to cite
// (handoff §2 "Leads only", CONTRIBUTING.md). Keep the two lists in sync.
// A URL on one of these domains, or any subdomain, is accepted with a warning.
var LeadsOnlyDomains = []string{
	"wood-database.com",
	"haz-map.com",
	"scrollsawvillage.com",
	"woodbin.com",
	"wooduchoose.com",
	"finewoodworking.com",
	"woodturner.org", // AAW
	"reddit.com",
	"lumberjocks.com",
	"sawmillcreek.org",
	"woodworkingtalk.com",
	"woodnet.net",
	"woodweb.com",
	"ukworkshop.co.uk",
	"wikipedia.org",
}

// leadsWarning is shown for a leads-only URL.
const leadsWarning = "This site is a lead, not a citable source. Please also give the primary study or agency page it relies on (DOI, PMID or an official URL)."

// Limits on request size (PLAN §6: size and link-count limits).
const (
	maxDetails     = 4000
	minDetails     = 20
	maxName        = 200
	maxCitations   = 10
	maxCitationTxt = 500
	maxURL         = 2000
	maxEmail       = 254
	maxLinksInText = 5
)

var (
	doiRe   = regexp.MustCompile(`^10\.\d{4,9}/\S+$`)
	pmidRe  = regexp.MustCompile(`^\d{1,9}$`)
	emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	linkRe  = regexp.MustCompile(`(?i)https?://|www\.`)
)

// Citation is one reference in a submission.
type Citation struct {
	DOI  string `json:"doi"`
	PMID string `json:"pmid"`
	URL  string `json:"url"`
	Text string `json:"text"`
}

// Request is the POST /api/v1/submissions body.
type Request struct {
	Kind          string     `json:"kind"`
	TargetSlug    string     `json:"target_slug"`
	SpeciesName   string     `json:"species_name"`
	BotanicalName string     `json:"botanical_name"`
	HazardType    string     `json:"hazard_type"`
	Details       string     `json:"details"`
	Citations     []Citation `json:"citations"`
	Email         string     `json:"email"`
	FormToken     string     `json:"form_token"`
	Website       string     `json:"website"` // honeypot
}

// CheckURL reports whether u is an acceptable http(s) URL and any warnings.
func CheckURL(raw string) (ok bool, warnings []string) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(raw) > maxURL {
		return false, nil
	}
	if d := leadsDomain(u.Hostname()); d != "" {
		warnings = append(warnings, d+": "+leadsWarning)
	}
	return true, warnings
}

func leadsDomain(host string) string {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	for _, d := range LeadsOnlyDomains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return d
		}
	}
	return ""
}

// Validate normalizes req in place and returns field errors and warnings.
// speciesExists reports whether a slug is published.
func Validate(req *Request, speciesExists func(string) bool) (fields map[string]string, warnings []string) {
	fields = map[string]string{}
	trim := func(s *string) { *s = strings.TrimSpace(*s) }
	trim(&req.Kind)
	trim(&req.TargetSlug)
	trim(&req.SpeciesName)
	trim(&req.BotanicalName)
	trim(&req.HazardType)
	trim(&req.Details)
	trim(&req.Email)

	if !validKind(req.Kind) {
		fields["kind"] = "Choose what kind of suggestion this is."
	}
	if req.TargetSlug != "" && !speciesExists(req.TargetSlug) {
		fields["target_slug"] = "That species is not listed."
	}
	switch req.Kind {
	case "new_species":
		if req.SpeciesName == "" {
			fields["species_name"] = "Give the wood's common name."
		}
		req.TargetSlug = ""
	case "correction", "new_source":
		if req.TargetSlug == "" {
			fields["target_slug"] = "Choose the species this is about."
		}
	case "reaction_report":
		if req.TargetSlug == "" && req.SpeciesName == "" {
			fields["species_name"] = "Choose a listed species or name the wood."
		}
	}
	if utf8.RuneCountInString(req.SpeciesName) > maxName {
		fields["species_name"] = "Too long."
	}
	if utf8.RuneCountInString(req.BotanicalName) > maxName {
		fields["botanical_name"] = "Too long."
	}
	if req.HazardType != "" {
		if _, ok := model.HazardTypeNames[req.HazardType]; !ok {
			fields["hazard_type"] = "Unknown hazard type."
		}
	}
	n := utf8.RuneCountInString(req.Details)
	switch {
	case n < minDetails:
		fields["details"] = "Please describe it in at least 20 characters."
	case n > maxDetails:
		fields["details"] = "Please keep it under 4000 characters."
	case len(linkRe.FindAllStringIndex(req.Details, -1)) > maxLinksInText:
		fields["details"] = "Too many links. Put sources in the citations instead."
	}
	if req.Email != "" && (len(req.Email) > maxEmail || !emailRe.MatchString(req.Email)) {
		fields["email"] = "That does not look like an email address."
	}

	// Citations: drop blank rows, then check each.
	var cites []Citation
	for _, c := range req.Citations {
		c.DOI, c.PMID, c.URL, c.Text = strings.TrimSpace(c.DOI), strings.TrimSpace(c.PMID), strings.TrimSpace(c.URL), strings.TrimSpace(c.Text)
		if c.DOI == "" && c.PMID == "" && c.URL == "" && c.Text == "" {
			continue
		}
		c.DOI = strings.TrimPrefix(strings.TrimPrefix(c.DOI, "https://doi.org/"), "doi:")
		cites = append(cites, c)
	}
	req.Citations = cites
	if len(cites) > maxCitations {
		fields["citations"] = "At most 10 citations."
	}
	needCite := req.Kind == "new_species" || req.Kind == "correction" || req.Kind == "new_source"
	if needCite && len(cites) == 0 {
		fields["citations"] = "Add at least one source with a DOI, PMID or URL."
	}
	for _, c := range cites {
		var msg string
		switch {
		case c.DOI == "" && c.PMID == "" && c.URL == "":
			msg = "Each source needs a DOI, PMID or URL."
		case c.DOI != "" && !doiRe.MatchString(c.DOI):
			msg = "A DOI looks like 10.1234/abcd."
		case c.PMID != "" && !pmidRe.MatchString(c.PMID):
			msg = "A PMID is digits only."
		case utf8.RuneCountInString(c.Text) > maxCitationTxt:
			msg = "Citation text is too long."
		}
		if msg == "" && c.URL != "" {
			ok, w := CheckURL(c.URL)
			if !ok {
				msg = "URLs must start with http:// or https://."
			}
			warnings = append(warnings, w...)
		}
		if msg != "" {
			fields["citations"] = msg
			break
		}
	}
	if warnings == nil {
		warnings = []string{}
	}
	return fields, warnings
}
