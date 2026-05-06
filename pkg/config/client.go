package config

import (
	"net/http"
)

// basicAuthRoundTripper is a roundtripper that adds basic authentication to a request.
type basicAuthRoundTripper struct {
	Username string
	Password OpaqueString
	Next     http.RoundTripper
}

// RoundTrip adds basic authentication to a request.
func (t *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.SetBasicAuth(t.Username, string(t.Password))
	return t.Next.RoundTrip(r)
}
