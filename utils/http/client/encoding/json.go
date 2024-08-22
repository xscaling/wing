package encoding

import (
	"encoding/json"
)

type JSONEncoding struct{}

func (j JSONEncoding) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j JSONEncoding) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (j JSONEncoding) SetContentType(setter contentTypeSetter) {
	setter("application/json")
}
