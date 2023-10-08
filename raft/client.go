package raft

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
)

type Client struct {
	protocol string
	server   string
	client   *resty.Client
}

func NewClient(server string) *Client {
	return &Client{
		protocol: "http",
		server:   server,
		client:   resty.New(),
	}
}

func (c *Client) GetNode(nodeId int64) (*NodeResponse, error) {
	url := fmt.Sprintf("%s://%s/nodes/%d", c.protocol, c.server, nodeId)
	return parseResponse(c.client.R().Get, url, &NodeResponse{})
}

func (c *Client) PostElection(electionRequest ElectionRequest) (*ElectionResponse, error) {
	url := fmt.Sprintf("%s://%s/election", c.protocol, c.server)
	return parseResponse(c.client.R().SetBody(electionRequest).Post, url, &ElectionResponse{})
}

type bodyType interface {
	*NodeResponse | *ElectionResponse
}

func parseResponse[T bodyType](methodFunc func(url string) (*resty.Response, error), url string, body T) (T, error) {
	res, err := methodFunc(url)
	if err != nil {
		return nil, err
	}
	if res.IsError() {
		return nil, errors.New(res.Error().(string))
	}
	err = json.Unmarshal(res.Body(), body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
