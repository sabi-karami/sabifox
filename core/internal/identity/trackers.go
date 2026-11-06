package identity

// Tracker blocking: the default routing lists (Chocolate4U's) carry
// curated tags of advertising and tracking domains (Persian and foreign),
// malware, phishing, and in-browser crypto miners. Trackers follow a
// person from site to site whatever the address; blocking them in the
// proxy works for every app, the phone's too.

// TrackerSiteTags are the geosite tags blocked, TrackerIPTags the geoip
// ones.
var (
	TrackerSiteTags = []string{"category-ads-all", "malware", "phishing", "cryptominers"}
	TrackerIPTags   = []string{"malware", "phishing"}
)

// TrackerRules are the block rules for the tags that a list has: has
// reports whether the geosite (site true) or geoip list carries a tag, so
// that another list's source, without some of them, still loads.
func TrackerRules(has func(site bool, tag string) bool) []string {
	var out []string
	for _, t := range TrackerSiteTags {
		if has(true, t) {
			out = append(out, "geosite:"+t)
		}
	}
	for _, t := range TrackerIPTags {
		if has(false, t) {
			out = append(out, "geoip:"+t)
		}
	}
	return out
}

// BlockTrackers reports the user's choice.
func (s *Syncer) BlockTrackers() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.BlockTrackers
}

// SetBlockTrackers saves it.
func (s *Syncer) SetBlockTrackers(on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.BlockTrackers = on
	return s.save()
}
