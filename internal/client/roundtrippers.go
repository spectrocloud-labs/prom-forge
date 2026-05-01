package client

import "net/http"

// BasicAuthRoundTripper is a roundtripper that adds basic authentication to a request.
type BasicAuthRoundTripper struct {
	Username, Password string
	Next               http.RoundTripper
}

// RoundTrip adds basic authentication to a request.
func (t *BasicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.SetBasicAuth(t.Username, t.Password)
	return t.Next.RoundTrip(r)
}
