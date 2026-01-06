package services

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt"

	"naimuBack/internal/models"
)

const (
	appStoreProdBase    = "https://api.storekit.itunes.apple.com"
	appStoreSandboxBase = "https://api.storekit-sandbox.itunes.apple.com"
	appleJWKSURL        = "https://apple.com/.well-known/appstoreconnect/keys"
)

type AppleIAPConfig struct {
	IssuerID   string
	BundleID   string
	KeyID      string
	PrivateKey string

	// Optional: force sandbox ("sandbox") or production ("production").
	Environment string
	HTTPClient  *http.Client
}

type AppleIAPService struct {
	issuerID string
	bundleID string
	keyID    string
	key      *ecdsa.PrivateKey

	defaultEnv string
	client     *http.Client

	jwksMu     sync.Mutex
	jwks       *jose.JSONWebKeySet
	jwksExpiry time.Time
}

func NewAppleIAPService(cfg AppleIAPConfig) (*AppleIAPService, error) {
	if strings.TrimSpace(cfg.IssuerID) == "" || strings.TrimSpace(cfg.KeyID) == "" || strings.TrimSpace(cfg.PrivateKey) == "" {
		return nil, fmt.Errorf("apple iap: issuer_id, key_id and private_key are required")
	}
	key, err := jwt.ParseECPrivateKeyFromPEM([]byte(cfg.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	env := strings.ToLower(strings.TrimSpace(cfg.Environment))
	if env != "sandbox" {
		env = "production"
	}
	return &AppleIAPService{
		issuerID:   strings.TrimSpace(cfg.IssuerID),
		bundleID:   strings.TrimSpace(cfg.BundleID),
		keyID:      strings.TrimSpace(cfg.KeyID),
		key:        key,
		defaultEnv: env,
		client:     client,
	}, nil
}

// VerifyTransaction fetches signedTransactionInfo from Apple, validates its signature
// and returns the decoded transaction payload.
func (s *AppleIAPService) VerifyTransaction(ctx context.Context, transactionID string) (models.AppleTransaction, error) {
	if strings.TrimSpace(transactionID) == "" {
		return models.AppleTransaction{}, fmt.Errorf("transaction_id is required")
	}
	envs := []string{s.defaultEnv}
	if s.defaultEnv == "production" {
		envs = append(envs, "sandbox")
	} else {
		envs = append(envs, "production")
	}

	var lastErr error
	for _, env := range envs {
		signed, err := s.fetchSignedTransaction(ctx, transactionID, env)
		if err != nil {
			lastErr = err
			continue
		}
		txn, err := s.DecodeSignedTransaction(ctx, signed)
		if err != nil {
			lastErr = err
			continue
		}
		if txn.TransactionID == "" {
			txn.TransactionID = transactionID
		}
		if txn.TransactionID != transactionID {
			return models.AppleTransaction{}, fmt.Errorf("transaction id mismatch: expected %s got %s", transactionID, txn.TransactionID)
		}
		if s.bundleID != "" && txn.BundleID != "" && txn.BundleID != s.bundleID {
			return models.AppleTransaction{}, fmt.Errorf("bundle id mismatch: %s", txn.BundleID)
		}
		if txn.Environment == "" {
			txn.Environment = env
		}
		return txn, nil
	}
	if lastErr == nil {
		lastErr = errors.New("failed to fetch transaction from apple api")
	}
	return models.AppleTransaction{}, lastErr
}

// ParseNotification verifies signedPayload from Apple server notifications and returns the decoded payload.
func (s *AppleIAPService) ParseNotification(ctx context.Context, signedPayload string) (models.AppleNotification, error) {
	data, err := s.verifyJWS(ctx, signedPayload)
	if err != nil {
		return models.AppleNotification{}, err
	}
	var notif models.AppleNotification
	if err := json.Unmarshal(data, &notif); err != nil {
		return models.AppleNotification{}, err
	}
	notif.Raw = signedPayload
	if s.bundleID != "" && notif.Data.BundleID != "" && notif.Data.BundleID != s.bundleID {
		return models.AppleNotification{}, fmt.Errorf("bundle id mismatch: %s", notif.Data.BundleID)
	}
	return notif, nil
}

// DecodeSignedTransaction verifies and decodes Apple's signedTransactionInfo JWS payload.
func (s *AppleIAPService) DecodeSignedTransaction(ctx context.Context, signedInfo string) (models.AppleTransaction, error) {
	payload, err := s.verifyJWS(ctx, signedInfo)
	if err != nil {
		return models.AppleTransaction{}, err
	}
	var txn models.AppleTransaction
	if err := json.Unmarshal(payload, &txn); err != nil {
		return models.AppleTransaction{}, err
	}
	txn.Raw = signedInfo
	return txn, nil
}

func (s *AppleIAPService) fetchSignedTransaction(ctx context.Context, transactionID, env string) (string, error) {
	token, err := s.signedToken()
	if err != nil {
		return "", err
	}

	base := appStoreProdBase
	if env == "sandbox" {
		base = appStoreSandboxBase
	}
	url := fmt.Sprintf("%s/inApps/v1/transactions/%s", base, transactionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("apple api %s: %s (%s)", env, resp.Status, strings.TrimSpace(string(body)))
	}

	var body struct {
		SignedTransactionInfo string `json:"signedTransactionInfo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if strings.TrimSpace(body.SignedTransactionInfo) == "" {
		return "", errors.New("empty signedTransactionInfo")
	}
	return body.SignedTransactionInfo, nil
}

func (s *AppleIAPService) signedToken() (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"iss": s.issuerID,
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
		"aud": "appstoreconnect-v1",
	}
	if s.bundleID != "" {
		claims["bid"] = s.bundleID
	}
	t := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	t.Header["kid"] = s.keyID
	return t.SignedString(s.key)
}

func (s *AppleIAPService) verifyJWS(ctx context.Context, token string) ([]byte, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("empty signed payload")
	}
	jws, err := jose.ParseSigned(token)
	if err != nil {
		return nil, err
	}
	if len(jws.Signatures) == 0 {
		return nil, errors.New("missing signature")
	}

	kid := jws.Signatures[0].Header.KeyID
	key, err := s.lookupKey(ctx, kid)
	if err != nil {
		return nil, err
	}

	payload, err := jws.Verify(&key)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *AppleIAPService) lookupKey(ctx context.Context, kid string) (jose.JSONWebKey, error) {
	set, err := s.fetchJWKS(ctx)
	if err != nil {
		return jose.JSONWebKey{}, err
	}
	keys := set.Key(kid)
	if len(keys) == 0 {
		return jose.JSONWebKey{}, fmt.Errorf("apple jwk not found: %s", kid)
	}
	return keys[0], nil
}

func (s *AppleIAPService) fetchJWKS(ctx context.Context) (*jose.JSONWebKeySet, error) {
	s.jwksMu.Lock()
	defer s.jwksMu.Unlock()

	if s.jwks != nil && time.Until(s.jwksExpiry) > 5*time.Minute {
		return s.jwks, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleJWKSURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("apple jwks: %s (%s)", resp.Status, strings.TrimSpace(string(body)))
	}

	var set jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, err
	}
	s.jwks = &set
	s.jwksExpiry = time.Now().Add(30 * time.Minute)
	return s.jwks, nil
}

// DecodeCompactJWS decodes payload without verification for debugging only.
// It is unused in production logic but handy when troubleshooting.
func DecodeCompactJWS(token string) ([]byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		return nil, errors.New("invalid jws format")
	}
	return base64.RawStdEncoding.DecodeString(parts[1])
}
