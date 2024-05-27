package common

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetSecret retrieves the Kubernetes Secret referenced by the CertificateConfig and handles errors if the Secret is not found.
func GetSecret(cl client.Client, ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := cl.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret); err != nil {
		return secret, err
	}

	return secret, nil
}
