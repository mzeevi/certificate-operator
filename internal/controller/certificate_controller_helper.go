package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/dana-team/certificate-operator/internal/clients/cert"

	v1alpha1 "github.com/dana-team/certificate-operator/api/v1alpha1"
	certhandler "github.com/dana-team/certificate-operator/internal/certhandler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	errFailedParseValidTo           = "failed to parse validTo: %v"
	errFailedParseValidFrom         = "failed to parse validFrom: %v"
	errFailedDownloadingCertificate = "failed downloading certificate: %v"
	errCreateOrUpdateTlsSecret      = "failed to create or update tls secret: %v"
)

const (
	ConditionParseValidToFailed            = "ParseValidToFailed"
	ConditionParseValidFromFailed          = "ParseValidFromFailed"
	ConditionSetOwnerRefFailed             = "SetOwnerRefFailed"
	ConditionCreateOrUpdateTLSSecretFailed = "CreateOrUpdateTLSSecretFailed"
)

// issueCertificate creates a certificate, obtains the certificate guid, and updates the Certificate status with the obtained guid.
// It returns an error if the operation fails.
func (r *CertificateReconciler) issueCertificate(ctx context.Context, certClient cert.Client, certificate *v1alpha1.Certificate) (condition metav1.Condition, err error) {
	if r.hasNotFoundErrorCondition(certificate) {
		return metav1.Condition{}, nil
	}

	guid, err := certClient.PostCertificate(ctx, certificate)
	if err != nil {
		return errorCondition(ConditionPostToCertAPIFailed, err), fmt.Errorf(errCreationFailed, err)
	}

	certificate.Status.Guid = guid
	if err = r.Status().Update(ctx, certificate); err != nil {
		return errorCondition(ConditionUpdateStatusFailed, err), fmt.Errorf(errCreationFailed, err)
	}

	return metav1.Condition{}, nil
}

// obtainCertificateData obtains certificate data, updates the Certificate status with the obtained data,
// and returns the validity information.
// It returns the validity information (validTo, validFrom, signatureHashAlgorithm), or an error if the operation fails.
func (r *CertificateReconciler) obtainCertificateData(ctx context.Context, certClient cert.Client, certificate *v1alpha1.Certificate) (validTo, validFrom, signatureHashAlgorithm string, condition metav1.Condition, err error) {
	getResponse, err := certClient.GetCertificate(ctx, certificate)
	if err != nil {
		return "", "", "", errorCondition(ConditionGetCertDataFromCertAPIFailed, err), err
	}

	return getResponse.ValidTo, getResponse.ValidFrom, getResponse.SignatureHashAlgorithm, metav1.Condition{}, nil
}

// updateCertValidity updates the certificate status with the validity information.
// It returns an error if the status update operation fails.
func (r *CertificateReconciler) updateCertValidity(ctx context.Context, certClient cert.Client, certificate *v1alpha1.Certificate) (metav1.Condition, error) {
	validTo, validFrom, signatureHashAlgorithm, condition, err := r.obtainCertificateData(ctx, certClient, certificate)
	if err != nil {
		return condition, err
	}

	validToTime, err := time.Parse(timeFormat, validTo)
	if err != nil {
		return errorCondition(ConditionParseValidToFailed, err), fmt.Errorf(errFailedParseValidTo, err)
	}

	validFromTime, err := time.Parse(timeFormat, validFrom)
	if err != nil {
		return errorCondition(ConditionParseValidFromFailed, err), fmt.Errorf(errFailedParseValidFrom, err)
	}

	certificate.Status.ValidTo = metav1.Time{Time: validToTime}
	certificate.Status.ValidFrom = metav1.Time{Time: validFromTime}
	certificate.Status.SignatureHashAlgorithm = signatureHashAlgorithm

	if err = r.Status().Update(ctx, certificate); err != nil {
		return errorCondition(ConditionUpdateStatusFailed, err), fmt.Errorf(errUpdateStatus, err)
	}

	return metav1.Condition{}, nil
}

// downloadCert downloads the certificate from the Cert API and decodes it into TLS data.
// It returns the TLS data containing the certificate and private key, or an error if the download or decoding fails.
func (r *CertificateReconciler) downloadCert(ctx context.Context, certClient cert.Client, certificate *v1alpha1.Certificate) (certhandler.TLSData, metav1.Condition, error) {
	downloadResponse, err := certClient.DownloadCertificate(ctx, certificate)
	if err != nil {
		return certhandler.TLSData{}, errorCondition(ConditionDownloadCertFromCertAPIFailed, err), fmt.Errorf(errFailedDownloadingCertificate, err)
	}

	tlsData, err := certhandler.Decoder(downloadResponse.Data, downloadResponse.Password)
	if err != nil {
		return certhandler.TLSData{}, errorCondition(ConditionDecodeCertFailed, err), fmt.Errorf(errFailedDownloadingCertificate, err)
	}

	return tlsData, metav1.Condition{}, nil
}

// createOrUpdateTlsSecret creates or updates a TLS secret with the provided TLS data and associates it with the certificate.
// It returns an error if the creation or update operation fails.
func (r *CertificateReconciler) createOrUpdateTlsSecret(ctx context.Context, certificate *v1alpha1.Certificate, tlsData certhandler.TLSData, namespace string) (metav1.Condition, error) {
	tlsSecret := certhandler.TlsSecret(tlsData, certificate, namespace)
	if err := controllerutil.SetOwnerReference(certificate, tlsSecret, r.Scheme); err != nil {
		return errorCondition(ConditionSetOwnerRefFailed, err), fmt.Errorf(fmt.Sprintf(errFailedToSetOwnerRefForSecret, tlsSecret.Name), err)
	}

	err := certhandler.CreateOrUpdateTLSSecret(ctx, r.Client, tlsSecret)
	if err != nil {
		return errorCondition(ConditionCreateOrUpdateTLSSecretFailed, err), fmt.Errorf(errCreateOrUpdateTlsSecret, err)
	}

	certificate.Status.SecretName = certificate.Spec.SecretName
	if err = r.Status().Update(ctx, certificate); err != nil {
		return errorCondition(ConditionUpdateStatusFailed, err), fmt.Errorf(errUpdateStatus, err)
	}

	return metav1.Condition{}, nil
}

func errorCondition(reason string, err error) metav1.Condition {
	return metav1.Condition{
		Type:    ConditionError,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: err.Error(),
	}
}
