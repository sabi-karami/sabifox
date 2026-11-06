package identity

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEveryZoneLoadsAndHasAWindowsName(t *testing.T) {
	zones := map[string]bool{}
	for _, z := range countryZones {
		zones[z] = true
	}
	for _, byCity := range cityZones {
		for _, z := range byCity {
			zones[z] = true
		}
	}
	for z := range zones {
		if _, err := time.LoadLocation(z); err != nil {
			t.Errorf("zone %s does not load: %v", z, err)
		}
		if WindowsZone(z) == "" {
			t.Errorf("zone %s has no Windows name", z)
		}
	}
}

func TestZoneFor(t *testing.T) {
	cases := []struct{ country, city, want string }{
		{"de", "Frankfurt", "Europe/Berlin"},
		{"US", "Los Angeles, CA", "America/Los_Angeles"},
		{"US", "Salt Lake City, UT", "America/Denver"},
		{"US", "", "America/New_York"},
		{"CA", "Vancouver", "America/Vancouver"},
		{"AU", "Perth", "Australia/Perth"},
		{"XX", "", ""},
	}
	for _, c := range cases {
		if got := ZoneFor(c.country, c.city); got != c.want {
			t.Errorf("ZoneFor(%q, %q) = %q, want %q", c.country, c.city, got, c.want)
		}
	}
}

func TestWindowsNames(t *testing.T) {
	if got := ZoneFromWindows("W. Europe Standard Time_dstoff"); got != "Europe/Amsterdam" {
		t.Errorf("ZoneFromWindows = %q", got)
	}
	if !IsIranZone("Iran Standard Time") || !IsIranZone("Asia/Tehran") || IsIranZone("Europe/Berlin") {
		t.Error("IsIranZone")
	}
	if off, ok := OffsetMinutes("Asia/Tehran", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); !ok || off != 210 {
		t.Errorf("Tehran offset = %d %v", off, ok)
	}
}

func TestPersianFonts(t *testing.T) {
	for _, f := range []string{"Vazirmatn-Regular.ttf", "B Nazanin.ttf", "IRANSansWeb(FaNum).woff2", "Sahel-Bold.ttf", "XB Zar.ttf", "Dana-FaNum.otf"} {
		if !IsPersianFont(f) {
			t.Errorf("%s should be Persian", f)
		}
	}
	for _, f := range []string{"arial.ttf", "RAVIE.TTF", "NotoNastaliqUrdu-Regular.ttf", "tahoma.ttf", "segoeui.ttf", "Veronika.ttf"} {
		if IsPersianFont(f) {
			t.Errorf("%s should not be Persian", f)
		}
	}
	dir := t.TempDir()
	for _, f := range []string{"a/Vazir.ttf", "b/arial.ttf", "Shabnam.woff"} {
		p := filepath.Join(dir, f)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, nil, 0o644)
	}
	got := PersianFonts([]string{dir, filepath.Join(dir, "missing")})
	if len(got) != 2 || got[0] != "Shabnam.woff" || got[1] != "Vazir.ttf" {
		t.Errorf("PersianFonts = %v", got)
	}
}

func TestIranLayout(t *testing.T) {
	if !hasIranLayout("[('xkb', 'us'), ('xkb', 'ir')]") || !hasIranLayout("X11 Layout: us,ir") || hasIranLayout("[('xkb', 'us'), ('xkb', 'irish')]") {
		t.Error("hasIranLayout")
	}
}

type fakeSystem struct {
	zone    string
	sets    int
	failSet bool
}

func (f *fakeSystem) Supported() bool         { return true }
func (f *fakeSystem) Zone() (string, error)   { return f.zone, nil }
func (f *fakeSystem) Name(iana string) string { return WindowsZone(iana) }
func (f *fakeSystem) IANA(n string) string    { return ZoneFromWindows(n) }
func (f *fakeSystem) SetZone(n string) error {
	f.sets++
	if f.failSet {
		return errors.New("denied")
	}
	f.zone = n
	return nil
}

func TestSyncerAppliesAndRestores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.json")
	sys := &fakeSystem{zone: "Iran Standard Time"}
	s := NewSyncer(path, sys)
	s.Apply("Europe/Berlin")
	if sys.sets != 0 {
		t.Fatal("applied while disabled")
	}
	if err := s.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	s.Apply("Europe/Berlin")
	if sys.zone != "W. Europe Standard Time" {
		t.Fatalf("zone = %s", sys.zone)
	}
	s.Apply("Europe/Berlin")
	if sys.sets != 1 {
		t.Fatalf("set again: %d", sys.sets)
	}
	// A crash: a new syncer still knows what to restore.
	s = NewSyncer(path, sys)
	s.Apply("Asia/Tokyo")
	if sys.zone != "Tokyo Standard Time" {
		t.Fatalf("zone = %s", sys.zone)
	}
	s.Apply("")
	if sys.zone != "Iran Standard Time" {
		t.Fatalf("not restored: %s", sys.zone)
	}
	if st := s.Status(); st.Applied || !st.Enabled {
		t.Fatalf("status %+v", st)
	}
}

func TestSyncerRespectsUserChanges(t *testing.T) {
	sys := &fakeSystem{zone: "Iran Standard Time"}
	s := NewSyncer(filepath.Join(t.TempDir(), "identity.json"), sys)
	_ = s.SetEnabled(true)
	s.Apply("Europe/Paris")
	sys.zone = "Pacific Standard Time" // the user picks another zone
	s.Apply("Europe/Paris")
	if sys.zone != "Pacific Standard Time" || !s.Status().Overridden {
		t.Fatalf("overrode the user: %s", sys.zone)
	}
	s.Restore()
	if sys.zone != "Pacific Standard Time" {
		t.Fatalf("restore replaced the user's zone: %s", sys.zone)
	}
}

func TestSyncerFailureIsNotRetried(t *testing.T) {
	sys := &fakeSystem{zone: "Asia/Tehran", failSet: true}
	s := NewSyncer(filepath.Join(t.TempDir(), "identity.json"), sys)
	_ = s.SetEnabled(true)
	s.Apply("Europe/London")
	s.Apply("Europe/London")
	if sys.sets != 1 || s.Status().Error == "" || s.Status().Applied {
		t.Fatalf("sets=%d status=%+v", sys.sets, s.Status())
	}
	_ = s.SetEnabled(true)
	s.Apply("Europe/London")
	if sys.sets != 2 {
		t.Fatalf("not retried after switching: %d", sys.sets)
	}
}
