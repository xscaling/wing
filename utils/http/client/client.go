package client

import (
	"net/http"

	"github.com/xscaling/wing/utils/http/client/sign"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	requester *Requester
	endpoint  string
	signer    sign.Signer
}

func NewClient(requester *Requester, endpoint string) *Client {
	return &Client{
		requester: requester,
		endpoint:  endpoint,
	}
}

func (c *Client) GetEndpoint() string {
	return c.endpoint
}

// Request function will use decoder to unmarshal for receiver first(if set),
// and check status code all the time(some case may lead unmarshal success with unexpected code).
func (c *Client) Request(method string, resourceFormat string, options ...Option) (*resty.Response, error) {
	opts := NewOptions()
	for _, opt := range options {
		if err := opt(opts); err != nil {
			return nil, err
		}
	}
	resp, err := c.requester.do(
		opts.GetSigner(), method, c.endpoint, opts.Body(), opts.Headers(), resourceFormat, opts.Args()...)
	if err != nil {
		return resp, err
	}
	receiver, defaultError := opts.GetReceiverOptions()
	if receiver != nil {
		// Try unmarshal first
		err := c.requester.Encoding.Unmarshal(resp.Body(), receiver)
		if err != nil {
			return resp, err
		}
		// If unmarshal success, turn status code check
	}
	// No receiver then check status code, expected code or < 400
	if opts.expectedStatusCode == resp.StatusCode() || resp.StatusCode() < http.StatusBadRequest {
		return resp, nil
	}
	return resp, defaultError
}
