package service

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sabi-karami/sabifox/core/internal/config"
	"github.com/sabi-karami/sabifox/core/internal/identity"
	"github.com/sabi-karami/sabifox/core/internal/logx"
	"github.com/sabi-karami/sabifox/core/internal/routing"
)

// identityFileName keeps the time zone sync's state (identity.Syncer).
const identityFileName = "identity.json"

// identitySyncInterval is how often the sync compares the system zone
// with the exit's.
const identitySyncInterval = 10 * time.Second

// identitySystem replaces the host's time zone in tests.
var identitySystem = identity.Host

func (d *Daemon) identity() *identity.Syncer {
	d.identityOnce.Do(func() {
		d.identitySync = identity.NewSyncer(filepath.Join(d.paths.Dir, identityFileName), identitySystem())
	})
	return d.identitySync
}

// exitPlace is the exit's country and city: the selected location's, or
// the exit check's for the recommended location; "" while unknown or off.
func (d *Daemon) exitPlace() (country, city string) {
	engine := d.currentEngine()
	if engine == nil {
		return "", ""
	}
	selected, _ := engine.selection()
	if code := strings.ToUpper(selected.CountryCode); len(code) == 2 {
		return code, selected.CityName
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.exit != nil && len(d.exit.CountryCode) == 2 {
		return strings.ToUpper(d.exit.CountryCode), d.exit.City
	}
	return "", ""
}

// syncIdentity moves the system zone to the exit's while the sync is on
// and the VPN connected, and restores it otherwise. An exit not known yet
// (the recommended location before its check) changes nothing.
func (d *Daemon) syncIdentity() {
	s := d.identity()
	if !s.Enabled() || !d.IsOn() {
		s.Restore()
		return
	}
	country, city := d.exitPlace()
	if zone := identity.ZoneFor(country, city); zone != "" {
		s.Apply(zone)
	}
}

// runIdentitySync keeps the zone in sync until the service stops, then
// restores the user's zone.
func (d *Daemon) runIdentitySync() {
	d.syncIdentity()
	t := time.NewTicker(identitySyncInterval)
	defer t.Stop()
	for {
		select {
		case <-d.stop:
			return
		case <-t.C:
			d.syncIdentity()
		}
	}
}

// IdentityView is GET /api/identity: the Identity Shield's report.
type IdentityView struct {
	Platform string `json:"platform"`
	// Connected: the VPN is on.
	Connected   bool   `json:"connected"`
	ExitCountry string `json:"exit_country,omitempty"`
	ExitCity    string `json:"exit_city,omitempty"`
	ExitZone    string `json:"exit_zone,omitempty"`
	// ExitOffsetMinutes is the exit zone's offset from UTC now.
	ExitOffsetMinutes *int `json:"exit_offset_minutes,omitempty"`
	// SystemZone is the system's zone by its own name, SystemZoneIANA its
	// IANA name, SystemIran whether it is Iran's.
	SystemZone          string `json:"system_zone,omitempty"`
	SystemZoneIANA      string `json:"system_zone_iana,omitempty"`
	SystemOffsetMinutes *int   `json:"system_offset_minutes,omitempty"`
	SystemIran          bool   `json:"system_iran"`
	// Sync is the time zone sync's state.
	Sync identity.Status `json:"sync"`
	// Desktop is the keyboard, languages and fonts (desktops only).
	Desktop *identity.Desktop `json:"desktop,omitempty"`
	// Leaks is what guards against traffic outside the VPN.
	Leaks IdentityLeaks `json:"leaks"`
	// BlockTrackers: advertising, tracking, malware, phishing and mining
	// sites are blocked; TrackerRules the rules in effect (none until the
	// lists are downloaded).
	BlockTrackers bool     `json:"block_trackers"`
	TrackerRules  []string `json:"tracker_rules"`
	// Browsers are the installed browsers the private browser can start
	// (computers only).
	Browsers []identity.Browser `json:"browsers"`
}

// IdentityLeaks reports the connection's leak protection.
type IdentityLeaks struct {
	// Mode is system_proxy.mode: tun guards every app, set only the apps
	// that use the system proxy.
	Mode string `json:"mode"`
	// TunSupported: this app can run the TUN mode here.
	TunSupported bool `json:"tun_supported"`
	// TunBlocksIPv6 and TunBlocksDNS: in the TUN mode, IPv6 and DNS
	// outside the VPN are blocked (Windows' filters, macOS', Linux's
	// routes).
	TunBlocksIPv6 bool `json:"tun_blocks_ipv6"`
	TunBlocksDNS  bool `json:"tun_blocks_dns"`
	// EncryptedDNS: every DNS server set is DoH or DoT.
	EncryptedDNS bool `json:"encrypted_dns"`
	// IranDirect: Iranian sites go directly (routing region ir), so they
	// see the real address, not the VPN's.
	IranDirect bool `json:"iran_direct"`
}

func (d *Daemon) identityView() IdentityView {
	cfg := d.config()
	s := d.identity()
	sys := identitySystem()
	now := time.Now()
	v := IdentityView{Platform: runtime.GOOS, Connected: d.IsOn(), Sync: s.Status()}
	if v.Connected {
		v.ExitCountry, v.ExitCity = d.exitPlace()
		v.ExitZone = identity.ZoneFor(v.ExitCountry, v.ExitCity)
		if off, ok := identity.OffsetMinutes(v.ExitZone, now); ok {
			v.ExitOffsetMinutes = &off
		}
	}
	if zone, err := sys.Zone(); err == nil {
		v.SystemZone = zone
		v.SystemZoneIANA = sys.IANA(zone)
		v.SystemIran = identity.IsIranZone(zone) || identity.IsIranZone(v.SystemZoneIANA)
		if off, ok := identity.OffsetMinutes(v.SystemZoneIANA, now); ok {
			v.SystemOffsetMinutes = &off
		}
	}
	switch runtime.GOOS {
	case "windows", "linux", "darwin":
		desktop := identity.ReadDesktop()
		v.Desktop = &desktop
	}
	encrypted := len(cfg.DNS.Servers) > 0
	for _, server := range cfg.DNS.Servers {
		server = strings.ToLower(strings.TrimSpace(server))
		if !strings.HasPrefix(server, "https://") && !strings.HasPrefix(server, "tls://") {
			encrypted = false
		}
	}
	v.Leaks = IdentityLeaks{
		Mode:          cfg.SystemProxy.Mode,
		TunSupported:  d.TunSupported,
		TunBlocksIPv6: d.TunSupported,
		TunBlocksDNS:  d.TunSupported && runtime.GOOS == "windows",
		EncryptedDNS:  encrypted,
		IranDirect:    cfg.Routing.Enabled && strings.EqualFold(cfg.Routing.Region, "ir"),
	}
	v.BlockTrackers = s.BlockTrackers()
	v.TrackerRules = []string{}
	if v.BlockTrackers {
		v.TrackerRules = identity.TrackerRules(d.listHas)
	}
	v.Browsers = []identity.Browser{}
	switch runtime.GOOS {
	case "windows", "linux", "darwin":
		if found := identity.FindBrowsers(); found != nil {
			v.Browsers = found
		}
	}
	if runtime.GOOS == "android" {
		// The phone's VPN carries every app, IPv6 and DNS included.
		v.Leaks.Mode = config.SystemProxyTun
		v.Leaks.TunBlocksIPv6, v.Leaks.TunBlocksDNS = true, true
	}
	return v
}

func (s *dashboardServer) handleIdentity(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.d.identityView())
}

type identityRequest struct {
	SyncTimezone  *bool `json:"sync_timezone"`
	BlockTrackers *bool `json:"block_trackers"`
}

func (s *dashboardServer) handleIdentityPut(w http.ResponseWriter, r *http.Request) {
	var req identityRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.SyncTimezone != nil {
		if err := s.d.identity().SetEnabled(*req.SyncTimezone); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, identity.ErrUnsupported) {
				status = http.StatusConflict
			}
			writeError(w, status, err.Error())
			return
		}
		s.d.syncIdentity()
	}
	if req.BlockTrackers != nil {
		if err := s.d.identity().SetBlockTrackers(*req.BlockTrackers); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.d.applyRouting(s.d.config().Routing)
		if *req.BlockTrackers {
			select {
			case s.d.routing.wake <- struct{}{}:
			default:
			}
		}
	}
	writeJSON(w, http.StatusOK, s.d.identityView())
}

// trackerTags caches the tags of the routing lists for the tracker rules,
// by the files' modification times.
var trackerTags struct {
	mu       sync.Mutex
	site, ip time.Time
	have     map[string]bool
}

func (d *Daemon) listHas(site bool, tag string) bool {
	sitePath := filepath.Join(d.paths.RoutingDir, routing.GeoSiteFile)
	ipPath := filepath.Join(d.paths.RoutingDir, routing.GeoIPFile)
	mod := func(p string) time.Time {
		if info, err := os.Stat(p); err == nil {
			return info.ModTime()
		}
		return time.Time{}
	}
	trackerTags.mu.Lock()
	defer trackerTags.mu.Unlock()
	if st, it := mod(sitePath), mod(ipPath); trackerTags.have == nil || !st.Equal(trackerTags.site) || !it.Equal(trackerTags.ip) {
		have := map[string]bool{}
		if tags, err := routing.Tags(sitePath); err == nil {
			for _, t := range tags {
				have["site:"+t] = true
			}
		}
		if tags, err := routing.Tags(ipPath); err == nil {
			for _, t := range tags {
				have["ip:"+t] = true
			}
		}
		trackerTags.have, trackerTags.site, trackerTags.ip = have, st, it
	}
	if site {
		return trackerTags.have["site:"+tag]
	}
	return trackerTags.have["ip:"+tag]
}

// effectiveRouting is the routing with the Identity Shield's tracker
// blocking: its rules added to Block, and routing on for them; with the
// user's routing off only the local networks go directly, as without
// routing. The lists download for it as for routing.
func (d *Daemon) effectiveRouting(cfg config.Routing) config.Routing {
	if !d.identity().BlockTrackers() {
		return cfg
	}
	if !cfg.Enabled {
		cfg.Enabled = true
		cfg.Direct = []string{"geoip:private"}
		cfg.Block = nil
	}
	rules := identity.TrackerRules(d.listHas)
	cfg.Block = append(slices.Clone(cfg.Block), rules...)
	return cfg
}

// The private browser's profiles live in the data folder.
const browserDirName = "private-browser"

func (s *dashboardServer) handleIdentityBrowser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Browser string `json:"browser"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	var chosen *identity.Browser
	for _, b := range identity.FindBrowsers() {
		if req.Browser == "" || b.ID == req.Browser {
			chosen = &b
			break
		}
	}
	if chosen == nil {
		writeError(w, http.StatusNotFound, identity.ErrNoBrowser.Error())
		return
	}
	addr, user, _ := s.d.ProxyEndpoint()
	if addr == "" || !s.d.IsOn() {
		writeError(w, http.StatusConflict, "connect first: the private browser goes only through the VPN")
		return
	}
	if user != "" {
		writeError(w, http.StatusConflict, "the local proxy has a login (proxy.username), which browsers cannot send; remove it to use the private browser")
		return
	}
	country, city := s.d.exitPlace()
	err := identity.Launch(*chosen, identity.LaunchOptions{
		Proxy:      addr,
		ProfileDir: filepath.Join(s.d.paths.Dir, browserDirName, chosen.ID),
		TimeZone:   identity.ZoneFor(country, city),
		URL:        identityCheckURL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logx.Info("private browser started browser=%s", chosen.ID)
	writeJSON(w, http.StatusOK, map[string]string{"browser": chosen.ID})
}

// identityCheckURL is the site's check of what a browser shows.
const identityCheckURL = "https://sabifox.sahebkarami.ir/check"
