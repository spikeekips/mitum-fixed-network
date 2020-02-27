package util

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
)

func GenerateED25519Privatekey() (ed25519.PrivateKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)

	return priv, err
}

func GenerateTLSCerts(host string, key ed25519.PrivateKey) ([]tls.Certificate, error) {
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	template.DNSNames = append(template.DNSNames, host)

	var certDER []byte
	if c, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		key.Public().(ed25519.PublicKey),
		key,
	); err != nil {
		return nil, err
	} else {
		certDER = c
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyBytes,
		},
	)
	certPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		},
	)

	var certificate tls.Certificate
	if c, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return nil, err
	} else {
		certificate = c
	}

	return []tls.Certificate{certificate}, nil
}
