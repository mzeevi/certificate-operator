package certhandler

import (
	"errors"

	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	v1alpha1 "github.com/dana-team/certificate-operator/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretName = "my-secret"
	namespace  = "default"
)

var (
	validCertKey    = []byte(`-----BEGIN CERTIFICATE-----`)
	validPrivateKey = []byte(`-----BEGIN PRIVATE KEY-----`)

	validSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       validCertKey,
			corev1.TLSPrivateKeyKey: validPrivateKey,
		},
	}
)

func Test_TlsSecret(t *testing.T) {
	type args struct {
		tlsData     TLSData
		certificate *v1alpha1.Certificate
		namespace   string
	}
	type want struct {
		secret *corev1.Secret
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnTlsSecret": {
			args: args{
				tlsData: TLSData{
					CertificateBytes: validCertKey,
					PrivateKeyBytes:  validPrivateKey,
				},
				certificate: &v1alpha1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert",
						Namespace: "default",
					},
					Spec: v1alpha1.CertificateSpec{
						SecretName: "my-created-secret",
					},
				},
				namespace: "default",
			},
			want: want{
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-created-secret",
						Namespace: "default",
					},
					Type: corev1.SecretTypeTLS,
					Data: map[string][]byte{
						corev1.TLSCertKey:       validCertKey,
						corev1.TLSPrivateKeyKey: validPrivateKey,
					},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			secret := TlsSecret(tc.args.tlsData, tc.args.certificate, tc.args.namespace)
			if diff := cmp.Diff(tc.want.secret, secret); diff != "" {
				t.Fatalf("TlsSecret(...): -want result, +got result: %v", diff)
			}
		})
	}
}

func Test_CreateOrUpdateTLSSecret(t *testing.T) {
	type args struct {
		localKube client.Client
		secret    *corev1.Secret
	}
	type want struct {
		tlsData TLSData
		err     error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldGetSuccessfully": {
			args: args{
				localKube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						secret, ok := obj.(*corev1.Secret)
						if !ok {
							return errors.New("object is not a Secret")
						}

						*secret = validSecret
						return nil
					},
					MockUpdate: test.NewMockUpdateFn(nil),
				},
				secret: &validSecret,
			},
			want: want{
				tlsData: TLSData{
					CertificateBytes: validPrivateKey,
					PrivateKeyBytes:  validPrivateKey,
				},
				err: nil,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := CreateOrUpdateTLSSecret(context.Background(), tc.args.localKube, tc.args.secret)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Fatalf("CreateOrUpdateTLSSecret(...): -want error, +got error: %v", diff)
			}
		})
	}
}
