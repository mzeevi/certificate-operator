package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/dana-team/certificate-operator/api/v1alpha1"
	"github.com/dana-team/certificate-operator/internal/certhandler"
	"github.com/dana-team/certificate-operator/internal/clients/cert"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MockPostCertificateFn func(ctx context.Context, certificate *v1alpha1.Certificate) (string, error)
type MockDownloadCertificateFn func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.DownloadCertificateResponse, error)
type MockGetCertificateFn func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.GetCertificateResponse, error)

var (
	errBoom                = errors.New("boom")
	errParsingDate         = errors.New(`parsing time "2024-10-1888T09:05:22" as "2006-01-02T15:04:05": cannot parse "88T09:05:22" as "T"`)
	errCannotDecodeB64Data = errors.New("cannot decode base64-encoded PKCS#12 data")
	validCertKey           = []byte(`-----BEGIN CERTIFICATE-----`)
	validPrivateKey        = []byte(`-----BEGIN PRIVATE KEY-----`)
)

const guid = "guid"

type MockCertClient struct {
	MockPostCertificate     MockPostCertificateFn
	MockDownloadCertificate MockDownloadCertificateFn
	MockGetCertificate      MockGetCertificateFn
}

func (c *MockCertClient) PostCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (string, error) {
	return c.MockPostCertificate(ctx, certificate)
}

func (c *MockCertClient) DownloadCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (cert.DownloadCertificateResponse, error) {
	return c.MockDownloadCertificate(ctx, certificate)
}

func (c *MockCertClient) GetCertificate(ctx context.Context, certificate *v1alpha1.Certificate) (cert.GetCertificateResponse, error) {
	return c.MockGetCertificate(ctx, certificate)
}

var (
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
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: "v1alpha1",
		},
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

func condition(reason string, err error) metav1.Condition {
	return metav1.Condition{
		Type:    ConditionError,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: err.Error(),
	}
}

func Test_issueCertificate(t *testing.T) {
	type args struct {
		localKube         client.Client
		certClient        cert.Client
		certificate       *v1alpha1.Certificate
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		condition metav1.Condition
		err       error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldIssueCertificateSuccessfully": {
			args: args{
				certificate:       &certificate,
				certificateConfig: &certificateConfig,
				certClient: &MockCertClient{
					MockPostCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (string, error) {
						return guid, nil
					},
					MockGetCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.GetCertificateResponse, error) {
						return cert.GetCertificateResponse{
							ValidTo:                "2024-10-18T09:05:22",
							ValidFrom:              "2024-04-18T09:05:22",
							SignatureHashAlgorithm: "sha384",
						}, nil
					},
				},
				localKube: &test.MockClient{
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(nil),
				},
			},
			want: want{
				condition: metav1.Condition{},
				err:       nil,
			},
		},
		"ShouldFailCreatingCertificate": {
			args: args{
				certificate:       &certificate,
				certificateConfig: &certificateConfig,
				certClient: &MockCertClient{
					MockPostCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (string, error) {
						return "", errBoom
					},
				},
				localKube: &test.MockClient{
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(nil),
				},
			},
			want: want{
				condition: condition(ConditionPostToCertAPIFailed, errBoom),
				err:       fmt.Errorf(errCreationFailed, errBoom),
			},
		},
		"ShouldFailUpdatingStatus": {
			args: args{
				certificate:       &certificate,
				certificateConfig: &certificateConfig,
				certClient: &MockCertClient{
					MockPostCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (string, error) {
						return guid, nil
					},
				},
				localKube: &test.MockClient{
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(errBoom),
				},
			},
			want: want{
				condition: condition(ConditionUpdateStatusFailed, errBoom),
				err:       fmt.Errorf(errCreationFailed, errBoom),
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
			CertClientBuilder: func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (cert.Client, error) {
				return &MockCertClient{}, nil
			},
		}

		t.Run(name, func(t *testing.T) {
			errCondition, gotErr := r.issueCertificate(context.Background(), tc.args.certClient, tc.args.certificate)
			if diff := cmp.Diff(tc.want.condition, errCondition); diff != "" {
				t.Fatalf("issueCertificate(...): -want result, +got result: %v", diff)
			}

			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("issueCertificate(...): -want error, +got error: %v", diff)
			}
		})
	}
}

func Test_obtainCertificateData(t *testing.T) {
	type args struct {
		localKube         client.Client
		certificate       *v1alpha1.Certificate
		certClient        cert.Client
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		validTo                string
		validFrom              string
		signatureHashAlgorithm string
		condition              metav1.Condition
		err                    error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldGetCertDataSuccessfully": {
			args: args{
				certificate:       &certificate,
				certificateConfig: &certificateConfig,
				certClient: &MockCertClient{
					MockPostCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (string, error) {
						return guid, nil
					},
					MockGetCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.GetCertificateResponse, error) {
						return cert.GetCertificateResponse{
							ValidTo:                "2024-10-18T09:05:22",
							ValidFrom:              "2024-04-18T09:05:22",
							SignatureHashAlgorithm: "sha384",
						}, nil
					},
				},
				localKube: &test.MockClient{
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(nil),
				},
			},
			want: want{
				validTo:                "2024-10-18T09:05:22",
				validFrom:              "2024-04-18T09:05:22",
				signatureHashAlgorithm: "sha384",
				condition:              metav1.Condition{},
				err:                    nil,
			},
		},
		"ShouldFailGettingCertData": {
			args: args{
				certificate:       &certificate,
				certificateConfig: &certificateConfig,
				certClient: &MockCertClient{
					MockGetCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.GetCertificateResponse, error) {
						return cert.GetCertificateResponse{}, errBoom
					},
				},
				localKube: &test.MockClient{},
			},
			want: want{
				validTo:                "",
				validFrom:              "",
				signatureHashAlgorithm: "",
				condition:              condition(ConditionGetCertDataFromCertAPIFailed, errBoom),
				err:                    errBoom,
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
			CertClientBuilder: func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (cert.Client, error) {
				return &MockCertClient{}, nil
			},
		}

		t.Run(name, func(t *testing.T) {
			validTo, validFrom, signatureHashAlgorithm, condition, gotErr := r.obtainCertificateData(context.Background(), tc.args.certClient, tc.args.certificate)
			if diff := cmp.Diff(tc.want.validTo, validTo); diff != "" {
				t.Fatalf("obtainCertificateData(...): -want validTo, +got validTo: %v", diff)
			}

			if diff := cmp.Diff(tc.want.validFrom, validFrom); diff != "" {
				t.Fatalf("obtainCertificateData(...): -want validFrom, +got validFrom: %v", diff)
			}

			if diff := cmp.Diff(tc.want.signatureHashAlgorithm, signatureHashAlgorithm); diff != "" {
				t.Fatalf("obtainCertificateData(...): -want signatureHashAlgorithm, +got signatureHashAlgorithm: %v", diff)
			}

			if diff := cmp.Diff(tc.want.condition, condition); diff != "" {
				t.Fatalf("obtainCertificateData(...): -want result, +got result: %v", diff)
			}

			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("obtainCertificateData(...): -want error, +got error: %v", diff)
			}
		})
	}
}

func Test_updateCertValidity(t *testing.T) {
	type args struct {
		localKube         client.Client
		certClient        cert.Client
		certificate       *v1alpha1.Certificate
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		condition metav1.Condition
		err       error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldUpdateCertValiditySuccessfully": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				certClient: &MockCertClient{
					MockGetCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.GetCertificateResponse, error) {
						return cert.GetCertificateResponse{
							ValidTo:                "2024-10-18T09:05:22",
							ValidFrom:              "2024-04-18T09:05:22",
							SignatureHashAlgorithm: "sha384",
						}, nil
					},
				},
				localKube: &test.MockClient{
					MockUpdate:       test.NewMockUpdateFn(nil),
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"ShouldFailParsingValidTo": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				certClient: &MockCertClient{
					MockGetCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.GetCertificateResponse, error) {
						return cert.GetCertificateResponse{
							ValidTo:                "2024-10-1888T09:05:22",
							ValidFrom:              "2024-04-18T09:05:22",
							SignatureHashAlgorithm: "sha384",
						}, nil
					},
				},
				localKube: &test.MockClient{
					MockUpdate:       test.NewMockUpdateFn(nil),
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(nil),
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						secret, ok := obj.(*corev1.Secret)
						if !ok {
							return errors.New("object is not a Secret")
						}

						*secret = corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      certificateConfig.Spec.SecretRef.Name,
								Namespace: certificateConfig.Spec.SecretRef.Namespace},
							Data: map[string][]byte{
								"token": []byte("value"),
							},
						}
						return nil
					},
				},
			},
			want: want{
				condition: condition(ConditionParseValidToFailed, errParsingDate),
				err:       fmt.Errorf(errFailedParseValidTo, errParsingDate),
			},
		},
		"ShouldFailParsingValidFrom": {
			args: args{
				certificateConfig: &certificateConfig,
				certificate:       &certificate,
				certClient: &MockCertClient{
					MockGetCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.GetCertificateResponse, error) {
						return cert.GetCertificateResponse{
							ValidTo:                "2024-10-18T09:05:22",
							ValidFrom:              "2024-10-1888T09:05:22",
							SignatureHashAlgorithm: "sha384",
						}, nil
					},
				},
				localKube: &test.MockClient{
					MockUpdate:       test.NewMockUpdateFn(nil),
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(nil),
				},
			},
			want: want{
				condition: condition(ConditionParseValidFromFailed, errParsingDate),
				err:       fmt.Errorf(errFailedParseValidFrom, errParsingDate),
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
			CertClientBuilder: func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (cert.Client, error) {
				return &MockCertClient{}, nil
			},
		}

		t.Run(name, func(t *testing.T) {
			condition, gotErr := r.updateCertValidity(context.Background(), tc.args.certClient, tc.args.certificate)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("updateCertValidity(...): -want error, +got error: %v", diff)
			}

			if diff := cmp.Diff(tc.want.condition, condition); diff != "" {
				t.Fatalf("updateCertValidity(...): -want result, +got result: %v", diff)
			}
		})
	}
}

func Test_downloadCert(t *testing.T) {
	type args struct {
		localKube         client.Client
		certClient        cert.Client
		certificate       *v1alpha1.Certificate
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		tlsData   certhandler.TLSData
		condition metav1.Condition
		err       error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldDownloadCertSuccessfully": {
			args: args{
				certificate:       &certificate,
				certificateConfig: &certificateConfig,
				certClient: &MockCertClient{
					MockDownloadCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.DownloadCertificateResponse, error) {
						return cert.DownloadCertificateResponse{
							Data:     "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",
							Password: "jtvdDUG0E7Ll",
						}, nil
					},
				},
				localKube: &test.MockClient{},
			},
			want: want{
				condition: metav1.Condition{},
				tlsData: certhandler.TLSData{
					CertificateBytes: []byte(`-----BEGIN CERTIFICATE-----`),
					PrivateKeyBytes:  []byte(`-----BEGIN PRIVATE KEY-----`),
				},
				err: nil,
			},
		},
		"ShouldFailToDecodeCert": {
			args: args{
				certificate:       &certificate,
				certificateConfig: &certificateConfig,
				certClient: &MockCertClient{
					MockDownloadCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.DownloadCertificateResponse, error) {
						return cert.DownloadCertificateResponse{
							Data:     "wrong-data",
							Password: "wrong-password",
						}, nil
					},
				},
				localKube: &test.MockClient{},
			},
			want: want{
				condition: condition(ConditionDecodeCertFailed, errors.New(errCannotDecodeB64Data.Error()+": illegal base64 data at input byte 5")),
				tlsData:   certhandler.TLSData{},
				err:       errors.New("failed downloading certificate: cannot decode base64-encoded PKCS#12 data: illegal base64 data at input byte 5"),
			},
		},
		"ShouldFailDownloadCert": {
			args: args{
				certificate:       &certificate,
				certificateConfig: &certificateConfig,
				certClient: &MockCertClient{
					MockDownloadCertificate: func(ctx context.Context, certificate *v1alpha1.Certificate) (cert.DownloadCertificateResponse, error) {
						return cert.DownloadCertificateResponse{}, errBoom
					},
				},
				localKube: &test.MockClient{},
			},
			want: want{
				condition: condition(ConditionDownloadCertFromCertAPIFailed, errBoom),
				tlsData:   certhandler.TLSData{},
				err:       fmt.Errorf(errFailedDownloadingCertificate, errBoom),
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
			CertClientBuilder: func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (cert.Client, error) {
				return &MockCertClient{}, nil
			},
		}

		t.Run(name, func(t *testing.T) {
			tlsData, errCondition, gotErr := r.downloadCert(context.Background(), tc.args.certClient, tc.args.certificate)
			if !bytes.Contains(tlsData.CertificateBytes, tc.want.tlsData.CertificateBytes) {
				t.Fatalf("downloadCert(...): expected certificate bytes not found in result")
			}

			if diff := cmp.Diff(tc.want.condition, errCondition); diff != "" {
				t.Fatalf("downloadCert(...): -want result, +got result: %v", diff)
			}

			if gotErr != nil {
				if diff := cmp.Diff(tc.want.err.Error(), gotErr.Error()); diff != "" {
					t.Fatalf("downloadCert(...): -want error, +got error: %v", diff)
				}
			}
		})
	}
}

func Test_isSecretUpToDate(t *testing.T) {
	type args struct {
		localKube   client.Client
		certificate *v1alpha1.Certificate
	}
	type want struct {
		result bool
		err    error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnSecretIsUpToDate": {
			args: args{
				certificate: &v1alpha1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert",
						Namespace: "default",
					},
					Spec: v1alpha1.CertificateSpec{
						SecretName: "my-secret",
					},
					Status: v1alpha1.CertificateStatus{
						SecretName: "my-secret",
					},
				},
				localKube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
			},
			want: want{
				result: true,
				err:    nil,
			},
		},
		"ShouldFailGettingSecret": {
			args: args{
				certificate: &v1alpha1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert",
						Namespace: "default",
					},
					Spec: v1alpha1.CertificateSpec{
						SecretName: "my-secret",
					},
					Status: v1alpha1.CertificateStatus{
						SecretName: "my-secret-new",
					},
				},
				localKube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
			},
			want: want{
				result: false,
				err:    errBoom,
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
			CertClientBuilder: func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (cert.Client, error) {
				return &MockCertClient{}, nil
			},
		}

		t.Run(name, func(t *testing.T) {
			result, gotErr := r.isSecretUpToDate(context.Background(), tc.args.certificate, tc.args.certificate.Namespace)
			if diff := cmp.Diff(tc.want.result, result); diff != "" {
				t.Fatalf("isSecretUpToDate(...): -want result, +got result: %v", diff)
			}

			if gotErr != nil {
				if diff := cmp.Diff(tc.want.err.Error(), gotErr.Error()); diff != "" {
					t.Fatalf("isSecretUpToDate(...): -want error, +got error: %v", diff)
				}
			}
		})
	}
}

func Test_isSecretNameChanged(t *testing.T) {
	type args struct {
		certificate *v1alpha1.Certificate
	}
	type want struct {
		result bool
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnSecretIsChanged": {
			args: args{
				certificate: &v1alpha1.Certificate{
					Spec: v1alpha1.CertificateSpec{
						SecretName: "my-secret",
					},
					Status: v1alpha1.CertificateStatus{
						SecretName: "my-secret-new",
					},
				},
			},
			want: want{
				result: true,
			},
		},
		"ShouldReturnSecretIsNotChanged": {
			args: args{
				certificate: &v1alpha1.Certificate{
					Spec: v1alpha1.CertificateSpec{
						SecretName: "my-secret",
					},
					Status: v1alpha1.CertificateStatus{
						SecretName: "my-secret",
					},
				},
			},
			want: want{
				result: false,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := isSecretNameChanged(tc.args.certificate)
			if diff := cmp.Diff(tc.want.result, result); diff != "" {
				t.Fatalf("isSecretNameChanged(...): -want result, +got result: %v", diff)
			}
		})
	}
}

func Test_isSecretDeleted(t *testing.T) {
	type args struct {
		localKube   client.Client
		certificate *v1alpha1.Certificate
	}
	type want struct {
		result bool
		err    error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnSecretIsUpToDate": {
			args: args{
				certificate: &v1alpha1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert",
						Namespace: "default",
					},
					Status: v1alpha1.CertificateStatus{
						SecretName: "my-secret",
					},
				},
				localKube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
			},
			want: want{
				result: false,
				err:    nil,
			},
		},
		"ShouldFailGettingSecret": {
			args: args{
				certificate: &v1alpha1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert",
						Namespace: "default",
					},
					Status: v1alpha1.CertificateStatus{
						SecretName: "my-secret-new",
					},
				},
				localKube: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			want: want{
				result: false,
				err:    errBoom,
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
			CertClientBuilder: func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (cert.Client, error) {
				return &MockCertClient{}, nil
			},
		}

		t.Run(name, func(t *testing.T) {
			result, gotErr := r.isSecretDeleted(context.Background(), tc.args.certificate, tc.args.certificate.Namespace)
			if diff := cmp.Diff(tc.want.result, result); diff != "" {
				t.Fatalf("isSecretDeleted(...): -want result, +got result: %v", diff)
			}

			if gotErr != nil {
				if diff := cmp.Diff(tc.want.err.Error(), gotErr.Error()); diff != "" {
					t.Fatalf("isSecretDeleted(...): -want error, +got error: %v", diff)
				}
			}
		})
	}
}

func Test_hasNotFoundErrorCondition(t *testing.T) {
	type args struct {
		certificate *v1alpha1.Certificate
		localKube   client.Client
	}
	type want struct {
		result bool
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldHaveNotFoundCondition": {
			args: args{
				certificate: &v1alpha1.Certificate{
					Status: v1alpha1.CertificateStatus{
						Conditions: []metav1.Condition{
							{
								Type:    ConditionError,
								Status:  metav1.ConditionTrue,
								Reason:  ConditionGetCertDataFromCertAPIFailed,
								Message: http.StatusText(http.StatusNotFound),
							},
						},
					},
				},
			},
			want: want{
				result: true,
			},
		},
		"ShouldNotHaveNotFoundCondition": {
			args: args{
				certificate: &v1alpha1.Certificate{
					Status: v1alpha1.CertificateStatus{
						Conditions: []metav1.Condition{},
					},
				},
			},
			want: want{
				result: false,
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
			CertClientBuilder: func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (cert.Client, error) {
				return &MockCertClient{}, nil
			},
		}

		t.Run(name, func(t *testing.T) {
			result := r.hasNotFoundErrorCondition(tc.args.certificate)
			if diff := cmp.Diff(tc.want.result, result); diff != "" {
				t.Fatalf("hasNotFoundErrorCondition(...): -want result, +got result: %v", diff)
			}
		})
	}
}

func Test_createOrUpdateTlsSecret(t *testing.T) {
	type args struct {
		localKube   client.Client
		certClient  cert.Client
		certificate *v1alpha1.Certificate
		tlsData     certhandler.TLSData
		namespace   string
	}
	type want struct {
		condition metav1.Condition
		err       error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldCreateSecretSuccessfully": {
			args: args{
				certificate: &certificate,
				namespace:   "default",
				tlsData: certhandler.TLSData{
					CertificateBytes: []byte(`-----BEGIN CERTIFICATE-----`),
					PrivateKeyBytes:  []byte(`-----BEGIN PRIVATE KEY-----`),
				},
				certClient: &MockCertClient{},
				localKube: &test.MockClient{
					MockUpdate:       test.NewMockUpdateFn(nil),
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(nil),
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						secret, ok := obj.(*corev1.Secret)
						if !ok {
							return errors.New("object is not a Secret")
						}

						*secret = corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      certificate.Spec.SecretName,
								Namespace: certificate.Namespace,
							},
							Type: corev1.SecretTypeTLS,
							Data: map[string][]byte{
								corev1.TLSCertKey:       validCertKey,
								corev1.TLSPrivateKeyKey: validPrivateKey,
							},
						}
						return nil
					},
				},
			},
			want: want{
				condition: metav1.Condition{},
				err:       nil,
			},
		},
		"ShouldFailUpdatingSecretNameInStatus": {
			args: args{
				certificate: &certificate,
				namespace:   "default",
				tlsData: certhandler.TLSData{
					CertificateBytes: []byte(`-----BEGIN CERTIFICATE-----`),
					PrivateKeyBytes:  []byte(`-----BEGIN PRIVATE KEY-----`),
				},
				certClient: &MockCertClient{},
				localKube: &test.MockClient{
					MockUpdate:       test.NewMockUpdateFn(nil),
					MockStatusUpdate: test.NewMockSubResourceUpdateFn(errBoom),
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						secret, ok := obj.(*corev1.Secret)
						if !ok {
							return errors.New("object is not a Secret")
						}

						*secret = corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      certificate.Spec.SecretName,
								Namespace: certificate.Namespace,
							},
							Type: corev1.SecretTypeTLS,
							Data: map[string][]byte{
								corev1.TLSCertKey:       validCertKey,
								corev1.TLSPrivateKeyKey: validPrivateKey,
							},
						}
						return nil
					},
				},
			},
			want: want{
				condition: condition(ConditionUpdateStatusFailed, errBoom),
				err:       fmt.Errorf(errUpdateStatus, errBoom),
			},
		},
		"ShouldFailSettingOwnerRef": {
			args: args{
				certificate: &certificate,
				namespace:   "different-namespace",
				tlsData: certhandler.TLSData{
					CertificateBytes: []byte(`-----BEGIN CERTIFICATE-----`),
					PrivateKeyBytes:  []byte(`-----BEGIN PRIVATE KEY-----`),
				},
				certClient: &MockCertClient{},
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(nil),
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						secret, ok := obj.(*corev1.Secret)
						if !ok {
							return errors.New("object is not a Secret")
						}

						*secret = corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      certificate.Spec.SecretName,
								Namespace: certificate.Namespace,
							},
							Type: corev1.SecretTypeTLS,
							Data: map[string][]byte{
								corev1.TLSCertKey:       validCertKey,
								corev1.TLSPrivateKeyKey: validPrivateKey,
							},
						}
						return nil
					},
				},
			},
			want: want{
				condition: condition(ConditionSetOwnerRefFailed, errors.New("cross-namespace owner references are disallowed, owner's namespace default, obj's namespace different-namespace")),
				err:       errors.New("failed to set owner reference for secret my-secret-new%!(EXTRA *errors.errorString=cross-namespace owner references are disallowed, owner's namespace default, obj's namespace different-namespace)"),
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateReconciler{
			Client: tc.args.localKube,
			Scheme: newScheme(),
			Log:    logr.Logger{},
			CertClientBuilder: func(logr.Logger, *v1alpha1.CertificateConfig, map[string][]byte) (cert.Client, error) {
				return &MockCertClient{}, nil
			},
		}

		t.Run(name, func(t *testing.T) {
			condition, gotErr := r.createOrUpdateTlsSecret(context.Background(), tc.args.certificate, tc.args.tlsData, tc.args.namespace)
			if gotErr != nil {
				if diff := cmp.Diff(tc.want.err.Error(), gotErr.Error()); diff != "" {
					t.Fatalf("createOrUpdateTlsSecret(...): -want error, +got error: %v", diff)
				}
			}
			if diff := cmp.Diff(tc.want.condition, condition); diff != "" {
				t.Fatalf("createOrUpdateTlsSecret(...): -want result, +got result: %v", diff)
			}
		})
	}
}

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = v1alpha1.AddToScheme(s)
	return s
}
