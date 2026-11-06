// Package identity keeps the computer's visible identity in line with the
// VPN's exit: the system time zone follows the exit country (where the
// system lets a program change it), and the report lists what still points
// at Iran (a Persian keyboard, Persian fonts, an Iranian time zone), for
// the app's Identity Shield page.
package identity

import (
	"strings"
	"time"

	// The exit's offset is computed with the program's own zone data, so
	// it is right also where the system has none (Windows).
	_ "time/tzdata"
)

// countryZones is each exit country's time zone: its capital's, or the
// most populous one's. Multi-zone countries are refined by city
// (cityZones).
var countryZones = map[string]string{
	"AE": "Asia/Dubai",
	"AL": "Europe/Tirane",
	"AR": "America/Argentina/Buenos_Aires",
	"AT": "Europe/Vienna",
	"AU": "Australia/Sydney",
	"BA": "Europe/Sarajevo",
	"BE": "Europe/Brussels",
	"BG": "Europe/Sofia",
	"BR": "America/Sao_Paulo",
	"CA": "America/Toronto",
	"CH": "Europe/Zurich",
	"CL": "America/Santiago",
	"CO": "America/Bogota",
	"CY": "Asia/Nicosia",
	"CZ": "Europe/Prague",
	"DE": "Europe/Berlin",
	"DK": "Europe/Copenhagen",
	"EE": "Europe/Tallinn",
	"ES": "Europe/Madrid",
	"FI": "Europe/Helsinki",
	"FR": "Europe/Paris",
	"GB": "Europe/London",
	"GE": "Asia/Tbilisi",
	"GR": "Europe/Athens",
	"HK": "Asia/Hong_Kong",
	"HR": "Europe/Zagreb",
	"HU": "Europe/Budapest",
	"ID": "Asia/Jakarta",
	"IE": "Europe/Dublin",
	"IL": "Asia/Jerusalem",
	"IN": "Asia/Kolkata",
	"IR": "Asia/Tehran",
	"IS": "Atlantic/Reykjavik",
	"IT": "Europe/Rome",
	"JP": "Asia/Tokyo",
	"KR": "Asia/Seoul",
	"LT": "Europe/Vilnius",
	"LU": "Europe/Luxembourg",
	"LV": "Europe/Riga",
	"MD": "Europe/Chisinau",
	"MK": "Europe/Skopje",
	"MT": "Europe/Malta",
	"MX": "America/Mexico_City",
	"MY": "Asia/Kuala_Lumpur",
	"NL": "Europe/Amsterdam",
	"NO": "Europe/Oslo",
	"NZ": "Pacific/Auckland",
	"PE": "America/Lima",
	"PH": "Asia/Manila",
	"PL": "Europe/Warsaw",
	"PT": "Europe/Lisbon",
	"RO": "Europe/Bucharest",
	"RS": "Europe/Belgrade",
	"SE": "Europe/Stockholm",
	"SG": "Asia/Singapore",
	"SI": "Europe/Ljubljana",
	"SK": "Europe/Bratislava",
	"TH": "Asia/Bangkok",
	"TR": "Europe/Istanbul",
	"TW": "Asia/Taipei",
	"UA": "Europe/Kiev",
	"UK": "Europe/London",
	"US": "America/New_York",
	"VN": "Asia/Ho_Chi_Minh",
	"ZA": "Africa/Johannesburg",
}

// cityZones refines a multi-zone country by the exit city (lower case,
// without the state after a comma).
var cityZones = map[string]map[string]string{
	"US": {
		"atlanta": "America/New_York", "ashburn": "America/New_York", "boston": "America/New_York",
		"miami": "America/New_York", "new york": "America/New_York", "raleigh": "America/New_York",
		"secaucus": "America/New_York", "washington": "America/New_York", "philadelphia": "America/New_York",
		"chicago": "America/Chicago", "dallas": "America/Chicago", "houston": "America/Chicago",
		"denver": "America/Denver", "salt lake city": "America/Denver",
		"phoenix":     "America/Phoenix",
		"los angeles": "America/Los_Angeles", "san jose": "America/Los_Angeles",
		"san francisco": "America/Los_Angeles", "seattle": "America/Los_Angeles", "las vegas": "America/Los_Angeles",
	},
	"CA": {
		"montreal": "America/Toronto", "toronto": "America/Toronto",
		"vancouver": "America/Vancouver", "calgary": "America/Edmonton",
	},
	"AU": {
		"sydney": "Australia/Sydney", "melbourne": "Australia/Melbourne", "brisbane": "Australia/Brisbane",
		"adelaide": "Australia/Adelaide", "perth": "Australia/Perth",
	},
}

// windowsZones is each zone above as Windows names it (tzutil, the
// registry's Time Zones keys), from CLDR's windowsZones.
var windowsZones = map[string]string{
	"Africa/Johannesburg":            "South Africa Standard Time",
	"America/Argentina/Buenos_Aires": "Argentina Standard Time",
	"America/Bogota":                 "SA Pacific Standard Time",
	"America/Chicago":                "Central Standard Time",
	"America/Denver":                 "Mountain Standard Time",
	"America/Edmonton":               "Mountain Standard Time",
	"America/Lima":                   "SA Pacific Standard Time",
	"America/Los_Angeles":            "Pacific Standard Time",
	"America/Mexico_City":            "Central Standard Time (Mexico)",
	"America/New_York":               "Eastern Standard Time",
	"America/Phoenix":                "US Mountain Standard Time",
	"America/Santiago":               "Pacific SA Standard Time",
	"America/Sao_Paulo":              "E. South America Standard Time",
	"America/Toronto":                "Eastern Standard Time",
	"America/Vancouver":              "Pacific Standard Time",
	"Asia/Bangkok":                   "SE Asia Standard Time",
	"Asia/Dubai":                     "Arabian Standard Time",
	"Asia/Ho_Chi_Minh":               "SE Asia Standard Time",
	"Asia/Hong_Kong":                 "China Standard Time",
	"Asia/Jakarta":                   "SE Asia Standard Time",
	"Asia/Jerusalem":                 "Israel Standard Time",
	"Asia/Kolkata":                   "India Standard Time",
	"Asia/Kuala_Lumpur":              "Singapore Standard Time",
	"Asia/Manila":                    "Singapore Standard Time",
	"Asia/Nicosia":                   "GTB Standard Time",
	"Asia/Seoul":                     "Korea Standard Time",
	"Asia/Singapore":                 "Singapore Standard Time",
	"Asia/Taipei":                    "Taipei Standard Time",
	"Asia/Tbilisi":                   "Georgian Standard Time",
	"Asia/Tehran":                    "Iran Standard Time",
	"Asia/Tokyo":                     "Tokyo Standard Time",
	"Atlantic/Reykjavik":             "Greenwich Standard Time",
	"Australia/Adelaide":             "Cen. Australia Standard Time",
	"Australia/Brisbane":             "E. Australia Standard Time",
	"Australia/Melbourne":            "AUS Eastern Standard Time",
	"Australia/Perth":                "W. Australia Standard Time",
	"Australia/Sydney":               "AUS Eastern Standard Time",
	"Europe/Amsterdam":               "W. Europe Standard Time",
	"Europe/Athens":                  "GTB Standard Time",
	"Europe/Belgrade":                "Central Europe Standard Time",
	"Europe/Berlin":                  "W. Europe Standard Time",
	"Europe/Bratislava":              "Central Europe Standard Time",
	"Europe/Brussels":                "Romance Standard Time",
	"Europe/Bucharest":               "GTB Standard Time",
	"Europe/Budapest":                "Central Europe Standard Time",
	"Europe/Chisinau":                "E. Europe Standard Time",
	"Europe/Copenhagen":              "Romance Standard Time",
	"Europe/Dublin":                  "GMT Standard Time",
	"Europe/Helsinki":                "FLE Standard Time",
	"Europe/Istanbul":                "Turkey Standard Time",
	"Europe/Kiev":                    "FLE Standard Time",
	"Europe/Lisbon":                  "GMT Standard Time",
	"Europe/Ljubljana":               "Central Europe Standard Time",
	"Europe/London":                  "GMT Standard Time",
	"Europe/Luxembourg":              "W. Europe Standard Time",
	"Europe/Madrid":                  "Romance Standard Time",
	"Europe/Malta":                   "W. Europe Standard Time",
	"Europe/Oslo":                    "W. Europe Standard Time",
	"Europe/Paris":                   "Romance Standard Time",
	"Europe/Prague":                  "Central Europe Standard Time",
	"Europe/Riga":                    "FLE Standard Time",
	"Europe/Rome":                    "W. Europe Standard Time",
	"Europe/Sarajevo":                "Central European Standard Time",
	"Europe/Skopje":                  "Central European Standard Time",
	"Europe/Sofia":                   "FLE Standard Time",
	"Europe/Stockholm":               "W. Europe Standard Time",
	"Europe/Tallinn":                 "FLE Standard Time",
	"Europe/Tirane":                  "Central Europe Standard Time",
	"Europe/Vienna":                  "W. Europe Standard Time",
	"Europe/Vilnius":                 "FLE Standard Time",
	"Europe/Warsaw":                  "Central European Standard Time",
	"Europe/Zagreb":                  "Central European Standard Time",
	"Europe/Zurich":                  "W. Europe Standard Time",
	"Pacific/Auckland":               "New Zealand Standard Time",
}

// ZoneFor is the IANA time zone of an exit: its city's in a multi-zone
// country, else its country's; "" for an unknown country.
func ZoneFor(country, city string) string {
	country = strings.ToUpper(strings.TrimSpace(country))
	if byCity, ok := cityZones[country]; ok {
		name := strings.ToLower(strings.TrimSpace(city))
		if i := strings.IndexByte(name, ','); i >= 0 {
			name = strings.TrimSpace(name[:i])
		}
		if zone, ok := byCity[name]; ok {
			return zone
		}
	}
	return countryZones[country]
}

// WindowsZone is an IANA zone's Windows name; "" when unknown.
func WindowsZone(iana string) string { return windowsZones[iana] }

// ZoneFromWindows is an IANA zone with this Windows name (the first in
// order, for a name several share); "" when unknown. A disabled daylight
// saving suffix (_dstoff) is ignored.
func ZoneFromWindows(name string) string {
	name = trimDSTOff(name)
	best := ""
	for iana, w := range windowsZones {
		if strings.EqualFold(w, name) && (best == "" || iana < best) {
			best = iana
		}
	}
	return best
}

// OffsetMinutes is the zone's current offset from UTC in minutes.
func OffsetMinutes(iana string, now time.Time) (int, bool) {
	if iana == "" {
		return 0, false
	}
	loc, err := time.LoadLocation(iana)
	if err != nil {
		return 0, false
	}
	_, off := now.In(loc).Zone()
	return off / 60, true
}

// IsIranZone reports whether a zone name (IANA or Windows) is Iran's.
func IsIranZone(name string) bool {
	n := strings.ToLower(trimDSTOff(strings.TrimSpace(name)))
	return n == "asia/tehran" || n == "iran" || n == "iran standard time"
}

func trimDSTOff(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(name), "_dstoff")
}

// SameZone compares two system zone names, ignoring case and Windows'
// _dstoff suffix.
func SameZone(a, b string) bool {
	return strings.EqualFold(trimDSTOff(a), trimDSTOff(b))
}
