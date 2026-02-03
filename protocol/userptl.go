package protocol

type RegistReq struct {
	ID        string `json:"id"` //key, sid
	PW        string `json:"pw"`
	Uid       uint64 `json:"uid"`
	Name      string `json:"name"`
	Gender    string `json:"gender"`
	Age       string `json:"age"`
	Birth     string `json:"birth"`
	Area      string `json:"area"`
	Email     string `json:"email"`
	Nick      string `json:"nick"`
	MainPic   string `json:"main_pic"`
	ThumbIcon string `json:"thmb_pic"`
	SPIntro   string `json:"spIntro"` //본인 인삿말
}

type LoginReq struct {
	ID string `json:"id"`
	PW string `json:"pw"`
}

type UserInfoResp struct {
	ID       string `json:"sid"`
	Uid      uint64 `json:"uid"`
	Did      string `json:"did"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Nick     string `json:"nick"`
	Gender   string `json:"gender"`
	Age      string `json:"age"`
	Birth    string `json:"birthday"`
	Area     string `json:"area"`
	Stat     string `json:"stat"`
	MainPic  string `json:"main_pic"`
	ThumbPic string `json:"thmb_pic"`
	SPIntro  string `json:"sp_intro"`
}

type MetaHeader struct {
	UID      uint64 `json:"uid"`
	SID      string `json:"sid"`
	DID      string `json:"did"`
	Nick     string `json:"nick"`
	Gender   string `json:"gender"`
	Age      string `json:"age"`
	Area     string `json:"area"`
	Email    string `json:"email"`
	MainPic  string `json:"main_pic"`
	ThumbPic string `json:"thmb_pic"`
	SPIntro  string `json:"sp_intro"`
}

// VideoConfig 비디오 설정
type VideoConfig struct {
	Enabled    bool `json:"enabled"`
	Width      int  `json:"width"`
	Height     int  `json:"height"`
	FrameRate  int  `json:"frameRate"`
	MaxBitrate int  `json:"maxBitrate"`
}

// AudioConfig 오디오 설정
type AudioConfig struct {
	Enabled          bool `json:"enabled"`
	EchoCancellation bool `json:"echoCancellation"`
	NoiseSuppression bool `json:"noiseSuppression"`
	AutoGainControl  bool `json:"autoGainControl"`
	MaxBitrate       int  `json:"maxBitrate"`
}

// ICEServer ICE 서버 설정 (STUN only)
type ICEServer struct {
	URLs []string `json:"urls"`
	Type string   `json:"type"` // stun only
}

// MediaConfig 미디어 설정
type MediaConfig struct {
	Video VideoConfig `json:"video"`
	Audio AudioConfig `json:"audio"`
}

// WebRTCConfig WebRTC 설정 정보 구조체 (STUN only)
type WebRTCConfig struct {
	ICEServers      []ICEServer `json:"iceServers"`
	SignalingServer string      `json:"signalingServer"`
	StunServers     []string    `json:"stunServers"`
	MediaSettings   MediaConfig `json:"mediaSettings"`
}

type LoginUserResp struct {
	Message      string       `json:"msg"`
	AccessToken  string       `json:"acTok"`
	RefreshToken string       `json:"refTok"`
	UID          string       `json:"uid"`
	MetaHeader   string       `json:"meta"`
	WebRTCConfig WebRTCConfig `json:"wrtc"`
}

type WTRoomUser struct {
	UID      uint64 `json:"uid"`
	SID      string `json:"sid"`
	DID      string `json:"did"`
	MainPic  string `json:"mainPic"`
	ThumbPic string `json:"thumbPic"`
	Intro    string `json:"intro"`
	Gender   string `json:"gender"`
	Nick     string `json:"nick"`
	Area     string `json:"area"`
	Age      string `json:"age"`
	NewStat  bool   `json:"newStat"`
}
