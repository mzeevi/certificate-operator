# certificate-operator

A Kubernetes operator designed to manage `Certificate` resources by interfacing with the `Cert` API.

It automates the process of obtaining and renewing `TLS certificates` from `Cert` and managing them as Kubernetes secrets.

##  Features
- [x] TLS Secret creation: Automatically creates a `secret` of type `tls` in the requested name and namespace. The `tls.crt` and `tls.key` are extracted from the `Certificate` obtained from `Cert`.
- [x] Automatic Certificate Renewal: Automatically renews `TLS Certificates` before they expire, ensuring continuous security for your applications.

## Resources

### Certificate
  - Manages specifications for creating certificates.
  - Contains details about the certificate's validity period (`validFrom` and `validTo`) and the current state of the certificate.
  - Provides insights into the certificate's signature `hash algorithm`, and `GUID`.

Note: The fields in the `Spec` are all optional, not all have to be specified.

```yaml
apiVersion: cert.dana.io/v1alpha1
kind: Certificate
metadata:
  name: certificate-sample
spec:
  certificateData:
    subject:
      commonName: "example"
      country: "ex"
      state: "example"
      locality: "example"
      organization: "example"
      organizationUnit: "example"
    san:
      dns:
        - "www.example.com"
      ips:
        - "192.168.1.1"
    template: "default"
    form: pfx
  configRef: 
    name: "certificateconfig-sample"
  secretName: my-secret-new
```

### CertificateConfig
  - Stores configuration details required for interacting with the external `Cert` API service.
  - Specifies settings such as `daysBeforeRenewal` and `waitTimeout`, which affect interaction with the external `Cert` API.

```yaml
apiVersion: cert.dana.io/v1alpha1
kind: CertificateConfig
metadata:
  name: certificateconfig-sample
spec:
  secretRef:
    name: cert-credentials
    namespace: default
  daysBeforeRenewal: 7
  waitTimeout: 5m
```

The `Secret` has a single key - `credentials` and contains a `json` with the needed keys, as specified below:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cert-credentials
  namespace: default
type: Opaque
stringData:
  credentials: |
    {
      "apiEndpoint": "https://cert.com/cert-route/",
      "token": "jwt-token",
      "downloadEndpoint": "/down"
    }
```

## Getting Started

### Prerequisites

1. A Kubernetes cluster (you can [use KinD](https://kind.sigs.k8s.io/docs/user/quick-start/)).

```bash
$ make prereq
```

### Deploying the controller

```bash
$ make deploy IMG=ghcr.io/dana-team/certificate-operator:<release>
```

#### Build your own image

```bash
$ make docker-build docker-push IMG=<registry>/certificate-operator:<tag>
```
