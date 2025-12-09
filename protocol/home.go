package protocol

type HomeDataReq struct {
	ID string `json:"id"`
}

type VideoChat struct {
	ID      string `json:"id"`
	VCID    string `json:"vcid"`
	MainPic string `json:"mainPic"`
	Intro   string `json:"intro"`
	Gender  string `json:"gender"`
	Nick    string `json:"nick"`
	Area    string `json:"area"`
	Age     string `json:"age"`
}

type Story struct {
	ID       string `json:"id"`
	STID     string `json:"stid"`
	StoryPic string `json:"storyPic"`
}

type Pre7Story struct {
	Idx  int    `json:"idx"`
	Nick string `json:"nick"`
	Pic1 string `json:"pic1"`
}

type VoiceChat struct {
	ID      string `json:"id"`
	VOID    string `json:"void"`
	MainPic string `json:"mainPic"`
	Intro   string `json:"intro"`
	Gender  string `json:"gender"`
	Nick    string `json:"nick"`
	Area    string `json:"area"`
	Age     string `json:"age"`
}

type HomeDataResp struct {
	Noti      bool        `json:"noti"`
	Msg       int         `json:"msg"`
	CheckIn   bool        `json:"checkIn"`
	VideoChat []VideoChat `json:"videoChat"`
	VoiceChat []VoiceChat `json:"voiceChat"`
	Story     []Story     `json:"story"`
	Terms     string      `json:"terms"`
	Policy    string      `json:"policy"`
}
