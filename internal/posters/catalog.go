// Package posters holds the shop posters (docs/posters.md): the catalog,
// stock QR codes, the ShortLinks client that mints a link per customized
// poster, the public link endpoint and the admin Settings card for the
// ShortLinks key.
package posters

// StockBase is the public host of the stock poster links. The stock QR codes
// are generated from it at build time (cmd/posterbuild), so it is fixed here
// rather than read from settings.
const StockBase = "https://go.dustalmanac.com"

// Poster is one sheet in the set.
type Poster struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slogan      string `json:"slogan"`
	Destination string `json:"destination"` // site path the QR code opens
	// Alias is the stock link's custom key on go.dustalmanac.com. ShortLinks
	// aliases are 1–12 letters and digits, so no hyphens.
	Alias string `json:"-"`
	// Published: shown on /posters and accepted by the link endpoint.
	Published bool `json:"-"`
}

// StockURL is the stock link printed on the downloadable sheets
// (utm_content=stock-v1, created by hand in ShortLinks).
func (p Poster) StockURL() string { return StockBase + "/u/" + p.Alias }

// Catalog is the set, in display order. Slogans are from docs/posters.md.
var Catalog = []Poster{
	{ID: "shave", Title: "Shave", Slogan: "Paul shaves for a reason.", Destination: "/protection/respirators", Alias: "shave", Published: true},
	{ID: "sunbeam", Title: "Sunbeam", Slogan: "You can't see what you're breathing.", Destination: "/risks", Alias: "sunbeam", Published: true},
	{ID: "tall-tale", Title: "Tall Tale", Slogan: "Catch it at the tool.", Destination: "/protection/dust-collection", Alias: "talltale"},
	{ID: "sweep", Title: "Sweep", Slogan: "No brooms.", Destination: "/protection/cleanup", Alias: "sweep"},
	{ID: "almanac", Title: "Almanac", Slogan: `No wood is "safe." Know yours.`, Destination: "/", Alias: "almanac"},
}

// Get returns a poster by id, published or not.
func Get(id string) (Poster, bool) {
	for _, p := range Catalog {
		if p.ID == id {
			return p, true
		}
	}
	return Poster{}, false
}

// Published returns the posters shown on the site.
func Published() []Poster {
	var out []Poster
	for _, p := range Catalog {
		if p.Published {
			out = append(out, p)
		}
	}
	return out
}
