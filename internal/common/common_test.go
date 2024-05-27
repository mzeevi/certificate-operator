package common

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errBoom = errors.New("boom")

var (
	secretName      = "testSecret"
	secretNamespace = "testNS"
)

func Test_GetSecret(t *testing.T) {
	type args struct {
		localKube client.Client
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldGetSecretSuccessfully": {
			args: args{
				localKube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						secret, ok := obj.(*corev1.Secret)
						if !ok {
							return errors.New("object is not a Secret")
						}

						*secret = corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      secretName,
								Namespace: secretNamespace,
							},
							Data: map[string][]byte{
								"token": []byte("value"),
							},
						}
						return nil
					},
				},
			},
			want: want{
				err: nil,
			},
		},
		"ShouldFailToGetSecret": {
			args: args{
				localKube: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			want: want{
				err: errBoom,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, gotErr := GetSecret(tc.args.localKube, context.Background(), secretName, secretNamespace)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("getSecret(...): -want error, +got error: %v", diff)
			}
		})
	}
}
