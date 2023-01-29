package prometheus

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/xscaling/wing/utils"

	"github.com/prometheus/common/model"
)

// ResponseMeta contains response information such as status, error
type ResponseMeta struct {
	Status    string   `json:"status"`
	ErrorType string   `json:"errorType"`
	Error     string   `json:"error"`
	Warnings  []string `json:"warnings"`
}

// VectorResponse is the payload for Vector result type response
type VectorResponse struct {
	ResponseMeta `json:",inline"`
	Data         VectorData `json:"data"`
}

// VectorData contains ResultType(which are most used for result type detection) and Result(model.Vector)
type VectorData struct {
	ResultType string       `json:"resultType"`
	Result     model.Vector `json:"result"`
}

type QueryClient interface {
	Query(server Server, query string, when time.Time) (float64, error)
}

type promQueryClient struct {
	httpClient         *http.Client
	insecureHTTPClient *http.Client
}

func NewQueryClient(timeout time.Duration) *promQueryClient {
	client := &promQueryClient{
		insecureHTTPClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
	return client
}

func (c *promQueryClient) Query(server Server, query string, when time.Time) (float64, error) {
	queryEscaped := url.QueryEscape(query)
	url := fmt.Sprintf("%s/api/v1/query?query=%s&time=%s", *server.ServerAddress, queryEscaped, when.Format(time.RFC3339))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return -1, err
	}
	// Set auth info
	if server.BearerToken != nil {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *server.BearerToken))
	} else if server.Username != nil {
		var password string
		if server.Password != nil {
			password = *server.Password
		}
		req.SetBasicAuth(*server.Username, password)
	}
	httpClient := c.httpClient
	if utils.GetPointerBoolValue(server.InsecureSSL, false) {
		httpClient = c.insecureHTTPClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	if resp.StatusCode/100 != 2 {
		return -1, fmt.Errorf("unable to fetch metric from prometheus server `%s`, status code %d body: `%s`", *server.ServerAddress, resp.StatusCode, body)
	}
	var vector VectorResponse
	if err := json.Unmarshal(body, &vector); err != nil {
		return -1, err
	}
	resultSize := len(vector.Data.Result)
	if resultSize == 0 {
		// empty value will be regarded as zero
		return 0, nil
	} else if resultSize > 1 {
		return -1, errors.New("this query returns multiple data")
	}
	return float64(vector.Data.Result[0].Value), nil
}
