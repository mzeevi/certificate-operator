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

// CertificateSpec defines the desired state of a Certificate.
type CertificateSpec struct {
	// CertificateData contains the data for generating the certificate.
	CertificateData CertificateData `json:"certificateData,omitempty"`
	// SecretName is the name of the Kubernetes Secret where the extracted certificate is stored.
	SecretName string `json:"secretName,omitempty"`
	// ConfigRef is the referance to the CertificateConfig associated with this Certificate.
	ConfigRef ConfigReference `json:"configRef,omitempty"`
}

// A ConfigReference is a reference to a CertificateConfig resource that will be used
// to configure the certificate.
type ConfigReference struct {
	// Name of the CertificateConfig.
	Name string `json:"name"`
}

// CertificateStatus defines the observed state of a Certificate.
type CertificateStatus struct {
	// Conditions represent the current conditions of the Certificate.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ValidFrom represents the time when the certificate becomes valid.
	ValidFrom metav1.Time `json:"validFrom,omitempty"`
	// ValidTo represents the time when the certificate expires.
	ValidTo metav1.Time `json:"validTo,omitempty"`
	// Issuer is the entity that issued the certificate.
	Issuer string `json:"issuer,omitempty"`
	// Guid is a unique identifier for the certificate.
	Guid string `json:"guid,omitempty"`
	// SignatureHashAlgorithm is the algorithm used to sign the certificate.
	SignatureHashAlgorithm string `json:"signatureHashAlgorithm,omitempty"`
	// SecretName is the name of the Kubernetes Secret where the extracted certificate is stored.
	SecretName string `json:"secretName,omitempty"`
}

// CertificateData contains data for generating a Certificate.
type CertificateData struct {
	// Subject represents the subject of the certificate.
	Subject Subject `json:"subject,omitempty"`
	// San represents Subject Alternative Names of the certificate.
	San San `json:"san,omitempty"`
	// Template is an optional field specifying the template for the certificate.
	Template string `json:"template,omitempty"`
	// Form is an optional field specifying the format of the certificate.
	// +kubebuilder:default:="pfx"
	// +kubebuilder:validation:Enum=pfx;
	Form string `json:"form,omitempty"`
}

// Subject represents the subject of a Certificate.
type Subject struct {
	// CommonName is the common name of the subject.
	CommonName         string `json:"commonName,omitempty"`
	Country            string `json:"country,omitempty"`
	State              string `json:"state,omitempty"`
	Locality           string `json:"locality,omitempty"`
	Organization       string `json:"organization,omitempty"`
	OrganizationalUnit string `json:"organizationUnit,omitempty"`
}

// San represents Subject Alternative Names of a Certificate.
type San struct {
	// DNS represents the DNS names included in the certificate.
	DNS []string `json:"dns,omitempty"`
	// IPs represents the IP addresses included in the certificate.
	IPs []string `json:"ips,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Certificate is the Schema for the certificates API.
type Certificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateSpec   `json:"spec,omitempty"`
	Status CertificateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CertificateList contains a list of Certificate.
type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Certificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Certificate{}, &CertificateList{})
}
