package cert

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/dana-team/certificate-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testAPIEndpoint      = "https://api.endpoint"
	testDownloadEndpoint = "https://download.endpoint"
	testToken            = "dummy-token"
	testTimeout          = 2 * time.Minute
)

const (
	withAPIEndpoint      = "WithAPIEndpoint"
	withDownloadEndpoint = "WithDownloadEndpoint"
	withToken            = "WithToken"
	withTimeout          = "WithTimeout"
)

func TestClientOptions(t *testing.T) {
	type args struct {
		name   string
		option func(*client)
	}
	type want struct {
		value interface{}
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldCreateSuccessfullyWithAPIEndpoint": {
			args: args{
				name:   withAPIEndpoint,
				option: WithAPIEndpoint(testAPIEndpoint),
			},
			want: want{
				value: testAPIEndpoint,
			},
		},
		"ShouldCreateSuccessfullyWithDownloadEndpoint": {
			args: args{
				name:   withDownloadEndpoint,
				option: WithDownloadEndpoint(testDownloadEndpoint),
			},
			want: want{
				value: testDownloadEndpoint,
			},
		},
		"ShouldCreateSuccessfullyWithToken": {
			args: args{
				name:   withToken,
				option: WithToken(testToken),
			},
			want: want{
				value: testToken,
			},
		},
		"ShouldCreateSuccessfullyWithTimeout": {
			args: args{
				name:   withTimeout,
				option: WithTimeout(testTimeout),
			},
			want: want{
				value: testTimeout,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cl := NewClient(logr.Logger{}, tc.args.option)
			switch tc.args.name {
			case withAPIEndpoint:
				if diff := cmp.Diff(tc.want.value, cl.(*client).apiEndpoint, test.EquateErrors()); diff != "" {
					t.Fatalf("createClient(...): -want error, +got error: %v", diff)
				}
			case withDownloadEndpoint:
				if diff := cmp.Diff(tc.want.value, cl.(*client).downloadEndpoint, test.EquateErrors()); diff != "" {
					t.Fatalf("createClient(...): -want error, +got error: %v", diff)
				}
			case withToken:
				if diff := cmp.Diff(tc.want.value, cl.(*client).token, test.EquateErrors()); diff != "" {
					t.Fatalf("createClient(...): -want error, +got error: %v", diff)
				}
			case withTimeout:
				if diff := cmp.Diff(tc.want.value, cl.(*client).timeout, test.EquateErrors()); diff != "" {
					t.Fatalf("createClient(...): -want error, +got error: %v", diff)
				}
			}

		})
	}
}

func Test_getWaitTimeout(t *testing.T) {
	type args struct {
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		value time.Duration
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldSetCustomTimeout": {
			args: args{
				certificateConfig: &v1alpha1.CertificateConfig{
					Spec: v1alpha1.CertificateConfigSpec{
						WaitTimeout: &metav1.Duration{Duration: testTimeout},
					},
				},
			},
			want: want{
				value: 2 * time.Minute,
			},
		},
		"ShouldSetDefaultTimeout": {
			args: args{
				certificateConfig: &v1alpha1.CertificateConfig{
					Spec: v1alpha1.CertificateConfigSpec{
						WaitTimeout: &metav1.Duration{Duration: testTimeout},
					},
				},
			},
			want: want{
				value: 2 * time.Minute,
			},
		},
	}

	tests := []struct {
		name                string
		certificateConfig   *v1alpha1.CertificateConfig
		expectedWaitTimeout time.Duration
	}{
		{
			name: "WithCustomTimeout",
			certificateConfig: &v1alpha1.CertificateConfig{
				Spec: v1alpha1.CertificateConfigSpec{
					WaitTimeout: nil,
				},
			},
			expectedWaitTimeout: defaultWaitTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedWaitTimeout, getWaitTimeout(tt.certificateConfig))
		})
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			gotValue := getWaitTimeout(tc.args.certificateConfig)
			if diff := cmp.Diff(tc.want.value, gotValue, test.EquateErrors()); diff != "" {
				t.Fatalf("getWaitTimeout(...): -want value, +got value: %v", diff)
			}
		})
	}
}

func Test_NewClientFromCertificateConfigAndSecretData(t *testing.T) {
	type args struct {
		credentials map[string]string
	}
	type want struct {
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldCreateClientSuccuessfully": {
			args: args{
				credentials: map[string]string{
					keyAPIEndpoint:      testAPIEndpoint,
					keyDownloadEndpoint: testDownloadEndpoint,
					keyToken:            testToken,
				},
			},
			want: want{
				err: nil,
			},
		},
		"ShouldFailWithMissingAPIEndpoint": {
			args: args{
				credentials: map[string]string{
					keyDownloadEndpoint: testDownloadEndpoint,
					keyToken:            testToken,
				},
			},
			want: want{
				err: errors.New(errMissingAPIEndpoint),
			},
		},
		"ShouldFailWithMissingDownloadEndpoint": {
			args: args{
				credentials: map[string]string{
					keyAPIEndpoint: testAPIEndpoint,
					keyToken:       testToken,
				},
			},
			want: want{
				err: errors.New(errMissingDownloadEndpoint),
			},
		},
		"ShouldFailWithMissingToken": {
			args: args{
				credentials: map[string]string{
					keyAPIEndpoint:      testAPIEndpoint,
					keyDownloadEndpoint: testDownloadEndpoint,
				},
			},
			want: want{
				err: errors.New(errMissingToken),
			},
		},
	}

	for name, tc := range cases {
		certConfig := &v1alpha1.CertificateConfig{}

		t.Run(name, func(t *testing.T) {
			credentialsJSON, err := json.Marshal(tc.args.credentials)
			if err != nil {
				t.Fatalf("Failed to marshal credentials: %v", err)
			}

			secretData := map[string][]byte{
				keyCredentials: credentialsJSON,
			}

			_, gotErr := NewClientFromCertificateConfigAndSecretData(logr.Logger{}, certConfig, secretData)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("getSecret(...): -want error, +got error: %v", diff)
			}
		})
	}
}
