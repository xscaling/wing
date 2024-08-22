package encoding

type Encoding interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
	SetContentType(contentTypeSetter)
}

type contentTypeSetter func(contentType string)
