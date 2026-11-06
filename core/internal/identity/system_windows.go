//go:build windows

package identity

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type hostSystem struct{}

func tzutil(args ...string) (string, error) {
	exe := "tzutil.exe"
	if root := os.Getenv("SystemRoot"); root != "" {
		exe = filepath.Join(root, "System32", "tzutil.exe")
	}
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tzutil %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (hostSystem) Supported() bool { return true }

func (hostSystem) Zone() (string, error) { return tzutil("/g") }

func (hostSystem) SetZone(name string) error {
	_, err := tzutil("/s", name)
	return err
}

func (hostSystem) Name(iana string) string { return WindowsZone(iana) }

func (hostSystem) IANA(name string) string { return ZoneFromWindows(name) }

// readKeyboardAndLanguages reads the user's keyboard layouts (Persian is
// 0429) and display languages from the registry.
func readKeyboardAndLanguages() (*bool, []string) {
	persian := false
	known := false
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Keyboard Layout\Preload`, registry.QUERY_VALUE); err == nil {
		known = true
		names, _ := k.ReadValueNames(0)
		for _, n := range names {
			if v, _, err := k.GetStringValue(n); err == nil && strings.HasSuffix(strings.ToLower(v), "0429") {
				persian = true
			}
		}
		k.Close()
	}
	var langs []string
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\International\User Profile`, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS); err == nil {
		known = true
		langs, _, _ = k.GetStringsValue("Languages")
		for _, l := range langs {
			if strings.HasPrefix(strings.ToLower(l), "fa") {
				if sub, err := registry.OpenKey(k, l, registry.QUERY_VALUE); err == nil {
					names, _ := sub.ReadValueNames(0)
					for _, n := range names {
						if strings.Contains(strings.ToLower(n), "0429") {
							persian = true
						}
					}
					sub.Close()
				}
			}
		}
		k.Close()
	}
	if !known {
		return nil, langs
	}
	return &persian, langs
}
