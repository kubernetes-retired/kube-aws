package amiregistry

import (
	"fmt"
	"net/http"
)

const MaxRetryCount = 2

type reliableHttp interface {
	Get(url string) (resp *http.Response, err error)
}

type reliableHttpImpl struct {
	underlyingGet func(url string) (resp *http.Response, err error)
}

func newHttp() reliableHttpImpl {
	return reliableHttpImpl{
		underlyingGet: http.Get,
	}
}

func (i reliableHttpImpl) Get(url string) (resp *http.Response, err error) {
	var r *http.Response
	var e error

	c := 0
	for c <= MaxRetryCount {
		c++
		r, e = i.underlyingGet(url)
		if e == nil {
			return r, nil
		}
	}

	return nil, fmt.Errorf("max retry count exceeded: %v", e)
}
