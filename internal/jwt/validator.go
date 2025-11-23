package jwt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// Claims represents GitLab CI JWT claims
type Claims struct {
	jwt.RegisteredClaims
	ProjectPath string `json:"project_path"`
	RefPath     string `json:"ref_path"`
	PipelineID  string `json:"pipeline_id"`
	JobID       string `json:"job_id"`
}

// Validator validates GitLab OIDC JWTs
type Validator struct {
	audience    string
	issuers     []string
	jwksURL     string
	keySet      jwk.Set
	keySetMutex sync.RWMutex
	lastFetch   time.Time
}

// NewValidator creates a new JWT validator
func NewValidator(audience string, issuers []string, jwksURL string) *Validator {
	return &Validator{
		audience: audience,
		issuers:  issuers,
		jwksURL:  jwksURL,
	}
}

// ValidateToken validates a JWT token string
func (v *Validator) ValidateToken(tokenString string) (*Claims, error) {
	// Ensure JWKS is loaded
	if err := v.refreshJWKS(); err != nil {
		return nil, fmt.Errorf("failed to refresh JWKS: %w", err)
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		// Find key in JWKS
		v.keySetMutex.RLock()
		defer v.keySetMutex.RUnlock()

		key, found := v.keySet.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key not found in JWKS")
		}

		var rawKey interface{}
		if err := key.Raw(&rawKey); err != nil {
			return nil, fmt.Errorf("failed to get raw key: %w", err)
		}

		return rawKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate issuer
	if !v.isValidIssuer(claims.Issuer) {
		return nil, fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Validate audience
	found := false
	for _, aud := range claims.Audience {
		if aud == v.audience {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("invalid audience")
	}

	// Validate expiration
	if claims.ExpiresAt == nil || claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// refreshJWKS fetches the JWKS from GitLab if needed
func (v *Validator) refreshJWKS() error {
	v.keySetMutex.RLock()
	needsRefresh := v.keySet == nil || time.Since(v.lastFetch) > 1*time.Hour
	v.keySetMutex.RUnlock()

	if !needsRefresh {
		return nil
	}

	v.keySetMutex.Lock()
	defer v.keySetMutex.Unlock()

	// Double-check after acquiring write lock
	if v.keySet != nil && time.Since(v.lastFetch) <= 1*time.Hour {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keySet, err := jwk.Fetch(ctx, v.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	v.keySet = keySet
	v.lastFetch = time.Now()

	return nil
}

// isValidIssuer checks if the issuer is in the allowed list
func (v *Validator) isValidIssuer(issuer string) bool {
	for _, allowed := range v.issuers {
		if allowed == issuer {
			return true
		}
	}
	return false
}
