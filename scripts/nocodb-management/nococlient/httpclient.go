package nococlient

import (
	"net/http"
)

// APIKeyTransport adds the API Key header "xc-token" to all requests.
type APIKeyTransport struct {
	APIKey string
	Base   http.RoundTripper
}

func (t *APIKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("xc-token", t.APIKey)
	if t.Base == nil {
		t.Base = http.DefaultTransport
	}
	return t.Base.RoundTrip(req)
}

// NewHTTPClient returns an HTTP client that automatically adds the "xc-token" header.
func NewHTTPClient(apiKey string) *http.Client {
	return &http.Client{
		Transport: &APIKeyTransport{
			APIKey: apiKey,
			Base:   http.DefaultTransport,
		},
	}
}
