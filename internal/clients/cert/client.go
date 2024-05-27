package cert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dana-team/certificate-operator/api/v1alpha1"
	httpClient "github.com/dana-team/certificate-operator/internal/clients/http"
	"github.com/go-logr/logr"
)

const (
	defaultWaitTimeout  = time.Minute
	keyAPIEndpoint      = "apiEndpoint"
	keyDownloadEndpoint = "downloadEndpoint"
	keyToken            = "token"
	keyCredentials      = "credentials"

	errMissingAPIEndpoint      = "missing API Endpoint in secret"
	errMissingDownloadEndpoint = "missing Download API Endpoint in secret"
	errMissingToken            = "missing token in secret"
	errUnmarshalCredentials    = "cannot unmarshal credentials as JSON: %v"
)

type ClientBuilder func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (Client, error)

// Client is the interface to interact with Cert API service.
type Client interface {
	PostCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (string, error)
	DownloadCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (DownloadCertificateResponse, error)
	GetCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (GetCertificateResponse, error)
}

type client struct {
	log              logr.Logger
	localHttpClient  httpClient.Client
	timeout          time.Duration
	apiEndpoint      string
	downloadEndpoint string
	token            string
}

// NewClient returns a new client.
func NewClient(log logr.Logger, options ...func(*client)) Client {
	cl := &client{}
	cl.localHttpClient = httpClient.NewClient(log)
	for _, o := range options {
		o(cl)
	}

	return cl
}

// WithAPIEndpoint returns a client with the API Endpoint field populated.
func WithAPIEndpoint(apiEndpoint string) func(*client) {
	return func(c *client) {
		c.apiEndpoint = apiEndpoint
	}
}

// WithDownloadEndpoint returns a client with the Download Endpoint field populated.
func WithDownloadEndpoint(downloadEndpoint string) func(*client) {
	return func(c *client) {
		c.downloadEndpoint = downloadEndpoint
	}
}

// WithToken returns a client with the Token field populated.
func WithToken(token string) func(*client) {
	return func(c *client) {
		c.token = token
	}
}

// WithTimeout returns a client with the Timeout field populated.
func WithTimeout(timeout time.Duration) func(*client) {
	return func(c *client) {
		c.timeout = timeout
	}
}

// NewClientFromCertificateConfigAndSecretData creates a new Client instance using the provided certificateConfig spec and secret data.
func NewClientFromCertificateConfigAndSecretData(log logr.Logger, certificateConfig *v1alpha1.CertificateConfig, secretData map[string][]byte) (Client, error) {
	creds := map[string]string{}

	if err := json.Unmarshal(secretData[keyCredentials], &creds); err != nil {
		return nil, fmt.Errorf(errUnmarshalCredentials, err)
	}

	apiEndpoint := creds[keyAPIEndpoint]
	if apiEndpoint == "" {
		return nil, errors.New(errMissingAPIEndpoint)
	}

	downloadEndpoint := creds[keyDownloadEndpoint]
	if downloadEndpoint == "" {
		return nil, errors.New(errMissingDownloadEndpoint)
	}

	token := creds[keyToken]
	if token == "" {
		return nil, errors.New(errMissingToken)
	}

	timeout := getWaitTimeout(certificateConfig)

	return NewClient(
		log,
		WithAPIEndpoint(apiEndpoint),
		WithDownloadEndpoint(downloadEndpoint),
		WithToken(token),
		WithTimeout(timeout),
	), nil

}

// getWaitTimeout returns the wait timeout duration specified in the CertificateConfig, or the default wait timeout if not specified.
func getWaitTimeout(certificateConfig *v1alpha1.CertificateConfig) time.Duration {
	if certificateConfig.Spec.WaitTimeout != nil {
		return certificateConfig.Spec.WaitTimeout.Duration
	}

	return defaultWaitTimeout
}
