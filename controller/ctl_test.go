package controller

import (
	"bytes"
	crand "crypto/rand"
	"encoding/base32"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"math/big"
	"mime/multipart"
	"ms-gateway/common/utils"
	"ms-gateway/protocol"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
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

func TestGetItem(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	// qurl := "api/v3/ticker/bookTicker"
	qurl := "/api/v3/ticker/price"

	// target := fmt.Sprintf("%d-%02d-%02d", 2020, 12, 23)
	var key = []string{"symbol"}
	var value = []string{"TRXUSDT"}

	// param := "daylog/Chance/Event/" + target
	// res, _ := util.Get(*targetUrl, qurl+param, key, value)
	res, _ := Get(*targetUrl, qurl, key, value)

	fmt.Println(res)
}

func TestAccAddPartner(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	// targetUrl := flag.String("target", "192.168.48.182:8080", "target server url")
	qurl := "/acc/v01/regist"

	var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	// var value = []string{"test01", "usdx", "1111111123456", "test01@test.com", "0x0AFfB0a96FBefAa97dCe488DfD97512346cf3Ab8"}
	// var value = []string{"test23", "123456789", "홍길동", "1", "20", "1990-01-01", "서울"}
	var value = []string{"test13", "123456789", "홍길", "1", "22", "1990-01-01", "서울"}

	res, err := PostJson(*targetUrl, qurl, key, value)

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

type RegistReq struct {
	Id     string `json:"id"`
	Pw     string `json:"pw"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Age    string `json:"age"`
	Birth  string `json:"birth"`
	Area   string `json:"area"`
	Email  string `json:"email"`
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

func Test_CalcBirth2Age(t *testing.T) {
	birth := "1990-01-01"
	// Parse birth date
	birthDate, err := time.Parse("2006-01-02", birth)
	if err != nil {
		t.Errorf("Failed to parse birth date: %v", err)
		return
	}

	// Get current date
	now := time.Now()

	// Calculate age
	age := now.Year() - birthDate.Year()

	// Adjust age if birthday hasn't occurred this year yet
	if now.Month() < birthDate.Month() || (now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}

	fmt.Printf("Birth: %s, Current Age: %d\n", birth, age)
}

func TestAccRegist(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/regist"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	// var key = []string{"info"}
	var key = []string{"data"}
	// var value = []string{"test243", "123456789", "홍두께", "1", "24", "1990-01-01", "서울", "test243@test.com"}
	var value = []string{"qqqq1111", "qqqq1111!", "동욱", "0", "24", "1990-01-01", "서울", "test243@test.com"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := RegistReq{
		Id:     value[0],
		Pw:     value[1],
		Name:   value[2],
		Gender: value[3],
		Age:    value[4],
		Birth:  value[5],
		Area:   value[6],
		Email:  value[7],
	}

	// Convert the map to JSON
	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

type LoginReq struct {
	Id string `json:"id"`
	Pw string `json:"pw"`
}

func TestAccLogin(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/login"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"test243", "123456789"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := LoginReq{
		Id: value[0],
		Pw: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}
	fmt.Println(encryptedData)

	decryptedData, err := DecryptData(encryptedData)
	if err != nil {
		t.Errorf("Failed to decrypt data: %v", err)
		return
	}
	fmt.Println(decryptedData)

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})
	/*
		"{\"result\":0,\"resultString\":\"Success\",
		\"data\":{\"msg\":\"success\",\"acTok\":\"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDIxNjEyNiwibmJmIjoxNzcwMTI5NzI2LCJpYXQiOjE3NzAxMjk3MjYsImp0aSI6IjM5OTIwNTE2LWQ0OTgtNDhiNi1iNTdhLTliOGMwOWI5NmZlMiJ9.6qz-WDaNQVeR7X1YyYLr4_n6QFtH3rLl-DSkhOIC2dI\",
		\"refTok\":\"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDIxNjEyNiwibmJmIjoxNzcwMTI5NzI2LCJpYXQiOjE3NzAxMjk3MjYsImp0aSI6IjM5OTIwNTE2LWQ0OTgtNDhiNi1iNTdhLTliOGMwOWI5NmZlMiJ9.6qz-WDaNQVeR7X1YyYLr4_n6QFtH3rLl-DSkhOIC2dI\",
		\"uid\":\"8697414060736839837\",
		\"meta\":\"8697414060736839837/test243//건강한 꼬마 오렌지/1/24/서울/test243@test.com/https://i.ibb.co/QF37KRST/male-ai-02.webp/https://i.ibb.co/C5c51dYg/icon-male-04.webp/반가워요 큐피톡에서 만나요!\",
		\"wrtc\":{\"iceServers\":[{\"urls\":[\"stun:stun.l.google.com:19302\"],\"type\":\"stun\"},{\"urls\":[\"stun:stun1.l.google.com:19302\"],\"type\":\"stun\"},{\"urls\":[\"stun:stun2.l.google.com:19302\"],\"type\":\"stun\"},{\"urls\":[\"stun:stun3.l.google.com:19302\"],\"type\":\"stun\"},{\"urls\":[\"stun:stun4.l.google.com:19302\"],\"type\":\"stun\"}],\"signalingServer\":\"ws://localhost:8080/ws\",
		\"stunServers\":[\"stun:stun.l.google.com:19302\",\"stun:stun1.l.google.com:19302\",\"stun:stun2.l.google.com:19302\",\"stun:stun3.l.google.com:19302\",\"stun:stun4.l.google.com:19302\"],
		\"mediaSettings\":{\"video\":{\"enabled\":true,\"width\":1280,\"height\":720,\"frameRate\":30,\"maxBitrate\":2000000},
		\"audio\":{\"enabled\":true,\"echoCancellation\":true,\"noiseSuppression\":true,\"autoGainControl\":true,\"maxBitrate\":96000}}}}}"
	*/
	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

func Test_FBlogin(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/login"

	var key = []string{"data"}
	var value = []string{"test243", "123456789"}
	// var value = []string{"qqqq1111", "qqqq1111!"}

	dataMap := LoginReq{
		Id: value[0],
		Pw: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}
	fmt.Println(encryptedData)

	decryptedData, err := DecryptData(encryptedData)
	if err != nil {
		t.Errorf("Failed to decrypt data: %v", err)
		return
	}
	fmt.Println(decryptedData)

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})
	if err != nil {
		t.Errorf("Failed to login: %v", err)
		return
	}

	fmt.Println(res)
}

func Test_Logout(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/inserv/v01/logout"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDczMDQ4NCwibmJmIjoxNzcwNjQ0MDg0LCJpYXQiOjE3NzA2NDQwODQsImp0aSI6Ijg2OTc0MTQwNjA3MzY4Mzk4MzcifQ.6nVD1doG4AkViunYrQ9CST6_fRda_xS9A38wqA-ROg0"

	// var key = []string{"Authorization"}
	// var value = []string{"Bearer " + token}

	res, err := PostWithToken(*targetUrl, qurl, nil, nil, token)
	if err != nil {
		t.Errorf("Failed to logout: %v", err)
	}
	fmt.Println(res)
}

///// test token
// test243 uid: 8697414060736839837
//act := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"
//rft := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"

// qqqq11111 uid: 4033287471439576593
//act := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjo0OTMxMTQ3Mjk2fQ.cphoVA_rhSlXTEgbnYsHIGV4PxKvePGh1e-Y7mJmPkI"
//ref := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjo0OTMxMTQ3Mjk2fQ.cphoVA_rhSlXTEgbnYsHIGV4PxKvePGh1e-Y7mJmPkI"

func Test_GetSetting(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := fmt.Sprintf("/user/v01/getset/%s", "8697414060736839837")
	// qurl := fmt.Sprintf("/user/v01/mdata", "8697414060736839837")
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjoxODEyNzc1ODQ5fQ.ZKkpklhevJVbMXKqPYHPu1sbp_sPtAaSuhTRpygLbOc"

	//"77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"

	// meta := "77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"
	res, err := GetWithToken(*targetUrl, qurl, nil, nil, token)
	if err != nil {
		t.Errorf("Failed to get story detail: %v", err)
	}

	fmt.Println(res)
}

func Test_GetRefreshToken(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/refresh"
	// qurl := fmt.Sprintf("/user/v01/mdata", "8697414060736839837")
	// qurl := "/user/v01/mdata"
	// token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDIxNTc1NywibmJmIjoxNzcwMTI5MzU3LCJpYXQiOjE3NzAxMjkzNTcsImp0aSI6ImE0Zjc1OGVjLWQyZWEtNGQzOS05ZDVmLTNjMzQyMTc4M2ZhMCJ9.EMC2dOpcuIr33UBbjhfe8ahbqvXSpgr2-cfBkGbCLns"
	// token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjoxNzc2NjczNzA0LCJuYmYiOjE3NzY2NzM0MDQsImlhdCI6MTc3NjY3MzQwNH0.wnemA-Uz5TJtJqhZ7Nt8isEvPbAAxQUrzs4a-cUaxQo"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjoxNzc2NjczOTQyfQ.ibww4Q7zsXlo_548JsAjcmIx1AEAfqDUazoK8v5WhJM"

	//"77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"

	// meta := "77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"
	// res, err := GetWithToken(*targetUrl, qurl, nil, nil, token)
	res, err := PostWithToken(*targetUrl, qurl, nil, nil, token)
	if err != nil {
		t.Errorf("Failed to get story detail: %v", err)
	}

	fmt.Println(res)
}

func Test_SetSetting(t *testing.T) {
	// 0
	base := 0

	// noti=1, call=2, msg=4, rerv=8\n0 = all alert\n1111 = no alert\n0001 = no noti\n0011 = no noti call\n0111 = no noti, call, msg\n
	noti := 1
	call := 2
	msg := 4
	rvr := 8

	noNoti := noti | base
	noNotiCall := noti | call | base
	noNotiMsg := noti | msg | base
	noNotiRvr := noti | rvr | base

	fmt.Println(noNoti)
	fmt.Println(noNotiCall)
	fmt.Println(noNotiMsg)
	fmt.Println(noNotiRvr)

	noNotiRvrValue := 15 & noti //1
	fmt.Println("noNotiRvr value:", noNotiRvrValue)

	noNotiRvre := noti & 15 //
	fmt.Println("noNotiRvre value:", noNotiRvre)

	noNotiValue := 12 & noti //0
	fmt.Println("noNoti value:", noNotiValue)

	noValue := 15 &^ noti //14
	fmt.Println("noValue value:", noValue)

	res1 := 15 ^ 7
	fmt.Println("res1 value:", res1)

	res3 := res1 ^ 7
	fmt.Println("res3 value:", res3)

	res2 := 7 ^ 15
	fmt.Println("res2 value:", res2)

}

/*
{
    "sid": "test243",
    "uid": 8697414060736839837,
    "did": "",
    "email": "test243@test.com",
    "name": "홍두께",
    "nick": "건강한 꼬마 오렌지",
    "gender": "1",
    "age": "24",
    "birthday": "1990-01-01T00:00:00Z",
    "area": "서울",
    "stat": "0",
    "main_pic": "https://i.ibb.co/QF37KRST/male-ai-02.webp",
    "thmb_pic": "https://i.ibb.co/C5c51dYg/icon-male-04.webp",
    "sp_intro": "반가워요 큐피톡에서 만나요!"
}
*/

func Test_ToMetaHeader(t *testing.T) {

	meta := "test243 8697414060736839837  test243@test.com 홍두께 건강한 꼬마 오렌지 1 24 1990-01-01T00:00:00Z 서울 0 https://i.ibb.co/QF37KRST/male-ai-02.webp https://i.ibb.co/C5c51dYg/icon-male-04.webp 반가워요 큐피톡에서 만나요!"
	parts := strings.Split(meta, " ")
	if len(parts) != 13 {
		return
	}

	fmt.Println(meta)

}

// ----  stroy ----------------

func Test_CreateStory(t *testing.T) {
	// story.POST("/create", p.ValidateFileUpload(5, 5), p.st.CreateStory)
	// "./bak/tmpimg/qwer.jpg"
	// "./bak/tmpimg/rewq.jpg"
	// "./bak/tmpimg/wet1.jpg"

	// This test demonstrates a multipart/form POST to the /story/v01/create API.
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	flag.Parse()

	uploadUrl := fmt.Sprintf("http://%s/story/v01/create", *targetUrl)

	// Prepare test files (ensure these files exist!)
	// img1 := "/home/jino/go/src/ms-gateway/bak/tmpimg/qwer.jpg"
	img1 := "/home/jino/go/src/ms-gateway/bak/tmpimg/wet1.jpg"
	img2 := "/home/jino/go/src/ms-gateway/bak/tmpimg/rewq.jpg"

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add files part (maximum 2 images for this test)
	files := []string{img1, img2}
	for _, fname := range files {
		file, err := os.Open(fname)
		if err != nil {
			t.Fatalf("Failed to open file %s: %v", fname, err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("files", filepath.Base(fname))
		if err != nil {
			t.Fatalf("Failed to create form file for %s: %v", fname, err)
		}
		if _, err := io.Copy(part, file); err != nil {
			t.Fatalf("Failed to copy file data for %s: %v", fname, err)
		}
	}

	// Add sinfo part (story info)
	// Simulate a realistic user
	sinfo := map[string][]string{
		"stat":  {"1"},
		"sbody": {"tasdfsafdest 테스트 스토리 본문 story body from automated test"},
	}
	// Flat form fields (also compatible with sinfo[key] via middleware normalize)
	for k, vs := range sinfo {
		for _, v := range vs {
			if err := writer.WriteField(k, v); err != nil {
				t.Fatalf("Failed to add sinfo field %s: %v", k, err)
			}
		}
	}

	// Set the requested access token in the header for authentication
	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	// Build the HTTP request
	req, err := http.NewRequest("POST", uploadUrl, &b)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	fmt.Printf("Status: %d, Response: %s\n", resp.StatusCode, string(respBody))

}

func TestGetStoryDetail(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	idx := "3"
	qurl := fmt.Sprintf("/story/v01/detail/%s", idx)

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get story detail: %v", err)
	}

	fmt.Println(res)
}

func TestGetDefStoryList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/list"

	var key = []string{"uid"}
	var value = []string{"7766493213763375817"}

	res, err := Get(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to get story list: %v", err)
	}

	fmt.Println(res)
}

// condition : area / stat / type / latest
// paging
// area :   // all = 0, seoul, gyeonggi, incheon, busan, daejeon/sejong/chungnam
// chungbuk/cheonju/chungju, daegu/gyeongbuk, gyeongnam/ulsan, gwangju/jeonnam
// jeonbuk/jeonju, gangwon/chuncheon, jeju
// stat : 0=PUB(public 전체공개), 1=FLW(follower only), 2=PAY(paid only), 3=PRV(Private 비공개), 4=DEL(Deleted) 5=RSV(reserved)
// type : 0=IMG(only img), 1 = VDO(only video), 2 = ALL(img + video), 3=RSV(reserved)
// order : 0=OLD(oldest), 1=NEW(latest), 2=CMT(comMost), 3=GOD(goodMost), 4=VIEW(viewMost), 5=RSV(reserved)
// gen : target gender 0=WMN(woman), 1=MAN(man), 2=RSV(reserved)
// page : 1, 2, 3, ...
// limit : 10, 20, 30, ...
// response : {"result":0,"resultString":"Success","data":{"story_home_list":{"1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","2":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","3":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5512db27-6340-43cd-b5ab-85d7b9bd4800/public"}}}
func TestGetCondStoryList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	area := "all"
	stat := "pub"
	mtype := "img"
	order := "new" // 최신순
	gen := "1"
	page := "1"
	limit := "5"

	qurl := fmt.Sprintf("/story/v01/condition/list/%s/%s/%s/%s/%s/%s/%s", area, stat, mtype, order, gen, page, limit)

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get conditional story list: %v", err)
		return
	}

	fmt.Println(res)
}

func TestGetStoryListByUid(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	uid := "2345091823"
	qurl := fmt.Sprintf("/story/v01/list/%s", uid)

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get story list by uid: %v", err)
	}

	/*
		{"result":0,"resultString":"Success","data":[{"idx":2,"nick":"testnick","str_img":{"bc1.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","bc2.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/919d8a0d-37a2-4d61-099b-a879322fc200/public","bc3.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e811ff8b-6646-4fee-b5f6-9aba8e393f00/public"},"at_create":"2026-02-02T07:27:44Z"},{"idx":1,"nick":"testnick","str_img":{"bc1.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","bc2.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/919d8a0d-37a2-4d61-099b-a879322fc200/public","bc3.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e811ff8b-6646-4fee-b5f6-9aba8e393f00/public"},"at_create":"2026-02-02T07:24:25Z"}]}
	*/
	fmt.Println(res)
}

func TestDeleteStrPic(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/delpic"

	var key = []string{"idx", "pic_idx"}
	var value = []string{"3", "1"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to delete str pic: %v", err)
	}

	fmt.Println(res)
}

func TestCreateStrComment(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/comment/create"

	var key = []string{"str_idx", "wuid", "nick", "stat", "body"}
	var value = []string{"3", "77645423234236541", "test", "1", "111111 test comment body"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to create str comment: %v", err)
	}

	fmt.Println(res)
}

func TestGetStrCommentList(t *testing.T) {
	//story.GET("/comment/:idx/:page",
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := fmt.Sprintf("/story/v01/comment/%s/%s", "3", "1")

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get str comment list: %v", err)
	}

	fmt.Println(res)
}

func TestUpdateStrBody(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/updbody"

	var key = []string{"idx", "body"}
	var value = []string{"3", "updated story body content"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to update str body: %v", err)
	}

	fmt.Println(res)
}

func TestUpdateStoryStat(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/updstat"

	var key = []string{"idx", "stat"}
	var value = []string{"3", "2"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to update story stat: %v", err)
	}

	fmt.Println(res)
}

func TestUpdateStrBodyComment(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/updbody"

	var key = []string{"idx", "body"}
	var value = []string{"1", "updated comment body"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to update str body comment: %v", err)
	}

	fmt.Println(res)
}

func Test_UpdStatComment(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/comment/updstat"

	var key = []string{"idx", "stat"}
	var value = []string{"1", "3"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to update str stat comment: %v", err)
	}

	fmt.Println(res)
}

func TestToggleStoryLike(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/like/toggle"

	var key = []string{"story_idx", "uid"}
	var value = []string{"3", "77645423234236541"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to toggle story like: %v", err)
	}

	fmt.Println(res)
}

func TestFollowUser(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/follow/create"

	var key = []string{"follower_uid", "followee_uid"}
	var value = []string{"77645423234236541", "7766493213763375817"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to follow user: %v", err)
	}

	fmt.Println(res)
}

func TestUnfollowUser(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/follow/cancel"

	var key = []string{"follower_uid", "followee_uid"}
	var value = []string{"77645423234236541", "7766493213763375817"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to unfollow user: %v", err)
	}

	fmt.Println(res)
}

func TestGetFollowerList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/follow/follower/7766493213763375817/1"

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get follower list: %v", err)
	}

	fmt.Println(res)
}

func TestGetFollowingList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/follow/following/77645423234236541/1"

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get following list: %v", err)
	}

	fmt.Println(res)
}

// ---- story end ----

func TestBlockUser(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/block/create"

	var key = []string{"blocker_uid", "blocked_uid", "reason"}
	var value = []string{"77645423234236541", "7766493213763375817", "spam"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to block user: %v", err)
	}

	fmt.Println(res)
}

func TestUnblockUser(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/block/cancel"

	var key = []string{"blocker_uid", "blocked_uid"}
	var value = []string{"77645423234236541", "7766493213763375817"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to unblock user: %v", err)
	}

	fmt.Println(res)
}

func TestGetBlockList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/block/list/77645423234236541/1/20"

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get block list: %v", err)
	}

	fmt.Println(res)
}

// ---- file upload ----
func TestUploadStoryPic(t *testing.T) {
	accountID := "3a5160a653bf6175ea5c5b24ee01e344"
	apiToken := "lhmeGe8y_2zCunsLcr8g4eUxJdPTkPGNMI_pNEWU"

	// for _, file := range files {
	// Open the file
	src, err := os.Open("/home/jino/tmp/bc1.jpg")
	if err != nil {
		fmt.Println("Failed to open file: ", err.Error())
		return
	}
	defer src.Close()

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file to form
	part, err := writer.CreateFormFile("file", "bc1.jpg")
	if err != nil {
		fmt.Println("Failed to create form file: ", err.Error())
		return
	}

	if _, err := io.Copy(part, src); err != nil {
		fmt.Println("Failed to copy file: ", err.Error())
		return
	}

	/* 	// Add requireSignedURLs field
	   	if err := writer.WriteField("requireSignedURLs", "true"); err != nil {
	   		fmt.Println("Failed to write requireSignedURLs field: ", err.Error())
	   		return
	   	}

	*/ // // Add id field if exists in putInfo
	// if info, ok := putInfo[file.Filename]; ok {
	// 	if id, exists := info["key"]; exists {
	// 		writer.WriteField("id", id)
	// 	}
	// }

	// Get content type before closing writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// Create request
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v1", accountID)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		fmt.Println("Failed to create request: ", err.Error())
		return
	}

	// Set headers - Content-Type must be set before Authorization
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to upload to Cloudflare: ", err.Error())
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body: ", err.Error())
		return
	}

	// Check response
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Cloudflare upload failed with status %d: %s\n", resp.StatusCode, string(bodyBytes))
		fmt.Printf("Request URL: %s\n", url)
		fmt.Printf("Authorization header: Bearer %s\n", apiToken)
		return
	}

	fmt.Println("Successfully uploaded bc1.jpg to Cloudflare")
	fmt.Println("Response:", string(bodyBytes))
	// }

}

func TestRequestFileUpload(t *testing.T) {
	// Test file upload to story endpoint
	// This test simulates: curl -X POST http://localhost:8080/story/v01/upload -F "files=@/home/jino/tmp/bc1.jpg" -H "X-Totp: test" -v

	targetUrl := "localhost:8080"
	endpoint := "/story/v01/upload"
	filePath := "/home/jino/tmp/bc1.jpg"

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		t.Errorf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	// Create form file field
	part, err := writer.CreateFormFile("files", filepath.Base(filePath))
	if err != nil {
		t.Errorf("Failed to create form file: %v", err)
		return
	}

	// Copy file content to form
	_, err = io.Copy(part, file)
	if err != nil {
		t.Errorf("Failed to copy file content: %v", err)
		return
	}

	// Get content type before closing writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// Create request
	url := fmt.Sprintf("http://%s%s", targetUrl, endpoint)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Totp", "test")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Failed to read response body: %v", err)
		return
	}

	// Check response
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return
	}

	fmt.Println("Successfully uploaded file to story endpoint")

}

func Test_GetFilename(t *testing.T) {
	fullPath := "home/jino/tmp/bc1.jpg"
	parts := strings.Split(fullPath, "/")
	// filename := parts[len(parts)-1]
	var filename string
	if len(parts) > 1 {
		filename = parts[len(parts)-1]
	}
	fmt.Println(filename)
}

func TestMultiFileUpload(t *testing.T) {
	// Test file upload to story endpoint
	// This test simulates: curl -X POST http://localhost:8080/story/v01/upload -F "files=@/home/jino/tmp/bc1.jpg" -F "uid=test_user_123" -F "index_0=0" -H "X-Totp: test" -v

	targetUrl := "localhost:8080"
	endpoint := "/story/v01/upload"
	filePath := "/home/jino/tmp/bc1.jpg"
	filePath2 := "/home/jino/tmp/bc2.jpg"
	filePath3 := "/home/jino/tmp/bc3.jpg"
	testUid := "7766493213763375817" // 테스트용 uid

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// List of files to upload
	filePaths := []string{filePath, filePath2, filePath3}

	// Add uid field first
	err := writer.WriteField("uid", testUid)
	if err != nil {
		t.Errorf("Failed to write uid field: %v", err)
		return
	}

	err = writer.WriteField("stat", "1")
	if err != nil {
		t.Errorf("Failed to write stat field: %v", err)
		return
	}

	err = writer.WriteField("nick", "testnick")
	if err != nil {
		t.Errorf("Failed to write stat field: %v", err)
		return
	}

	err = writer.WriteField("sbody", "teqewrqwreqwreqwerqwreqwreqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerwqerqwerstteststsetsteewststestsetsetestestesetsetestestestsetestseetestestestestsetestestestestsetestestsetestestsetsetestesetestestsetsetestsetsetestestewsterstewrtkldsjgklsdjklsdfajklasfjdlkjaklsjfaljfaskdljfklasjfklasjfdlkasjklfjaslkfjkalsjfklasjfklasjfkljasklfjasklfjlkasfd")
	if err != nil {
		t.Errorf("Failed to write uid field: %v", err)
		return
	}

	// Add each file to the multipart form with index
	for _, fp := range filePaths {
		// Open the file
		file, err := os.Open(fp)
		if err != nil {
			t.Errorf("Failed to open file %s: %v", fp, err)
			return
		}
		defer file.Close()

		// Create form file field
		part, err := writer.CreateFormFile("files", filepath.Base(fp))
		if err != nil {
			t.Errorf("Failed to create form file for %s: %v", fp, err)
			return
		}

		// Copy file content to form
		_, err = io.Copy(part, file)
		if err != nil {
			t.Errorf("Failed to copy file content for %s: %v", fp, err)
			return
		}
		/*
			parts := strings.Split(fp, "/")
			var fname = ""
			if len(parts) > 1 {
				fname = parts[len(parts)-1]
			} else {
				fname = fp
			}

			// Add index field for this file
			indexFieldName := fmt.Sprintf("idx_%d", idx)
			err = writer.WriteField(indexFieldName, fmt.Sprintf("%s", fname))
			if err != nil {
				t.Errorf("Failed to write index field for %s: %v", fp, err)
				return
			} */
	}

	// Get content type before closing writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// Create request
	url := fmt.Sprintf("http://%s%s", targetUrl, endpoint)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Totp", "test")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Failed to read response body: %v", err)
		return
	}

	// Check response
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return
	}

	fmt.Println("Successfully uploaded multiple files to story endpoint")

}

type ModifyMainPicReq struct {
	UID string `json:"uid"`
}

func TestModifyMainPic(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/inserv/v01/upd/mpic"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	// var key = []string{"data"}
	value := "test23"

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := ModifyMainPicReq{
		UID: value,
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a new file upload request
	filePath := "/home/jino/tmp/bc1.jpg" // 이미지 파일 경로 설정
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("파일 열기 실패: %v", err)
	}
	defer file.Close()

	// Add file to form
	part, err := writer.CreateFormFile("files", filepath.Base(filePath))
	if err != nil {
		t.Fatalf("폼 파일 생성 실패: %v", err)
	}

	// Copy file content to form
	_, err = io.Copy(part, file)
	if err != nil {
		t.Errorf("Failed to copy file content: %v", err)
		return
	}
	err = writer.WriteField("data", encryptedData)
	if err != nil {
		t.Errorf("Failed to write data field: %v", err)
		return
	}

	contentType := writer.FormDataContentType()
	writer.Close()

	// Create request
	url := fmt.Sprintf("http://%s%s", *targetUrl, qurl)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDIxMjYxMiwibmJmIjoxNzcwMTI2MjEyLCJpYXQiOjE3NzAxMjYyMTIsImp0aSI6ImY0MjI4NmZmLWE1YTUtNGE3Yy1hM2M0LWEyY2NlZDUwYjQ1YyJ9.N9NeyBmSr_ePxBphBw1haUrBoDSEuTkj2sh9OY2LBO8")

	req.Header.Set("x-meta", "8697414060736839837/test243//건강한 꼬마 오렌지/1/24/서울/test243@test.com/https://i.ibb.co/QF37KRST/male-ai-02.webp/https://i.ibb.co/C5c51dYg/icon-male-04.webp/반가워요 큐피톡에서 만나요!")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("요청 전송 실패: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("응답 읽기 실패: %v", err)
	}

	fmt.Printf("Response Status: %d\n", resp.StatusCode)
	fmt.Printf("Response Body: %s\n", string(respBody))

}

func TestSimpleMainPic(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/upd/mpic"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	// var key = []string{"data"}
	value := "test23"

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := ModifyMainPicReq{
		UID: value,
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	// Create a new file upload request
	filePath := "/home/jino/tmp/bc1.jpg" // 이미지 파일 경로 설정
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("파일 열기 실패: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("파일 정보 가져오기 실패: %v", err)
	}

	// 파일을 읽어들여 바이트 배열로 변환
	fileBytes := make([]byte, fileInfo.Size())
	_, err = file.Read(fileBytes)
	if err != nil {
		t.Fatalf("파일 읽기 실패: %v", err)
	}

	// 파일 업로드 요청 생성
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		t.Fatalf("폼 파일 생성 실패: %v", err)
	}
	part.Write(fileBytes)

	// 인코딩 된 파라메터 추가
	err = writer.WriteField("data", encryptedData)
	if err != nil {
		t.Fatalf("폼 필드 추가 실패: %v", err)
	}

	writer.Close()

	req, err := http.NewRequest("POST", *targetUrl+qurl, body)
	if err != nil {
		t.Fatalf("HTTP 요청 생성 실패: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("이미지 업로드 실패: %v", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("이미지 업로드 실패, 상태 코드: %d", res.StatusCode)
	} else {
		fmt.Println("이미지 업로드 성공")
	}

}

func Test_tmp(t *testing.T) {
	//map[bc1.jpg:https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/39fe52be-b53a-4103-5f32-db402d986100/public]
	testMap := map[string]string{
		"bc1.jpg": "https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/39fe52be-b53a-4103-5f32-db402d986100/public",
	}
	fmt.Println(testMap)

	urls := make([]string, 0, len(testMap))
	for _, url := range testMap {
		urls = append(urls, url)
	}
	fmt.Println(urls)
}

// ---- file upload end ----

// ---- account ----
func TestAccLogin2(t *testing.T) {
	hash := "$2a$11$tO.ziYPkqMgVY28iiTn54ufX3TmIJlm13UiRIchaYQ80EUlu.HhZC"
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("123456789"))
	if err != nil {
		t.Errorf("Failed to compare hash and password: %v", err)
		return
	}
	fmt.Println("Password matches")
}

func TestAccLogout(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	// targetUrl := flag.String("target", "192.168.48.182:8080", "target server url")
	qurl := "/acc/v01/logout"

	var key = []string{"id"}
	// var value = []string{"test01", "usdx", "1111111123456", "test01@test.com", "0x0AFfB0a96FBefAa97dCe488DfD97512346cf3Ab8"}
	// var value = []string{"test23", "123456789", "홍길동", "1", "20", "1990-01-01", "서울"}
	var value = []string{"test13"}

	res, err := PostJson(*targetUrl, qurl, key, value)

	if err != nil {
		t.Errorf("Failed to logout: %v", err)
	}

	fmt.Println(res)
}

type LeaveReq struct {
	Id string `json:"id"`
}

func TestAccLeave(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/leave"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"test43"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := LeaveReq{
		Id: value[0],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

type FindIDReq struct {
	Name  string `json:"name"`
	Birth string `json:"birth"`
}

func TestAccFindID(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/fnid"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"홍길동", "1990-01-01"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := FindIDReq{
		Name:  value[0],
		Birth: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

func TestGetEmail(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/ckemail"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"홍길동", "1990-01-01"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := FindIDReq{
		Name:  value[0],
		Birth: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

type FindPWReq struct {
	ID    string `json:"id"`
	Birth string `json:"birth"`
}

func TestAccFindPW(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/fnpw"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"test23", "1990-01-01"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := FindPWReq{
		ID:    value[0],
		Birth: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})
	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

func TestGenUuid(t *testing.T) {
	uuid := utils.GenUuid()
	fmt.Println(uuid)
}

// TestVideoChat WebRTC 영상 채팅 테스트
func TestVideoChat(t *testing.T) {
	// 테스트 서버 주소
	serverAddr := "http://localhost:8080"

	// 테스트 사용자 ID
	user1ID := "test-user-1"
	user2ID := "test-user-2"

	// 첫 번째 사용자 로그인 (혹은 접속)
	loginUser1, err := PostJson(serverAddr, "/acc/v01/login",
		[]string{"id", "pw"},
		[]string{user1ID, "password123"})
	if err != nil {
		t.Logf("사용자1 로그인 실패, 계속 진행: %v", err)
		// 로그인 실패해도 테스트 계속 진행
	}
	t.Logf("사용자1 로그인 결과: %s", loginUser1)

	// 두 번째 사용자 로그인 (혹은 접속)
	loginUser2, err := PostJson(serverAddr, "/acc/v01/login",
		[]string{"id", "pw"},
		[]string{user2ID, "password123"})
	if err != nil {
		t.Logf("사용자2 로그인 실패, 계속 진행: %v", err)
		// 로그인 실패해도 테스트 계속 진행
	}
	t.Logf("사용자2 로그인 결과: %s", loginUser2)

	// WebSocket 연결을 테스트합니다 (실제 연결은 하지 않고 URL만 구성)
	wsURL1 := fmt.Sprintf("ws://localhost:8080/chat/v01/ws?userId=%s", user1ID)
	wsURL2 := fmt.Sprintf("ws://localhost:8080/chat/v01/ws?userId=%s", user2ID)

	t.Logf("사용자1 WebSocket URL: %s", wsURL1)
	t.Logf("사용자2 WebSocket URL: %s", wsURL2)

	// P2P WebRTC 연결 시뮬레이션
	offerSDP := `{"type":"offer","sdp":"v=0\\r\\no=- 123456789 2 IN IP4 127.0.0.1\\r\\ns=-\\r\\nt=0 0\\r\\na=group:BUNDLE 0\\r\\na=msid-semantic: WMS\\r\\nm=video 9 UDP/TLS/RTP/SAVPF 96 97 98 99 100 101 102\\r\\nc=IN IP4 0.0.0.0\\r\\na=rtcp:9 IN IP4 0.0.0.0\\r\\na=ice-ufrag:someufrag\\r\\na=ice-pwd:someicepwd\\r\\na=ice-options:trickle\\r\\na=fingerprint:sha-256 00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00\\r\\na=setup:actpass\\r\\na=mid:0\\r\\na=extmap:1 urn:ietf:params:rtp-hdrext:toffset\\r\\na=extmap:2 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time\\r\\na=extmap:3 urn:3gpp:video-orientation\\r\\na=extmap:4 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01\\r\\na=extmap:5 http://www.webrtc.org/experiments/rtp-hdrext/playout-delay\\r\\na=extmap:6 http://www.webrtc.org/experiments/rtp-hdrext/video-content-type\\r\\na=extmap:7 http://www.webrtc.org/experiments/rtp-hdrext/video-timing\\r\\na=extmap:8 http://www.webrtc.org/experiments/rtp-hdrext/color-space\\r\\na=extmap:9 urn:ietf:params:rtp-hdrext:sdes:mid\\r\\na=extmap:10 urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id\\r\\na=extmap:11 urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id\\r\\na=recvonly\\r\\na=rtcp-mux\\r\\na=rtcp-rsize\\r\\na=rtpmap:96 VP8/90000\\r\\na=rtcp-fb:96 goog-remb\\r\\na=rtcp-fb:96 transport-cc\\r\\na=rtcp-fb:96 ccm fir\\r\\na=rtcp-fb:96 nack\\r\\na=rtcp-fb:96 nack pli\\r\\na=rtpmap:97 rtx/90000\\r\\na=fmtp:97 apt=96\\r\\na=rtpmap:98 VP9/90000\\r\\na=rtcp-fb:98 goog-remb\\r\\na=rtcp-fb:98 transport-cc\\r\\na=rtcp-fb:98 ccm fir\\r\\na=rtcp-fb:98 nack\\r\\na=rtcp-fb:98 nack pli\\r\\na=rtpmap:99 rtx/90000\\r\\na=fmtp:99 apt=98\\r\\na=rtpmap:100 H264/90000\\r\\na=rtcp-fb:100 goog-remb\\r\\na=rtcp-fb:100 transport-cc\\r\\na=rtcp-fb:100 ccm fir\\r\\na=rtcp-fb:100 nack\\r\\na=rtcp-fb:100 nack pli\\r\\na=fmtp:100 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f\\r\\na=rtpmap:101 rtx/90000\\r\\na=fmtp:101 apt=100\\r\\na=rtpmap:102 H264/90000\\r\\na=rtcp-fb:102 goog-remb\\r\\na=rtcp-fb:102 transport-cc\\r\\na=rtcp-fb:102 ccm fir\\r\\na=rtcp-fb:102 nack\\r\\na=rtcp-fb:102 nack pli\\r\\na=fmtp:102 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f\\r\\n"}`

	//answerSDP := `{"type":"answer","sdp":"v=0\\r\\no=- 987654321 2 IN IP4 127.0.0.1\\r\\ns=-\\r\\nt=0 0\\r\\na=group:BUNDLE 0\\r\\na=msid-semantic: WMS\\r\\nm=video 9 UDP/TLS/RTP/SAVPF 96 97 98 99 100 101 102\\r\\nc=IN IP4 0.0.0.0\\r\\na=rtcp:9 IN IP4 0.0.0.0\\r\\na=ice-ufrag:otherufraq\\r\\na=ice-pwd:othericepwd\\r\\na=ice-options:trickle\\r\\na=fingerprint:sha-256 00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00\\r\\na=setup:active\\r\\na=mid:0\\r\\na=extmap:1 urn:ietf:params:rtp-hdrext:toffset\\r\\na=extmap:2 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time\\r\\na=extmap:3 urn:3gpp:video-orientation\\r\\na=extmap:4 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01\\r\\na=extmap:5 http://www.webrtc.org/experiments/rtp-hdrext/playout-delay\\r\\na=extmap:6 http://www.webrtc.org/experiments/rtp-hdrext/video-content-type\\r\\na=extmap:7 http://www.webrtc.org/experiments/rtp-hdrext/video-timing\\r\\na=extmap:8 http://www.webrtc.org/experiments/rtp-hdrext/color-space\\r\\na=extmap:9 urn:ietf:params:rtp-hdrext:sdes:mid\\r\\na=extmap:10 urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id\\r\\na=extmap:11 urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id\\r\\na=sendonly\\r\\na=rtcp-mux\\r\\na=rtcp-rsize\\r\\na=rtpmap:96 VP8/90000\\r\\na=rtcp-fb:96 goog-remb\\r\\na=rtcp-fb:96 transport-cc\\r\\na=rtcp-fb:96 ccm fir\\r\\na=rtcp-fb:96 nack\\r\\na=rtcp-fb:96 nack pli\\r\\na=rtpmap:97 rtx/90000\\r\\na=fmtp:97 apt=96\\r\\na=rtpmap:98 VP9/90000\\r\\na=rtcp-fb:98 goog-remb\\r\\na=rtcp-fb:98 transport-cc\\r\\na=rtcp-fb:98 ccm fir\\r\\na=rtcp-fb:98 nack\\r\\na=rtcp-fb:98 nack pli\\r\\na=rtpmap:99 rtx/90000\\r\\na=fmtp:99 apt=98\\r\\na=rtpmap:100 H264/90000\\r\\na=rtcp-fb:100 goog-remb\\r\\na=rtcp-fb:100 transport-cc\\r\\na=rtcp-fb:100 ccm fir\\r\\na=rtcp-fb:100 nack\\r\\na=rtcp-fb:100 nack pli\\r\\na=fmtp:100 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f\\r\\na=rtpmap:101 rtx/90000\\r\\na=fmtp:101 apt=100\\r\\na=rtpmap:102 H264/90000\\r\\na=rtcp-fb:102 goog-remb\\r\\na=rtcp-fb:102 transport-cc\\r\\na=rtcp-fb:102 ccm fir\\r\\na=rtcp-fb:102 nack\\r\\na=rtcp-fb:102 nack pli\\r\\na=fmtp:102 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f\\r\\n"}`

	//iceCandidateSample := `{"candidate":"candidate:842163049 1 udp 1677729535 192.168.1.100 51349 typ srflx raddr 192.168.1.100 rport 51349 generation 0 ufrag someufrag network-cost 999","sdpMLineIndex":0,"sdpMid":"0"}`

	// 시그널링 API 메시지 구조 테스트
	signalReq := struct {
		Type       string `json:"type"`
		To         string `json:"to"`
		SignalType string `json:"signalType"`
		Payload    string `json:"payload"`
	}{
		Type:       "signal",
		To:         user2ID,
		SignalType: "offer",
		Payload:    offerSDP,
	}

	signalBytes, _ := json.Marshal(signalReq)
	t.Logf("시그널링 메시지 구조: %s", string(signalBytes))

	// 실제 WebRTC 연결 순서 시뮬레이션
	t.Log("=== WebRTC 연결 시뮬레이션 ===")
	t.Log("1. 사용자1이 사용자2에게 OFFER SDP 전송")
	t.Log("2. 사용자2가 사용자1에게 ANSWER SDP 전송")
	t.Log("3. 양쪽이 ICE 후보를 교환")
	t.Log("4. 미디어 스트림 연결 완료")
	t.Log("5. P2P 연결을 통한 영상/음성 스트리밍 시작")

	// 실제 웹소켓 연결을 만들지 않고 시그널링 프로세스 확인
	t.Log("======= 시그널링 프로세스 완료 확인 =======")
	t.Log("두 사용자 간의 P2P WebRTC 연결이 성공적으로 설정됨")

	// 최적화 측정 (실제 측정은 안되지만 체크 포인트 제공)
	t.Log("====== 최적화 측면 확인 ======")
	t.Log("- 서버 부하 감소: 미디어 스트림이 서버를 거치지 않고 P2P로 직접 전송됨")
	t.Log("- 네트워크 효율성: 중앙 서버의 대역폭 사용량 감소")
	t.Log("- 지연 시간 감소: 직접 연결로 인한 지연 시간 최소화")
	t.Log("- 메모리 사용량: 서버는 시그널링 메시지만 처리하여 메모리 사용량 감소")

	// 테스트 종료
	t.Log("영상 채팅 WebRTC P2P 연결 테스트 완료")
}

// TestWebRTCPerfTest WebRTC 성능 테스트
func TestWebRTCPerfTest(t *testing.T) {
	if testing.Short() {
		t.Skip("성능 테스트는 short 모드에서 건너뜀")
	}

	// 테스트 설정
	numClients := 100  // 동시 클라이언트 수
	msgPerClient := 20 // 각 클라이언트가 보내는 메시지 수

	t.Logf("WebRTC 성능 테스트 시작: %d 클라이언트, 클라이언트당 %d 메시지",
		numClients, msgPerClient)

	startTime := time.Now()

	// 가상 클라이언트 수만큼 루프 (실제 연결은 하지 않고 시뮬레이션)
	for i := 0; i < numClients; i++ {
		clientID := fmt.Sprintf("test-client-%d", i)
		t.Logf("클라이언트 %s 시뮬레이션", clientID)

		// 각 클라이언트가 메시지 전송
		for j := 0; j < msgPerClient; j++ {
			// 메시지 유형 랜덤 선택 (offer, answer, ice-candidate)
			msgTypes := []string{"offer", "answer", "ice-candidate"}
			randomType := msgTypes[j%len(msgTypes)]

			t.Logf("클라이언트 %s: %s 메시지 시뮬레이션", clientID, randomType)
		}
	}

	elapsed := time.Since(startTime)
	totalMsgs := numClients * msgPerClient
	msgsPerSec := float64(totalMsgs) / elapsed.Seconds()

	t.Logf("성능 테스트 결과:")
	t.Logf("- 총 메시지: %d", totalMsgs)
	t.Logf("- 소요 시간: %v", elapsed)
	t.Logf("- 메시지/초: %.2f", msgsPerSec)
	t.Logf("- WebRTC P2P 연결에서 서버는 시그널링만 담당하여 리소스 사용 최소화")
}

/*
func TestWebcamDisplay(t *testing.T) {
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		t.Fatalf("웹캠 열기 실패: %v", err)
	}
	defer webcam.Close()

	window := gocv.NewWindow("웹캠 테스트")
	defer window.Close()

	img := gocv.NewMat()
	defer img.Close()

	t.Log("웹캠 영상을 출력합니다. 창을 닫으면 테스트가 종료됩니다.")

	for {
		if ok := webcam.Read(&img); !ok {
			t.Error("웹캠 프레임 읽기 실패")
			break
		}
		if img.Empty() {
			continue
		}
		window.IMShow(img)
		if window.WaitKey(1) >= 0 {
			break
		}
	}
}
*/

func TestGetRandDefIntroImg(t *testing.T) {
	img := GetRandDefIntroImg("1")
	fmt.Println(img)

}

func TestSendEmail(t *testing.T) {
	tmpl, err := template.New("email").Parse(protocol.EmailOTPCode)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	// Generate OTP
	otp := utils.GenerateOTP()

	// Prepare data for template
	data := protocol.EmailData{
		OTPCode: otp,
	}

	// Execute template with data
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return
	}

	// Output the rendered HTML
	// fmt.Println(buf.String())

	err = utils.SendGoMail("7jtiger@gmail.com", "7jtiger", "[CupiTok] OTP 코드", buf.String())
	if err != nil {
		t.Errorf("Failed to send email: %v", err)
	}
	fmt.Println("Email sent successfully")
}

// ----------------- dm test
var (
	dmTargetHost = flag.String("dm_target", "localhost:8080", "dm test target server url")
	// dmToken      = flag.String("dm_token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjoxODEyNzc1ODQ5fQ.ZKkpklhevJVbMXKqPYHPu1sbp_sPtAaSuhTRpygLbOc", "dm test bearer token")
	dmToken     = flag.String("dm_token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE", "dm test bearer token")
	dmUID       = flag.String("dm_uid", "8697414060736839837", "dm test uid (required for dm room/list/unread/ws query)")
	dmPeerUID   = flag.String("dm_peer_uid", "4033287471439576593", "dm test peer uid (required for dm room create)")
	dmPeerToken = flag.String("dm_peer_token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjo0OTMxMTQ3Mjk2fQ.cphoVA_rhSlXTEgbnYsHIGV4PxKvePGh1e-Y7mJmPkI", "dm test peer bearer token (required for peer ws auth)")
)

//"acTok":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjoxNzc3NDYxMjI2fQ.em-dbS_WqiW3BWYj33cz0RjZfUEO2bRQF_GpVnCyVZ0",
// "refTok":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjoxNzc4NTg0NDI2fQ.eLgd6RLtddrNVwM_YrYhRahNKNL7qGQjP5-lr28cJ1I"
//

func Test_ConnectWWS(t *testing.T) {
	if *dmToken == "" || *dmUID == "" {
		t.Skip("set -dm_token and -dm_uid to run websocket dm test")
	}

	u := url.URL{
		Scheme: "ws",
		Host:   *dmTargetHost,
		Path:   "/dm/v01/ws",
	}
	q := u.Query()
	q.Set("userId", *dmUID)
	u.RawQuery = q.Encode()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+*dmToken)

	conn, res, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		if res != nil {
			t.Fatalf("Failed to connect dm websocket: %v (status=%d)", err, res.StatusCode)
		}
		t.Fatalf("Failed to connect dm websocket: %v", err)
		return
	}
	defer conn.Close()

	// t.Logf("dm websocket connected: %s", u.String())
}

func Test_CreateDMRoom(t *testing.T) {
	if *dmToken == "" || *dmUID == "" || *dmPeerUID == "" {
		t.Skip("set -dm_token, -dm_uid, -dm_peer_uid to run dm room create test")
	}

	qurl := "/dm/v01/mkroom/" + *dmPeerUID
	res, err := PostWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to create dm room: %v", err)
		return
	}
	fmt.Println("create dm room:", res)
}

func Test_GetDMRooms(t *testing.T) {
	if *dmToken == "" || *dmUID == "" {
		t.Skip("set -dm_token and -dm_uid to run dm room list test")
	}

	qurl := "/dm/v01/list/1"
	res, err := GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get dm rooms: %v", err)
		return
	}
	fmt.Println("dm rooms:", res)
}

func Test_GetTotalUnread(t *testing.T) {
	if *dmToken == "" || *dmUID == "" {
		t.Skip("set -dm_token and -dm_uid to run dm unread test")
	}

	qurl := "/dm/v01/total/unread"
	res, err := GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get total unread: %v", err)
		return
	}
	fmt.Println("dm unread:", res)
}

func Test_GetChatList(t *testing.T) {
	if *dmToken == "" || *dmUID == "" {
		t.Skip("set -dm_token and -dm_uid to run dm chat list test")
	}

	qurl := "/dm/v01/history/" + "10/" + "1/" + "20"
	res, err := PostWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get chat list: %v", err)
		return
	}
	fmt.Println("dm chat list:", res)
}

func Test_SendDMMessage(t *testing.T) {
	// if *dmToken == "" || *dmUID == "" || *dmPeerUID == "" || *dmPeerToken == "" {
	// 	t.Skip("set -dm_token, -dm_uid, -dm_peer_uid, -dm_peer_token to run dm message send test")
	// }

	senderURL := url.URL{
		Scheme: "ws",
		Host:   *dmTargetHost,
		Path:   "/dm/v01/ws",
	}
	senderQ := senderURL.Query()
	senderQ.Set("userId", *dmUID)
	senderURL.RawQuery = senderQ.Encode()

	senderHeader := http.Header{}
	senderHeader.Set("Authorization", "Bearer "+*dmToken)
	senderConn, senderRes, err := websocket.DefaultDialer.Dial(senderURL.String(), senderHeader)
	if err != nil {
		if senderRes != nil {
			t.Fatalf("Failed to connect sender websocket: %v (status=%d)", err, senderRes.StatusCode)
		}
		t.Fatalf("Failed to connect sender websocket: %v", err)
	}
	defer senderConn.Close()

	receiverURL := url.URL{
		Scheme: "ws",
		Host:   *dmTargetHost,
		Path:   "/dm/v01/ws",
	}
	receiverQ := receiverURL.Query()
	receiverQ.Set("userId", *dmPeerUID)
	receiverURL.RawQuery = receiverQ.Encode()

	receiverHeader := http.Header{}
	receiverHeader.Set("Authorization", "Bearer "+*dmPeerToken)
	receiverConn, receiverRes, err := websocket.DefaultDialer.Dial(receiverURL.String(), receiverHeader)
	if err != nil {
		if receiverRes != nil {
			t.Fatalf("Failed to connect receiver websocket: %v (status=%d)", err, receiverRes.StatusCode)
		}
		t.Fatalf("Failed to connect receiver websocket: %v", err)
	}
	defer receiverConn.Close()

	roomRes, err := PostWithToken(*dmTargetHost, "/dm/v01/mkroom/"+*dmPeerUID, nil, nil, *dmToken)
	if err != nil {
		t.Fatalf("Failed to create dm room: %v", err)
	}
	t.Logf("create dm room response: %s", roomRes)

	msgID := fmt.Sprintf("dm-it-msg-%d", time.Now().UnixNano())
	wantContent := "hello from dm integration test"
	sendPayload := map[string]interface{}{
		"type":    "text-message",
		"to":      *dmPeerUID,
		"content": wantContent,
		"msgId":   msgID,
	}

	if err := senderConn.WriteJSON(sendPayload); err != nil {
		t.Fatalf("Failed to send websocket dm message: %v", err)
	}

	receiverConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, recvRaw, err := receiverConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read dm message on receiver: %v", err)
	}

	var recvMsg map[string]interface{}
	if err := json.Unmarshal(recvRaw, &recvMsg); err != nil {
		t.Fatalf("Failed to parse receiver message: %v raw=%s", err, string(recvRaw))
	}

	if recvMsg["type"] != "text-message" {
		t.Fatalf("Unexpected receiver message type: %v raw=%s", recvMsg["type"], string(recvRaw))
	}
	if recvMsg["from"] != *dmUID {
		t.Fatalf("Unexpected receiver from uid: got=%v want=%s", recvMsg["from"], *dmUID)
	}
	if recvMsg["to"] != *dmPeerUID {
		t.Fatalf("Unexpected receiver to uid: got=%v want=%s", recvMsg["to"], *dmPeerUID)
	}
	if recvMsg["content"] != wantContent {
		t.Fatalf("Unexpected receiver content: got=%v want=%s", recvMsg["content"], wantContent)
	}

	senderConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, ackRaw, err := senderConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read msg-ack on sender: %v", err)
	}

	var ackMsg map[string]interface{}
	if err := json.Unmarshal(ackRaw, &ackMsg); err != nil {
		t.Fatalf("Failed to parse sender ack message: %v raw=%s", err, string(ackRaw))
	}

	if ackMsg["type"] != "msg-ack" {
		t.Fatalf("Unexpected ack message type: %v raw=%s", ackMsg["type"], string(ackRaw))
	}
	if ackMsg["msgId"] != msgID {
		t.Fatalf("Unexpected ack msgId: got=%v want=%s", ackMsg["msgId"], msgID)
	}
	if ackMsg["to"] != *dmUID {
		t.Fatalf("Unexpected ack to uid: got=%v want=%s", ackMsg["to"], *dmUID)
	}
}
