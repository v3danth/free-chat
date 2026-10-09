package geo

import (
	"os"
	"testing"
)

// Uses the downloaded database when present (make geoip); skipped otherwise.
func TestCountry(t *testing.T) {
	const path = "../../data/dbip-country-lite.mmdb"
	if _, err := os.Stat(path); err != nil {
		t.Skip("no GeoIP database; run make geoip")
	}
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for ip, want := range map[string]string{
		"8.8.8.8":    "US",
		"1.1.1.1":    "AU",
		"49.204.0.1": "IN",
		"127.0.0.1":  "",
		"not an ip":  "",
	} {
		if got := db.Country(ip); got != want {
			t.Errorf("%s: got %q, want %q", ip, got, want)
		}
	}
	if got := db.Country("2001:4860::1"); len(got) != 2 {
		t.Errorf("IPv6: got %q, want a country code", got)
	}
}
