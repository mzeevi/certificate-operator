package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/dana-team/certificate-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	errorspkg "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	finalizers = []string{dependenciesFinalizer}
)

func Test_setFinalizers(t *testing.T) {
	type args struct {
		localKube         client.Client
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldSetFinalizerSuccessfully": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(nil),
				},
			},
			want: want{
				err: nil,
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateConfigReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
		}

		t.Run(name, func(t *testing.T) {
			gotErr := r.setFinalizers(context.Background(), tc.args.certificateConfig)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("setFinalizers(...): -want error, +got error: %v", diff)
			}
		})
	}
}

func Test_handleDelete(t *testing.T) {
	deletionTime := metav1.NewTime(time.Now())
	certificateConfig.ObjectMeta.DeletionTimestamp = &deletionTime
	certificateConfig.ObjectMeta.Finalizers = finalizers

	type args struct {
		localKube         client.Client
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldHandleDeleteSuccessfully": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(nil),
					MockList:   test.NewMockListFn(nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"ShouldFailHandleDelete": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(nil),
					MockList:   test.NewMockListFn(errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errListingCertificates, errBoom),
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateConfigReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
		}

		t.Run(name, func(t *testing.T) {
			gotErr := r.handleDelete(context.Background(), tc.args.certificateConfig, tc.args.certificateConfig.Name)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("handleDelete(...): -want error, +got error: %v", diff)
			}
		})
	}
}

func Test_removeFinalizer(t *testing.T) {
	deletionTime := metav1.NewTime(time.Now())
	certificateConfig.ObjectMeta.DeletionTimestamp = &deletionTime
	certificateConfig.ObjectMeta.Finalizers = finalizers

	type args struct {
		localKube         client.Client
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldCleanupFinalizerSuccessfully": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(nil),
					MockList:   test.NewMockListFn(nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"ShouldFailListingCertificates": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(nil),
					MockList:   test.NewMockListFn(errBoom),
				},
			},
			want: want{
				err: nil,
			},
		},
		"ShouldFailRemoveFinalizer": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(errBoom),
					MockList:   test.NewMockListFn(nil),
				},
			},
			want: want{
				err: errorspkg.New(errDeletingFinalizer),
			},
		},
		"ShouldNotRemoveFinalizer": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(errBoom),
					MockList: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
						certList, ok := list.(*v1alpha1.CertificateList)
						if !ok {
							return errors.New("object list is not a Certificates list")
						}

						*certList = v1alpha1.CertificateList{
							Items: []v1alpha1.Certificate{
								certificate,
							},
						}
						return nil
					},
				},
			},
			want: want{
				err: errorspkg.New(errDeletingFinalizer),
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateConfigReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
		}

		t.Run(name, func(t *testing.T) {
			gotErr := r.removeFinalizer(context.Background(), tc.args.certificateConfig)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("removeFinalizer(...): -want error, +got error: %v", diff)
			}
		})
	}
}

func Test_shouldRemoveFinalizer(t *testing.T) {
	type args struct {
		localKube         client.Client
		certificateConfig *v1alpha1.CertificateConfig
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnNoError": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(nil),
					MockList:   test.NewMockListFn(nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"ShouldFailListingCertificates": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(nil),
					MockList:   test.NewMockListFn(errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errListingCertificates, errBoom),
			},
		},
		"ShouldNotRemoveFinalizer": {
			args: args{
				certificateConfig: &certificateConfig,
				localKube: &test.MockClient{
					MockUpdate: test.NewMockUpdateFn(errBoom),
					MockList: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
						certList, ok := list.(*v1alpha1.CertificateList)
						if !ok {
							return errors.New("object list is not a Certificates list")
						}

						*certList = v1alpha1.CertificateList{
							Items: []v1alpha1.Certificate{
								certificate,
							},
						}
						return nil
					},
				},
			},
			want: want{
				err: fmt.Errorf(errCertificatesExist),
			},
		},
	}
	for name, tc := range cases {
		r := &CertificateConfigReconciler{
			Client: tc.args.localKube,
			Scheme: runtime.NewScheme(),
			Log:    logr.Logger{},
		}

		t.Run(name, func(t *testing.T) {
			gotErr := r.shouldRemoveFinalizer(context.Background(), tc.args.certificateConfig.Name)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("shouldRemoveFinalizer(...): -want error, +got error: %v", diff)
			}
		})
	}
}
