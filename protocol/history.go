package protocol

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
