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
	Body      string          `json:"body"`
	MType     int             `json:"type"`
	Stat      int             `json:"stat"`
	QtGood    int             `json:"qt_good"`
	QtChecked int             `json:"qt_checked"`
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

type StoryListResp struct {
	Idx      int             `json:"idx"`
	Nick     string          `json:"nick"`
	StrImg   json.RawMessage `json:"str_img"`
	AtCreate time.Time       `json:"at_create"`
}

type StoryDetailResp struct {
	Nick        string          `json:"nick"`
	Body        string          `json:"body"`
	StrImg      json.RawMessage `json:"str_img"`
	AtCreate    time.Time       `json:"at_create"`
	CommentList []StrComment    `json:"comment_list"`
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
	Idx      int       `json:"idx"`
	StrIdx   int       `json:"str_idx"`
	Wuid     uint64    `json:"wuid"`
	Nick     string    `json:"nick"`
	ThumbUrl string    `json:"thumb_url"`
	WGender  string    `json:"wgender"`
	WAge     string    `json:"wage"`
	WArea    string    `json:"warea"`
	Body     string    `json:"body"`
	Stat     int       `json:"stat"`
	AtCreate time.Time `json:"at_create"`
	AtUpdate time.Time `json:"at_update"`
}

type StrCmtCreateReq struct {
	StrIdx string `json:"str_idx"`
	Wuid   string `json:"wuid"`
	Nick   string `json:"nick"`
	Stat   string `json:"stat"`
	Body   string `json:"body"`
}

type StrCmtUpdStatReq struct {
	Idx  string `json:"idx"`
	Stat string `json:"stat"`
}

type StrCmtUpdBodyReq struct {
	Idx  string `json:"idx"`
	Body string `json:"body"`
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

// stat : 0=PUB(public 전체공개), 1=FLW(follower only), 2=PAY(paid only), 3=PRV(Private 비공개), 4=DEL(Deleted) 5=RSV(reserved)
func GetStatCode(stat string) int {
	switch strings.ToLower(stat) {
	case "pub":
		return 0
	case "flw":
		return 1
	case "pay":
		return 2
	case "prv":
		return 3
	case "del":
		return 4
	case "rsv":
		return 5
	}
	return 0
}

// order : 0=OLD(oldest), 1=NEW(latest), 2=CMT(comMost), 3=GOD(goodMost), 4=VIEW(viewMost), 5=RSV(reserved)
func GetOrderQuery(order string) string {
	switch strings.ToLower(order) {
	case "new":
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

// type : 0=IMG(only img), 1 = VDO(only video), 2 = ALL(img + video), 3=RSV(reserved)
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
	case "daejeon", "sejong", "chungnam", "대전/세종/충남", "대전", "세종", "충남":
		return 5
	case "chungbuk", "cheonju", "chungju", "충북/청주/충주", "충북", "청주", "충주":
		return 6
	case "daegu", "gyeongbuk", "대구/경북", "대구", "경북":
		return 7
	case "gyeongnam", "ulsan", "경남/울산", "경남", "울산":
		return 8
	case "gwangju", "jeonnam", "광주/전남", "광주", "전남":
		return 9
	case "jeonbuk", "jeonju", "전북/전주", "전북", "전주":
		return 10
	case "gangwon", "chuncheon", "강원/춘천", "강원", "춘천":
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
