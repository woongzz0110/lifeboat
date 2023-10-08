package background

import (
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type HttpProbe struct {
	client *resty.Client
	method string
	url    string
}

func NewHttpProbe(method string, url string) IProbe {
	return &HttpProbe{
		client: resty.New(),
		method: method,
		url:    url,
	}
}

func (h HttpProbe) exec() bool {
	execute, err := h.client.R().Execute(h.method, h.url)
	if err != nil {
		log.Warnf("Background probe error. %+v", err)
		return false
	}
	if !execute.IsSuccess() {
		log.Warnf("Background probe failed. status: %d", execute.StatusCode())
		return false
	}
	return true
}
