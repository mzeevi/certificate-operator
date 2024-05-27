package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	jsonutil "github.com/dana-team/certificate-operator/internal/jsonutil"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

// Client is the interface to interact with HTTP
type Client interface {
	SendRequest(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp Response, err error)
}

type client struct {
	log logr.Logger
}

// Response represents an HTTP response.
type Response struct {
	Body       string
	Headers    map[string][]string
	StatusCode int
}

// Request represents an HTTP request.
type Request struct {
	Method  string              `json:"method"`
	Body    string              `json:"body,omitempty"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers,omitempty"`
}

// SendRequest sends an HTTP request and returns the response.
func (c *client) SendRequest(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (Response, error) {
	requestBody := []byte(body)
	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(requestBody))

	if err != nil {
		return Response{}, err
	}

	for key, values := range headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	hclient := &http.Client{
		Transport: &http.Transport{
			// #nosec G402
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLSVerify},
		},
		Timeout: timeout,
	}

	response, err := hclient.Do(request)
	c.log.Info(fmt.Sprint("http request sent: ", jsonutil.ToJSON(Request{URL: url, Body: body, Method: method})))

	if err != nil {
		return Response{}, fmt.Errorf("http request to %q failed: %v", url, err)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return Response{}, fmt.Errorf("failed reading response body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		c.log.Info(fmt.Sprintf("request failed, method: %v, status code: %v, body: %v", method, response.StatusCode, responseBody))
		return Response{}, errors.New(http.StatusText(response.StatusCode))
	}

	beautifiedResponse := Response{
		Body:       string(responseBody),
		Headers:    response.Header,
		StatusCode: response.StatusCode,
	}

	err = response.Body.Close()
	if err != nil {
		return beautifiedResponse, err
	}

	return beautifiedResponse, nil
}

// NewClient returns a new Http Client
func NewClient(log logr.Logger) Client {
	return &client{
		log: log,
	}
}
