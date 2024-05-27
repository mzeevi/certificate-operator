package cert

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/dana-team/certificate-operator/api/v1alpha1"
	httpClient "github.com/dana-team/certificate-operator/internal/clients/http"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockSendRequestFn func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error)

type MockHttpClient struct {
	MockSendRequest MockSendRequestFn
}

func (c *MockHttpClient) SendRequest(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
	return c.MockSendRequest(ctx, method, url, body, headers, skipTLSVerify, timeout)
}

var (
	errBoom        = errors.New("boom")
	errBodyNotJson = errors.New("response body is not JSON")
)

var (
	apiEndpoint      = "https://example.com/cert/"
	downloadEndpoint = "download"
	token            = "jwt-test"
	timeout          = time.Minute

	certificateConfig = v1alpha1.CertificateConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-conf",
			Namespace: "default",
		},
		Spec: v1alpha1.CertificateConfigSpec{
			SecretRef: v1alpha1.SecretRef{
				Name:      "secret",
				Namespace: "default",
			},
			DaysBeforeRenewal: 7,
		},
	}

	certificate = v1alpha1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cert",
			Namespace: "default",
		},
		Spec: v1alpha1.CertificateSpec{
			CertificateData: v1alpha1.CertificateData{
				Subject: v1alpha1.Subject{
					CommonName: "example",
				},
				San: v1alpha1.San{
					DNS: []string{
						"www.example.com",
					},
					IPs: []string{
						"192.168.1.1",
					},
				},
				Template: "default",
				Form:     "pfx",
			},

			ConfigRef: v1alpha1.ConfigReference{
				Name: "certificateconfig-sample",
			},
			SecretName: "my-secret-new",
		},
	}
)

func Test_PostCertificate(t *testing.T) {
	type args struct {
		http              httpClient.Client
		certificate       *v1alpha1.Certificate
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		result string
		err    error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnGuid": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{
							Body:       `{"taskId": "83729jsdjd92819w1yhdsduy288yhduwdbd"}`,
							Headers:    nil,
							StatusCode: 200,
						}, nil
					},
				},
			},
			want: want{
				result: `83729jsdjd92819w1yhdsduy288yhduwdbd`,
				err:    nil,
			},
		},
		"ShouldFailSendingRequest": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{}, errBoom
					},
				},
			},
			want: want{
				result: "",
				err:    fmt.Errorf(errPostToCertFailed, errBoom),
			},
		},
		"ShouldFailParsingResponse": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{
							Body:       `{ "83729jsdjd92819w1yhdsduy288yhduwdbd"}`,
							Headers:    nil,
							StatusCode: 200,
						}, nil
					},
				},
			},
			want: want{
				result: "",
				err:    fmt.Errorf(errFailedToUnmarshalBody, errBodyNotJson),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cc := &client{
				log:              logr.Logger{},
				localHttpClient:  tc.args.http,
				timeout:          timeout,
				apiEndpoint:      apiEndpoint,
				downloadEndpoint: downloadEndpoint,
				token:            token,
			}

			got, gotErr := cc.PostCertificate(context.Background(), tc.args.certificate)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("PostCertificate(...): -want error, +got error: %v", diff)
			}
			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("PostCertificate(...): -want result, +got result: %v", diff)
			}
		})
	}
}

func Test_DownloadCertificate(t *testing.T) {
	type args struct {
		http              httpClient.Client
		certificate       *v1alpha1.Certificate
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		result DownloadCertificateResponse
		err    error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnResponseBody": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{
							Body:       `{"form":"pfx","format":"PEM","data":"string","password":"string"}`,
							Headers:    nil,
							StatusCode: 200,
						}, nil
					},
				},
			},
			want: want{
				result: DownloadCertificateResponse{Form: "pfx", Format: "PEM", Data: "string", Password: "string"},
				err:    nil,
			},
		},
		"ShouldFailSendingRequest": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{}, errBoom
					},
				},
			},
			want: want{
				result: DownloadCertificateResponse{},
				err:    fmt.Errorf(errDownloadToCertFailed, errBoom),
			},
		},
		"ShouldFailParsingResponse": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{
							Body:       `{ "83729jsdjd92819w1yhdsduy288yhduwdbd"}`,
							Headers:    nil,
							StatusCode: 200,
						}, nil
					},
				},
			},
			want: want{
				result: DownloadCertificateResponse{},
				err:    fmt.Errorf(errFailedToUnmarshalBody, errBodyNotJson),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cc := &client{
				log:              logr.Logger{},
				localHttpClient:  tc.args.http,
				timeout:          timeout,
				apiEndpoint:      apiEndpoint,
				downloadEndpoint: downloadEndpoint,
				token:            token,
			}

			got, gotErr := cc.DownloadCertificate(context.Background(), tc.args.certificate)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("DownloadCertificate(...): -want error, +got error: %v", diff)
			}
			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("DownloadCertificate(...): -want result, +got result: %v", diff)
			}
		})
	}
}

func Test_GetCertificate(t *testing.T) {
	type args struct {
		http              httpClient.Client
		certificate       *v1alpha1.Certificate
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		result GetCertificateResponse
		err    error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnResponseBody": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{
							Body:       `{"validTo":"2024-10-18T09:05:22","validFrom":"2024-04-18T09:05:22","signatureHashAlgorithm":"sha384"}`,
							Headers:    nil,
							StatusCode: 200,
						}, nil
					},
				},
			},
			want: want{
				result: GetCertificateResponse{ValidTo: "2024-10-18T09:05:22", ValidFrom: "2024-04-18T09:05:22", SignatureHashAlgorithm: "sha384"},
				err:    nil,
			},
		},
		"ShouldFailSendingRequest": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{}, errBoom
					},
				},
			},
			want: want{
				result: GetCertificateResponse{},
				err:    fmt.Errorf(errGetDataToCertFailed, errBoom),
			},
		},
		"ShouldFailParsingResponse": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				http: &MockHttpClient{
					MockSendRequest: func(ctx context.Context, method string, url string, body string, headers map[string][]string, skipTLSVerify bool, timeout time.Duration) (resp httpClient.Response, err error) {
						return httpClient.Response{
							Body:       `{ "83729jsdjd92819w1yhdsduy288yhduwdbd"}`,
							Headers:    nil,
							StatusCode: 200,
						}, nil
					},
				},
			},
			want: want{
				result: GetCertificateResponse{},
				err:    fmt.Errorf(errFailedToUnmarshalBody, errBodyNotJson),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cc := &client{
				log:              logr.Logger{},
				localHttpClient:  tc.args.http,
				timeout:          timeout,
				apiEndpoint:      apiEndpoint,
				downloadEndpoint: downloadEndpoint,
				token:            token,
			}

			got, gotErr := cc.GetCertificate(context.Background(), tc.args.certificate)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("GetCertificate(...): -want error, +got error: %v", diff)
			}
			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("GetCertificate(...): -want result, +got result: %v", diff)
			}
		})
	}
}
