package protocol

import (
	"encoding/json"
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

/*
CREATE TABLE `story` (

	`idx` int unsigned NOT NULL AUTO_INCREMENT,
	`uid` bigint NOT NULL,
	`nick` varchar(20) NOT NULL,
	`body` varchar(512) DEFAULT NULL,
	`stat` tinyint DEFAULT '1' COMMENT 'stat=0 : del, stat=1 : pub, stat=2 : private, stat=3 : limit, stat=4 : ',
	`qt_good` int DEFAULT NULL,
	`qt_checked` int DEFAULT NULL COMMENT '조회수',
	`str_img` json NOT NULL,
	`at_update` datetime DEFAULT NULL,
	`at_create` datetime DEFAULT NULL,
	`str_imgbak` json DEFAULT NULL,
	PRIMARY KEY (`idx`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
*/
type StoryImage struct {
	Idx       int             `json:"idx"`
	Uid       uint64          `json:"uid"`
	Nick      string          `json:"nick"`
	Body      string          `json:"body"`
	Stat      int             `json:"stat"`
	QtGood    int             `json:"qt_good"`
	QtChecked int             `json:"qt_checked"`
	StrImg    json.RawMessage `json:"str_img"`
	AtUpdate  time.Time       `json:"at_update"`
	AtCreate  time.Time       `json:"at_create"`
	StrImgBak json.RawMessage `json:"str_imgbak"`
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
