package apns

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"nofrillz/internal/config"
)

const providerTokenTTL = 50 * time.Minute

type deliveryProvider interface {
	Send(ctx context.Context, deviceToken string, title string, body string) (*DeliveryResult, error)
}

type noopProvider struct{}

type provider struct {
	logger      *zerolog.Logger
	topic       string
	endpoint    string
	keyID       string
	teamID      string
	privateKey  *ecdsa.PrivateKey
	httpClient  *http.Client
	tokenMu     sync.Mutex
	bearerToken string
	expiresAt   time.Time
}

type apnsErrorResponse struct {
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type providerTokenHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

type providerTokenClaims struct {
	Issuer   string `json:"iss"`
	IssuedAt int64  `json:"iat"`
}

type apnsPayload struct {
	APS apsPayload `json:"aps"`
}

type apsPayload struct {
	Alert alertPayload `json:"alert"`
	Sound string       `json:"sound,omitempty"`
}

type alertPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type ecdsaSignature struct {
	R, S *big.Int
}

func NewProviderFromConfig(cfg *config.APNSConfig, logger *zerolog.Logger) (deliveryProvider, error) {
	if cfg == nil || !cfg.Enabled {
		return noopProvider{}, nil
	}

	if strings.TrimSpace(cfg.Topic) == "" {
		return nil, fmt.Errorf("apns.topic is required when apns is enabled")
	}
	if strings.TrimSpace(cfg.KeyID) == "" {
		return nil, fmt.Errorf("apns.key_id is required when apns is enabled")
	}
	if strings.TrimSpace(cfg.TeamID) == "" {
		return nil, fmt.Errorf("apns.team_id is required when apns is enabled")
	}

	privateKey, err := loadPrivateKey(cfg)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &provider{
		logger:     logger,
		topic:      strings.TrimSpace(cfg.Topic),
		endpoint:   endpointForEnvironment(cfg.Environment),
		keyID:      strings.TrimSpace(cfg.KeyID),
		teamID:     strings.TrimSpace(cfg.TeamID),
		privateKey: privateKey,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				ForceAttemptHTTP2: true,
				TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
			},
		},
	}, nil
}

func (noopProvider) Send(ctx context.Context, deviceToken string, title string, body string) (*DeliveryResult, error) {
	return &DeliveryResult{}, nil
}

func (p *provider) Send(ctx context.Context, deviceToken string, title string, body string) (*DeliveryResult, error) {
	result, err := p.send(ctx, deviceToken, title, body)
	if err == nil || result == nil {
		return result, err
	}

	if result.Reason == "ExpiredProviderToken" || result.Reason == "InvalidProviderToken" {
		p.resetBearerToken()
		return p.send(ctx, deviceToken, title, body)
	}

	return result, err
}

func (p *provider) send(ctx context.Context, deviceToken string, title string, body string) (*DeliveryResult, error) {
	bearerToken, err := p.providerToken()
	if err != nil {
		return nil, err
	}

	payloadBody, err := json.Marshal(apnsPayload{
		APS: apsPayload{
			Alert: alertPayload{Title: title, Body: body},
			Sound: "default",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal apns payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/3/device/"+deviceToken, strings.NewReader(string(payloadBody)))
	if err != nil {
		return nil, fmt.Errorf("create apns request: %w", err)
	}
	request.Header.Set("authorization", "bearer "+bearerToken)
	request.Header.Set("apns-topic", p.topic)
	request.Header.Set("apns-push-type", "alert")
	request.Header.Set("apns-priority", "10")
	request.Header.Set("content-type", "application/json")

	response, err := p.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send apns request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return &DeliveryResult{StatusCode: response.StatusCode}, nil
	}

	apiErr := apnsErrorResponse{}
	if err := json.NewDecoder(response.Body).Decode(&apiErr); err != nil {
		return &DeliveryResult{StatusCode: response.StatusCode}, fmt.Errorf("apns returned %d and response decode failed: %w", response.StatusCode, err)
	}

	result := &DeliveryResult{
		StatusCode:   response.StatusCode,
		Reason:       apiErr.Reason,
		InvalidToken: isInvalidTokenReason(apiErr.Reason),
	}

	return result, fmt.Errorf("apns returned %d: %s", response.StatusCode, apiErr.Reason)
}

func (p *provider) providerToken() (string, error) {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	now := time.Now().UTC()
	if p.bearerToken != "" && now.Before(p.expiresAt) {
		return p.bearerToken, nil
	}

	header, err := json.Marshal(providerTokenHeader{Algorithm: "ES256", KeyID: p.keyID})
	if err != nil {
		return "", fmt.Errorf("marshal apns jwt header: %w", err)
	}
	claims, err := json.Marshal(providerTokenClaims{Issuer: p.teamID, IssuedAt: now.Unix()})
	if err != nil {
		return "", fmt.Errorf("marshal apns jwt claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claims)
	signingInput := encodedHeader + "." + encodedClaims

	digest := sha256.Sum256([]byte(signingInput))
	signature, err := ecdsa.SignASN1(rand.Reader, p.privateKey, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign apns jwt: %w", err)
	}

	var parsed ecdsaSignature
	if _, err := asn1.Unmarshal(signature, &parsed); err != nil {
		return "", fmt.Errorf("unmarshal apns jwt signature: %w", err)
	}

	rBytes := parsed.R.FillBytes(make([]byte, 32))
	sBytes := parsed.S.FillBytes(make([]byte, 32))
	rawSignature := append(rBytes, sBytes...)
	encodedSignature := base64.RawURLEncoding.EncodeToString(rawSignature)

	p.bearerToken = signingInput + "." + encodedSignature
	p.expiresAt = now.Add(providerTokenTTL)

	return p.bearerToken, nil
}

func (p *provider) resetBearerToken() {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()
	p.bearerToken = ""
	p.expiresAt = time.Time{}
}

func endpointForEnvironment(environment string) string {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "production", "prod":
		return "https://api.push.apple.com"
	default:
		return "https://api.sandbox.push.apple.com"
	}
}

func loadPrivateKey(cfg *config.APNSConfig) (*ecdsa.PrivateKey, error) {
	var pemBytes []byte
	if raw := strings.TrimSpace(cfg.AuthKeyBase64); raw != "" {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("decode apns auth key base64: %w", err)
		}
		pemBytes = decoded
	} else if path := strings.TrimSpace(cfg.AuthKeyPath); path != "" {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read apns auth key: %w", err)
		}
		pemBytes = bytes
	} else {
		return nil, fmt.Errorf("apns.auth_key_path or apns.auth_key_base64 is required when apns is enabled")
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("decode apns auth key pem: no pem block found")
	}

	switch block.Type {
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse apns pkcs8 private key: %w", err)
		}
		ecdsaKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("apns private key is not an ecdsa key")
		}
		return ecdsaKey, nil
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse apns ec private key: %w", err)
		}
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported apns private key type %q", block.Type)
	}
}

func isInvalidTokenReason(reason string) bool {
	switch reason {
	case "BadDeviceToken", "DeviceTokenNotForTopic", "Unregistered":
		return true
	default:
		return false
	}
}

var _ crypto.Signer = (*ecdsa.PrivateKey)(nil)
