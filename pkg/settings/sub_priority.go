package settings

// SubPriority decides which downloaded subtitle is kept when several fit.
type SubPriority struct {
	SiteOrder    []string `json:"site_order"`    // subtitle sites, most preferred first
	Language     string   `json:"language"`      // "bilingual" (default), "chs" or "cht"
	MatchRelease *bool    `json:"match_release"` // prefer subs whose name matches the video's release tags; nil = true
}

const (
	SubLangBilingual = "bilingual"
	SubLangChs       = "chs"
	SubLangCht       = "cht"
)

// DefaultSiteOrder is the order used before this setting existed, minus dead sites.
var DefaultSiteOrder = []string{"assrt", "shooter", "xunlei"}

func NewSubPriority() *SubPriority {
	on := true
	return &SubPriority{SiteOrder: append([]string{}, DefaultSiteOrder...), Language: SubLangBilingual, MatchRelease: &on}
}

// Check fills defaults for configs saved before this setting existed and keeps the site
// list to known sites: unknown names are dropped, missing ones appended in default order.
func (p *SubPriority) Check() {
	known := map[string]bool{}
	for _, s := range DefaultSiteOrder {
		known[s] = true
	}
	seen := map[string]bool{}
	order := make([]string, 0, len(DefaultSiteOrder))
	for _, s := range p.SiteOrder {
		if known[s] && !seen[s] {
			order = append(order, s)
			seen[s] = true
		}
	}
	for _, s := range DefaultSiteOrder {
		if !seen[s] {
			order = append(order, s)
		}
	}
	p.SiteOrder = order
	if p.Language != SubLangChs && p.Language != SubLangCht {
		p.Language = SubLangBilingual
	}
	if p.MatchRelease == nil {
		on := true
		p.MatchRelease = &on
	}
}
