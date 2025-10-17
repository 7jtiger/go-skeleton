package protocol

type ResultCode int

const (
	Success             ResultCode = 0
	Failed                         = 1   // 요청이 실패하였습니다.
	IpInvalid                      = 2   // 요청한 아이피가 정상적이지 않습니다.
	UserIDNotFound                 = 13  // 유저 아이디가 존재하지 않습니다.
	IDDuplicate                    = 14  // 유저 아이디가 중복되었습니다.
	EmailDuplicate                 = 15  // 유저 이메일이 중복되었습니다.
	AccessTokenInvalid             = 101 // 접속 토큰이 유효하지 않음
	InvalidParam                   = 102 // 요청 파라미터가 유효하지 않음
	UserAlreadyExists              = 103 // 이미 존재하는 유저
	UserRegistFailed               = 104 // 회원가입 실패
	UserRegistSuccess              = 105 // 회원가입 성공
	JsonParseFailed                = 106 // JSON 파싱 실패
	UserLoginFailed                = 107 // 로그인 실패
	UserLogoutFailed               = 108 // 로그아웃 실패
	UserChangePWFailed             = 109 // 비밀번호 변경 실패
	UserChangePWSuccess            = 110 // 비밀번호 변경 성공
	UserFindIDFailed               = 111 // 아이디 찾기 실패
	UserFindPWFailed               = 112 // 비밀번호 찾기 실패
	UserLeaveFailed                = 113 // 회원탈퇴 실패
	UserInfoFailed                 = 114 // 유저 정보 조회 실패
	UserDeleteFailed               = 115 // 유저 삭제 실패
	UserAuthOTPFailed              = 116 // 이메일 인증 실패
	// codeNN            = 102 //
	// codeNN            = 103 //
	// codeNN            = 104 //
	// codeNN            = 105 //
	// codeNN            = 106 //
	// codeNN            = 107 //
	// codeNN            = 300 //
	// codeNN            = 301 //
	// codeNN            = 302 //
	// codeNN            = 303 //
	// codeNN            = 304 //
	// codeNN            = 305 //
	// codeNN            = 306 //
	// codeNN            = 307 //
	// codeNN            = 308 //
	// codeNN            = 600 //
	// codeNN            = 601 //
	// codeNN            = 602 //
	// codeNN            = 700 //
	// codeNN            = 701 //
	// codeNN            = 702 //
)

func (r ResultCode) toString() string {
	switch r {
	case Success:
		return "Success"
	case Failed:
		return "Failed"
	case IpInvalid:
		return "IpInvalid"
	}
	return ""
}

type EmailData struct {
	OTPCode string
}

var EmailOTPCode = `<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OTP 확인 이메일</title>
</head>
<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 20px; text-align: center;">
    <div style="max-width: 600px; margin: auto; background-color: #fff; padding: 20px; border-radius: 8px;">
        <h2>고객 확인 OTP 코드</h2>
        <p>아래 6자리 OTP 코드를 사용해 계정을 확인하세요:</p>
        <div style="font-size: 24px; font-weight: bold; color: #2c3e50; letter-spacing: 5px; margin: 20px 0; background-color: #f8f9fa; padding: 10px; border-radius: 5px; display: inline-block;">{{.OTPCode}}</div>
        <p>이 코드는 10분간 유효합니다. 타인에 유출을 주의하세요</p>
        <p style="font-size: 12px; color: #777; margin-top: 20px;">© 2025 CupiTok</p>
    </div>
</body>
</html>`
