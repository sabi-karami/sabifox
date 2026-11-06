package identity

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// The private browser: an installed browser started with its own profile
// in the app's data folder, every connection through the local proxy (the
// VPN) with the names resolved there (no DNS outside it), WebRTC kept to
// the proxy, English as the language, and in Firefox its fingerprinting
// resistance. When the VPN is off the proxy is gone and the browser
// cannot connect at all: it never falls back to the real address.

// Browser is an installed browser the private browser can start.
type Browser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Kind is firefox (Firefox, LibreWolf) or chromium (Chrome, Brave,
	// Edge, Chromium).
	Kind string `json:"kind"`
	Path string `json:"-"`
}

type browserCandidate struct {
	id, name, kind string
	windows        []string // under Program Files, Program Files (x86), or LocalAppData
	mac            []string // under /Applications
	linux          []string // commands
}

var browserCandidates = []browserCandidate{
	{"firefox", "Firefox", "firefox", []string{`Mozilla Firefox\firefox.exe`}, []string{"Firefox.app/Contents/MacOS/firefox"}, []string{"firefox", "firefox-esr"}},
	{"librewolf", "LibreWolf", "firefox", []string{`LibreWolf\librewolf.exe`}, []string{"LibreWolf.app/Contents/MacOS/librewolf"}, []string{"librewolf"}},
	{"brave", "Brave", "chromium", []string{`BraveSoftware\Brave-Browser\Application\brave.exe`}, []string{"Brave Browser.app/Contents/MacOS/Brave Browser"}, []string{"brave-browser", "brave"}},
	{"chrome", "Google Chrome", "chromium", []string{`Google\Chrome\Application\chrome.exe`}, []string{"Google Chrome.app/Contents/MacOS/Google Chrome"}, []string{"google-chrome", "google-chrome-stable"}},
	{"chromium", "Chromium", "chromium", nil, []string{"Chromium.app/Contents/MacOS/Chromium"}, []string{"chromium", "chromium-browser"}},
	{"edge", "Microsoft Edge", "chromium", []string{`Microsoft\Edge\Application\msedge.exe`}, []string{"Microsoft Edge.app/Contents/MacOS/Microsoft Edge"}, []string{"microsoft-edge", "microsoft-edge-stable"}},
}

// FindBrowsers lists the installed browsers, Firefox first: its
// fingerprinting resistance hides the most.
func FindBrowsers() []Browser {
	var out []Browser
	for _, c := range browserCandidates {
		if path := findBrowser(c); path != "" {
			out = append(out, Browser{ID: c.id, Name: c.name, Kind: c.kind, Path: path})
		}
	}
	return out
}

func findBrowser(c browserCandidate) string {
	exists := func(p string) bool {
		info, err := os.Stat(p)
		return err == nil && !info.IsDir()
	}
	switch runtime.GOOS {
	case "windows":
		for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")} {
			for _, rel := range c.windows {
				if root != "" && exists(filepath.Join(root, rel)) {
					return filepath.Join(root, rel)
				}
			}
		}
	case "darwin":
		home, _ := os.UserHomeDir()
		for _, root := range []string{"/Applications", filepath.Join(home, "Applications")} {
			for _, rel := range c.mac {
				if exists(filepath.Join(root, rel)) {
					return filepath.Join(root, rel)
				}
			}
		}
	case "linux":
		for _, name := range c.linux {
			if p, err := exec.LookPath(name); err == nil {
				return p
			}
		}
	}
	return ""
}

// LaunchOptions configure a private browser start.
type LaunchOptions struct {
	// Proxy is the local proxy's host:port (SOCKS5, no login).
	Proxy string
	// ProfileDir is the browser's own profile folder.
	ProfileDir string
	// TimeZone, an IANA zone, is given to Chromium browsers (TZ, which
	// they honor on Linux and macOS).
	TimeZone string
	// URL opens first.
	URL string
}

// ErrNoBrowser: the browser asked for is not installed.
var ErrNoBrowser = errors.New("this browser is not installed")

// FirefoxPrefs is the private profile's user.js: written at every start,
// so it always holds.
func FirefoxPrefs(host string, port int, home string) string {
	prefs := []struct {
		name  string
		value any
	}{
		// Everything through the VPN, names resolved by the proxy.
		{"network.proxy.type", 1},
		{"network.proxy.socks", host},
		{"network.proxy.socks_port", port},
		{"network.proxy.socks_version", 5},
		{"network.proxy.socks_remote_dns", true},
		{"network.proxy.socks5_remote_dns", true},
		{"network.proxy.no_proxies_on", ""},
		{"network.proxy.failover_direct", false},
		{"network.proxy.allow_bypass", false},
		{"network.trr.mode", 5},
		{"network.dns.disablePrefetch", true},
		{"network.dns.disablePrefetchFromHTTPS", true},
		{"network.prefetch-next", false},
		{"network.predictor.enabled", false},
		{"network.http.speculative-parallel-limit", 0},
		{"browser.urlbar.speculativeConnect.enabled", false},
		{"network.http.http3.enable", false},
		// WebRTC only through the proxy, without local addresses.
		{"media.peerconnection.ice.proxy_only_if_behind_proxy", true},
		{"media.peerconnection.ice.default_address_only", true},
		{"media.peerconnection.ice.no_host", true},
		// The fingerprint: Firefox's resistance (UTC time zone, English,
		// standard fonts, screen and canvas protection), as Tor Browser.
		{"privacy.resistFingerprinting", true},
		{"privacy.resistFingerprinting.letterboxing", true},
		{"privacy.spoof_english", 2},
		{"intl.accept_languages", "en-US, en"},
		{"javascript.use_us_english_locale", true},
		{"intl.regional_prefs.use_os_locales", false},
		{"browser.display.use_document_fonts", 1},
		{"geo.enabled", false},
		{"dom.battery.enabled", false},
		{"device.sensors.enabled", false},
		// Tracking.
		{"privacy.trackingprotection.enabled", true},
		{"privacy.trackingprotection.socialtracking.enabled", true},
		{"privacy.trackingprotection.cryptomining.enabled", true},
		{"privacy.trackingprotection.fingerprinting.enabled", true},
		{"network.cookie.cookieBehavior", 5},
		{"privacy.partition.network_state", true},
		{"privacy.query_stripping.enabled", true},
		{"network.http.referer.XOriginTrimmingPolicy", 2},
		{"beacon.enabled", false},
		{"privacy.globalprivacycontrol.enabled", true},
		{"privacy.donottrackheader.enabled", true},
		{"dom.security.https_only_mode", true},
		// Left behind at exit.
		{"privacy.sanitize.sanitizeOnShutdown", true},
		{"privacy.clearOnShutdown.cache", true},
		{"privacy.clearOnShutdown.history", true},
		{"privacy.clearOnShutdown.formdata", true},
		{"privacy.clearOnShutdown.sessions", true},
		{"privacy.clearOnShutdown.downloads", true},
		{"privacy.clearOnShutdown.cookies", false},
		{"browser.formfill.enable", false},
		// No reports to Mozilla, no first-run pages.
		{"toolkit.telemetry.enabled", false},
		{"toolkit.telemetry.unified", false},
		{"datareporting.healthreport.uploadEnabled", false},
		{"datareporting.policy.dataSubmissionEnabled", false},
		{"datareporting.policy.dataSubmissionPolicyBypassNotification", true},
		{"app.shield.optoutstudies.enabled", false},
		{"app.normandy.enabled", false},
		{"browser.discovery.enabled", false},
		{"browser.newtabpage.activity-stream.feeds.telemetry", false},
		{"browser.newtabpage.activity-stream.telemetry", false},
		{"browser.newtabpage.activity-stream.showSponsored", false},
		{"browser.newtabpage.activity-stream.showSponsoredTopSites", false},
		{"extensions.pocket.enabled", false},
		{"browser.shell.checkDefaultBrowser", false},
		{"browser.aboutwelcome.enabled", false},
		{"startup.homepage_welcome_url", ""},
		{"browser.startup.homepage_override.mstone", "ignore"},
		{"browser.startup.homepage", home},
		{"browser.startup.page", 1},
	}
	var b strings.Builder
	b.WriteString("// SABIFOX private browser: written at every start; change the app's settings, not this file.\n")
	for _, p := range prefs {
		var v string
		switch x := p.value.(type) {
		case string:
			v = fmt.Sprintf("%q", x)
		default:
			v = fmt.Sprint(x)
		}
		fmt.Fprintf(&b, "user_pref(%q, %s);\n", p.name, v)
	}
	return b.String()
}

// ChromiumArgs are a Chromium browser's flags for the private profile.
func ChromiumArgs(host string, port int, profile, url string) []string {
	return []string{
		"--user-data-dir=" + profile,
		fmt.Sprintf("--proxy-server=socks5://%s", net.JoinHostPort(host, fmt.Sprint(port))),
		// Names only through the proxy: a local lookup fails.
		"--host-resolver-rules=MAP * ~NOTFOUND , EXCLUDE " + host,
		"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
		"--webrtc-ip-handling-policy=disable_non_proxied_udp",
		"--disable-quic",
		"--lang=en-US",
		"--accept-lang=en-US,en",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-sync",
		"--new-window",
		url,
	}
}

// Command is the private browser's command, without starting it.
func Command(b Browser, o LaunchOptions) (*exec.Cmd, error) {
	host, portText, err := net.SplitHostPort(o.Proxy)
	if err != nil {
		return nil, fmt.Errorf("proxy address %q: %w", o.Proxy, err)
	}
	var port int
	if _, err := fmt.Sscan(portText, &port); err != nil || port <= 0 {
		return nil, fmt.Errorf("proxy port %q", portText)
	}
	if err := os.MkdirAll(o.ProfileDir, 0o700); err != nil {
		return nil, err
	}
	var cmd *exec.Cmd
	switch b.Kind {
	case "firefox":
		if err := os.WriteFile(filepath.Join(o.ProfileDir, "user.js"), []byte(FirefoxPrefs(host, port, o.URL)), 0o600); err != nil {
			return nil, err
		}
		cmd = exec.Command(b.Path, "-no-remote", "-new-instance", "-profile", o.ProfileDir, o.URL)
	case "chromium":
		cmd = exec.Command(b.Path, ChromiumArgs(host, port, o.ProfileDir, o.URL)...)
	default:
		return nil, ErrNoBrowser
	}
	cmd.Env = os.Environ()
	if o.TimeZone != "" && b.Kind == "chromium" {
		cmd.Env = append(cmd.Env, "TZ="+o.TimeZone)
	}
	return cmd, nil
}

// Launch starts the private browser and leaves it running.
func Launch(b Browser, o LaunchOptions) error {
	cmd, err := Command(b, o)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
