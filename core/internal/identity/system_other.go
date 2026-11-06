//go:build !windows && (!linux || android)

package identity

type hostSystem struct{}

func (hostSystem) Supported() bool                { return false }
func (hostSystem) Zone() (string, error)          { return "", ErrUnsupported }
func (hostSystem) SetZone(string) error           { return ErrUnsupported }
func (hostSystem) Name(iana string) string        { return iana }
func (hostSystem) IANA(name string) string        { return name }
func readKeyboardAndLanguages() (*bool, []string) { return nil, nil }
