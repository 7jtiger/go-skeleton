package client

import (
	"fmt"
	"ms-gateway/protocol"
	"os"
)

// ExampleClient 사용 예제
// 이 함수는 실제 테스트가 아니라 사용 예제를 보여주는 함수입니다.
func ExampleClient() {
	// 환경 변수에서 설정 읽기 (실제 사용 시)
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	aesKeyHex := os.Getenv("AES_KEY_HEX")
	if aesKeyHex == "" {
		// 예제용 키 (실제로는 환경 변수나 설정 파일에서 읽어야 함)
		aesKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	}

	// 클라이언트 생성
	client, err := NewAPIClient(baseURL, aesKeyHex)
	if err != nil {
		fmt.Printf("클라이언트 생성 실패: %v\n", err)
		return
	}

	// 1. 회원가입 예제
	fmt.Println("=== 회원가입 예제 ===")
	registReq := protocol.RegistReq{
		ID:      "testuser",
		PW:      "password123",
		Name:    "테스트 사용자",
		Gender:  "1", // 1: 남성, 0: 여성
		Age:     "25",
		Birth:   "1998-01-01",
		Area:    "서울",
		Email:   "test@example.com",
		SPIntro: "반가워요 큐피톡에서 만나요!",
	}

	err = client.RegistUser(registReq)
	if err != nil {
		fmt.Printf("회원가입 실패: %v\n", err)
	} else {
		fmt.Println("회원가입 성공!")
	}

	// 2. 로그인 예제
	fmt.Println("\n=== 로그인 예제 ===")
	loginReq := protocol.LoginReq{
		ID: "testuser",
		PW: "password123",
	}

	loginResp, err := client.LoginUser(loginReq)
	if err != nil {
		fmt.Printf("로그인 실패: %v\n", err)
		return
	}

	fmt.Printf("로그인 성공!\n")
	fmt.Printf("  - UID: %s\n", loginResp.UID)
	fmt.Printf("  - Access Token: %s\n", loginResp.AccessToken[:20]+"...")
	fmt.Printf("  - Refresh Token: %s\n", loginResp.RefreshToken[:20]+"...")
	fmt.Printf("  - Message: %s\n", loginResp.Message)

	// 3. 홈데이터 조회 예제
	fmt.Println("\n=== 홈데이터 조회 예제 ===")
	homeData, err := client.GetHomeData(loginResp.AccessToken)
	if err != nil {
		fmt.Printf("홈데이터 조회 실패: %v\n", err)
		return
	}

	fmt.Printf("홈데이터 조회 성공!\n")
	fmt.Printf("  - 신규 알림: %d개\n", homeData.NotiNew)
	fmt.Printf("  - 쪽지 개수: %d개\n", homeData.MsgQty)
	fmt.Printf("  - 출석체크: %v\n", homeData.CheckIn)

	if homeData.VdChatList != nil {
		fmt.Printf("  - 영상통화 리스트: %d개\n", len(*homeData.VdChatList))
	}

	if homeData.VoChatList != nil {
		fmt.Printf("  - 음성통화 리스트: %d개\n", len(*homeData.VoChatList))
	}

	if homeData.StoryList != nil {
		fmt.Printf("  - 스토리 리스트: %d개\n", len(*homeData.StoryList))
	}

	fmt.Printf("  - 이용약관 링크: %s\n", homeData.TermsLink)
	fmt.Printf("  - 개인정보처리방침 링크: %s\n", homeData.PolicyLink)
}

// TestClient 전체 플로우 테스트 함수
// 실제 테스트를 실행하려면 이 함수를 테스트 함수로 변경하세요.
func TestClient() {
	// 환경 변수 설정 확인
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		fmt.Println("경고: API_BASE_URL 환경 변수가 설정되지 않았습니다. 기본값 사용: http://localhost:8080")
		baseURL = "http://localhost:8080"
	}

	aesKeyHex := os.Getenv("AES_KEY_HEX")
	if aesKeyHex == "" {
		fmt.Println("에러: AES_KEY_HEX 환경 변수가 설정되지 않았습니다.")
		fmt.Println("설정 예시: export AES_KEY_HEX=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
		return
	}

	// 클라이언트 생성
	client, err := NewAPIClient(baseURL, aesKeyHex)
	if err != nil {
		fmt.Printf("클라이언트 생성 실패: %v\n", err)
		return
	}

	fmt.Println("=== 전체 플로우 테스트 시작 ===\n")

	// 1. 회원가입
	fmt.Println("1. 회원가입 시도...")
	registReq := protocol.RegistReq{
		ID:      "testuser123",
		PW:      "testpassword",
		Name:    "테스트 사용자",
		Gender:  "1",
		Age:     "25",
		Birth:   "1998-01-01",
		Area:    "서울",
		Email:   "testuser123@example.com",
		SPIntro: "반가워요 큐피톡에서 만나요!",
	}

	err = client.RegistUser(registReq)
	if err != nil {
		fmt.Printf("   회원가입 실패: %v\n", err)
		// 이미 존재하는 사용자일 수 있으므로 계속 진행
	} else {
		fmt.Println("   회원가입 성공!")
	}

	// 2. 로그인
	fmt.Println("\n2. 로그인 시도...")
	loginReq := protocol.LoginReq{
		ID: "testuser123",
		PW: "testpassword",
	}

	loginResp, err := client.LoginUser(loginReq)
	if err != nil {
		fmt.Printf("   로그인 실패: %v\n", err)
		return
	}
	fmt.Println("   로그인 성공!")
	fmt.Printf("   - UID: %s\n", loginResp.UID)

	// 3. 홈데이터 조회
	fmt.Println("\n3. 홈데이터 조회 시도...")
	homeData, err := client.GetHomeData(loginResp.AccessToken)
	if err != nil {
		fmt.Printf("   홈데이터 조회 실패: %v\n", err)
		return
	}
	fmt.Println("   홈데이터 조회 성공!")
	fmt.Printf("   - 신규 알림: %d개\n", homeData.NotiNew)
	fmt.Printf("   - 쪽지 개수: %d개\n", homeData.MsgQty)
	fmt.Printf("   - 출석체크: %v\n", homeData.CheckIn)

	fmt.Println("\n=== 전체 플로우 테스트 완료 ===")
}
