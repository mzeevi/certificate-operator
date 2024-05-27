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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CertificateConfigSpec defines the desired state of CertificateConfig.
type CertificateConfigSpec struct {
	// SecretRef is a reference to the Kubernetes Secret containing credentials for authenticating with the cert API.
	SecretRef SecretRef `json:"secretRef"`
	// DaysBeforeRenewal represents the number of days to renew the certificate before expiration.
	DaysBeforeRenewal int `json:"daysBeforeRenewal"`
	// WaitTimeout specifies the maximum time duration for waiting for response from cert.
	WaitTimeout *metav1.Duration `json:"waitTimeout,omitempty"`
	// ForceExpirationUpdate indicates whether to force an update of the Certificate details even when it's valid.
	ForceExpirationUpdate bool `json:"forceExpirationUpdate,omitempty"`
}

// SecretRef is a reference to the Kubernetes Secret containing credentials for authenticating with the cert API.
type SecretRef struct {
	// Name is the name of the Secret.
	Name string `json:"name"`
	// Namespace is the namespace where the Secret is located.
	Namespace string `json:"namespace"`
}

// CertificateConfigStatus defines the observed state of CertificateConfig.
type CertificateConfigStatus struct {
	// This section is intentionally left blank.
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// CertificateConfig is the Schema for the certificateconfigs API.
type CertificateConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateConfigSpec   `json:"spec,omitempty"`
	Status CertificateConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CertificateConfigList contains a list of CertificateConfig.
type CertificateConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CertificateConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CertificateConfig{}, &CertificateConfigList{})
}
