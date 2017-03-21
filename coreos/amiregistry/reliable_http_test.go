package amiregistry

import (
	"fmt"
	"net/http"
	"testing"
)

type fakeGetter struct {
	numSimulatedErrors int
}

func (g *fakeGetter) Get(url string) (resp *http.Response, err error) {
	if g.numSimulatedErrors > 0 {
		g.numSimulatedErrors--
		return nil, fmt.Errorf("simulated error: errors left=%d", g.numSimulatedErrors)
	}
	return &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}, nil
}

func TestAuthTokenGeneration(t *testing.T) {
	twoErrors := &fakeGetter{
		numSimulatedErrors: 2,
	}
	h1 := reliableHttpImpl{
		underlyingGet: twoErrors.Get,
	}
	r1, e1 := h1.Get("http://example.com/status")
	if e1 != nil {
		t.Errorf("expected an reliable http get to succeed after implicit retries, but it didn't: %v", e1)
	}
	if r1.StatusCode != 200 {
		t.Errorf("expected an reliable http get to succeed after implicit retries, but it didn't: invalid status code: %d", r1.StatusCode)
	}

	threeErrors := &fakeGetter{
		numSimulatedErrors: 3,
	}
	h2 := reliableHttpImpl{
		underlyingGet: threeErrors.Get,
	}
	_, e2 := h2.Get("http://example.com/status")
	if e2 == nil {
		t.Error("expected an reliable http get to fail after exceeding max retry count, but it didn't")
	}
}
