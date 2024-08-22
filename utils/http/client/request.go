package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/xscaling/wing/utils/http/client/encoding"
	"github.com/xscaling/wing/utils/http/client/sign"

	"github.com/go-resty/resty/v2"
)

const HeaderUserAgent = "User-Agent"

type requestSignerContext struct{}

type Requester struct {
	Encoding  encoding.Encoding
	userAgent string
	client    *resty.Client
}

func NewRequester(encoding encoding.Encoding, timeout time.Duration) *Requester {
	requester := &Requester{
		Encoding: encoding,
	}
	client := resty.New().
		SetTimeout(timeout).
		OnBeforeRequest(requester.onBeforeRequest).
		SetPreRequestHook(requester.preRequestHook)
	requester.client = client
	return requester
}

func (r Requester) onBeforeRequest(c *resty.Client, request *resty.Request) error {
	if r.userAgent != "" && c.Header.Get(HeaderUserAgent) == "" {
		request.SetHeader(HeaderUserAgent, r.userAgent)
	}
	r.Encoding.SetContentType(func(contentType string) {
		request.SetHeader("Content-Type", contentType)
	})

	return nil
}

func (r Requester) preRequestHook(_ *resty.Client, request *http.Request) error {
	// Must sign after all thing set
	rawSigner := request.Context().Value(requestSignerContext{})
	if rawSigner == nil {
		return nil
	}
	signer, ok := rawSigner.(sign.Signer)
	if !ok {
		return fmt.Errorf("invalid signer type in context: %T", rawSigner)
	}
	signer.Sign(request)
	return nil
}

func (r Requester) do(
	signer sign.Signer, method, endpoint string, requestBody interface{},
	headers map[string]string, resourceFormat string, query url.Values,
) (*resty.Response, error) {
	fullURL := fmt.Sprintf("%s/%s?%s", endpoint, resourceFormat, query.Encode())
	// As using dynamic signer potentially, we need to set preRequestHook every request to avoid polluting Requester
	request := r.client.R()
	request.SetHeaders(headers).
		SetContext(context.WithValue(request.Context(), requestSignerContext{}, signer))
	if requestBody != nil {
		// Though we can use auto marshal by resty(but only supports JSON and XML),
		// considering extension ability decide to manually marshal
		encodedRequestBody, err := r.Encoding.Marshal(requestBody)
		if err != nil {
			return nil, err
		}
		request.SetBody(encodedRequestBody)
	}
	return request.Execute(method, fullURL)
}
