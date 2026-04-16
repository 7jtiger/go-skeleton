package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	// "github.com/google/uuid"
)

// JWTClaims JWT 클레임 구조체 - 표준 claims + 커스텀 필드
type JWTClaims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

func GetJWTClaims(uidStr string, expiration time.Duration) *JWTClaims {
	now := time.Now()
	return &JWTClaims{
		UserID: uidStr,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cupitok.com",
			Subject:   "Authentication",
			Audience:  jwt.ClaimStrings{uidStr},
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uidStr,
		},
	}

	// ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	// IssuedAt:  jwt.NewNumericDate(time.Now()),
	// NotBefore: jwt.NewNumericDate(time.Now()),
}

// CreateJWTTokenWithConfig 설정을 포함한 JWT 토큰 생성
// func CreateJWTToken(secret string, claims *JWTClaims, expiration time.Duration) (string, error) {
func CreateJWTToken(secret string, claims *JWTClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyJWTToken JWT 토큰 검증
func VerifyJWTToken(tokenString, secret string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

// VerifyJWTTokenWithAudience 특정 Audience를 검증하는 JWT 토큰 검증
func VerifyJWTTokenWithAudience(tokenString, secret, expectedAudience string) (*JWTClaims, error) {
	claims, err := VerifyJWTToken(tokenString, secret)
	if err != nil {
		return nil, err
	}

	// Audience 검증
	if expectedAudience != "" {
		audienceValid := false
		for _, aud := range claims.Audience {
			if aud == expectedAudience {
				audienceValid = true
				break
			}
		}
		if !audienceValid {
			return nil, jwt.ErrTokenInvalidAudience
		}
	}

	return claims, nil
}

func IsTokenExpiringSoon(tokenString, secret string, threshold time.Duration) (bool, error) {
	claims, err := VerifyJWTToken(tokenString, secret)
	if err != nil {
		return false, err
	}

	if claims.RegisteredClaims.ExpiresAt.Before(time.Now().Add(threshold)) {
		return true, nil
	} else {
		return false, nil
	}
}
