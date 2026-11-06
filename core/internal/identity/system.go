package identity

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// System reads and sets the computer's time zone by its own names: IANA
// names on Linux, Windows names on Windows.
type System interface {
	// Supported reports whether this program can set the zone here.
	Supported() bool
	// Zone is the current system zone.
	Zone() (string, error)
	// SetZone sets it.
	SetZone(name string) error
	// Name is an IANA zone's system name; "" when the system has none.
	Name(iana string) string
	// IANA is a system name's IANA zone; "" when unknown.
	IANA(name string) string
}

// ErrUnsupported: the system zone cannot be set by this program here
// (macOS needs an administrator, Android and iOS do not allow it).
var ErrUnsupported = errors.New("the system time zone cannot be changed by the app on this system")

// Host is this computer's System.
func Host() System { return hostSystem{} }

// persianFontTokens are parts of the file names of Persian fonts (lower
// case, without spaces, hyphens or underscores). Sites detect such fonts
// by measuring text; they are rare outside Iran.
var persianFontTokens = []string{
	"vazir", "iransans", "iranyekan", "irannastaliq", "iraniansans", "irsans", "irancell",
	"bnazanin", "bzar", "bmitra", "blotus", "btitr", "bkoodak", "byekan", "byas", "broya",
	"bhoma", "btraffic", "bkamran", "bbadr", "bjadid", "bfantezy", "bkourosh",
	"sahel", "shabnam", "samim", "tanha", "parastoo", "gandom", "lalezar", "estedad",
	"mikhak", "kalameh", "morabba", "peyda", "anjoman", "dirooz", "nahid", "behdad", "nika",
	"xbzar", "xbniloofar", "xbriyaz", "xbkayhan", "xbroya", "xbyas", "xbtitre", "xbyagut",
	"xbshafigh", "xbsols", "xbtabriz", "xbkhoramshahr", "farsi", "persian", "yekanbakh", "dana",
}

// shortFontTokens match only at the start of a name (they are short
// enough to appear inside other fonts' names).
var shortFontTokens = map[string]bool{"dana": true, "nika": true, "nahid": true}

// fontDirs are the folders with installed fonts.
func fontDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{}
	if windir := os.Getenv("WINDIR"); windir != "" {
		dirs = append(dirs, filepath.Join(windir, "Fonts"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		dirs = append(dirs, filepath.Join(local, "Microsoft", "Windows", "Fonts"))
	}
	dirs = append(dirs, "/usr/share/fonts", "/usr/local/share/fonts", "/Library/Fonts")
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "fonts"), filepath.Join(home, ".fonts"), filepath.Join(home, "Library", "Fonts"))
	}
	return dirs
}

func normalizeFontName(name string) string {
	name = strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '_' || r == '.' {
			return -1
		}
		return r
	}, name)
}

// IsPersianFont reports whether a font file's name is a Persian font's.
func IsPersianFont(file string) bool {
	n := normalizeFontName(filepath.Base(file))
	for _, t := range persianFontTokens {
		if shortFontTokens[t] {
			if strings.HasPrefix(n, t) {
				return true
			}
			continue
		}
		if strings.Contains(n, t) {
			return true
		}
	}
	return false
}

// PersianFonts lists the installed Persian fonts' file names (at most
// 40), from the system and user font folders.
func PersianFonts(dirs []string) []string {
	found := map[string]bool{}
	scanned := 0
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, e os.DirEntry, err error) error {
			if err != nil {
				if e != nil && e.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			scanned++
			if scanned > 50000 || len(found) >= 40 {
				return filepath.SkipAll
			}
			if e.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".ttf", ".otf", ".ttc", ".woff", ".woff2", ".fon":
			default:
				return nil
			}
			if IsPersianFont(path) {
				found[filepath.Base(path)] = true
			}
			return nil
		})
	}
	out := make([]string, 0, len(found))
	for f := range found {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// Desktop is what the computer itself shows about its owner.
type Desktop struct {
	// PersianKeyboard: a Persian keyboard layout is installed; nil when
	// unknown on this system.
	PersianKeyboard *bool `json:"persian_keyboard,omitempty"`
	// Languages are the system's display languages (BCP 47), when known.
	Languages []string `json:"languages,omitempty"`
	// PersianFonts are the installed Persian fonts' files.
	PersianFonts []string `json:"persian_fonts"`
}

var desktopCache struct {
	mu sync.Mutex
	at time.Time
	d  Desktop
}

// ReadDesktop reads the keyboard, languages and fonts, at most once every
// two minutes (the fonts' scan reads many folders).
func ReadDesktop() Desktop {
	desktopCache.mu.Lock()
	defer desktopCache.mu.Unlock()
	if !desktopCache.at.IsZero() && time.Since(desktopCache.at) < 2*time.Minute {
		return desktopCache.d
	}
	d := Desktop{PersianFonts: PersianFonts(fontDirs())}
	d.PersianKeyboard, d.Languages = readKeyboardAndLanguages()
	desktopCache.d, desktopCache.at = d, time.Now()
	return d
}
