package alexa

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	certCache     = make(map[string]*x509.Certificate)
	certCacheLock sync.RWMutex
)

// VerifyRequest validates that an incoming HTTP request was sent by Amazon Alexa.
// It checks the certificate chain URL, downloads and validates the signing cert,
// and verifies the request body signature.
func VerifyRequest(r *http.Request, body []byte) error {
	certURL := r.Header.Get("SignatureCertChainUrl")
	if certURL == "" {
		return errors.New("missing SignatureCertChainUrl header")
	}

	signature := r.Header.Get("Signature")
	if signature == "" {
		return errors.New("missing Signature header")
	}

	if err := validateCertURL(certURL); err != nil {
		return fmt.Errorf("invalid cert URL: %w", err)
	}

	cert, err := fetchAndValidateCert(certURL)
	if err != nil {
		return fmt.Errorf("cert validation failed: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	hash := sha1.Sum(body)
	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("certificate does not contain an RSA public key")
	}

	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA1, hash[:], sigBytes); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

func validateCertURL(certURL string) error {
	u, err := url.Parse(certURL)
	if err != nil {
		return err
	}

	if !strings.EqualFold(u.Scheme, "https") {
		return errors.New("scheme must be https")
	}
	if !strings.EqualFold(u.Host, "s3.amazonaws.com") &&
		!strings.HasSuffix(strings.ToLower(u.Host), ".s3.amazonaws.com") {
		return errors.New("host must be s3.amazonaws.com")
	}
	if !strings.HasPrefix(u.Path, "/echo.api/") {
		return errors.New("path must start with /echo.api/")
	}
	if u.Port() != "" && u.Port() != "443" {
		return errors.New("port must be 443 or empty")
	}

	return nil
}

func fetchAndValidateCert(certURL string) (*x509.Certificate, error) {
	certCacheLock.RLock()
	if cert, ok := certCache[certURL]; ok {
		if time.Now().Before(cert.NotAfter) {
			certCacheLock.RUnlock()
			return cert, nil
		}
	}
	certCacheLock.RUnlock()

	resp, err := http.Get(certURL)
	if err != nil {
		return nil, fmt.Errorf("download cert: %w", err)
	}
	defer resp.Body.Close()

	certData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read cert body: %w", err)
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, errors.New("certificate is expired or not yet valid")
	}

	foundSAN := false
	for _, name := range cert.DNSNames {
		if strings.EqualFold(name, "echo-api.amazon.com") {
			foundSAN = true
			break
		}
	}
	if !foundSAN {
		return nil, errors.New("certificate SAN does not contain echo-api.amazon.com")
	}

	certCacheLock.Lock()
	certCache[certURL] = cert
	certCacheLock.Unlock()

	return cert, nil
}
