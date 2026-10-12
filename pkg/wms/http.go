package wms

import (
	"fmt"
	"net/url"
)

type Connection struct {
	uri     *url.URL
	auth    Auth
	country CountryCode
}

type Endpoint struct {
	Country string
	Enabled bool
	Auth    Auth
	URI     string
}

func (e *Endpoint) Connection() (*Connection, error) {
	code := countryByName(e.Country)
	if code == CountryUnknown {
		return nil, fmt.Errorf("wms: country code by name: unknown country %s", e.Country)
	}

	parsedURL, err := url.ParseRequestURI(e.URI)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("wms: parsing endpoint uri: invalid %s: %w", e.URI, err)
	}

	return &Connection{
		uri:     parsedURL,
		auth:    e.Auth,
		country: code,
	}, nil
}
