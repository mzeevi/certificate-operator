package certhandler

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"software.sslmate.com/src/go-pkcs12"
)

const (
	errCannotDecodeData          = "cannot decode PKCS#12 data: %v"
	errCannotDecodeB64Data       = "cannot decode base64-encoded PKCS#12 data: %v"
	errCannotCastToRSAPrivateKey = "cannot cast to RSA Private Key"

	certificateBlockType = "CERTIFICATE"
	rsaBlockType         = "RSA PRIVATE KEY"
)

// TLSData represents TLS data containing a private key and certificate bytes.
type TLSData struct {
	PrivateKeyBytes  []byte
	CertificateBytes []byte
}

// Decoder decodes the PKCS#12 formatted TLS data.
func Decoder(data, password string) (TLSData, error) {
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return TLSData{}, fmt.Errorf(errCannotDecodeB64Data, err)
	}

	privateKey, certificate, _, err := pkcs12.DecodeChain(decodedData, password)
	if err != nil {
		return TLSData{}, fmt.Errorf(errCannotDecodeData, err)
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return TLSData{}, errors.New(errCannotCastToRSAPrivateKey)
	}

	// Encode certificate to PEM format
	certificateBytes := pem.EncodeToMemory(&pem.Block{Type: certificateBlockType, Bytes: certificate.Raw})
	privateKeyBytes := pem.EncodeToMemory(&pem.Block{Type: rsaBlockType, Bytes: x509.MarshalPKCS1PrivateKey(rsaPrivateKey)})

	return TLSData{
		PrivateKeyBytes:  privateKeyBytes,
		CertificateBytes: certificateBytes,
	}, nil
}
