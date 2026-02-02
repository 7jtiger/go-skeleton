package models

import (
	"fmt"
	"ms-gateway/common/utils"
	ptl "ms-gateway/protocol"
	"testing"
)

// TestAccountDBLoginUser AccountDB의 LoginUser 함수 단위 테스트
func TestAccountDBLoginUser(t *testing.T) {
	// 실제 데이터베이스 연결이 필요한 테스트
	// 실제 환경에서는 테스트 데이터베이스를 사용해야 합니다
	t.Skip("Skipping database integration test - requires test database setup")

	// 테스트용 데이터베이스 연결 (실제 구현시 설정 필요)
	// db := setupTestDatabase()
	// defer db.Close()

	// accountDB := &AccountDB{conndb: db}

	// 테스트 케이스
	tests := []struct {
		name          string
		req           ptl.LoginReq
		expectError   bool
		expectedError string
	}{
		{
			name:        "Valid Login",
			req:         ptl.LoginReq{ID: "testuser", PW: "correctpassword"},
			expectError: false,
		},
		{
			name:          "Invalid Password",
			req:           ptl.LoginReq{ID: "testuser", PW: "wrongpassword"},
			expectError:   true,
			expectedError: "password does not match",
		},
		{
			name:          "Non-existent User",
			req:           ptl.LoginReq{ID: "nonexistentuser", PW: "password"},
			expectError:   true,
			expectedError: "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// result, err := accountDB.LoginUser(tt.req)

			// if tt.expectError {
			// 	if err == nil {
			// 		t.Errorf("Expected error but got none")
			// 	} else if tt.expectedError != "" && err.Error() != tt.expectedError {
			// 		t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
			// 	}
			// } else {
			// 	if err != nil {
			// 		t.Errorf("Expected no error but got: %v", err)
			// 	}
			// 	if result == nil {
			// 		t.Errorf("Expected user info but got nil")
			// 	}
			// }
		})
	}
}

// MockAccountDB 목 데이터베이스 구조체
type MockAccountDB struct {
	users map[string]MockUser
}

type MockUser struct {
	UID      uint64
	SID      string
	PWHash   string
	Nick     string
	Name     string
	Email    string
	Gender   string
	Age      string
	Birth    string
	Area     string
	MainPic  string
	ThumbPic string
	SPIntro  string
}

// NewMockAccountDB 목 데이터베이스 생성
func NewMockAccountDB() *MockAccountDB {
	// 테스트용 암호화된 패스워드 생성
	key := "Cupitok-testuid-Gateway"
	encryptedPW, _ := utils.EncryptChaCha20("1234", key)

	return &MockAccountDB{
		users: map[string]MockUser{
			"testuser": {
				UID:      1234567890,
				SID:      "testuser",
				PWHash:   encryptedPW,
				Nick:     "테스트유저",
				Name:     "테스트",
				Email:    "test@example.com",
				Gender:   "M",
				Age:      "25",
				Birth:    "19990101",
				Area:     "서울",
				MainPic:  "main.jpg",
				ThumbPic: "thumb.jpg",
				SPIntro:  "안녕하세요",
			},
		},
	}
}

// LoginUser 목 구현
func (m *MockAccountDB) LoginUser(req ptl.LoginReq) (*ptl.UserInfoResp, error) {
	user, exists := m.users[req.ID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// 패스워드 복호화 및 검증
	key := fmt.Sprintf("Cupitok-%d-Gateway", user.UID)
	decPW, err := utils.DecryptChaCha20(user.PWHash, key)
	if err != nil {
		return nil, fmt.Errorf("error decrypting password: %v", err)
	}

	if decPW != req.PW {
		return nil, fmt.Errorf("password does not match")
	}

	// 성공시 사용자 정보 반환
	return &ptl.UserInfoResp{
		ID:       user.SID,
		Uid:      user.UID,
		Email:    user.Email,
		Name:     user.Name,
		Nick:     user.Nick,
		Gender:   user.Gender,
		Age:      user.Age,
		Birth:    user.Birth,
		Area:     user.Area,
		MainPic:  user.MainPic,
		ThumbPic: user.ThumbPic,
		SPIntro:  user.SPIntro,
	}, nil
}

// TestMockAccountDBLoginUser 목을 사용한 LoginUser 테스트
func TestMockAccountDBLoginUser(t *testing.T) {
	mockDB := NewMockAccountDB()

	tests := []struct {
		name          string
		req           ptl.LoginReq
		expectError   bool
		expectedError string
	}{
		{
			name:        "Valid Login",
			req:         ptl.LoginReq{ID: "testuser", PW: "1234"},
			expectError: false,
		},
		{
			name:          "Invalid Password",
			req:           ptl.LoginReq{ID: "testuser", PW: "wrongpassword"},
			expectError:   true,
			expectedError: "password does not match",
		},
		{
			name:          "Non-existent User",
			req:           ptl.LoginReq{ID: "nonexistentuser", PW: "1234"},
			expectError:   true,
			expectedError: "user not found",
		},
		{
			name:          "Empty ID",
			req:           ptl.LoginReq{ID: "", PW: "1234"},
			expectError:   true,
			expectedError: "user not found",
		},
		{
			name:          "Empty Password",
			req:           ptl.LoginReq{ID: "testuser", PW: ""},
			expectError:   true,
			expectedError: "password does not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mockDB.LoginUser(tt.req)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.expectedError != "" && err.Error() != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
				}
				if result != nil {
					t.Errorf("Expected nil result on error, but got: %+v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result == nil {
					t.Errorf("Expected user info but got nil")
				} else {
					// 반환된 사용자 정보 검증
					if result.ID != "testuser" {
						t.Errorf("Expected ID 'testuser', got '%s'", result.ID)
					}
					if result.Uid != 1234567890 {
						t.Errorf("Expected UID '1231241231', got '%d'", result.Uid)
					}
					if result.Nick != "테스트유저" {
						t.Errorf("Expected Nick '테스트유저', got '%s'", result.Nick)
					}
				}
			}
		})
	}
}

// TestPasswordEncryption 패스워드 암호화/복호화 테스트
func TestPasswordEncryption(t *testing.T) {
	uid := "testuid"
	password := "testpassword123!@#"
	key := fmt.Sprintf("Cupitok-%s-Gateway", uid)

	// 암호화
	encryptedPW, err := utils.EncryptChaCha20(password, key)
	if err != nil {
		t.Errorf("Failed to encrypt password: %v", err)
		return
	}

	// 복호화
	decryptedPW, err := utils.DecryptChaCha20(encryptedPW, key)
	if err != nil {
		t.Errorf("Failed to decrypt password: %v", err)
		return
	}

	// 검증
	if decryptedPW != password {
		t.Errorf("Password mismatch: expected '%s', got '%s'", password, decryptedPW)
	}

	t.Logf("Password encryption/decryption test passed")
}

// TestPasswordValidation 다양한 패스워드 유효성 검사
func TestPasswordValidation(t *testing.T) {
	mockDB := NewMockAccountDB()

	passwords := []struct {
		name     string
		password string
		valid    bool
	}{
		{"Correct Password", "1234", true},
		{"Wrong Password", "wrong", false},
		{"Empty Password", "", false},
		{"Numeric Password", "123456", false},
		{"Special Characters", "!@#$%", false},
		{"Long Password", "verylongpasswordthatexceedsusuallengthlimits", false},
	}

	for _, p := range passwords {
		t.Run(p.name, func(t *testing.T) {
			req := ptl.LoginReq{ID: "testuser", PW: p.password}
			result, err := mockDB.LoginUser(req)

			if p.valid {
				if err != nil {
					t.Errorf("Expected valid password but got error: %v", err)
				}
				if result == nil {
					t.Errorf("Expected user info but got nil")
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for invalid password but got none")
				}
				if result != nil {
					t.Errorf("Expected nil result for invalid password")
				}
			}
		})
	}
}

// BenchmarkLoginUser LoginUser 함수 성능 벤치마크
func BenchmarkLoginUser(b *testing.B) {
	mockDB := NewMockAccountDB()
	req := ptl.LoginReq{ID: "testuser", PW: "1234"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mockDB.LoginUser(req)
		if err != nil {
			b.Errorf("Benchmark failed: %v", err)
		}
	}
}

// BenchmarkPasswordDecryption 패스워드 복호화 성능 벤치마크
func BenchmarkPasswordDecryption(b *testing.B) {
	uid := "testuid"
	password := "testpassword123"
	key := fmt.Sprintf("Cupitok-%s-Gateway", uid)
	encryptedPW, _ := utils.EncryptChaCha20(password, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := utils.DecryptChaCha20(encryptedPW, key)
		if err != nil {
			b.Errorf("Benchmark failed: %v", err)
		}
	}

}
