package sign

import (
	"net/http"
)

type Signer interface {
	Sign(req *http.Request)
}
