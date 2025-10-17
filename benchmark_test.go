package main

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims 커스텀 클레임 구조체
type JWTClaims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

// 첫 번째 방식: 커스텀 구조체 + 값 타입
func BenchmarkJWTClaims_ValueType(b *testing.B) {
	userID := "test-user-123"
	expiration := 24 * time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		now := time.Now()
		claims := JWTClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "ms-gateway",
				Subject:   "user-auth",
				Audience:  jwt.ClaimStrings{"ms-gateway-api"},
				ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
				IssuedAt:  jwt.NewNumericDate(now),
				ID:        uuid.New().String(),
			},
		}
		_ = claims
	}
}

// 두 번째 방식: 표준 구조체 + 포인터 타입
func BenchmarkJWTClaims_PointerType(b *testing.B) {
	expirationMinutes := 1440 // 24시간

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		now := time.Now()
		claims := &jwt.RegisteredClaims{
			Issuer:    "ms-gateway",
			Subject:   "user-auth",
			Audience:  jwt.ClaimStrings{"ms-gateway-api"},
			IssuedAt:  &jwt.NumericDate{Time: now},
			ExpiresAt: &jwt.NumericDate{Time: now.Add(time.Minute * time.Duration(expirationMinutes))},
			ID:        uuid.New().String(),
		}
		_ = claims
	}
}

// 메모리 할당 테스트
func BenchmarkJWTClaims_Memory_ValueType(b *testing.B) {
	b.ReportAllocs()
	userID := "test-user-123"
	expiration := 24 * time.Hour

	for i := 0; i < b.N; i++ {
		now := time.Now()
		claims := JWTClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "ms-gateway",
				Subject:   "user-auth",
				Audience:  jwt.ClaimStrings{"ms-gateway-api"},
				ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
				IssuedAt:  jwt.NewNumericDate(now),
				ID:        uuid.New().String(),
			},
		}
		_ = claims
	}
}

func BenchmarkJWTClaims_Memory_PointerType(b *testing.B) {
	b.ReportAllocs()
	expirationMinutes := 1440

	for i := 0; i < b.N; i++ {
		now := time.Now()
		claims := &jwt.RegisteredClaims{
			Issuer:    "ms-gateway",
			Subject:   "user-auth",
			Audience:  jwt.ClaimStrings{"ms-gateway-api"},
			IssuedAt:  &jwt.NumericDate{Time: now},
			ExpiresAt: &jwt.NumericDate{Time: now.Add(time.Minute * time.Duration(expirationMinutes))},
			ID:        uuid.New().String(),
		}
		_ = claims
	}
}
