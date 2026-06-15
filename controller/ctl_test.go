package controller

import (
	"bytes"
	crand "crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"ms-gateway/common/utils"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	// "gocv.io/x/gocv"
)

func genOtp() string {
	secret := "123456"
	encodedSecret := base32.StdEncoding.EncodeToString([]byte(secret))
	otp, err := totp.GenerateCode(encodedSecret, time.Now())
	if err != nil {
		fmt.Println("Error generating OTP:", err)
		return ""
	}

	return otp
}

func TestGenOtp(t *testing.T) {
	secret := "123456"
	encodedSecret := base32.StdEncoding.EncodeToString([]byte(secret))
	otp, err := totp.GenerateCode(encodedSecret, time.Now())
	fmt.Println(otp)
	if err != nil {
		fmt.Println("Error generating OTP:", err)

	}

	fmt.Println(otp)
}

func Get(host, relativePath string, keys []string, values []string) (string, error) {
	// if len(keys) != len(values) {
	// 	return "", fmt.Errorf("mismatch length of keys and values")
	// }
	u := url.URL{Scheme: "http", Host: host, Path: relativePath}
	if request, err := http.NewRequest("GET", u.String(), nil); err != nil {
		return "", err
	} else {
		q := request.URL.Query()
		for i, key := range keys {
			q.Add(key, values[i])
		}
		request.URL.RawQuery = q.Encode()

		client := http.Client{
			//Timeout: 50e9,
		}

		if response, err := client.Do(request); err != nil {
			return "", err
		} else {
			defer response.Body.Close()

			if body, err := io.ReadAll(response.Body); err != nil {
				return "", err
			} else {
				return string(body), nil
			}
		}
	}
}

func GetWithToken(host, relativePath string, keys []string, values []string, token string) (string, error) {
	// if len(keys) != len(values) {
	// 	return "", fmt.Errorf("mismatch length of keys and values")
	// }
	u := url.URL{Scheme: "http", Host: host, Path: relativePath}
	if request, err := http.NewRequest("GET", u.String(), nil); err != nil {
		return "", err
	} else {
		q := request.URL.Query()
		for i, key := range keys {
			q.Add(key, values[i])
		}
		request.URL.RawQuery = q.Encode()

		// Set Authorization header with Bearer token
		request.Header.Set("Authorization", "Bearer "+token)

		client := http.Client{
			//Timeout: 50e9,
		}

		if response, err := client.Do(request); err != nil {
			return "", err
		} else {
			defer response.Body.Close()

			if body, err := io.ReadAll(response.Body); err != nil {
				return "", err
			} else {
				return string(body), nil
			}
		}
	}
}

func PostWithToken(host, relativePath string, keys []string, values []string, token string) (string, error) {
	u := url.URL{Scheme: "http", Host: host, Path: relativePath}

	// Create a map to hold the JSON data
	jsonData := make(map[string]string)
	for i, key := range keys {
		jsonData[key] = values[i]
	}

	// Convert map to JSON
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return "", err
	}

	if request, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonBytes)); err != nil {
		return "", err
	} else {
		request.Header.Set("Content-Type", "application/json")

		// Set Authorization header with Bearer token
		request.Header.Set("Authorization", "Bearer "+token)

		client := http.Client{
			//Timeout: 50e9,
		}

		if response, err := client.Do(request); err != nil {
			return "", err
		} else {
			defer response.Body.Close()

			if body, err := io.ReadAll(response.Body); err != nil {
				return "", err
			} else {
				return string(body), nil
			}
		}
	}
}

func PostJson(host, relativePath string, keys []string, values []string) (string, error) {
	// if len(keys) != len(values) {
	// 	return "", fmt.Errorf("mismatch length of keys and values")
	// }
	u := url.URL{Scheme: "http", Host: host, Path: relativePath}

	// Create a map to hold the JSON data
	jsonData := make(map[string]string)
	for i, key := range keys {
		jsonData[key] = values[i]
	}

	// Convert map to JSON
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return "", err
	}

	if request, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonBytes)); err != nil {
		return "", err
	} else {
		request.Header.Set("Content-Type", "application/json")

		// Generate OTP and set it in the header
		otp := genOtp()
		request.Header.Set("X-Otp", otp)

		client := http.Client{
			//Timeout: 50e9,
		}

		if response, err := client.Do(request); err != nil {
			return "", err
		} else {
			defer response.Body.Close()

			if body, err := io.ReadAll(response.Body); err != nil {
				return "", err
			} else {
				return string(body), nil
			}
		}
	}
}

func PostEncJson(host, relativePath string, keys []string, values []string) (string, error) {
	// if len(keys) != len(values) {
	// 	return "", fmt.Errorf("mismatch length of keys and values")
	// }
	u := url.URL{Scheme: "http", Host: host, Path: relativePath}

	// Create a map to hold the JSON data
	jsonData := make(map[string]string)
	for i, key := range keys {
		jsonData[key] = values[i]
	}

	// Convert map to JSON
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return "", err
	}
	/*
		keyBytes := []byte("testtesttesttesttesttesttesttest") // 32 bytes key
		encryptedData, err := utils.EncryptGCM(jsonBytes, keyBytes)
		if err != nil {
			return "", err
		} */

	if request, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonBytes)); err != nil {
		// if request, err := http.NewRequest("POST", u.String(), bytes.NewBuffer([]byte(encryptedData))); err != nil {
		return "", err
	} else {
		request.Header.Set("Content-Type", "application/json")

		// Generate OTP and set it in the header
		otp := genOtp()
		request.Header.Set("OTP-Auth", otp)

		client := http.Client{
			//Timeout: 50e9,
		}

		if response, err := client.Do(request); err != nil {
			return "", err
		} else {
			defer response.Body.Close()

			if body, err := io.ReadAll(response.Body); err != nil {
				return "", err
			} else {
				return string(body), nil
			}
		}
	}
}

func EncryptData(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Encrypt the JSON data using EncryptGCM
	keyBytes := []byte("JX03yhFNF2sh0Zoiu8yLzeCzjPCoCz87") // 32 bytes key
	encryptedData, err := utils.EncryptGCM(jsonData, keyBytes)
	if err != nil {
		return "", err
	}

	return encryptedData, nil
}

func Test_EncryptData(t *testing.T) {
	data := "77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Failed to marshal data:", err)
	}

	// Encrypt the JSON data using EncryptGCM
	keyBytes := []byte("JX03yhFNF2sh0Zoiu8yLzeCzjPCoCz87") // 32 bytes key
	encryptedData, err := utils.EncryptGCM(jsonData, keyBytes)
	if err != nil {
		fmt.Println("Failed to encrypt data: ", err)
	}

	fmt.Println(encryptedData)
}

func DecryptData(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Encrypt the JSON data using EncryptGCM
	keyBytes := []byte("JX03yhFNF2sh0Zoiu8yLzeCzjPCoCz87") // 32 bytes key
	encryptedData, err := utils.EncryptGCM(jsonData, keyBytes)
	if err != nil {
		return "", err
	}

	return encryptedData, nil
}

func Test_DecryptData(t *testing.T) {
	// data := "BNvbgARYucDQhlGQ43bB3MYtqw52ADS08YDy85ZnyNaYK1LWphA6qQ9L1O+9+Dd2p3DKVj2ZY6cfd3SAiCiV8Aah6v02Qd9zWfqXMKLY3N3XNQpgGOHhuwNQN8ol63ypn7SGlnEN3OqY/uRQsKAGqSM/ogvSe8x7M40UwCxcZD1veNZIL8Gr1N1JorqCTgJyFHq4l2a8JUcwDuAF/Gd50A=="
	// data := "bX58xL87yL4eRyQCtgF/CBtAWzTt6Mgrcn6xdPoCDDUDcuDay29eEYK96B8W8kkqTkTfNCfA+1qerXxoyb8waoHYf7ovsKU/bnG5cnWnPjl3m5tDoVZz2y0UAm4in1sbWagUjSTaTXe58wBRl7r83TsBRVBmY030Lt2WQkwRvxJJz1Eo9kryQLWY8QHozgaUk7Fx4VEcZAXLr90="
	// data := "+f7doUUBGS/tqQZy/toBSAFqa/Wny4oq61dv11hjVsvNFdF9Wh1q9mIa6nF/oGDZ3SEqSn3cTPCKesZHZcZu4VA2Mzh941ACLecyri9ImPuDIsCHJ42hwmbCImdaJa38v579Jf6VOnZSF3XMvQLmgaFdIbckIG/pL7r17TweNudoG6RfOyYbFfzwkNl+CaWmJ1GkQpRSqYmSefzfug=="
	// data := "6XMm5LGE6ETe4HRsQbEJWxypTmyYiFEnZaKlyOOhbxaCzGNwTScIFxJGnLQ="
	data := "K3s3VvPcUdZD1X8cDq96OWfb4oE9QuMfcDkKWGZdZju+puABUfQXx8CGvnufyG1Xrq2glSzSsURQ5VplMA1vjbQI5ESoXi6soYjpxcG3dypsYvCcuScHn2jkZHExe4IsvUNTz++fSzIUOggPLU0CsNWv6nw="

	// Encrypt the JSON data using EncryptGCM
	keyBytes := []byte("JX03yhFNF2sh0Zoiu8yLzeCzjPCoCz87") // 32 bytes key
	encryptedData, err := utils.DecryptGCM(data, keyBytes)
	if err != nil {
		t.Errorf("Failed to decrypt data: %v", err)
		return
	}

	fmt.Println(string(encryptedData))
}

func Test_GenUuid(t *testing.T) {
	bytes := make([]byte, 8)
	_, err := crand.Read(bytes)
	if err != nil {
		t.Errorf("Failed to generate random bytes: %v", err)
		return
	}

	// 바이트를 big.Int로 변환
	uid := new(big.Int).SetBytes(bytes)

	// MySQL BIGINT UNSIGNED 범위 제한 (2^63-1)
	maxBigInt := new(big.Int).SetUint64(1<<63 - 1) // 9223372036854775807
	uid.Mod(uid, maxBigInt)

	fmt.Println(uid.String())
}
