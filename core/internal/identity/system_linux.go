//go:build linux && !android

package identity

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type hostSystem struct{}

func (hostSystem) Supported() bool {
	_, err := exec.LookPath("timedatectl")
	return err == nil
}

func (hostSystem) Zone() (string, error) {
	if out, err := exec.Command("timedatectl", "show", "-p", "Timezone", "--value").Output(); err == nil {
		if z := strings.TrimSpace(string(out)); z != "" {
			return z, nil
		}
	}
	link, err := os.Readlink("/etc/localtime")
	if err != nil {
		return "", err
	}
	if i := strings.Index(link, "zoneinfo/"); i >= 0 {
		return link[i+len("zoneinfo/"):], nil
	}
	return "", fmt.Errorf("unknown /etc/localtime: %s", link)
}

func (hostSystem) SetZone(name string) error {
	out, err := exec.Command("timedatectl", "set-timezone", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("timedatectl set-timezone %s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (hostSystem) Name(iana string) string { return iana }

func (hostSystem) IANA(name string) string { return name }

// readKeyboardAndLanguages reads the desktop's keyboard layouts (GNOME's
// input sources, else the system's X11 layout) and the session language.
func readKeyboardAndLanguages() (*bool, []string) {
	var langs []string
	for _, v := range []string{"LANGUAGE", "LC_ALL", "LANG"} {
		if s := os.Getenv(v); s != "" {
			for _, p := range strings.Split(s, ":") {
				p = strings.SplitN(strings.SplitN(p, ".", 2)[0], "@", 2)[0]
				if p != "" && p != "C" && p != "POSIX" {
					langs = append(langs, strings.ReplaceAll(p, "_", "-"))
				}
			}
			break
		}
	}
	var layouts string
	if out, err := exec.Command("gsettings", "get", "org.gnome.desktop.input-sources", "sources").Output(); err == nil {
		layouts = string(out)
	} else if out, err := exec.Command("localectl", "status").Output(); err == nil {
		layouts = string(out)
	} else if out, err := exec.Command("setxkbmap", "-query").Output(); err == nil {
		layouts = string(out)
	} else {
		return nil, langs
	}
	persian := hasIranLayout(layouts)
	return &persian, langs
}
