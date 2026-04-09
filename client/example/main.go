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
		fmt.Println("경고: API_BASE_URL 환경 변수가 설정되지 않았습니다.")
		fmt.Println("기본값 사용: http://localhost:8080")
		baseURL = "http://localhost:8080"
	}

	aesKeyHex := os.Getenv("AES_KEY_HEX")
	if aesKeyHex == "" {
		fmt.Println("에러: AES_KEY_HEX 환경 변수가 설정되지 않았습니다.")
		fmt.Println("설정 예시:")
		fmt.Println("  export AES_KEY_HEX=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
		fmt.Println("\n또는 conf/config.toml 파일의 Server.BaseKey 값을 hex 문자열로 변환하여 사용하세요.")
		os.Exit(1)
	}

	// 클라이언트 생성
	apiClient, err := client.NewAPIClient(baseURL, aesKeyHex)
	if err != nil {
		fmt.Printf("클라이언트 생성 실패: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== API 클라이언트 테스트 시작 ===")

	// 1. 회원가입
	fmt.Println("1. 회원가입 시도...")
	registReq := protocol.RegistReq{
		ID:      "testuser123",
		PW:      "testpassword",
		Name:    "테스트 사용자",
		Gender:  "1", // 1: 남성, 0: 여성
		Age:     "25",
		Birth:   "1998-01-01",
		Area:    "서울",
		Email:   "testuser123@example.com",
		SPIntro: "반가워요 큐피톡에서 만나요!",
	}

	err = apiClient.RegistUser(registReq)
	if err != nil {
		fmt.Printf("   회원가입 실패: %v\n", err)
		fmt.Println("   (이미 존재하는 사용자일 수 있습니다. 계속 진행합니다.)")
	} else {
		fmt.Println("   회원가입 성공!")
	}

	// 2. 로그인
	fmt.Println("2. 로그인 시도...")
	loginReq := protocol.LoginReq{
		ID: "testuser123",
		PW: "testpassword",
	}

	loginResp, err := apiClient.LoginUser(loginReq)
	if err != nil {
		fmt.Printf("   로그인 실패: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   로그인 성공!")
	fmt.Printf("   - UID: %s\n", loginResp.UID)
	fmt.Printf("   - Access Token: %s...\n", loginResp.AccessToken[:min(20, len(loginResp.AccessToken))])
	fmt.Printf("   - Message: %s\n\n", loginResp.Message)

	// 3. 홈데이터 조회
	fmt.Println("3. 홈데이터 조회 시도...")
	homeData, err := apiClient.GetHomeData(loginResp.AccessToken)
	if err != nil {
		fmt.Printf("   홈데이터 조회 실패: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   홈데이터 조회 성공!")
	fmt.Printf("   - 신규 알림: %d개\n", homeData.NotiNew)
	fmt.Printf("   - 쪽지 개수: %d개\n", homeData.MsgQty)
	fmt.Printf("   - 출석체크: %v\n", homeData.CheckIn)

	if homeData.VdChatList != nil {
		fmt.Printf("   - 영상통화 리스트: %d개\n", len(*homeData.VdChatList))
	} else {
		fmt.Printf("   - 영상통화 리스트: 없음\n")
	}

	if homeData.VoChatList != nil {
		fmt.Printf("   - 음성통화 리스트: %d개\n", len(*homeData.VoChatList))
	} else {
		fmt.Printf("   - 음성통화 리스트: 없음\n")
	}

	if homeData.StoryList != nil {
		fmt.Printf("   - 스토리 리스트: %d개\n", len(*homeData.StoryList))
	} else {
		fmt.Printf("   - 스토리 리스트: 없음\n")
	}

	fmt.Printf("   - 이용약관 링크: %s\n", homeData.TermsLink)
	fmt.Printf("   - 개인정보처리방침 링크: %s\n", homeData.PolicyLink)

	fmt.Println("\n=== API 클라이언트 테스트 완료 ===")
}

// min 함수는 Go 1.21 이전 버전을 위한 헬퍼 함수
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
