package cert

// postCertificateBody represents the request body structure for sending a POST request to the Cert service.
type postCertificateBody struct {
	Subject  Subject `json:"subject,omitempty"`
	San      San     `json:"san,omitempty"`
	Template string  `json:"template,omitempty"`
}

// Subject represents the subject of a certificate, including common name, country, state, locality,
// organization, and organizational unit.
type Subject struct {
	CommonName         string `json:"commonName,omitempty"`
	Country            string `json:"country,omitempty"`
	State              string `json:"state,omitempty"`
	Locality           string `json:"locality,omitempty"`
	Organization       string `json:"organization,omitempty"`
	OrganizationalUnit string `json:"organizationalUnit,omitempty"`
}

// San represents the subject alternative name (SAN) of a certificate, including DNS names and IP addresses.
type San struct {
	DNS []string `json:"dns,omitempty"`
	IPs []string `json:"ips,omitempty"`
}

// PostCertificateResponse represents the structure of the JSON response body for obtaining a certificate.
type PostCertificateResponse struct {
	Guid string `json:"taskId"`
}

// DownloadCertificateResponse represents the response received when downloading a certificate.
type DownloadCertificateResponse struct {
	Form     string `json:"form"`
	Format   string `json:"format"`
	Data     string `json:"data"`
	Password string `json:"password"`
}

// GetCertificateResponse represents the response received when getting certificate data.
type GetCertificateResponse struct {
	ValidTo                string `json:"validTo"`
	ValidFrom              string `json:"validFrom"`
	SignatureHashAlgorithm string `json:"signatureHashAlgorithm"`
}
