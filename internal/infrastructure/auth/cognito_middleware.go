package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"rbac-project/internal/domain"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

type jwkCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	ttl       time.Duration
	url       string
	client    *http.Client
}

func newJWKCache(url string, ttl time.Duration) *jwkCache {
	return &jwkCache{
		keys:   map[string]*rsa.PublicKey{},
		ttl:    ttl,
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *jwkCache) keyForKid(kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if key, ok := c.keys[kid]; ok && time.Now().Before(c.expiresAt) {
		c.mu.RUnlock()
		return key, nil
	}
	c.mu.RUnlock()

	if err := c.refresh(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	key, ok := c.keys[kid]
	if !ok {
		return nil, errors.New("jwk key not found")
	}
	return key, nil
}

func (c *jwkCache) refresh() error {
	resp, err := c.client.Get(c.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("unable to fetch jwks")
	}
	var parsed jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey, len(parsed.Keys))
	for _, key := range parsed.Keys {
		if key.Kty != "RSA" || key.Kid == "" || key.N == "" || key.E == "" {
			continue
		}
		pubKey, err := rsaFromJWK(key.N, key.E)
		if err != nil {
			continue
		}
		keys[key.Kid] = pubKey
	}
	if len(keys) == 0 {
		return errors.New("no valid jwk keys")
	}
	c.mu.Lock()
	c.keys = keys
	c.expiresAt = time.Now().Add(c.ttl)
	c.mu.Unlock()
	return nil
}

func rsaFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nRaw, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eRaw, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	var eInt int
	for _, b := range eRaw {
		eInt = eInt<<8 + int(b)
	}
	if eInt == 0 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nRaw), E: eInt}, nil
}

type CognitoMiddleware struct {
	userPoolID string
	region     string
	cache      *jwkCache
}

func NewCognitoMiddleware(userPoolID, region string) *CognitoMiddleware {
	jwksURL := "https://cognito-idp." + region + ".amazonaws.com/" + userPoolID + "/.well-known/jwks.json"
	return &CognitoMiddleware{
		userPoolID: userPoolID,
		region:     region,
		cache:      newJWKCache(jwksURL, 15*time.Minute),
	}
}

func (m *CognitoMiddleware) Handler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization token"})
		}
		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
		if tokenString == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization token"})
		}
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			kid, ok := token.Header["kid"].(string)
			if !ok || kid == "" {
				return nil, errors.New("missing kid")
			}
			return m.cache.keyForKid(kid)
		}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
		if err != nil || !token.Valid {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		}
		sub, _ := claims["sub"].(string)
		if sub == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": domain.ErrInvalidInput.Error()})
		}
		c.Set("user_id", sub)
		return next(c)
	}
}
