# API Client

회원가입, 로그인, 홈데이터 조회 기능을 제공하는 API 클라이언트입니다.

## 기능

- ✅ 회원가입 (AES 암호화 지원)
- ✅ 로그인 (AES 암호화 지원)
- ✅ 홈데이터 조회 (JWT 인증 지원)

## 사용 방법

### 1. 클라이언트 생성

```go
package main

import (
    "ms-gateway/client"
)

func main() {
    // 서버 기본 URL과 AES 키 설정
    baseURL := "http://localhost:8080"
    aesKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 32바이트 hex 문자열
    
    // 클라이언트 생성
    apiClient, err := client.NewAPIClient(baseURL, aesKeyHex)
    if err != nil {
        panic(err)
    }
}
```

### 2. 회원가입

```go
import "ms-gateway/protocol"

registReq := protocol.RegistReq{
    ID:      "testuser",
    PW:      "password123",
    Name:    "테스트 사용자",
    Gender:  1, // 1: 남성, 0: 여성
    Age:     "25",
    Birth:   "1998-01-01",
    Area:    "서울",
    Email:   "test@example.com",
    SPIntro: "반가워요 큐피톡에서 만나요!",
}

err := apiClient.RegistUser(registReq)
if err != nil {
    fmt.Printf("회원가입 실패: %v\n", err)
    return
}
fmt.Println("회원가입 성공!")
```

### 3. 로그인

```go
loginReq := protocol.LoginReq{
    ID: "testuser",
    PW: "password123",
}

loginResp, err := apiClient.LoginUser(loginReq)
if err != nil {
    fmt.Printf("로그인 실패: %v\n", err)
    return
}

fmt.Printf("로그인 성공!\n")
fmt.Printf("UID: %s\n", loginResp.UID)
fmt.Printf("Access Token: %s\n", loginResp.AccessToken)
fmt.Printf("Refresh Token: %s\n", loginResp.RefreshToken)
```

### 4. 홈데이터 조회

```go
homeData, err := apiClient.GetHomeData(loginResp.AccessToken)
if err != nil {
    fmt.Printf("홈데이터 조회 실패: %v\n", err)
    return
}

fmt.Printf("신규 알림: %d개\n", homeData.NotiNew)
fmt.Printf("쪽지 개수: %d개\n", homeData.MsgQty)
fmt.Printf("출석체크: %v\n", homeData.CheckIn)

if homeData.VdChatList != nil {
    fmt.Printf("영상통화 리스트: %d개\n", len(*homeData.VdChatList))
}

if homeData.VoChatList != nil {
    fmt.Printf("음성통화 리스트: %d개\n", len(*homeData.VoChatList))
}

if homeData.StoryList != nil {
    fmt.Printf("스토리 리스트: %d개\n", len(*homeData.StoryList))
}
```

## 전체 예제

`api_client_test.go` 파일의 `TestClient()` 함수를 참고하세요.

```go
package main

import (
    "fmt"
    "ms-gateway/client"
    "ms-gateway/protocol"
    "os"
)

func main() {
    // 환경 변수에서 설정 읽기
    baseURL := os.Getenv("API_BASE_URL")
    if baseURL == "" {
        baseURL = "http://localhost:8080"
    }

    aesKeyHex := os.Getenv("AES_KEY_HEX")
    if aesKeyHex == "" {
        fmt.Println("에러: AES_KEY_HEX 환경 변수가 설정되지 않았습니다.")
        return
    }

    // 클라이언트 생성
    apiClient, err := client.NewAPIClient(baseURL, aesKeyHex)
    if err != nil {
        panic(err)
    }

    // 1. 회원가입
    registReq := protocol.RegistReq{
        ID:      "testuser",
        PW:      "password123",
        Name:    "테스트 사용자",
        Gender:  1,
        Age:     "25",
        Birth:   "1998-01-01",
        Area:    "서울",
        Email:   "test@example.com",
        SPIntro: "반가워요 큐피톡에서 만나요!",
    }

    err = apiClient.RegistUser(registReq)
    if err != nil {
        fmt.Printf("회원가입 실패: %v\n", err)
    }

    // 2. 로그인
    loginReq := protocol.LoginReq{
        ID: "testuser",
        PW: "password123",
    }

    loginResp, err := apiClient.LoginUser(loginReq)
    if err != nil {
        fmt.Printf("로그인 실패: %v\n", err)
        return
    }

    // 3. 홈데이터 조회
    homeData, err := apiClient.GetHomeData(loginResp.AccessToken)
    if err != nil {
        fmt.Printf("홈데이터 조회 실패: %v\n", err)
        return
    }

    fmt.Printf("홈데이터 조회 성공!\n")
    fmt.Printf("신규 알림: %d개\n", homeData.NotiNew)
    fmt.Printf("쪽지 개수: %d개\n", homeData.MsgQty)
}
```

## 환경 변수 설정

테스트를 실행하기 전에 다음 환경 변수를 설정하세요:

```bash
export API_BASE_URL=http://localhost:8080
export AES_KEY_HEX=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

## 주의사항

1. **AES 키**: AES 키는 반드시 32바이트(256비트) hex 문자열이어야 합니다.
2. **JWT 토큰**: 홈데이터 조회 시 로그인에서 받은 Access Token을 사용해야 합니다.
3. **에러 처리**: 모든 함수는 에러를 반환하므로 적절히 처리해야 합니다.

## API 엔드포인트

- `POST /acc/v01/regist`: 회원가입 (AES 암호화 필요)
- `POST /acc/v01/login`: 로그인 (AES 암호화 필요)
- `GET /home/v01/mdata`: 홈데이터 조회 (JWT 인증 필요)

