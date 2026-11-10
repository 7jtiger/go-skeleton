package protocol

import (
	"strings"
	"time"
)

type RegistReq struct {
	ID        string `json:"id"` //key, sid
	PW        string `json:"pw"`
	Uid       uint64 `json:"uid"`
	DID       string `json:"did"`
	DOS       string `json:"dos"`
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
	ID  string `json:"id"`
	PW  string `json:"pw"`
	DID string `json:"did"`
	DOS string `json:"dos"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refTok"`
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
	// UID      uint64 `json:"uid"`
	UID      string `json:"uid"`
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

type RefreshTokenResp struct {
	Message      string `json:"msg"`
	AccessToken  string `json:"acTok"`
	RefreshToken string `json:"refTok"`
	UID          string `json:"uid"`
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

/*
	 type HeaderUser struct {
		SID      string `json:"sid"`
		UID      string `json:"uid"`
		DID      string `json:"did"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Nick     string `json:"nick"`
		Gender   string `json:"gender"`
		Age      string `json:"age"`
		Birth    string `json:"birth"`
		Area     string `json:"area"`
		Stat     string `json:"stat"`
		MainPic  string `json:"main_pic"`
		ThumbPic string `json:"thmb_pic"`
		SPIntro  string `json:"sp_intro"`
	}
*/
func ToMetaHeader(meta string) *MetaHeader {
	parts := strings.Split(meta, " ")
	if len(parts) != 8 {
		return nil
	}

	/* 	meta := ptl.MetaHeader{
		SID:      parts[0],
		UID:      parts[1],
		DID:      parts[2],
		Nick:     parts[3],
		Gender:   parts[4],
		Age:      parts[5],
		Area:     parts[6],
		Email:    parts[7],
		MainPic:  parts[8],
		ThumbPic: parts[9],
		SPIntro:  parts[10],
	} */
	return nil
}

type FavoriteUserItem struct {
	UID      uint64
	SID      string
	Nick     string
	Gender   int
	Age      int
	Area     string
	MainPic  string
	ThumbPic string
	SPIntro  string
	FavAt    time.Time
}

type PrfInfo struct {
	Uid     uint64    `json:"uid" example:"8697414060736839837"`
	Nick    string    `json:"nick" example:"건강한 꼬마 오렌지"`
	Gender  string    `json:"gender" example:"1"`
	Birth   time.Time `json:"birth" example:"1990-01-01T00:00:00Z"`
	Age     string    `json:"age" example:"-2"`
	Area    string    `json:"area" example:"1"`
	Intro   string    `json:"intro" example:"반가워요 큐피톡에서 만나요!"`
	MainPic string    `json:"main_pic" example:""`
}

// PrfInfoData is GetPrfInfo response payload (data field).
type PrfInfoData struct {
	PrfInfo PrfInfo `json:"prf_Info"`
	FlwCnt  int64   `json:"flw_cnt" example:"0"`
}

// PrfInfoResp is full API envelope for GetPrfInfo Swagger (@Success).
type PrfInfoResp struct {
	Result       int         `json:"result" example:"0"`
	ResultString string      `json:"resultString" example:"Success"`
	Data         PrfInfoData `json:"data"`
}

// PrfUpdData is UpdatePrfInfo response payload (data field).
type PrfUpdData struct {
	Msg string `json:"msg" example:"ok"`
}

// PrfUpdResp is full API envelope for UpdatePrfInfo Swagger (@Success).
type PrfUpdResp struct {
	Result       int        `json:"result" example:"0"`
	ResultString string     `json:"resultString" example:"Success"`
	Data         PrfUpdData `json:"data"`
}

// PrfAssetsData is GetPrfAssets response payload (data field).
type PrfAssetsData struct {
	HoldPoint int64 `json:"hold_point" example:"0"`
	HoldCash  int64 `json:"hold_cash" example:"0"`
	MsgCount  int64 `json:"msg_count" example:"3"`
}

// PrfAssetsResp is full API envelope for GetPrfAssets Swagger (@Success).
type PrfAssetsResp struct {
	Result       int           `json:"result" example:"0"`
	ResultString string        `json:"resultString" example:"Success"`
	Data         PrfAssetsData `json:"data"`
}

type CreateStoryReq struct {
	Stat  int         `json:"stat" example:"0" enums:"0,1,2,3,4,5"` // 0=pub, 1=flw, 2=pay, 3=prv, 4=del, 5=rsv; 6=all(query only)
	Sbody string      `json:"sbody" example:"스토리 본문"`               // 최대 512자
	Price int         `json:"price" example:"10"`                   // 스토리 가격
	Media StoryStrImg `json:"media"`                                // story.str_img와 동일 슬롯 맵
}

// Story media type constants for story.str_img slot objects
const (
	StoryMediaTypeImg = "img"
	StoryMediaTypeVdo = "vdo"
)

// StoryStrImg is story.str_img JSON shape (max 5 slots; empty for text-only stories, keys "1".."5"):
//
//	{"1":{"type":"img","url":"..."},"2":{"type":"vdo","url":"...m3u8","thumb1":"..."},"3":{"type":"img","url":"..."}}
type StoryStrImg map[string]StoryMediaItem

// StoryMediaItem is one slot in story.str_img (img | vdo).
type StoryMediaItem struct {
	Type  string `json:"type" example:"img"` // "img" | "vdo"
	URL   string `json:"url" example:"https://imagedelivery.net/.../public"`
	Thumb string `json:"thumb1,omitempty" example:"https://imagedelivery.net/.../public"` // vdo only
}

// PrfPicInfo is stored in prf_info.sub_pic{1..5} as {"url":"...","stat":N} (slot = column number).
type PrfPicInfo struct {
	Url  string `json:"url" example:"https://imagedelivery.net/.../public"`
	Stat int    `json:"stat" example:"3"` // 1=요청중, 2=승인대기, 3=승인, 4=거절:삭제
}

// PrfPicMap is list/query response: {"1":{"url":"...","stat":3},"2":{"url":"...","stat":3}}
type PrfPicMap map[string]PrfPicInfo

// PrfPicListResp is full API envelope for GetPrfPicLists Swagger (@Success).
// data is PrfPicMap (non-empty slots only).
type PrfPicListResp struct {
	Result       int       `json:"result" example:"0"`
	ResultString string    `json:"resultString" example:"Success"`
	Data         PrfPicMap `json:"data"`
}

// PrfPicUploadData is UploadPrfPic response payload (data field).
// pics 배열 순서 = sub_pic1..N 슬롯.
type PrfPicUploadData struct {
	Msg  string       `json:"msg" example:"ok"`
	Pics []PrfPicInfo `json:"pics"`
}

// PrfPicUploadResp is full API envelope for UploadPrfPic Swagger (@Success).
type PrfPicUploadResp struct {
	Result       int              `json:"result" example:"0"`
	ResultString string           `json:"resultString" example:"Success"`
	Data         PrfPicUploadData `json:"data"`
}

// PrfStoryListsData is GetPrfStoryLists response payload (data field).
// Keys are story idx (JSON object keys). Up to four readable media stories;
// text-only and locked entries are skipped before filling the result.
type PrfStoryListsData struct {
	StrImgs   map[string]string `json:"str_imgs" example:"{\"12\":\"https://imagedelivery.net/.../public\"}"`
	StrThmbls map[string]string `json:"str_thmbls" example:"{\"15\":\"https://imagedelivery.net/.../public\"}"`
}

// PrfStoryListsResp is full API envelope for GetPrfStoryLists Swagger (@Success).
type PrfStoryListsResp struct {
	Result       int               `json:"result" example:"0"`
	ResultString string            `json:"resultString" example:"Success"`
	Data         PrfStoryListsData `json:"data"`
}

// PrfPicWaitingItem 관리자용 — sub_pic 슬롯 중 stat=1|2 대기 항목
type PrfPicWaitingItem struct {
	Uid  uint64 `json:"uid" example:"8697414060736839837"`
	Nick string `json:"nick" example:"건강한 꼬마 오렌지"`
	Slot int    `json:"slot" example:"1"` // sub_pic1~5 (컬럼 번호)
	Url  string `json:"url" example:"https://imagedelivery.net/.../public"`
	Stat int    `json:"stat" example:"1"` // 1=요청중, 2=승인대기
}

// PrfPicWaitingListData is GetPrfPicWaitingList response payload (data field).
type PrfPicWaitingListData struct {
	Msg        string              `json:"msg" example:"ok"`
	TotalCount int                 `json:"total_count" example:"2"`
	List       []PrfPicWaitingItem `json:"list"`
}

// PrfPicWaitingListResp is full API envelope for GetPrfPicWaitingList Swagger (@Success).
type PrfPicWaitingListResp struct {
	Result       int                   `json:"result" example:"0"`
	ResultString string                `json:"resultString" example:"Success"`
	Data         PrfPicWaitingListData `json:"data"`
}

// SetPrfPicStatReq 관리자용 — 특정 슬롯 프로필 사진 승인(stat=3)
type SetPrfPicStatReq struct {
	Uid  uint64 `json:"uid" example:"8697414060736839837"`
	Slot int    `json:"slot" example:"1"` // sub_pic1~5
}

// SetPrfPicStatData is SetPrfPicStat response payload (data field).
type SetPrfPicStatData struct {
	Msg string `json:"msg" example:"ok"`
	Res int64  `json:"res" example:"1"` // 0이면 대상 없음/이미 처리됨
}

// SetPrfPicStatResp is full API envelope for SetPrfPicStat Swagger (@Success).
type SetPrfPicStatResp struct {
	Result       int               `json:"result" example:"0"`
	ResultString string            `json:"resultString" example:"Success"`
	Data         SetPrfPicStatData `json:"data"`
}
