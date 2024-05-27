package cert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dana-team/certificate-operator/api/v1alpha1"
	jsonutil "github.com/dana-team/certificate-operator/internal/jsonutil"
	"github.com/pkg/errors"
)

const (
	authorizationToken     = "Bearer %v"
	authorizationHeaderKey = "Authorization"
	acceptHeaderKey        = "accept"
	acceptHeaderValue      = "application/json"
)

const (
	errBodyIsNotJson         = "response body is not JSON"
	errFailedToUnmarshalBody = "failed to unmarshal response body: %v"
	errPostToCertFailed      = "POST to cert failed: %v"
	errDownloadToCertFailed  = "download request to Cert API failed: %v"
	errGetDataToCertFailed   = "GET request to Cert API failed: %v"
)

// PostCertificate sends a POST request to cert to create a new certificate and returns the GUID.
func (c *client) PostCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (string, error) {
	body := createPostBody(certificate)

	response, err := c.localHttpClient.SendRequest(ctx, http.MethodPost, c.apiEndpoint, jsonutil.ToJSON(body), c.getAuthorizationHeader(), true, c.timeout)
	if err != nil {
		return "", fmt.Errorf(errPostToCertFailed, err)
	}

	var responseBody PostCertificateResponse
	if err = parseResponseBody(response.Body, &responseBody); err != nil {
		return "", fmt.Errorf(errFailedToUnmarshalBody, err)
	}

	return responseBody.Guid, nil
}

// DownloadCertificate downloads a certificate from the Cert API.
func (c *client) DownloadCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (DownloadCertificateResponse, error) {
	url := fmt.Sprintf("%s%s%s%s", c.apiEndpoint, certificate.Status.Guid, c.downloadEndpoint, certificate.Spec.CertificateData.Form)

	response, err := c.localHttpClient.SendRequest(ctx, http.MethodGet, url, "", c.getAuthorizationHeader(), true, c.timeout)
	if err != nil {
		return DownloadCertificateResponse{}, fmt.Errorf(errDownloadToCertFailed, err)
	}

	var responseBody DownloadCertificateResponse
	if err = parseResponseBody(response.Body, &responseBody); err != nil {
		return DownloadCertificateResponse{}, fmt.Errorf(errFailedToUnmarshalBody, err)
	}

	return responseBody, nil
}

// GetCertificate gets certificate data from the Cert API.
func (c *client) GetCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (GetCertificateResponse, error) {
	url := fmt.Sprintf("%s%s", c.apiEndpoint, certificate.Status.Guid)

	response, err := c.localHttpClient.SendRequest(ctx, http.MethodGet, url, "", c.getAuthorizationHeader(), true, c.timeout)
	if err != nil {
		return GetCertificateResponse{}, fmt.Errorf(errGetDataToCertFailed, err)
	}

	var responseBody GetCertificateResponse
	if err = parseResponseBody(response.Body, &responseBody); err != nil {
		return GetCertificateResponse{}, fmt.Errorf(errFailedToUnmarshalBody, err)
	}

	return responseBody, nil
}

// getAuthorizationHeader retrieves the authorization header for communicating with the Cert API.
func (c *client) getAuthorizationHeader() map[string][]string {
	return map[string][]string{
		authorizationHeaderKey: {fmt.Sprintf(authorizationToken, c.token)},
		acceptHeaderKey:        {acceptHeaderValue},
	}
}

// createPostBody creates the post request body for obtaining a certificate.
func createPostBody(certificate *v1alpha1.Certificate) postCertificateBody {
	return postCertificateBody{
		Subject: Subject{
			CommonName:         certificate.Spec.CertificateData.Subject.CommonName,
			Country:            certificate.Spec.CertificateData.Subject.Country,
			State:              certificate.Spec.CertificateData.Subject.State,
			Locality:           certificate.Spec.CertificateData.Subject.Locality,
			Organization:       certificate.Spec.CertificateData.Subject.Organization,
			OrganizationalUnit: certificate.Spec.CertificateData.Subject.OrganizationalUnit,
		},
		San: San{
			DNS: certificate.Spec.CertificateData.San.DNS,
			IPs: certificate.Spec.CertificateData.San.IPs,
		},
		Template: certificate.Spec.CertificateData.Template,
	}
}

// parseResponseBody parses the response body received from the Cert API.
func parseResponseBody(body string, response interface{}) error {
	if !jsonutil.IsJSONString(body) {
		return errors.New(errBodyIsNotJson)
	}

	return json.Unmarshal([]byte(body), response)
}
