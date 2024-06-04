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
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/dana-team/certificate-operator/internal/common"

	"github.com/dana-team/certificate-operator/internal/clients/cert"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "github.com/dana-team/certificate-operator/api/v1alpha1"
)

const (
	errCreationFailed               = "failed to create Certificate: %v"
	errGetFailed                    = "failed to get Certificate: %v"
	errFailedToSetOwnerRefForSecret = "failed to set owner reference for secret %v"
	errUpdateStatus                 = "failed to update Certificate status: %v"
	errFailedBuildingCertClient     = "failed to build Cert client: %v"
)

const (
	ConditionError                         = "Error"
	ConditionPostToCertAPIFailed           = "PostToCertAPIFailed"
	ConditionDownloadCertFromCertAPIFailed = "DownloadCertFromCertAPIFailed"
	ConditionGetCertDataFromCertAPIFailed  = "GetCertDataFromCertAPIFailed"
	ConditionUpdateStatusFailed            = "StatusUpdateFailed"
	ConditionDecodeCertFailed              = "DecodeCertFailed"
)

const (
	timeFormat = "2006-01-02T15:04:05"
)

const requeueAfterNotFoundError = time.Second * 5

// CertificateReconciler reconciles a Certificate object
type CertificateReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Log               logr.Logger
	CertClientBuilder cert.ClientBuilder
}

//+kubebuilder:rbac:groups=cert.dana.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cert.dana.io,resources=certificates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cert.dana.io,resources=certificates/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;create

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Certificate{}).
		Owns(&corev1.Secret{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return false
			},
			UpdateFunc: func(event.UpdateEvent) bool {
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return true
			},
			GenericFunc: func(event.GenericEvent) bool {
				return false
			},
		}).
		Complete(r)
}

// Reconcile handles reconciliation of Certificate objects.
func (r *CertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = r.Log.WithValues("certificate", req.NamespacedName)
	r.Log.Info("Starting Reconcile")

	certificate := &v1alpha1.Certificate{}
	if err := r.Client.Get(ctx, req.NamespacedName, certificate); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf(errGetFailed, err)
	}

	certificateConfig := &v1alpha1.CertificateConfig{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: certificate.Spec.ConfigRef.Name}, certificateConfig); err != nil {
		err = r.updateCertificateConditions(ctx, certificate, errorCondition("ConfigRetrievalFailed", err))
		if err != nil {
			return ctrl.Result{}, fmt.Errorf(errCreationFailed, err)
		}
		return ctrl.Result{}, fmt.Errorf(errCreationFailed, err)
	}

	secret, err := common.GetSecret(r.Client, ctx, certificateConfig.Spec.SecretRef.Name, certificateConfig.Spec.SecretRef.Namespace)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(errFailedToGetSecret, err)
	}

	certClient, err := r.CertClientBuilder(r.Log, certificateConfig, secret.Data)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(errFailedBuildingCertClient, err)
	}

	if isCertificateValid(certificate, certificateConfig) {
		if err := r.removeErrorConditions(ctx, certificate); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.forceExpirationUpdate(ctx, certClient, certificate, certificateConfig.Spec.ForceExpirationUpdate); err != nil {
			return ctrl.Result{}, err
		}

		if upToDate, err := r.isSecretUpToDate(ctx, certificate, req.Namespace); err != nil {
			return ctrl.Result{}, err
		} else if upToDate {
			return ctrl.Result{}, nil
		}
	}

	condition, err := r.issueCertificate(ctx, certClient, certificate)
	if err != nil {
		if updateErr := r.updateCertificateConditions(ctx, certificate, condition); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	condition, err = r.updateCertValidity(ctx, certClient, certificate)
	if err != nil {
		if updateErr := r.updateCertificateConditions(ctx, certificate, condition); updateErr != nil {
			return ctrl.Result{}, updateErr
		}

		if strings.Contains(err.Error(), http.StatusText(http.StatusNotFound)) {
			return ctrl.Result{RequeueAfter: requeueAfterNotFoundError}, err
		}

		return ctrl.Result{}, err
	}

	tlsData, condition, err := r.downloadCert(ctx, certClient, certificate)
	if err != nil {
		if updateErr := r.updateCertificateConditions(ctx, certificate, condition); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	condition, err = r.createOrUpdateTlsSecret(ctx, certificate, tlsData, req.Namespace)
	if err != nil {
		if updateErr := r.updateCertificateConditions(ctx, certificate, condition); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	err = r.removeErrorConditions(ctx, certificate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return reconcile.Result{}, nil
}

// updateCertificateConditions updates the conditions of the Certificate resource
func (r *CertificateReconciler) updateCertificateConditions(ctx context.Context, certificate *v1alpha1.Certificate, condition metav1.Condition) error {
	meta.SetStatusCondition(&certificate.Status.Conditions, condition)
	err := r.Client.Status().Update(ctx, certificate)
	if err != nil {
		return fmt.Errorf(errUpdateStatus, err)
	}

	return nil
}

// removeErrorConditions removes the error conditions of the Certificate resource
func (r *CertificateReconciler) removeErrorConditions(ctx context.Context, certificate *v1alpha1.Certificate) error {
	meta.RemoveStatusCondition(&certificate.Status.Conditions, ConditionError)
	err := r.Client.Status().Update(ctx, certificate)
	if err != nil {
		return fmt.Errorf(errUpdateStatus, err)
	}

	return nil
}

// isCertificateValid checks if the certificate is valid based on the renewal criteria specified in the CertificateConfig.
// It calculates the renewal date by subtracting the specified number of days before renewal from the current time.
// Returns true if the certificate is valid and false otherwise.
func isCertificateValid(certificate *v1alpha1.Certificate, certificateConfig *v1alpha1.CertificateConfig) bool {
	renewDate := time.Now().AddDate(0, 0, -certificateConfig.Spec.DaysBeforeRenewal)
	return !certificate.Status.ValidTo.IsZero() && certificate.Status.ValidTo.Time.After(renewDate)
}

// isSecretUpToDate checks if the secret associated with the certificate is up to date.
// It returns true if the reconciliation process should stop because the secret is up to date or an error occurred.
// It returns false if the reconciliation should continue.
func (r *CertificateReconciler) isSecretUpToDate(ctx context.Context, certificate *v1alpha1.Certificate, namespace string) (stopReconcile bool, err error) {
	if isSecretNameChanged(certificate) {
		// If the current secret name doesn't match the desired secret name, continue reconciliation.
		return false, nil
	}

	if isSecretDeleted, err := r.isSecretDeleted(ctx, certificate, namespace); err != nil {
		return true, err
	} else if isSecretDeleted {
		return false, nil
	}

	return true, nil
}

// isSecretNameChanged checks if the secret name was changed.
func isSecretNameChanged(certificate *v1alpha1.Certificate) bool {
	return certificate.Status.SecretName != certificate.Spec.SecretName
}

// isSecretDeleted checks if the secret was deleted.
func (r *CertificateReconciler) isSecretDeleted(ctx context.Context, certificate *v1alpha1.Certificate, namespace string) (bool, error) {
	if _, err := common.GetSecret(r.Client, ctx, certificate.Status.SecretName, namespace); err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	return false, nil
}

// forceExpirationUpdate updates the validity period of the certificate based on the certificate configuration.
// If ForceExpirationUpdate is set to true in the CertificateConfig, it updates the certificate's validity period.
// returns an error if any occurred during the update process.
func (r *CertificateReconciler) forceExpirationUpdate(ctx context.Context, certClient cert.Client, certificate *v1alpha1.Certificate, force bool) error {
	if !force {
		return nil
	}

	condition, err := r.updateCertValidity(ctx, certClient, certificate)
	if err != nil {
		err = r.updateCertificateConditions(ctx, certificate, condition)
		return err
	}

	return nil
}

// hasNotFoundErrorCondition checks if the Certificate resource has a condition indicating a NotFound error.
func (r *CertificateReconciler) hasNotFoundErrorCondition(certificate *v1alpha1.Certificate) bool {
	for _, condition := range certificate.Status.Conditions {
		if condition.Type == ConditionError && strings.Contains(condition.Message, http.StatusText(http.StatusNotFound)) {
			return true
		}
	}
	return false
}
