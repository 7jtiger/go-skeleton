package protocol

import (
	"encoding/json"
	"strings"
	"time"
)

/*
CREATE TABLE `noti_his` (
	`idx` int NOT NULL AUTO_INCREMENT,
	`uid` bigint unsigned NOT NULL,
	`nt_type` tinyint(4) unsigned zerofill NOT NULL,
	`nt_title` varchar(20) NOT NULL,
	`nt_msg` varchar(50) NOT NULL,
	`at_noti` datetime NOT NULL,
	`stat` tinyint(1) unsigned zerofill NOT NULL,
	`frm_uid` bigint unsigned NOT NULL,
	`frm_url` varchar(100) NOT NULL,
	`frm_nick` varchar(15) NOT NULL,
	PRIMARY KEY (`idx`)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='noti list'
*/

type Noti struct {
	Idx      int    `json:"idx"`
	Ntype    int    `json:"nt_type"`
	Title    string `json:"nt_title"`
	Msg      string `json:"nt_msg"`
	AtMsg    string `json:"at_noti"`
	Stat     int    `json:"stat"`
	FromUid  uint64 `json:"frm_uid"`
	FromUrl  string `json:"frm_url"`
	FromNick string `json:"frm_nick"`
}

/*
CREATE TABLE `anuc_his` (
`idx` int NOT NULL AUTO_INCREMENT,
`an_title` varchar(45) NOT NULL,
`an_body` tinytext,
`an_url` varchar(150) DEFAULT NULL,
`at_msg` datetime NOT NULL,
PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
*/
type Announcement struct {
	Idx   int    `json:"idx"`
	Title string `json:"an_title"`
	Body  string `json:"an_body"`
	Url   string `json:"an_url"`
	AtMsg string `json:"at_msg"`
	Stat  int    `json:"stat"`
}

type StoryImage struct {
	Idx       int             `json:"idx"`
	Uid       uint64          `json:"uid"`
	Nick      string          `json:"nick"`
	Birth     string          `json:"birth"`
	Area      int             `json:"area"`
	Gender    int             `json:"gender"`
	MainPic   string          `json:"main_pic"`
	Body      string          `json:"body"`
	MType     int             `json:"type"`
	Stat      int             `json:"stat"`
	QtGood    int             `json:"qt_good"`
	QtChecked int             `json:"qt_checked"`
	Price     int             `json:"price"`
	StrImg    json.RawMessage `json:"str_img"`
	AtUpdate  time.Time       `json:"at_update"`
	AtCreate  time.Time       `json:"at_create"`
	StrImgBak json.RawMessage `json:"str_imgbak"`
}

type StorySearchReq struct {
	Area  string `json:"area"`
	Stat  string `json:"stat"`
	MType string `json:"type"`
	Order string `json:"order"`
	Gen   string `json:"gender"`
	Page  string `json:"page"`
	Limit string `json:"limit"`
}

/* type StoryListResp struct {
	Idx      int         `json:"idx"`
	Uid      uint64      `json:"uid"`
	Nick     string      `json:"nick"`
	StrImg   StoryStrImg `json:"str_img"` // {"1":{"type":"img","url":"..."},"2":{"type":"vdo","url":"...","thumb":"..."}}
	AtCreate time.Time   `json:"at_create"`
}
*/

type StoryListResp struct {
	Idx      int       `json:"idx"`
	Uid      uint64    `json:"uid"`
	Nick     string    `json:"nick"`
	Stat     int       `json:"stat"`
	CntCount int       `json:"cnt_count"`
	CntUrl   string    `json:"cnt_url"`
	CntType  string    `json:"cnt_type"`
	CntThumb string    `json:"cnt_thumb"`
	AtCreate time.Time `json:"at_create"`
}

type StoryDetailResp struct {
	Idx          int          `json:"idx"`
	Uid          uint64       `json:"uid"`
	Nick         string       `json:"nick"`
	MainPic      string       `json:"main_pic"`
	Body         string       `json:"body"`
	Stat         int          `json:"stat"`
	QtGood       int          `json:"qt_good"`
	QtView       int          `json:"qt_view"`
	ContentTotal int          `json:"content_total"`
	Price        int          `json:"price"`
	IsPaid       int          `json:"is_paid"`     // 0=not paid, 1=paid
	IsFollowed   int          `json:"is_followed"` // 0=not followed, 1=followed
	StrImg       StoryStrImg  `json:"str_img"`     // story.str_img 슬롯 맵 (img|vdo)
	AtCreate     time.Time    `json:"at_create"`
	CmtCount     int          `json:"cmt_count"`
	CommentList  []StrComment `json:"comment_list"`
}

type UpdateStrStatReq struct {
	Idx  string `json:"idx"`
	Stat string `json:"stat"`
}

type UpdateStrBodyReq struct {
	Idx  string `json:"idx"`
	Body string `json:"body"`
}

type DeleteStrPicReq struct {
	Idx string `json:"idx"`
	Pic string `json:"pic_idx"`
}

/*
CREATE TABLE `str_cmt` (

	`idx` int unsigned NOT NULL AUTO_INCREMENT,
	`str_idx` int unsigned NOT NULL,
	`wuid` bigint NOT NULL,
	`nick` varchar(20) DEFAULT NULL,
	`thumb_url` varchar(256) DEFAULT NULL,
	`wgender` varchar(2) DEFAULT NULL,
	`wage` varchar(45) DEFAULT NULL,
	`warea` varchar(45) DEFAULT NULL,
	`body` varchar(256) DEFAULT NULL,
	`stat` tinyint DEFAULT NULL COMMENT 'stat=0:default, stat=1:private, stat=2::rerv, stat=3:rerv,stat=4:del',
	`at_create` datetime DEFAULT CURRENT_TIMESTAMP,
	`at_update` datetime DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (`idx`),
	UNIQUE KEY `idx_UNIQUE` (`idx`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
*/
type StrComment struct {
	Idx      int       `json:"idx" example:"2"`
	StrIdx   int       `json:"str_idx" example:"3"`
	Wuid     uint64    `json:"wuid" example:"77645423236541"`
	Nick     string    `json:"nick" example:"test"`
	ThumbUrl string    `json:"thumb_url" example:"https://imagedelivery.net/.../public"`
	WGender  string    `json:"wgender" example:"1"`
	WAge     string    `json:"wage" example:"30"`
	WArea    string    `json:"warea" example:"1"`
	Body     string    `json:"body" example:"111111 test comment body"`
	Stat     int       `json:"stat" example:"1"` // 0,1 visible in list
	AtCreate time.Time `json:"at_create" example:"2026-02-02T13:50:06Z"`
	AtUpdate time.Time `json:"at_update" example:"2026-02-02T13:50:06Z"`
}

type StrCmtCreateReq struct {
	StrIdx string `json:"str_idx" example:"3"`
	Wuid   string `json:"wuid" example:"77645423236541"`
	Nick   string `json:"nick" example:"test"`
	Stat   string `json:"stat" example:"1"`                        // 0=default, 1=private, 2=reserved, 3=reserved, 4=deleted
	Body   string `json:"body" example:"111111 test comment body"` // max 256
}

// StrCmtCreateResp is CreateStrComment response (SimpleRespOK — no result envelope).
type StrCmtCreateResp struct {
	Msg    string `json:"msg" example:"Successfully created str comment"`
	LastID int64  `json:"lastID" example:"1"`
}

// StrCmtDetailData is GetStrCmtDetail response payload (data field).
type StrCmtDetailData struct {
	Comment    []StrComment `json:"comment"`
	TotalCount int          `json:"total_count" example:"52"`
}

// StrCmtDetailResp is full API envelope for GetStrCmtDetail Swagger (@Success).
type StrCmtDetailResp struct {
	Result       int              `json:"result" example:"0"`
	ResultString string           `json:"resultString" example:"Success"`
	Data         StrCmtDetailData `json:"data"`
}

type StrCmtUpdStatReq struct {
	Idx  string `json:"idx" example:"3"`
	Stat string `json:"stat" example:"2"` // 0=default, 1=private, 2=reserved, 3=reserved, 4=deleted
}

// StrCmtUpdStatResp is UpdateStrStatComment response (SimpleRespOK — no result envelope).
type StrCmtUpdStatResp struct {
	Msg      string `json:"msg" example:"Successfully updated str stat comment"`
	Affected int64  `json:"affected" example:"1"`
}

type StrCmtUpdBodyReq struct {
	Idx  string `json:"idx" example:"3"`
	Body string `json:"body" example:"updated comment body content"`
}

// StrCmtUpdBodyResp is UpdateStrBodyComment response (SimpleRespOK — no result envelope).
type StrCmtUpdBodyResp struct {
	Msg      string `json:"msg" example:"Successfully updated str body comment"`
	Affected int64  `json:"affected" example:"1"`
}

type StoryLikeToggleReq struct {
	StoryIdx string `json:"story_idx"`
	Uid      string `json:"uid"`
}

type FollowReq struct {
	FollowerUid string `json:"follower_uid"`
	FolloweeUid string `json:"followee_uid"`
}

type FollowUserItem struct {
	Uid      uint64    `json:"uid"`
	Nick     string    `json:"nick"`
	ThumbPic string    `json:"thumb_pic"`
	AtUpdate time.Time `json:"at_update"`
}

type BlockReq struct {
	Uid    string `json:"uid"`
	Bid    string `json:"bid"`
	Reason string `json:"reason"`
}

type BlockUserItem struct {
	Uid      uint64    `json:"uid"`
	Nick     string    `json:"nick"`
	ThumbPic string    `json:"thumb_pic"`
	Reason   string    `json:"reason"`
	AtUpdate time.Time `json:"at_update"`
}

// stat : 0 = pub, 1 = private(follower only), 2 = limit(paid only), 3 = all(pub/prv/limit), 4 : del
func GetStatCode(stat string) int {
	switch strings.ToLower(stat) {
	case "pub":
		return 0
	case "prv":
		return 1
	case "lmt":
		return 2
	case "all":
		return 3
	}
	return 0
}

// order : 0=old(oldest), 1=new(latest)/all, 2=cmt(comMost), 3=god(goodMost), 4=view(viewMost), 5=rsv(reserved)
func GetOrderQuery(order string) string {
	switch strings.ToLower(order) {
	case "new", "all":
		return "at_create DESC"
	case "old":
		return "at_create ASC"
	case "cmt":
		return "qt_checked DESC"
	case "god":
		return "qt_good DESC"
	case "view":
		return "qt_checked DESC"
	case "rsv":
		return "at_create DESC"
	}
	return "at_create DESC"
}

// type : 0=img, 1 = vdo, 2 = all, 3=rsv(reserved)
func GetTypeCode(mtype string) int {
	switch strings.ToLower(mtype) {
	case "img":
		return 0
	case "vdo":
		return 1
	case "all":
		return 2
	case "rsv":
		return 3
	}
	return 0
}

func GetAreaCode(area string) int {
	//전국, 서울, 경기, 인천, 부산, 대전/세종/충남, 충북/청주/충주, 대구/경북,
	// 경남/울산, 광주/전남, 전북/전주, 강원/춘천, 제주
	// all = 0, seoul = 1, gyeonggi = 2, incheon = 3, busan = 4,
	// daejeon/sejong/chungnam = 5, chungbuk/cheonju = 6,
	// daegu/gyeongbuk = 7, gyeongnam/ulsan = 8, gwangju/jeonnam = 9,
	// jeonbuk/jeonju = 10, gangwon/chuncheon = 11, jeju = 12
	switch area {
	case "all", "전국":
		return 0
	case "seoul", "서울":
		return 1
	case "gyeonggi", "경기":
		return 2
	case "incheon", "인천":
		return 3
	case "busan", "부산":
		return 4
	case "daejeon", "sejong", "chungnam", "daejeon/sejong/chungnam", "대전/세종/충남", "대전", "세종", "충남":
		return 5
	case "chungbuk", "cheonju", "chungju", "chungbuk/cheonju/chungju", "충북/청주/충주", "충북", "청주", "충주":
		return 6
	case "daegu", "gyeongbuk", "daegu/gyeongbuk", "대구/경북", "대구", "경북":
		return 7
	case "gyeongnam", "ulsan", "gyeongnam/ulsan", "경남/울산", "경남", "울산":
		return 8
	case "gwangju", "jeonnam", "gwangju/jeonnam", "광주/전남", "광주", "전남":
		return 9
	case "jeonbuk", "jeonju", "jeonbuk/jeonju", "전북/전주", "전북", "전주":
		return 10
	case "gangwon", "chuncheon", "gangwon/chuncheon", "강원/춘천", "강원", "춘천":
		return 11
	case "jeju", "제주":
		return 12
	}

	return 0
}

func GetAreaName(code int) string {
	//전국, 서울, 경기, 인천, 부산, 대전/세종/충남, 충북/청주/충주, 대구/경북,
	// 경남/울산, 광주/전남, 전북/전주, 강원/춘천, 제주
	// all = 0, seoul = 1, gyeonggi = 2, incheon = 3, busan = 4,
	// daejeon/sejong/chungnam = 5, chungbuk/cheonju/chungju = 6,
	// daegu/gyeongbuk = 7, gyeongnam/ulsan = 8, gwangju/jeonnam = 9,
	// jeonbuk/jeonju = 10, gangwon/chuncheon = 11, jeju = 12
	switch code {
	case 0:
		return "all"
	case 1:
		return "seoul"
	case 2:
		return "gyeonggi"
	case 3:
		return "incheon"
	case 4:
		return "busan"
	case 5:
		return "daejeon/sejong/chungnam"
	case 6:
		return "chungbuk/cheonju/chungju"
	case 7:
		return "daegu/gyeongbuk"
	case 8:
		return "gyeongnam/ulsan"
	case 9:
		return "gwangju/jeonnam"
	case 10:
		return "jeonbuk/jeonju"
	case 11:
		return "gangwon/chuncheon"
	case 12:
		return "jeju"
	}
	return "all"
}
