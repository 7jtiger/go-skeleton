package utils

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20"
)

func GenerateRandomBytes(n int) ([]byte, error) {
	bytes := make([]byte, n)
	_, err := crand.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// AES-256 GCM 암호화
func EncryptGCM(plaintext []byte, key []byte) (string, error) {
	// 키 길이 확인
	if len(key) != 32 {
		return "", fmt.Errorf("key length must be 32 bytes (256 bits)")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// GCM 모드 생성
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 논스(Nonce) 생성
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return "", err
	}

	// 암호화 및 인증 태그 추가
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// base64로 인코딩
	encodedText := base64.StdEncoding.EncodeToString(ciphertext)

	return encodedText, nil
}

func DecryptGCM(encryptedText string, key []byte) ([]byte, error) {

	// 키 길이 확인
	// 256비트 = 32바이트
	if len(key) != 32 {
		return nil, fmt.Errorf("key length must be 32 bytes (256 bits)")
	}

	// base64 디코딩
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return nil, err
	}

	//GCM 블록 객체
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// GCM 모드 생성
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 논스 크기 확인
	// 논스 포함된 암호문은 최소 논스 크기 이상이어야 한다.
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext is too short")
	}

	// 암호문에서 논스 추출 및 복호화
	// 나머지 부분은 실제 암호화된 데이터
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	// 추출한 논스와 암호문을 사용해서 복호화 및 인증
	// 데이터 변조시 에러 발생
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptChaCha20 encrypts data using ChaCha20 with a key in the format {prefix}_{uuid}_{?}.
func EncryptChaCha20(data, strkey string) (string, error) {
	// Ensure key is 32 bytes (ChaCha20 requires 256-bit key)
	key := []byte(strkey)
	if len(key) < 32 {
		key = append(key, make([]byte, 32-len(key))...) // Pad with zeros
	} else if len(key) > 32 {
		key = key[:32] // Truncate to 32 bytes
	}

	// Generate random nonce (8 bytes for ChaCha20)
	nonce := make([]byte, chacha20.NonceSize)
	if _, err := crand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Create ChaCha20 cipher
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	// Encrypt data
	dataBytes := []byte(data)
	ciphertext := make([]byte, len(dataBytes))
	cipher.XORKeyStream(ciphertext, dataBytes)

	// Combine nonce and ciphertext, encode to base64
	encrypted := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptChaCha20 decrypts data using ChaCha20 with a key in the format {prefix}_{uuid}_{?}.
func DecryptChaCha20(encryptedData, strkey string) (string, error) {
	// Decode base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %v", err)
	}

	// Validate data length
	if len(data) < chacha20.NonceSize {
		return "", fmt.Errorf("encrypted data too short")
	}

	// Split nonce and ciphertext
	nonce, ciphertext := data[:chacha20.NonceSize], data[chacha20.NonceSize:]

	// Generate key
	key := []byte(strkey)
	if len(key) < 32 {
		key = append(key, make([]byte, 32-len(key))...) // Pad with zeros
	} else if len(key) > 32 {
		key = key[:32] // Truncate to 32 bytes
	}

	// Create ChaCha20 cipher
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	// Decrypt data
	plaintext := make([]byte, len(ciphertext))
	cipher.XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}
