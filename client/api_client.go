package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ms-gateway/common/utils"
	"ms-gateway/protocol"
)

// APIClient API 클라이언트 구조체
type APIClient struct {
	BaseURL    string
	AESKey     []byte
	HTTPClient *http.Client
}

// NewAPIClient 새로운 API 클라이언트 생성
// baseURL: 서버 기본 URL (예: "http://localhost:8080")
// aesKeyHex: AES 암호화 키 (32바이트 hex 문자열)
func NewAPIClient(baseURL, aesKeyHex string) (*APIClient, error) {
	// AES 키를 hex 디코딩
	aesKey, err := hex.DecodeString(aesKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	// 키 길이 확인 (32바이트 = 256비트)
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("AES key must be 32 bytes (256 bits), got %d bytes", len(aesKey))
	}

	return &APIClient{
		BaseURL: baseURL,
		AESKey:  aesKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// encryptRequest 요청 데이터를 AES 암호화하여 AesDataForm으로 변환
func (c *APIClient) encryptRequest(data interface{}) (*protocol.AesDataForm, error) {
	// JSON으로 직렬화
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// AES 암호화
	encryptedData, err := utils.EncryptGCM(jsonData, c.AESKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	return &protocol.AesDataForm{
		Data: encryptedData,
	}, nil
}

// decryptResponse 응답 데이터를 복호화
func (c *APIClient) decryptResponse(encryptedData string) ([]byte, error) {
	return utils.DecryptGCM(encryptedData, c.AESKey)
}

// RegistUser 회원가입
// registReq: 회원가입 요청 데이터
func (c *APIClient) RegistUser(registReq protocol.RegistReq) error {
	// 요청 데이터 암호화
	encryptedReq, err := c.encryptRequest(registReq)
	if err != nil {
		return fmt.Errorf("failed to encrypt request: %w", err)
	}

	// JSON으로 직렬화
	reqBody, err := json.Marshal(encryptedReq)
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted request: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequest("POST", c.BaseURL+"/acc/v01/regist", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 요청 전송
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 성공 응답 확인
	if resp.StatusCode != http.StatusOK {
		var errResp protocol.RespHeader
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("registration failed: %s (code: %d)", errResp.Desc, errResp.Result)
		}
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// LoginUser 로그인
// loginReq: 로그인 요청 데이터
// 반환값: 로그인 응답 (AccessToken, RefreshToken, UID, WebRTCConfig 포함)
func (c *APIClient) LoginUser(loginReq protocol.LoginReq) (*protocol.LoginUserResp, error) {
	// 요청 데이터 암호화
	encryptedReq, err := c.encryptRequest(loginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt request: %w", err)
	}

	// JSON으로 직렬화
	reqBody, err := json.Marshal(encryptedReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal encrypted request: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequest("POST", c.BaseURL+"/acc/v01/login", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 요청 전송
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 성공 응답 확인
	if resp.StatusCode != http.StatusOK {
		var errResp protocol.RespHeader
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("login failed: %s (code: %d)", errResp.Desc, errResp.Result)
		}
		return nil, fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 응답 파싱
	var loginResp protocol.LoginUserResp
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	return &loginResp, nil
}

// GetHomeData 홈데이터 조회
// accessToken: JWT Access Token
// 반환값: 홈데이터 (알림, 메시지, 출석체크, 영상/음성 통화 리스트, 스토리 리스트 등)
func (c *APIClient) GetHomeData(accessToken string) (*HomeData, error) {
	// HTTP 요청 생성
	req, err := http.NewRequest("GET", c.BaseURL+"/home/v01/mdata", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// JWT 토큰을 Authorization 헤더에 추가
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// 요청 전송
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 성공 응답 확인
	if resp.StatusCode != http.StatusOK {
		var errResp protocol.RespHeader
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("failed to get home data: %s (code: %d)", errResp.Desc, errResp.Result)
		}
		return nil, fmt.Errorf("failed to get home data with status %d: %s", resp.StatusCode, string(body))
	}

	// 응답 파싱
	var homeData HomeData
	if err := json.Unmarshal(body, &homeData); err != nil {
		return nil, fmt.Errorf("failed to parse home data response: %w", err)
	}

	return &homeData, nil
}

// HomeData 홈데이터 구조체 (controller/home.go의 HomeData와 동일)
type HomeData struct {
	NotiNew    int                    `json:"notiNew"`
	MsgQty     int                    `json:"msgQuantity"`
	CheckIn    bool                   `json:"checkIn"`
	VdChatList *[]protocol.WTRoomUser `json:"videoChatList"`
	VoChatList *[]protocol.WTRoomUser `json:"voiceChatList"`
	StoryList  *[]protocol.Pre7Story  `json:"storyList"`
	TermsLink  string                 `json:"termsLink"`
	PolicyLink string                 `json:"policyLink"`
}
