package identity

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestTrackerRulesKeepTheListsTags(t *testing.T) {
	all := TrackerRules(func(bool, string) bool { return true })
	want := []string{"geosite:category-ads-all", "geosite:malware", "geosite:phishing", "geosite:cryptominers", "geoip:malware", "geoip:phishing"}
	if !slices.Equal(all, want) {
		t.Fatalf("all = %v", all)
	}
	some := TrackerRules(func(site bool, tag string) bool { return site && tag == "category-ads-all" })
	if !slices.Equal(some, []string{"geosite:category-ads-all"}) {
		t.Fatalf("some = %v", some)
	}
}

func TestFirefoxPrefsKeepEverythingInTheProxy(t *testing.T) {
	p := FirefoxPrefs("127.0.0.1", 10810, "https://example.com/check")
	for _, want := range []string{
		`user_pref("network.proxy.socks", "127.0.0.1");`,
		`user_pref("network.proxy.socks_port", 10810);`,
		`user_pref("network.proxy.socks_remote_dns", true);`,
		`user_pref("network.trr.mode", 5);`,
		`user_pref("media.peerconnection.ice.proxy_only_if_behind_proxy", true);`,
		`user_pref("privacy.resistFingerprinting", true);`,
		`user_pref("intl.accept_languages", "en-US, en");`,
		`user_pref("browser.startup.homepage", "https://example.com/check");`,
	} {
		if !strings.Contains(p, want) {
			t.Errorf("missing %s", want)
		}
	}
}

func TestChromiumArgs(t *testing.T) {
	args := strings.Join(ChromiumArgs("127.0.0.1", 10810, "/p", "https://x"), " ")
	for _, want := range []string{"--proxy-server=socks5://127.0.0.1:10810", "--user-data-dir=/p", "MAP * ~NOTFOUND , EXCLUDE 127.0.0.1", "disable_non_proxied_udp", "--lang=en-US"} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %s in %s", want, args)
		}
	}
}

func TestCommandWritesTheFirefoxProfile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ff")
	cmd, err := Command(Browser{ID: "firefox", Kind: "firefox", Path: "/usr/bin/firefox"}, LaunchOptions{Proxy: "127.0.0.1:10810", ProfileDir: dir, URL: "https://x"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(cmd.Args, "-profile") || !slices.Contains(cmd.Args, dir) {
		t.Fatalf("args %v", cmd.Args)
	}
	if _, err := Command(Browser{Kind: "chromium", Path: "/x"}, LaunchOptions{Proxy: "bad", ProfileDir: dir}); err == nil {
		t.Fatal("a bad proxy address was accepted")
	}
}
