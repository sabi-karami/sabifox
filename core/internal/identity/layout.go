package identity

import "strings"

// hasIranLayout finds the XKB layout "ir" in gsettings, localectl or
// setxkbmap output.
func hasIranLayout(s string) bool {
	s = strings.ToLower(s)
	for _, f := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if f == "ir" {
			return true
		}
	}
	return false
}
