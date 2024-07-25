package protocol

type ResultCode int

const (
	Success            ResultCode = 0
	Failed                        = 1   // 요청이 실패하였습니다.
	IpInvalid                     = 2   // 요청한 아이피가 정상적이지 않습니다.
	UserIDNotFound                = 13  // 유저 아이디가 존재하지 않습니다
	AccessTokenInvalid            = 101 // 접속 토큰이 유효하지 않음
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
