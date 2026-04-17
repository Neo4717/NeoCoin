package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"
)

type SecurityConfig struct {
	TLSEnabled        bool
	CertFile          string
	KeyFile           string
	AutoTLS           bool
	MinTLSVersion     uint16
	RequireClientCert bool
}

const (
	TLS10 uint16 = 0x0301
	TLS11 uint16 = 0x0302
	TLS12 uint16 = 0x0303
	TLS13 uint16 = 0x0303
)

func TLSConfigFromEnv() *SecurityConfig {
	return &SecurityConfig{
		TLSEnabled:        envBool("TLS_ENABLED", false),
		CertFile:          os.Getenv("TLS_CERT_FILE"),
		KeyFile:           os.Getenv("TLS_KEY_FILE"),
		AutoTLS:           envBool("TLS_AUTO_GENERATE", false),
		MinTLSVersion:     uint16(envInt("TLS_MIN_VERSION", 0x0303)),
		RequireClientCert: envBool("TLS_REQUIRE_CLIENT_CERT", false),
	}
}

func (sc *SecurityConfig) MakeTLSConfig() (*tls.Config, error) {
	if !sc.TLSEnabled && !sc.AutoTLS {
		return nil, nil
	}

	var cert tls.Certificate
	var err error

	if sc.AutoTLS {
		cert, err = generateSelfSignedTLS()
		if err != nil {
			return nil, fmt.Errorf("generate self-signed cert: %w", err)
		}
	} else {
		cert, err = tls.LoadX509KeyPair(sc.CertFile, sc.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load cert: %w", err)
		}
	}

	minVersion := sc.MinTLSVersion
	if minVersion == 0 {
		minVersion = TLS13
	}

	cfg := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               minVersion,
		PreferServerCipherSuites: true,
		ClientAuth:               tls.NoClientCert,
	}

	if sc.RequireClientCert {
		cfg.ClientAuth = tls.RequestClientCert
	}

	return cfg, nil
}

func generateSelfSignedTLS() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial := big.NewInt(1)
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"NeoCoin"},
			CommonName:   "neocoin.local",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "127.0.0.1"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

type EncryptedStore struct {
	mu     sync.Mutex
	store  ChainStore
	cipher *Cipher
	key    [32]byte
}

type Cipher struct {
	key [32]byte
}

func NewCipher(password string) *Cipher {
	sum := sha256.Sum256([]byte(password))
	var key [32]byte
	copy(key[:], sum[:])
	return &Cipher{key: key}
}

func (c *Cipher) Encrypt(data []byte) ([]byte, error) {
	key := c.key
	out := make([]byte, len(data))
	copy(out, data)
	for i := range out {
		out[i] ^= key[i%len(key)]
	}
	return out, nil
}

func (c *Cipher) Decrypt(data []byte) ([]byte, error) {
	return c.Encrypt(data)
}
