package http

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientConfig is http client config
type ClientConfig struct {
	Dial      time.Duration
	Timeout   time.Duration
	KeepAlive time.Duration
}

// Client is http client
type Client struct {
	Timeout   time.Duration
	client    *http.Client
	dialer    *net.Dialer
	transport http.RoundTripper
}

// NewClient new a http client
func NewClient(c *ClientConfig) *Client {
	client := new(Client)
	client.Timeout = c.Timeout
	client.dialer = &net.Dialer{
		Timeout:   c.Dial,
		KeepAlive: c.KeepAlive,
	}
	originTransport := &http.Transport{
		DialContext:     client.dialer.DialContext,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.transport = &TraceTransport{RoundTripper: originTransport}
	client.client = &http.Client{Transport: client.transport}
	return client
}

// NewRequest create http request
func (c *Client) NewRequest(method, uri string, params url.Values) (req *http.Request, err error) {
	if params == nil {
		params = url.Values{}
	}
	if method == http.MethodGet {
		un := uri
		if params != nil {
			un = un + "?" + params.Encode()
		}
		req, err = http.NewRequest(http.MethodGet, un, nil)
	} else {
		req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(params.Encode()))
	}
	return
}

// Do http do
func (c *Client) Do(ctx context.Context, req *http.Request, res interface{}) (err error) {
	var resp *http.Response
	timeout := time.Duration(c.Timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req = req.WithContext(ctx)
	if resp, err = c.client.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if res != nil {
		if err = json.Unmarshal(body, &res); err != nil {
			return
		}
	}
	return
}

// Get send get request
func (c *Client) Get(ctx context.Context, uri string, params url.Values, res interface{}) (err error) {
	req, err := c.NewRequest(http.MethodGet, uri, params)
	if err != nil {
		return
	}
	return c.Do(ctx, req, res)
}

// Post send post request
func (c *Client) Post(ctx context.Context, uri string, params url.Values, res interface{}) (err error) {
	req, err := c.NewRequest(http.MethodPost, uri, params)
	if err != nil {
		return
	}
	return c.Do(ctx, req, res)
}
