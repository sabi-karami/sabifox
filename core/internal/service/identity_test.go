package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sabi-karami/sabifox/core/internal/identity"
)

type fakeZones struct{ zone string }

func (f *fakeZones) Supported() bool         { return true }
func (f *fakeZones) Zone() (string, error)   { return f.zone, nil }
func (f *fakeZones) SetZone(n string) error  { f.zone = n; return nil }
func (f *fakeZones) Name(iana string) string { return iana }
func (f *fakeZones) IANA(name string) string { return name }

// The Identity Shield's API reports the system zone and switches the sync;
// with the VPN off the sync changes nothing.
func TestTheIdentityAPI(t *testing.T) {
	zones := &fakeZones{zone: "Asia/Tehran"}
	old := identitySystem
	identitySystem = func() identity.System { return zones }
	t.Cleanup(func() { identitySystem = old })
	d := newTestDaemon(t)
	d.Off()
	c := newDashboardClient(t, d)
	rec := c.do("GET", "/api/identity", nil)
	var v IdentityView
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &v) != nil {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if !v.SystemIran || v.SystemZoneIANA != "Asia/Tehran" || v.SystemOffsetMinutes == nil || v.Sync.Enabled || !v.Sync.Supported {
		t.Fatalf("view %+v", v)
	}
	rec = c.do("PUT", "/api/identity", map[string]any{"sync_timezone": true})
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &v) != nil || !v.Sync.Enabled {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if zones.zone != "Asia/Tehran" {
		t.Fatalf("changed the zone while off: %s", zones.zone)
	}
	if rec := c.do("PUT", "/api/identity", map[string]any{"other": 1}); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field: %d", rec.Code)
	}
}

// Tracker blocking turns the routing on for its rules alone when the
// user's routing is off, and adds them to the user's when it is on.
func TestTrackerBlockingRouting(t *testing.T) {
	d := newTestDaemon(t)
	cfg := d.config().Routing
	cfg.Enabled = false
	if got := d.effectiveRouting(cfg); got.Enabled {
		t.Fatal("on without the choice")
	}
	if err := d.identity().SetBlockTrackers(true); err != nil {
		t.Fatal(err)
	}
	got := d.effectiveRouting(cfg)
	if !got.Enabled || len(got.Direct) != 1 || got.Direct[0] != "geoip:private" {
		t.Fatalf("routing off: %+v", got)
	}
	cfg.Enabled, cfg.Block = true, []string{"full:bad.example"}
	got = d.effectiveRouting(cfg)
	if !got.Enabled || got.Block[0] != "full:bad.example" {
		t.Fatalf("routing on: %+v", got)
	}
	if len(d.config().Routing.Block) != 0 && d.config().Routing.Block[0] != "full:bad.example" {
		t.Fatal("changed the config")
	}
}
