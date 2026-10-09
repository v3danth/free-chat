// Package geo maps a client IP to an ISO 3166-1 alpha-2 country code using a
// local MaxMind-format database (e.g. DB-IP "IP to Country Lite"), so no
// request ever leaves the server.
package geo

import (
	"net/netip"

	"github.com/oschwald/maxminddb-golang/v2"
)

type Locator interface {
	// Country returns "" when the country is unknown.
	Country(ip string) string
}

// None is used when no database is configured.
type None struct{}

func (None) Country(string) string { return "" }

type DB struct {
	r *maxminddb.Reader
}

func Open(path string) (*DB, error) {
	r, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	return &DB{r: r}, nil
}

func (d *DB) Close() error { return d.r.Close() }

func (d *DB) Country(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ""
	}
	var rec struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	if err := d.r.Lookup(addr.Unmap()).Decode(&rec); err != nil || len(rec.Country.ISOCode) != 2 {
		return ""
	}
	return rec.Country.ISOCode
}
