package certhandler

import (
	"context"
	"fmt"

	v1alpha1 "github.com/dana-team/certificate-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errCreatingSecret = "cannot create secret %q in the namespace %q: %v"
	errGettingSecret  = "cannot get secret %q in the namespace %q: %v"
	errUpdatingSecret = "cannot update secret %q in the namespace %q: %v"
)

// TlsSecret creates a TLS secret from the provided TLS data and Certificate object.
func TlsSecret(tlsData TLSData, certificate *v1alpha1.Certificate, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certificate.Spec.SecretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       tlsData.CertificateBytes,
			corev1.TLSPrivateKeyKey: tlsData.PrivateKeyBytes,
		},
	}
}

// CreateOrUpdateTLSSecret creates or updates a TLS secret in the Kubernetes cluster.
func CreateOrUpdateTLSSecret(ctx context.Context, kubeClient client.Client, secret *corev1.Secret) error {
	existingSecret := &corev1.Secret{}

	if err := kubeClient.Get(ctx, client.ObjectKey{Namespace: secret.Namespace, Name: secret.Name}, existingSecret); err != nil {
		if errors.IsNotFound(err) {
			if createErr := kubeClient.Create(ctx, secret); createErr != nil {
				return fmt.Errorf(errCreatingSecret, secret.Name, secret.Namespace, createErr)
			}
			return nil
		} else {
			return fmt.Errorf(errGettingSecret, secret.Name, secret.Namespace, err)
		}
	}

	existingSecret.Data = secret.Data
	err := kubeClient.Update(ctx, existingSecret)
	if err != nil {
		return fmt.Errorf(errUpdatingSecret, secret.Name, secret.Namespace, err)
	}

	return nil
}
