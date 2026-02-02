package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"

	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	ptl "ms-gateway/protocol"
)

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

func (p *AccountDB) RegistUser(req ptl.RegistReq, encpw []byte) error {
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
	query := "INSERT INTO user_info (sid, uid, email, pw_hash, name, nick, gender, age, birthday, area, main_pic, thmb_pic, sp_intro, at_join, at_upd) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	_, err = p.conndb.Exec(query, req.ID, req.Uid, encEmail, encpw, encName, req.Nick, req.Gender, age, req.Birth, req.Area, req.MainPic, req.ThumbIcon, req.SPIntro, time.Now(), time.Now())
	// _, err = p.conndb.Exec(query, req.ID, req.Uid, req.Email, encpw, encName, req.Nick, req.Gender, req.Age, req.Birth, req.Area, req.MainPic, req.ThumbIcon, req.SPIntro, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) updatedLastest(sid string) error {
	query := "UPDATE user_info SET at_upd = ? WHERE sid = ?"
	_, err := p.conndb.Exec(query, time.Now(), sid)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) LoginUser(req ptl.LoginReq, pw []byte) (*ptl.UserInfoResp, error) {
	query := "SELECT uid, pw_hash, nick FROM user_info WHERE sid = ? LIMIT 1"
	row := p.conndb.QueryRow(query, req.ID)
	var pwHash, nick string
	var uid uint64

	err := row.Scan(&uid, &pwHash, &nick)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	} else if err != nil {
		return nil, fmt.Errorf("error querying user: %v", err)
	}

	fmt.Println("pwHash:", pwHash)
	fmt.Println("pw:", string(pw))
	err = bcrypt.CompareHashAndPassword([]byte(pwHash), []byte(string(pw)))
	if err != nil {
		return nil, fmt.Errorf("password does not match: %v", err)
	}

	// if !bytes.Equal([]byte(pwHash), hsedPw) {
	// 	return nil, fmt.Errorf("password does not match")
	// }

	err = p.updatedLastest(req.ID)
	if err != nil {
		return nil, fmt.Errorf("error updating lastest: %v", err)
	}

	user, err := p.GetUserInfo(req.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting user info: %v", err)
	}

	return &user, nil
}

func (p *AccountDB) LogoutUser(id string) error {
	err := p.updatedLastest(id)
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
		query = fmt.Sprintf("UPDATE user_info SET nick = %s WHERE sid = %s AND uid = %s AND email = %s", value, id, uid, email)
	case "area":
		query = fmt.Sprintf("UPDATE user_info SET area = %s WHERE sid = %s AND uid = %s AND email = %s", value, id, uid, email)
	case "email":
		key := fmt.Sprintf("Cupitok-%s-Gateway", uid)
		encEmail, err := utils.EncryptChaCha20(value, key)
		if err != nil {
			return fmt.Errorf("error encrypting email: %v", err)
		}

		encNewEmail, err := utils.EncryptChaCha20(email, key)
		if err != nil {
			return fmt.Errorf("error encrypting email: %v", err)
		}

		query = fmt.Sprintf("UPDATE user_info SET email = %s WHERE sid = %s AND uid = %s AND email = %s", encNewEmail, id, uid, encEmail)
	default:
		return fmt.Errorf("invalid category: %s", cate)
	}

	_, err := p.conndb.Exec(query)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	return nil
}

func (p *AccountDB) GetUserInfo(id string) (ptl.UserInfoResp, error) {
	// query := "SELECT sid, email, name, nick, gender, age, birthday, area, at_join FROM user_info WHERE sid = ?"
	query := "SELECT sid, uid, did, email, name, nick, gender, age, birthday, area, stat, main_pic, thmb_pic, sp_intro FROM user_info WHERE sid = ? LIMIT 1"
	row := p.conndb.QueryRow(query, id)
	var user ptl.UserInfoResp
	var encEmail, encName string
	var did sql.NullString

	// 먼저 데이터베이스에서 암호화된 값들을 스캔 (NULL 가능한 컬럼은 sql.NullString 사용)
	err := row.Scan(&user.ID, &user.Uid, &did, &encEmail, &encName, &user.Nick, &user.Gender, &user.Age, &user.Birth, &user.Area, &user.Stat, &user.MainPic, &user.ThumbPic, &user.SPIntro)
	if err != nil {
		return ptl.UserInfoResp{}, err
	}

	// NULL 처리: did가 NULL이면 빈 문자열로 설정
	if did.Valid {
		user.Did = did.String
	} else {
		user.Did = ""
	}

	// uid를 사용하여 키 생성 (CheckNameBirth와 동일한 방식)
	key := fmt.Sprintf("Cupitok-%d-Gateway", user.Uid)

	// 이메일 복호화
	if encEmail != "" {
		decEmail, err := utils.DecryptChaCha20(encEmail, key)
		if err != nil {
			return ptl.UserInfoResp{}, fmt.Errorf("error decrypting email: %v", err)
		}
		user.Email = decEmail
	}

	// 이름 복호화
	if encName != "" {
		decName, err := utils.DecryptChaCha20(encName, key)
		if err != nil {
			return ptl.UserInfoResp{}, fmt.Errorf("error decrypting name: %v", err)
		}
		user.Name = decName
	}

	return user, nil
}

func (p *AccountDB) DeleteUser(id string) error {
	query := "UPDATE user_info SET stat = 4 WHERE sid = ?"
	_, err := p.conndb.Exec(query, id)
	if err != nil {
		return err
	}

	return nil
}

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

func (p *AccountDB) GetStory7List(uid uint64) (*[]ptl.Pre7Story, error) {
	query := "SELECT idx, nick, pic1 FROM story WHERE stat = 0 ORDER BY at_create DESC LIMIT 7"
	rows, err := p.conndb.Query(query, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pre7Story []ptl.Pre7Story
	for rows.Next() {
		var s ptl.Pre7Story
		err := rows.Scan(&s.Idx, &s.Nick, &s.Pic1)
		if err != nil {
			return nil, err
		}
		pre7Story = append(pre7Story, s)
	}

	return &pre7Story, nil
}
