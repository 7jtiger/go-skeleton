package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"

	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	ptc "ms-gateway/protocol"
)

/*
CREATE TABLE `acc_info` (
  `idx` int NOT NULL,
  `uid` bigint unsigned NOT NULL,
  `aname` varchar(45) DEFAULT NULL COMMENT 'account name',
  `bname` varchar(45) DEFAULT NULL COMMENT 'bank name',
  `acc_num` varchar(45) DEFAULT NULL COMMENT 'account number',
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`),
  UNIQUE KEY `acc_num_UNIQUE` (`acc_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `pf_info` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` bigint unsigned NOT NULL,
  `sp_intro` varchar(128) DEFAULT NULL,
  `intro` varchar(256) DEFAULT NULL,
  `fw_cnt` int DEFAULT NULL,
  `fwing_cnt` int DEFAULT NULL,
  `fw_list` json DEFAULT NULL,
  `fwing_list` json DEFAULT NULL,
  `birthday` date DEFAULT NULL,
  `pf_pic` json DEFAULT NULL,
  `sub_pic` json DEFAULT NULL,
  `at_upd` datetime DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  UNIQUE KEY `uid_UNIQUE` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

CREATE TABLE `user_fav` (
  `uid` bigint unsigned NOT NULL COMMENT '즐겨찾기 한 사용자 uid',
  `tid` bigint unsigned NOT NULL COMMENT '즐겨찾기 대상 uid',
  `stat` tinyint NOT NULL DEFAULT 1 COMMENT '1=active, 0=removed',
  `at_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `at_upd` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`uid`, `tid`),
  KEY `idx_user_stat_upd` (`uid`, `stat`, `at_upd` DESC),
  KEY `idx_user_target_stat_owner` (`tid`, `stat`, `uid`),
  CONSTRAINT `chk_user_not_self` CHECK (`uid` <> `tid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `user_block` (
  `idx` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `uid` BIGINT UNSIGNED NOT NULL,
  `bid` BIGINT UNSIGNED NOT NULL,
  `reason` VARCHAR(128) DEFAULT NULL,
  `stat` TINYINT NOT NULL DEFAULT 1 COMMENT '1:block, 0:unblock',
  `at_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `at_update` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `uk_block_pair` (`uid`, `bid`),
  KEY `idx_blocker_stat` (`uid`, `stat`, `at_update`),
  KEY `idx_blocked_stat` (`bid`, `stat`, `at_update`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `user_info` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `sid` varchar(20) NOT NULL COMMENT 'login id',
  `uid` bigint unsigned NOT NULL COMMENT 'base id, bigint',
  `did` varchar(200) DEFAULT NULL COMMENT 'device id',
  `dos` varchar(10) DEFAULT NULL COMMENT 'device os',
  `email` varchar(50) DEFAULT NULL,
  `pnum` varchar(50) DEFAULT NULL COMMENT 'phone number',
  `pw_hash` varchar(100) DEFAULT NULL COMMENT 'password hash\n',
  `join_plf` tinyint(1) DEFAULT NULL COMMENT 'aos = 0, ios = 1, etc = 2',
  `name` varchar(30) DEFAULT NULL COMMENT 'real name',
  `nick` varchar(20) DEFAULT NULL COMMENT 'nick name',
  `gender` tinyint(1) DEFAULT NULL COMMENT 'woman = 0, main = 1',
  `age` int DEFAULT NULL,
  `birthday` date DEFAULT NULL COMMENT '1900-12-30',
  `area` varchar(10) DEFAULT NULL,
  `stat` tinyint(1) NOT NULL DEFAULT '0' COMMENT '0=default 1= 2= 3= 4=inactive',
  `st_title` varchar(50) DEFAULT NULL COMMENT 'user card title',
  `hold_point` double DEFAULT NULL COMMENT 'own amount',
  `hold_cash` double DEFAULT NULL COMMENT 'own cash won',
  `main_pic` varchar(145) DEFAULT NULL COMMENT 'main picture url',
  `sub_pic` json DEFAULT NULL COMMENT '1~5ea pic',
  `thmb_pic` varchar(145) DEFAULT NULL,
  `sp_intro` varchar(100) DEFAULT NULL COMMENT 'simple message',
  `at_join` datetime DEFAULT NULL COMMENT '1900-12-30 11:30',
  `at_upd` datetime DEFAULT NULL,
  `at_latest` datetime DEFAULT NULL,
  `set_alert` tinyint DEFAULT '0' COMMENT 'noti=1, call=2, msg=4, rerv=8\n0 = all alert\n1111 = no alert\n0001 = no noti\n0011 = no noti call\n0111 = no noti, call, msg\n',
  PRIMARY KEY (`idx`),
  UNIQUE KEY `sid_UNIQUE` (`sid`),
  UNIQUE KEY `email_UNIQUE` (`email`),
  UNIQUE KEY `pnum_UNIQUE` (`pnum`),
  KEY `idx_user_info_find_id` (`name`,`birthday`,`gender`,`sid`)
) ENGINE=InnoDB AUTO_INCREMENT=27 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
*/

// ScopeDB : 유저정보를 제공
type AccountDB struct {
	conndb *sql.DB
	cfg    *conf.Config
	root   *Repositories

	// cacheChainLock sync.RWMutex
	quit     chan struct{}
	quitWait sync.WaitGroup
}

// NewAccountDB : 객체 할당 및 반환
func NewAccountDB(cf *conf.Config, root *Repositories) (IRepository, error) {
	r := &AccountDB{
		cfg:  cf,
		root: root,
		quit: make(chan struct{}),
	}

	var err error
	c := r.cfg.DB["adb"]
	uri := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True",
		c["user"],
		c["pass"],
		c["host"],
		c["name"])

	r.conndb, err = sql.Open("mysql", uri)
	if err != nil {
		return nil, fmt.Errorf("database connection error: %v", err)
	}

	r.conndb.SetMaxOpenConns(300)
	r.conndb.SetMaxIdleConns(10)
	r.conndb.SetConnMaxLifetime(time.Minute * 3)

	go r.heartbeat()

	log.Info("load repository : account db")
	return r, nil
}

func (p *AccountDB) Start() error {
	return nil
}

func (p *AccountDB) Terminate() {
	close(p.quit)
	p.quitWait.Wait()

	log.Info("Terminated Database")
}

func (p *AccountDB) Close() error {
	return p.conndb.Close()
}

func (p *AccountDB) Ping() error {
	return p.conndb.Ping()
}

func (p *AccountDB) heartbeat() {
	p.quitWait.Add(1)
	defer p.quitWait.Done()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.quit:
			return
		case <-ticker.C:
			for _, e := range p.root.elems {
				if repo, ok := e.Interface().(IRepository); ok {
					if err := repo.Ping(); err != nil {
						log.Error("mysql ping fail.", err)
					}
				} else {
					log.Error("element does not implement IRepository")
				}
			}
		}
	}
}

/*
func (p *AccountDB) getAccountQuery(query string, params ...interface{}) (*sql.Rows, error) {
	rows, err := p.conndb.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("error querying partners: %v", err)
	}
	// defer rows.Close()

	return rows, nil
}
*/

/*
func (p *AccountDB) getCounterQuery(query string, params ...interface{}) (int, error) {
	var count int
	err := p.conndb.QueryRow(query, params...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error querying count of active partners: %v", err)
	}
	return count, nil
}
*/

/*
func (p *AccountDB) updateAccount(query string, params ...interface{}) error {
	_, err := p.conndb.Exec(query, params...)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}
	return nil
}
*/

func (p *AccountDB) IsExistID(id string) bool {
	query := "SELECT idx FROM user_info WHERE sid = ? LIMIT 1"
	row := p.conndb.QueryRow(query, id)
	var idx int

	err := row.Scan(&idx)
	if err == sql.ErrNoRows {
		return false // 존재하지 않음
	} else if err != nil {
		log.Error("IsExistID database error:", err)
		return false
	}

	return true // 존재함
}

func (p *AccountDB) IsExistEmail(email string) bool {
	query := "SELECT idx FROM user_info WHERE email = ? LIMIT 1"
	row := p.conndb.QueryRow(query, email)
	var idx int

	err := row.Scan(&idx)
	if err == sql.ErrNoRows {
		return false // 존재하지 않음
	} else if err != nil {
		log.Error("IsExistEmail database error:", err)
		return false
	}

	return true // 존재함
}

func (p *AccountDB) IsExistUid(uid uint64) bool {
	query := "SELECT COUNT(*) FROM user_info WHERE uid = ?"
	row := p.conndb.QueryRow(query, uid)
	var count int

	err := row.Scan(&count)
	if err == sql.ErrNoRows {
		return false // 존재하지 않음
	} else if err != nil {
		log.Error("IsExistUid database error:", err)
		return true
	}

	if count > 0 {
		return true // 존재함
	} else {
		return false // 존재하지 않음
	}

	// return count > 0 // 존재함
}

func (p *AccountDB) RegistUser(req ptc.RegistReq, encpw []byte) error {
	key := fmt.Sprintf("Cupitok-%d-Gateway", req.Uid)
	// encpw, err := utils.EncryptChaCha20(req.PW, key)
	// if err != nil {
	// 	return fmt.Errorf("error encrypting password: %v", err)
	// }

	// 이메일 암호화
	encEmail, err := utils.EncryptChaCha20(req.Email, key)
	if err != nil {
		return fmt.Errorf("error encrypting email: %v", err)
	}

	// 이름 암호화
	encName, err := utils.EncryptChaCha20(req.Name, key)
	if err != nil {
		return fmt.Errorf("error encrypting name: %v", err)
	}

	age := utils.CalcBirth2Age(req.Birth)
	area := ptc.GetAreaCode(req.Area)
	query := "INSERT INTO user_info (sid, uid, did, dos, email, pw_hash, name, nick, gender, age, birthday, area, main_pic, thmb_pic, sp_intro, at_join, at_upd) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	_, err = p.conndb.Exec(query, req.ID, req.Uid, req.DID, req.DOS, encEmail, encpw, encName, req.Nick, req.Gender, age, req.Birth, area, req.MainPic, req.ThumbIcon, req.SPIntro, time.Now(), time.Now())
	// _, err = p.conndb.Exec(query, req.ID, req.Uid, req.Email, encpw, encName, req.Nick, req.Gender, req.Age, req.Birth, req.Area, req.MainPic, req.ThumbIcon, req.SPIntro, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) updatedLastest(uid uint64) error {
	query := "UPDATE user_info SET at_upd = ? WHERE uid = ?"
	_, err := p.conndb.Exec(query, time.Now(), uid)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) updatedDid(uid uint64, did, dos string) error {
	query := "UPDATE user_info SET did = ?, dos = ?, at_upd = ? WHERE uid = ?"
	_, err := p.conndb.Exec(query, did, dos, time.Now(), uid)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) LoginUser(req ptc.LoginReq, pw []byte) (*ptc.UserInfoResp, error) {
	query := "SELECT uid, pw_hash, nick, did, dos FROM user_info WHERE sid = ? LIMIT 1"
	// log.Info("LoginUser: ", req)
	row := p.conndb.QueryRow(query, req.ID)
	var pwHash, nick string
	var uid uint64

	var didNull, dosNull sql.NullString
	err := row.Scan(&uid, &pwHash, &nick, &didNull, &dosNull)
	// log.Info("LoginUser: ", uid, pwHash, nick, didNull.String, dosNull.String)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	} else if err != nil {
		return nil, fmt.Errorf("error querying user: %v", err)
	}

	// fmt.Println("pwHash:", pwHash)
	// fmt.Println("pw:", string(pw))
	err = bcrypt.CompareHashAndPassword([]byte(pwHash), []byte(string(pw)))
	if err != nil {
		return nil, fmt.Errorf("password does not match: %v", err)
	}

	if didNull.Valid {
		// if didNull.String == req.DID {
		// }
		if didNull.String != req.DID {
			if err := p.updatedDid(uid, req.DID, req.DOS); err != nil {
				return nil, fmt.Errorf("error updating did: %v", err)
			}
		}
	} else {
		if err := p.updatedDid(uid, req.DID, req.DOS); err != nil {
			return nil, fmt.Errorf("error updating did: %v", err)
		}
	}

	user, err := p.GetUserInfo(req.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting user info: %v", err)
	}

	return &user, nil
}

func (p *AccountDB) LogoutUser(uid uint64) error {
	err := p.updatedLastest(uid)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) LeaveUser(id string) error {
	query := "UPDATE user_info SET stat = 4 WHERE sid = ?"
	_, err := p.conndb.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) FindID(name, birth, gender string) *[]string {
	query := "SELECT sid FROM user_info WHERE name = ? AND birthday = ? AND gender = ? LIMIT 10"
	rows, err := p.conndb.Query(query, name, birth, gender)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var sid string

	sumid := []string{}
	for rows.Next() {
		err := rows.Scan(&sid)
		if err != nil {
			return nil
		}
		sumid = append(sumid, sid)
	}

	return &sumid
}

func (p *AccountDB) CheckNameBirth(sid, name, birth, gender string) (int, string) {
	// query := "SELECT idx FROM user_info WHERE sid = ? AND name = ? AND birthday = ? AND gender = ? LIMIT 1"
	query := "SELECT uid, email FROM user_info WHERE sid = ? AND name = ? AND birthday = ? AND gender = ?"
	row := p.conndb.QueryRow(query, sid, name, birth, gender)
	var uid, email string

	err := row.Scan(&uid, &email)
	if err == sql.ErrNoRows {
		log.Error("CheckNameBirth : no rows")
		return 1, ""
	} else if err != nil {
		log.Error("CheckNameBirth database error:", err)
		return 0, ""
	}

	if email != "" {
		key := fmt.Sprintf("Cupitok-%s-Gateway", uid)
		decEmail, err := utils.DecryptChaCha20(email, key)
		if err != nil {
			log.Error("CheckNameBirth decrypt email error:", err)
			return 4, ""
		}
		return 2, decEmail
	} else {
		return 3, ""
	}
}

func (p *AccountDB) FindPW(id, email, birth string) bool {
	query := "SELECT idx FROM user_info WHERE sid = ? AND email = ? AND birthday = ?"
	row := p.conndb.QueryRow(query, id, email, birth)
	var idx int

	err := row.Scan(&idx)
	if err == sql.ErrNoRows {
		return false
	} else if err != nil {
		return false
	}

	return true
}

func (p *AccountDB) ChangePW(id, uid, email, pw, newpw string) error {
	key := fmt.Sprintf("Cupitok-%s-Gateway", uid)
	encPw, err := utils.EncryptChaCha20(pw, key)
	if err != nil {
		return fmt.Errorf("error encrypting password: %v", err)
	}

	encNewPw, err := utils.EncryptChaCha20(newpw, key)
	if err != nil {
		return fmt.Errorf("error encrypting password: %v", err)
	}

	query := "UPDATE user_info SET pw_hash = ? WHERE sid = ? AND uid = ? AND email = ? AND pw_hash = ?"
	_, err = p.conndb.Exec(query, encNewPw, id, uid, email, encPw)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) ModifyUserInfo(id, uid, email, cate, value string) error {
	var query string
	switch cate {
	case "nick":
		// query = fmt.Sprintf("UPDATE user_info SET nick = %s WHERE sid = %s AND uid = %s AND email = %s", value, id, uid, email)
		query = "UPDATE user_info SET nick = ? WHERE sid = ? AND uid = ? AND email = ?"
	case "area":
		// query = fmt.Sprintf("UPDATE user_info SET area = %s WHERE sid = %s AND uid = %s AND email = %s", value, id, uid, email)
		query = "UPDATE user_info SET area = ? WHERE sid = ? AND uid = ? AND email = ?"
	case "email":
		// key := fmt.Sprintf("Cupitok-%s-Gateway", uid)
		key := fmt.Sprintf("Cupitok-%v-Gateway", uid)
		encEmail, err := utils.EncryptChaCha20(value, key)
		if err != nil {
			return fmt.Errorf("error encrypting email: %v", err)
		}

		encNewEmail, err := utils.EncryptChaCha20(email, key)
		if err != nil {
			return fmt.Errorf("error encrypting email: %v", err)
		}

		query = fmt.Sprintf("UPDATE user_info SET email = %s WHERE sid = %s AND uid = %s AND email = %s", encNewEmail, id, uid, encEmail)
	case "main_pic":
		query = fmt.Sprintf("UPDATE user_info SET main_pic = %s WHERE uid = %s", value, uid)
	default:
		return fmt.Errorf("invalid category: %s", cate)
	}

	_, err := p.conndb.Exec(query)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) GetUserInfo(id string) (ptc.UserInfoResp, error) {
	// query := "SELECT sid, email, name, nick, gender, age, birthday, area, at_join FROM user_info WHERE sid = ?"
	query := "SELECT sid, uid, did, email, name, nick, gender, birthday, area, stat, main_pic, thmb_pic, sp_intro FROM user_info WHERE sid = ? LIMIT 1"
	row := p.conndb.QueryRow(query, id)
	var user ptc.UserInfoResp
	var encEmail, encName string
	var did sql.NullString
	var area int

	// 먼저 데이터베이스에서 암호화된 값들을 스캔 (NULL 가능한 컬럼은 sql.NullString 사용)
	err := row.Scan(&user.ID, &user.Uid, &did, &encEmail, &encName, &user.Nick, &user.Gender, &user.Birth, &area, &user.Stat, &user.MainPic, &user.ThumbPic, &user.SPIntro)
	if err != nil {
		return ptc.UserInfoResp{}, err
	}

	// NULL 처리: did가 NULL이면 빈 문자열로 설정
	if did.Valid {
		user.Did = did.String
	} else {
		user.Did = ""
	}

	user.Area = ptc.GetAreaName(area)
	user.Age = strconv.Itoa(utils.CalcBirth2Age(user.Birth))
	// uid를 사용하여 키 생성 (CheckNameBirth와 동일한 방식)
	key := fmt.Sprintf("Cupitok-%d-Gateway", user.Uid)

	// 이메일 복호화
	if encEmail != "" {
		decEmail, err := utils.DecryptChaCha20(encEmail, key)
		if err != nil {
			return ptc.UserInfoResp{}, fmt.Errorf("error decrypting email: %v", err)
		}
		user.Email = decEmail
	}

	// 이름 복호화
	if encName != "" {
		decName, err := utils.DecryptChaCha20(encName, key)
		if err != nil {
			return ptc.UserInfoResp{}, fmt.Errorf("error decrypting name: %v", err)
		}
		user.Name = decName
	}

	return user, nil
}

// GetUserInfoByUID uid 기반 사용자 조회
func (p *AccountDB) GetUserInfoByUID(uid uint64) (ptc.UserInfoResp, error) {
	query := "SELECT sid, uid, did, email, name, nick, gender, birthday, area, stat, main_pic, thmb_pic, sp_intro FROM user_info WHERE uid = ? LIMIT 1"
	row := p.conndb.QueryRow(query, uid)
	var user ptc.UserInfoResp
	var encEmail, encName string
	var did sql.NullString
	var area int

	err := row.Scan(&user.ID, &user.Uid, &did, &encEmail, &encName, &user.Nick, &user.Gender, &user.Birth, &area, &user.Stat, &user.MainPic, &user.ThumbPic, &user.SPIntro)
	if err != nil {
		return ptc.UserInfoResp{}, err
	}

	if did.Valid {
		user.Did = did.String
	} else {
		user.Did = ""
	}

	user.Area = ptc.GetAreaName(area)
	user.Age = strconv.Itoa(utils.CalcBirth2Age(user.Birth))

	key := fmt.Sprintf("Cupitok-%d-Gateway", user.Uid)
	if encEmail != "" {
		decEmail, decErr := utils.DecryptChaCha20(encEmail, key)
		if decErr != nil {
			return ptc.UserInfoResp{}, fmt.Errorf("error decrypting email: %v", decErr)
		}
		user.Email = decEmail
	}

	if encName != "" {
		decName, decErr := utils.DecryptChaCha20(encName, key)
		if decErr != nil {
			return ptc.UserInfoResp{}, fmt.Errorf("error decrypting name: %v", decErr)
		}
		user.Name = decName
	}

	return user, nil
}

func (p *AccountDB) GetSetAlert(uid uint64) (int, error) {
	query := "SELECT set_alert FROM user_info WHERE uid = ?"
	row := p.conndb.QueryRow(query, uid)
	var setAlert int
	err := row.Scan(&setAlert)
	if err != nil {
		return -1, err
	}

	return setAlert, nil
}

func (p *AccountDB) SetAlert(uid uint64, value int) error {
	query := "UPDATE user_info SET set_alert = ? WHERE uid = ?"
	_, err := p.conndb.Exec(query, value, uid)
	if err != nil {
		return err
	}

	return nil
}

func (p *AccountDB) DeleteUser(id string) error {
	query := "UPDATE user_info SET stat = 4 WHERE sid = ?"
	_, err := p.conndb.Exec(query, id)
	if err != nil {
		return err
	}

	return nil
}

// ==== block =================================================================================

func (p *AccountDB) SetBlock(uid, bid uint64, reason string) (int64, error) {
	query := `
		INSERT INTO user_block (uid, bid, reason, stat, at_create, at_update)
		VALUES (?, ?, ?, 1, NOW(), NOW())
		ON DUPLICATE KEY UPDATE stat = 1, reason = VALUES(reason), at_update = NOW()
	`
	result, err := p.conndb.Exec(query, uid, bid, reason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *AccountDB) SetUnblock(uid, bid uint64) (int64, error) {
	query := `UPDATE user_block SET stat = 0, at_update = NOW() WHERE uid = ? AND bid = ?`
	result, err := p.conndb.Exec(query, uid, bid)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *AccountDB) GetBlockList(uid uint64, page, limit int) (*[]ptc.BlockUserItem, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT ub.bid, COALESCE(u.nick, ''), COALESCE(u.thmb_pic, ''), COALESCE(ub.reason, ''), ub.at_update
		FROM user_block ub
		LEFT JOIN user_info u ON u.uid = ub.bid
		WHERE ub.uid = ? AND ub.stat = 1
		ORDER BY ub.at_update DESC
		LIMIT ? OFFSET ?
	`
	rows, err := p.conndb.Query(query, uid, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []ptc.BlockUserItem{}
	for rows.Next() {
		var item ptc.BlockUserItem
		if err = rows.Scan(&item.Uid, &item.Nick, &item.ThumbPic, &item.Reason, &item.AtUpdate); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return &list, nil
}

func (p *AccountDB) IsBlockedPair(uidA, uidB uint64) (bool, error) {
	query := `
		SELECT 1
		FROM user_block
		WHERE stat = 1
			AND ((uid = ? AND bid = ?) OR (uid = ? AND bid = ?))
		LIMIT 1
	`
	var one int
	err := p.conndb.QueryRow(query, uidA, uidB, uidB, uidA).Scan(&one)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ==== block =================================================================================

// ==== favorite =================================================================================
func (p *AccountDB) SetFavoriteUser(uid, targetUid uint64, enable bool) error {
	if uid == 0 || targetUid == 0 {
		return fmt.Errorf("invalid uid")
	}
	if uid == targetUid {
		return fmt.Errorf("cannot favorite self")
	}
	if enable {
		query := `
			INSERT INTO user_fav (uid, tid, stat, at_create, at_upd)
			VALUES (?, ?, 1, NOW(), NOW())
			ON DUPLICATE KEY UPDATE
				stat = 1,
				at_upd = NOW()
		`
		_, err := p.conndb.Exec(query, uid, targetUid)
		if err != nil {
			return fmt.Errorf("set favorite failed: %v", err)
		}
		return nil
	}
	query := `
		UPDATE user_fav
		SET stat = 0, at_upd = NOW()
		WHERE uid = ? AND tid = ? AND stat = 1
	`
	_, err := p.conndb.Exec(query, uid, targetUid)
	if err != nil {
		return fmt.Errorf("unset favorite failed: %v", err)
	}
	return nil
}

func (p *AccountDB) GetFavoriteUsers(uid uint64, limit, offset int) ([]ptc.FavoriteUserItem, error) {
	if uid == 0 {
		return nil, fmt.Errorf("invalid owner uid")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			u.uid, u.sid, u.nick, u.gender, u.birthday, u.area, u.main_pic, u.thmb_pic, u.sp_intro,
			uf.at_upd
		FROM user_fav uf
		INNER JOIN user_info u ON u.uid = uf.tid
		WHERE uf.uid = ?
		  AND uf.stat = 1
		  AND u.stat <> 4
		ORDER BY uf.at_upd DESC
		LIMIT ? OFFSET ?
	`
	rows, err := p.conndb.Query(query, uid, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get favorites query failed: %v", err)
	}
	defer rows.Close()

	result := make([]ptc.FavoriteUserItem, 0, limit)
	for rows.Next() {
		var item ptc.FavoriteUserItem
		var area int
		var birthday string
		if err := rows.Scan(
			&item.UID, &item.SID, &item.Nick, &item.Gender, &birthday, &area,
			&item.MainPic, &item.ThumbPic, &item.SPIntro, &item.FavAt,
		); err != nil {
			return nil, fmt.Errorf("get favorites scan failed: %v", err)
		}
		item.Area = ptc.GetAreaName(area)
		item.Age = utils.CalcBirth2Age(birthday)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get favorites rows error: %v", err)
	}
	return result, nil
}

func (p *AccountDB) CountFavoriteUsers(uid uint64) (int, error) {
	query := `SELECT COUNT(*) FROM user_fav WHERE uid = ? AND stat = 1`
	var cnt int
	if err := p.conndb.QueryRow(query, uid).Scan(&cnt); err != nil {
		return 0, fmt.Errorf("count favorites failed: %v", err)
	}
	return cnt, nil
}

//==== favorite =================================================================================

//==== story =================================================================================
/*
CREATE TABLE `story` (
  `idx` int unsigned NOT NULL AUTO_INCREMENT,
  `uid` bigint NOT NULL,
  `nick` varchar(20) NOT NULL,
  `body` varchar(256) NOT NULL,
  `pic1` varchar(145) DEFAULT NULL,
  `pic2` varchar(145) DEFAULT NULL,
  `pic3` varchar(145) DEFAULT NULL,
  `pic4` varchar(145) DEFAULT NULL,
  `pic5` varchar(145) DEFAULT NULL,
  `stat` tinyint DEFAULT NULL COMMENT 'stat = 0 공개, stat = 1 비공객, stat = 2 유료공개, stat=4 삭제',
  `at_create` datetime DEFAULT CURRENT_TIMESTAMP,
  `at_update` datetime DEFAULT CURRENT_TIMESTAMP,
  `qt_good` int DEFAULT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

===============================================================================
CREATE TABLE `story_comment` (
  `idx` int unsigned NOT NULL AUTO_INCREMENT,
  `str_idx` int unsigned NOT NULL,
  `uid` bigint NOT NULL,
  `nick` varchar(20) DEFAULT NULL,
  `body` varchar(256) DEFAULT NULL,
  `stat` tinyint DEFAULT '0' COMMENT 'stat = 0 정상, stat = 1 삭제, stat = 2 신고',
  `at_create` datetime DEFAULT CURRENT_TIMESTAMP,
  `at_update` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`idx`,`str_idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci

*/

func (p *AccountDB) GetStory7List(uid uint64) (*[]ptc.Pre7Story, error) {
	query := "SELECT idx, nick, pic1 FROM story WHERE stat = 0 ORDER BY at_create DESC LIMIT 7"
	rows, err := p.conndb.Query(query, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pre7Story []ptc.Pre7Story
	for rows.Next() {
		var s ptc.Pre7Story
		err := rows.Scan(&s.Idx, &s.Nick, &s.Pic1)
		if err != nil {
			return nil, err
		}
		pre7Story = append(pre7Story, s)
	}

	return &pre7Story, nil
}

/*
//TODO:
3) API별 최소 수정 전략
A. CheckEmail

 기존: WHERE email = ?

 변경: 입력 email 정규화 → email_bidx 생성 → WHERE email_bidx = ?
B. RegistUserInfo

 등록 시 email 암호화 저장은 유지

 동시에 email_bidx, name_bidx도 같이 INSERT

 중복 체크는 IsExistEmail(bidx 기반)으로 수행
C. FindID

 기존: WHERE name = ? AND birthday = ? AND gender = ?

 변경: name→name_bidx로 변환 후
WHERE name_bidx = ? AND birthday = ? AND gender = ?

 필요하면 sid까지 포함해 오탐 줄임
D. FindPW (현재 CheckNameBirth 경유)

 기존 CheckNameBirth: sid + name + birth + gender 평문 비교

 변경: sid + name_bidx + birth + gender 조건으로 조회

 email은 기존처럼 조회 후 복호화 반환
E. ChangePW

 기존: WHERE sid=? AND uid=? AND email=? AND pw_hash=?

 변경(최소):
email_bidx를 조건으로 사용
또는 더 단순히 uid 중심으로 인증 체인 정리

 권장: WHERE uid=? AND pw_hash=? + 별도 소유자 검증(JWT uid)
F. ModifyUserInfo

 cate=email일 때:
새 email 암호화 저장 + 새 email_bidx 함께 업데이트

 cate=nick/area일 때의 WHERE ... email = ? 제거
uid 또는 sid+uid로 변경 권장

 SQL 바인딩(?) 일관화 유지 (현재 일부는 이미 개선됨)
*/
