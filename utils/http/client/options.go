package client

import (
	"github.com/xscaling/wing/utils/http/client/sign"
	"net/url"
)

type Options struct {
	signer             sign.Signer
	query              url.Values
	body               interface{}
	headers            map[string]string
	receiver           interface{}
	expectedStatusCode int
	defaultError       error
}

func NewOptions() *Options {
	return &Options{
		expectedStatusCode: -1,
	}
}

func (o Options) Signer() sign.Signer {
	return o.signer
}

func (o Options) Query() url.Values {
	return o.query
}

func (o Options) Body() interface{} {
	return o.body
}

func (o Options) Headers() map[string]string {
	return o.headers
}

func (o Options) GetSigner() sign.Signer {
	return o.signer
}

func (o Options) GetReceiverOptions() (interface{}, error) {
	return o.receiver, o.defaultError
}

type Option func(*Options) error

func WithSigner(signer sign.Signer) Option {
	return func(o *Options) error {
		o.signer = signer
		return nil
	}
}

func WithQuery(query url.Values) Option {
	return func(o *Options) error {
		o.query = query
		return nil
	}
}

func WithReceiver(receiver interface{}, defaultError error) Option {
	return func(o *Options) error {
		o.receiver = receiver
		o.defaultError = defaultError
		return nil
	}
}

func WithRequestBody(body interface{}) Option {
	return func(o *Options) error {
		o.body = body
		return nil
	}
}

func WithHeaders(headers map[string]string) Option {
	return func(o *Options) error {
		o.headers = headers
		return nil
	}
}

func WithExpectedStatusCode(statusCode int) Option {
	return func(o *Options) error {
		o.expectedStatusCode = statusCode
		return nil
	}
}
