/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/dana-team/certificate-operator/internal/common"

	v1alpha1 "github.com/dana-team/certificate-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	errCertificatesExist            = "cannot delete CertificateConfig because associated Certificates exist"
	errFailedToGetCertificateConfig = "failed to get CertificateConfig %q: %v"
	errFailedToGetSecret            = "failed to get secret: %v"
	errSettingFinalizer             = "error occurred while setting the finalizers of the CertificateConfig resource: %v"
	errDeletingFinalizer            = "error occurred while deleting the finalizers of the CertificateConfig resource"
	errListingCertificates          = "failed to list Certificates: %v"
)

const (
	dependenciesFinalizer = "cert.dana.io/check-dependencies"
)

// CertificateConfigReconciler reconciles a CertificateConfig object
type CertificateConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cert.dana.io,resources=certificateconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cert.dana.io,resources=certificateconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cert.dana.io,resources=certificateconfigs/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &v1alpha1.Certificate{}, "spec.configRef.Name", func(obj client.Object) []string {
		return []string{obj.(*v1alpha1.Certificate).Spec.ConfigRef.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CertificateConfig{}).
		Complete(r)
}

// Reconcile handles reconciliation of CertificateConfig objects.
func (r *CertificateConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = r.Log.WithValues("certificateConfig", req.Name)
	r.Log.Info("Starting Reconcile")

	certificateConfig := &v1alpha1.CertificateConfig{}
	if err := r.Get(ctx, req.NamespacedName, certificateConfig); err != nil {
		return ctrl.Result{}, fmt.Errorf(errFailedToGetCertificateConfig, req.Name, err)
	}

	_, err := common.GetSecret(r.Client, ctx, certificateConfig.Spec.SecretRef.Name, certificateConfig.Spec.SecretRef.Namespace)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(errFailedToGetSecret, err)
	}

	err = r.setFinalizers(ctx, certificateConfig)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(errSettingFinalizer, err)
	}

	err = r.handleDelete(ctx, certificateConfig, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// setFinalizers sets the finalizers on the CertificateConfig if it has not been marked for deletion and the finalizers need updating.
// It returns an error if the update operation fails.
func (r *CertificateConfigReconciler) setFinalizers(ctx context.Context, certificateConfig *v1alpha1.CertificateConfig) error {
	controllerutil.AddFinalizer(certificateConfig, dependenciesFinalizer)
	if err := r.Update(ctx, certificateConfig); err != nil {
		r.Log.Error(err, errSettingFinalizer)
		return err
	}

	return nil
}

// handleDelete checks if the CertificateConfig has been marked for deletion and performs cleanup if necessary.
// It returns an error if any operation fails.
func (r *CertificateConfigReconciler) handleDelete(ctx context.Context, certificateConfig *v1alpha1.CertificateConfig, name string) error {
	if !certificateConfig.GetDeletionTimestamp().IsZero() {
		r.Log.Info("deletion detected! Proceeding to cleanup the finalizers...")

		err := r.shouldRemoveFinalizer(ctx, name)
		if err != nil {
			return err
		}

		err = r.removeFinalizer(ctx, certificateConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

// removeFinalizer removes the finalizer, and updates the CertificateConfig accordingly.
// It returns an error if any operation fails.
func (r *CertificateConfigReconciler) removeFinalizer(ctx context.Context, certificateConfig *v1alpha1.CertificateConfig) error {
	controllerutil.RemoveFinalizer(certificateConfig, dependenciesFinalizer)
	if err := r.Update(ctx, certificateConfig); err != nil {
		return errors.New(errDeletingFinalizer)
	}

	r.Log.Info("cleaned up the '" + dependenciesFinalizer + "' finalizer successfully")
	return nil
}

// shouldRemoveFinalizer checks if there are associated Certificates with the CertificateConfig, if there are, returns false, otherwise returns true
// It returns an error if any operation fails.
func (r *CertificateConfigReconciler) shouldRemoveFinalizer(ctx context.Context, name string) error {
	certificateList := &v1alpha1.CertificateList{}
	if err := r.Client.List(ctx, certificateList, client.MatchingFields{"spec.configRef.Name": name}); err != nil {
		return fmt.Errorf(errListingCertificates, err)
	}

	if len(certificateList.Items) > 0 {
		r.Log.Info(fmt.Sprintf("found %d associated Certificates", len(certificateList.Items)))
		return fmt.Errorf(errCertificatesExist)
	}

	return nil
}
