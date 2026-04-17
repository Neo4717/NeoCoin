package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

type AuthConfig struct {
	JWTSecret   string
	JWTExpiry   time.Duration
	JWTIssuer   string
	RequireAuth bool
}

func AuthConfigFromEnv() *AuthConfig {
	secret := os.Getenv("JWT_SECRET")
	requireAuth := envBool("JWT_REQUIRE_AUTH", false)
	expiryHours := envInt("JWT_EXPIRY_HOURS", 24)
	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "neocoin"
	}

	return &AuthConfig{
		JWTSecret:   secret,
		JWTExpiry:   time.Duration(expiryHours) * time.Hour,
		JWTIssuer:   issuer,
		RequireAuth: requireAuth,
	}
}

type JWTClaims struct {
	Sub    string   `json:"sub"`
	Iss    string   `json:"iss"`
	Exp    int64    `json:"exp"`
	Iat    int64    `json:"iat"`
	Nonce  string   `json:"nonce,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

type JWTManager struct {
	mu        sync.RWMutex
	secret    []byte
	expiry    time.Duration
	issuer    string
	blacklist map[string]time.Time
}

func NewJWTManager(cfg *AuthConfig) *JWTManager {
	secret := []byte(cfg.JWTSecret)
	if len(secret) == 0 {
		var entropy [32]byte
		rand.Read(entropy[:])
		secret = entropy[:]
	}

	return &JWTManager{
		secret:    secret,
		expiry:    cfg.JWTExpiry,
		issuer:    cfg.JWTIssuer,
		blacklist: make(map[string]time.Time),
	}
}

func (j *JWTManager) Issue(subject string, scopes []string) (string, error) {
	var nonce [16]byte
	rand.Read(nonce[:])

	now := time.Now().Unix()
	claims := JWTClaims{
		Sub:    subject,
		Iss:    j.issuer,
		Exp:    now + int64(j.expiry.Seconds()),
		Iat:    now,
		Nonce:  hex.EncodeToString(nonce[:]),
		Scopes: scopes,
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	signature := j.sign([]byte(header + "." + payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(signature)

	return header + "." + payloadB64 + "." + sigB64, nil
}

func (j *JWTManager) Validate(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	header, payload, sig := parts[0], parts[1], parts[2]

	headerStr, _ := base64.RawURLEncoding.DecodeString(header)
	var h struct{ Alg string }
	json.Unmarshal(headerStr, &h)
	if h.Alg != "HS256" {
		return nil, errors.New("unsupported algorithm")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}

	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}

	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}

	if claims.Iss != j.issuer {
		return nil, errors.New("invalid issuer")
	}

	expectedSig := j.sign([]byte(header + "." + payload))
	gotSig, _ := base64.RawURLEncoding.DecodeString(sig)

	if !hmac.Equal(expectedSig, gotSig) {
		return nil, errors.New("invalid signature")
	}

	j.mu.RLock()
	if exp, ok := j.blacklist[claims.Nonce]; ok && time.Now().Before(exp) {
		j.mu.RUnlock()
		return nil, errors.New("token revoked")
	}
	j.mu.RUnlock()

	return &claims, nil
}

func (j *JWTManager) Revoke(token string) error {
	claims, err := j.Validate(token)
	if err != nil {
		return err
	}

	j.mu.Lock()
	j.blacklist[claims.Nonce] = time.Now().Add(j.expiry)
	j.mu.Unlock()

	return nil
}

func (j *JWTManager) sign(msg []byte) []byte {
	h := hmac.New(sha256.New, j.secret)
	h.Write(msg)
	return h.Sum(nil)
}

type AuthMiddleware struct {
	jwt *JWTManager
}

func NewAuthMiddleware(jwt *JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwt: jwt}
}

func (a *AuthMiddleware) ValidateRequest(token string) (*JWTClaims, error) {
	if token == "" {
		return nil, errors.New("missing token")
	}
	return a.jwt.Validate(token)
}
