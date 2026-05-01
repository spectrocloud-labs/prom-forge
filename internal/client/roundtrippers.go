package client

import (
	"net/http"

	"github.com/spectrocloud-labs/prom-forge/internal/config"
)

// BasicAuthRoundTripper is a roundtripper that adds basic authentication to a request.
type BasicAuthRoundTripper struct {
	Username string
	Password config.OpaqueString
	Next     http.RoundTripper
}

// RoundTrip adds basic authentication to a request.
func (t *BasicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.SetBasicAuth(t.Username, string(t.Password))
	return t.Next.RoundTrip(r)
}
