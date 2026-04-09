package models

import (
	// "bytes"
	"context"
	"encoding/json"
	"hash/fnv"
	"math/rand"
	"strconv"
	"testing"

	// "crypto/rand"
	// "crypto/sha256"
	"database/sql"
	// "encoding/base64"
	// "encoding/hex"
	// "encoding/json"
	// "errors"
	"fmt"
	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	ptl "ms-gateway/protocol"

	// "ms-gateway/common/utils"
	// "math"
	// "math/big"
	// "strings"
	// "testing"
	"time"

	// "github.com/go-redis/redis/v8"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	// "github.com/hashicorp/vault/shamir"
	// "go.mongodb.org/mongo-driver/bson/primitive"
	// "github.com/google/uuid"
)

func ConnectDB(dname string) *sql.DB {
	// uri := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", "admin", "guruAdm0621#!", "133.186.150.59:3306", "bankerdb")
	// uri := "admin:guruAdm0621#!@tcp(133.186.150.59:3306)/bankerdb?parseTime=true"
	var uri string
	if dname == "adb" {
		uri = "root:qwer@tcp(192.168.48.196:3306)/adb?parseTime=true"
	} else if dname == "hdb" {
		uri = "root:qwer@tcp(192.168.48.196:3306)/hdb?parseTime=true"
	} else if dname == "idb" {
		uri = "root:qwer@tcp(192.168.48.196:3306)/idb?parseTime=true"
	} else if dname == "sdb" {
		uri = "root:qwer@tcp(192.168.48.196:3306)/sdb?parseTime=true"
	} else {
		return nil
	}

	conndb, err := sql.Open("mysql", uri)
	if err != nil {
		return nil
	}

	conndb.SetMaxIdleConns(30)
	conndb.SetMaxOpenConns(300)
	conndb.SetConnMaxLifetime(time.Minute * 3)

	return conndb
}

func ConnRedis() *redis.Client {
	redisOption := &redis.Options{
		Addr: "192.168.48.196:6379",
		// Username:  "root",
		Password: "qwer",
		DB:       0,
		// TLSConfig: &tls.Config{InsecureSkipVerify: false},
	}

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	client := redis.NewClient(redisOption)
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Error("Redis 연결 실패:", err)
		return nil
	}

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		fmt.Println("Redis : ", err)
	}

	return client
}

func Test_CheckID(t *testing.T) {
	adb := ConnectDB("adb")
	if adb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer adb.Close()

	query := "SELECT idx FROM user_info WHERE sid = ? LIMIT 1"
	row := adb.QueryRow(query, "test")
	var idx int
	//"sql: no rows in result set"

	err := row.Scan(&idx)
	if err == sql.ErrNoRows {
		fmt.Println("존재하지 않는 아이디")
	} else if err != nil {
		fmt.Println("오류 발생")
	}

	fmt.Println(row)
}

func Test_GetUserInfo(t *testing.T) {
	adb := ConnectDB("adb")
	if adb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer adb.Close()

	query := "SELECT sid, uid, did, email, name, nick, gender, age, birthday, area, stat, main_pic, thmb_pic, sp_intro FROM user_info WHERE sid = ? LIMIT 1"
	row := adb.QueryRow(query, "test23")
	var uinfo ptl.UserInfoResp
	// var idx int
	//"sql: no rows in result set"

	// NULL 값을 처리하기 위해 sql.NullString 사용
	var mainPic, thumbPic, spIntro sql.NullString

	err := row.Scan(&uinfo.ID, &uinfo.Uid, &uinfo.Did, &uinfo.Email, &uinfo.Name, &uinfo.Nick, &uinfo.Gender, &uinfo.Age, &uinfo.Birth, &uinfo.Area, &uinfo.Stat, &mainPic, &thumbPic, &spIntro)
	if err == sql.ErrNoRows {
		fmt.Println("존재하지 않는 아이디")
	} else if err != nil {
		fmt.Println("오류 발생:", err)
	}

	// NULL 값 처리 후 구조체에 할당
	if mainPic.Valid {
		uinfo.MainPic = mainPic.String
	} else {
		uinfo.MainPic = ""
	}

	if thumbPic.Valid {
		uinfo.ThumbPic = thumbPic.String
	} else {
		uinfo.ThumbPic = ""
	}

	if spIntro.Valid {
		uinfo.SPIntro = spIntro.String
	} else {
		uinfo.SPIntro = ""
	}

	fmt.Println(uinfo)
}

func Test_SetUserInfo(t *testing.T) {
	adb := ConnectDB("adb")
	if adb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer adb.Close()

	// insert 10 users with random data
	query := "INSERT INTO user_info (sid, uid, did, email, name, nick, gender, age, birthday, area, stat, main_pic, thmb_pic, sp_intro) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	// 한국 지방도시 리스트
	cities := []string{"부산", "대구", "인천", "광주", "대전", "울산", "수원", "창원", "성남", "고양", "용인", "부천", "안산", "안양", "남양주", "천안", "전주", "김해", "포항", "제주"}

	// 랜덤 이름 리스트
	firstNames := []string{"김", "이", "박", "최", "정", "강", "조", "윤", "장", "임"}
	lastNames := []string{"민수", "영희", "철수", "지은", "현우", "수진", "동혁", "미영", "준호", "은지", "상훈", "예은", "태현", "소영", "진우"}

	// 랜덤 문장 리스트
	sentences := []string{
		"안녕하세요! 반갑습니다.",
		"새로운 인연을 기대합니다.",
		"좋은 사람들과 만나고 싶어요.",
		"취미는 영화감상과 독서입니다.",
		"여행을 좋아하는 사람입니다.",
		"음악과 카페를 좋아해요.",
		"운동과 건강관리에 관심이 많습니다.",
		"맛있는 음식 탐방을 즐깁니다.",
		"새로운 도전을 좋아합니다.",
		"긍정적인 에너지를 가진 사람입니다.",
	}

	for i := 0; i < 10; i++ {
		// sid: 일반적인 한단어 형태
		sids := []string{"user11", "user21", "user31", "admin", "test", "sample", "demo", "guest", "member", "client"}
		sid := sids[i]

		// uid, did: 난수 형태 (8자리)
		// uid := rand.Int63n(9223372036854775807)
		uid, err := utils.GenRandomUID()
		if err != nil {
			fmt.Printf("사용자 %d 난수 생성 오류: %v\n", i+1, err)
		}
		key := fmt.Sprintf("Cupitok-%v-Gateway", uid)
		did := fmt.Sprintf("%08d", rand.Intn(100000000))

		// email: 이메일 형태
		puremail := fmt.Sprintf("user1%d@example.com", i+1)
		email, err := utils.EncryptChaCha20(puremail, key)
		if err != nil {
			fmt.Printf("사용자 %d 이메일 암호화 오류: %v\n", i+1, err)
		}

		// name, nick: 랜덤 이름
		// purname := firstNames[rand.Intn(len(firstNames))] + lastNames[rand.Intn(len(lastNames))]
		name := firstNames[rand.Intn(len(firstNames))] + lastNames[rand.Intn(len(lastNames))]
		// name, err := utils.EncryptChaCha20(purname, key)

		nick := utils.GenDefNick()

		// gender: 0, 1 랜덤값
		gender := rand.Intn(2)

		// age: 20~40 랜덤값
		age := 20 + rand.Intn(21)

		// birthday: 나이에 맞는 랜덤일
		currentYear := 2024
		birthYear := currentYear - age
		birthMonth := 1 + rand.Intn(12)
		birthDay := 1 + rand.Intn(28) // 안전하게 28일까지
		birthday := fmt.Sprintf("%04d-%02d-%02d", birthYear, birthMonth, birthDay)

		// area: 한국 지방도시 랜덤
		area := cities[rand.Intn(len(cities))]

		// stat: 0
		stat := 0

		// main_pic, thmb_pic, sp_intro: 랜덤 문장
		mainPic := sentences[rand.Intn(len(sentences))]
		thmbPic := "https://i.ibb.co/99MhfMXt/icon-male-03.webp"
		spIntro := sentences[rand.Intn(len(sentences))]

		_, err = adb.Exec(query, sid, uid, did, email, name, nick, gender, age, birthday, area, stat, mainPic, thmbPic, spIntro)
		if err != nil {
			fmt.Printf("사용자 %d 입력 오류: %v\n", i+1, err)
		} else {
			fmt.Printf("사용자 %d (%s) 입력 성공\n", i+1, sid)
		}
	}

	fmt.Println("성공")
}

func Test_GenNick(t *testing.T) {
	nick := utils.GenDefNick()
	fmt.Println(nick)
}

func Test_HSetAccess(t *testing.T) {
	rdb := ConnRedis()
	if rdb == nil {
		fmt.Println("Redis 연결 실패")
		return
	}
	defer rdb.Close()

	err := rdb.HSet(context.Background(), "access", []string{"token", "userID"}).Err()
	if err != nil {
		fmt.Println("HSet 실패:", err)
	}

	res := rdb.HGet(context.Background(), "access", "token")
	// res = "userID"
	fmt.Println(res.Val())
	// rdb.HSetAccess([]string{"token", "userID"})
}

func Test_SetJWTToken(t *testing.T) {
	rdb := ConnRedis()
	if rdb == nil {
		fmt.Println("Redis 연결 실패")
		return
	}
	defer rdb.Close()

	options := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		// ExpirationVal:  3600, //sec //1hour
		ExpirationVal: 86400, //sec //1day
	}

	if err := rdb.HSetEXWithArgs(context.Background(), "AUTH:ACCESS", options, "token", "userID").Err(); err != nil {
		fmt.Println("HSetEXWithArgs 실패:", err)
	}

	// err := rdb.HSet(context.Background(), "access", []string{"token", "userID"}).Err()
	// if err != nil {
	// 	fmt.Println("HSet 실패:", err)
	// }

	res := rdb.HGet(context.Background(), "AUTH:ACCESS", "token")
	// res = "userID"
	fmt.Println(res.Val())
	// rdb.HSetAccess([]string{"token", "userID"})
}

func Test_HDel(t *testing.T) {
	rdb := ConnRedis()
	if rdb == nil {
		fmt.Println("Redis 연결 실패")
		return
	}
	defer rdb.Close()

	// sample_bicycle:* 패턴에 매칭되는 모든 키를 찾아서 일괄 삭제
	keys, err := rdb.Keys(context.Background(), "sample_*").Result()
	if err != nil {
		fmt.Println("Keys 조회 실패:", err)
		return
	}

	if len(keys) > 0 {
		err = rdb.Del(context.Background(), keys...).Err()
		if err != nil {
			fmt.Println("일괄 삭제 실패:", err)
			return
		}
		fmt.Printf("%d개의 sample_bicycle 키가 삭제되었습니다\n", len(keys))
	} else {
		fmt.Println("삭제할 sample_bicycle 키가 없습니다")
	}

}

func Test_chaCha20(t *testing.T) {
	email := "test@example.com"
	key := "Cupitok-testuid-Gateway"
	encEmail, err := utils.EncryptChaCha20(email, key)
	if err != nil {
		fmt.Println("EncryptChaCha20 실패:", err)
	}
	fmt.Println(encEmail)
	decEmail, err := utils.DecryptChaCha20(encEmail, key)
	if err != nil {
		fmt.Println("DecryptChaCha20 실패:", err)
	}
	fmt.Println(decEmail)
}

func Test_2Hash(t *testing.T) {
	// 64비트 정수 해시값 반환
	input := "input@example.com"

	h := fnv.New64a()
	h.Write([]byte(input))
	hash := h.Sum64()

	fmt.Println(hash)
}

func Test_HSetChatRoomList(t *testing.T) {
	rdb := ConnRedis()
	if rdb == nil {
		fmt.Println("Redis 연결 실패")
		return
	}
	defer rdb.Close()

	room := ChatRoomListItem{
		RoomID:       "1",
		RoomName:     "test",
		LastActivity: time.Now(),
		UnreadCount:  0,
		LastMessage:  "test",
		Priority:     0,
	}

	roomJSON, err := json.Marshal(room)
	if err != nil {
		fmt.Println("JSON 직렬화 실패:", err)
		return
	}

	fmt.Println(roomJSON)
}

// ===============Story db test===============================================
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
func Test_SetStory(t *testing.T) {
	sdb := ConnectDB("sdb")
	if sdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer sdb.Close()

	sinfo := map[string]string{
		"uid":   "2345091823",
		"nick":  "testnick",
		"stat":  "1",
		"idx_0": "bc1.jpg",
		"idx_1": "bc2.jpg",
		"idx_2": "bc3.jpg",
		"sbody": "테스트 스토리 내용입니다",
	}

	// Test data: map of filenames to Cloudflare image URLs
	imageURLs := map[string]string{
		"bc1.jpg": "https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public",
		"bc2.jpg": "https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/919d8a0d-37a2-4d61-099b-a879322fc200/public",
		"bc3.jpg": "https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e811ff8b-6646-4fee-b5f6-9aba8e393f00/public",
	}

	// Sample story data
	uid := uint64(2345091823)
	// sbody := sinfo["sbody"]

	simg := &ptl.StoryImage{}
	var err error
	simg.StrImg, err = json.Marshal(imageURLs)
	if err != nil {
		fmt.Println("JSON 직렬화 실패:", err)
		return
	}

	stat, err := strconv.Atoi(sinfo["stat"])
	if err != nil {
		fmt.Println("stat 변환 실패:", err)
		return
	}

	simg.Body = sinfo["sbody"]
	simg.Nick = sinfo["nick"]
	simg.Stat = stat

	// for filename, url := range imageURLs {
	// 	fmt.Println(filename, url)

	// 	imageURLsJSON = append(imageURLsJSON, url)
	// }

	// Insert story into database
	query := `INSERT INTO story (uid, nick, body, stat, str_img, at_create, at_update) 
				VALUES (?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := sdb.Exec(query, uid, simg.Nick, simg.Body, simg.Stat, simg.StrImg)
	if err != nil {
		fmt.Println("스토리 삽입 실패:", err)
		return
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		fmt.Println("LastInsertId 조회 실패:", err)
	}
	fmt.Println(result)
	// result, err := sdb.Exec(query, uid, sbody, simg.StrImg01, simg.StrImg02, simg.StrImg03, simg.StrImg04, simg.StrImg05, simg.AtCreate, simg.AtUpdate)

	/* 	result, err := sdb.Exec(query, uid, sbody, string(imageURLsJSON))
	   	if err != nil {
	   		fmt.Println("스토리 삽입 실패:", err)
	   		return
	   	}

	   	lastID, err := result.LastInsertId()
	   	if err != nil {
	   		fmt.Println("LastInsertId 조회 실패:", err)
	   		return
	   	}
	*/
	fmt.Printf("스토리 삽입 성공 - ID: %d\n", lastID)
	fmt.Printf("이미지 URLs: %v\n", imageURLs)

}

// ===============history db test===============================================
func Test_SetTmpNoti(t *testing.T) {
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	// // Insert 10 test notification records
	// query := `INSERT INTO noti_his (uid, nt_type, nt_title, nt_msg, at_noti, stat, frm_uid, frm_url, frm_nick)
	// 			VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?)`

	query := `INSERT INTO noti_his (uid, nt_type, nt_title, nt_msg, at_noti, stat, frm_uid, frm_url, frm_nick) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Sample data arrays
	titles := []string{"새 메시지", "친구 요청", "시스템 알림", "이벤트", "공지사항", "채팅 초대", "프로필 방문", "좋아요", "댓글", "업데이트"}
	messages := []string{
		"새로운 메시지가 도착했습니다",
		"친구 요청을 받았습니다",
		"시스템 점검 안내",
		"특별 이벤트에 참여하세요",
		"중요한 공지사항입니다",
		"채팅방에 초대되었습니다",
		"누군가 프로필을 방문했습니다",
		"게시물에 좋아요를 받았습니다",
		"새로운 댓글이 달렸습니다",
		"앱이 업데이트되었습니다",
	}
	nicknames := []string{"김철수", "이영희", "박민수", "최지은", "정현우", "강수진", "조동혁", "윤미영", "장준호", "임은지"}
	urls := []string{
		"https://example.com/profile1",
		"https://example.com/profile2",
		"https://example.com/profile3",
		"https://example.com/profile4",
		"https://example.com/profile5",
		"https://example.com/profile6",
		"https://example.com/profile7",
		"https://example.com/profile8",
		"https://example.com/profile9",
		"https://example.com/profile10",
	}

	for i := 0; i < 10; i++ {
		// uid: random 8-digit number
		uid := uint64(10000000 + rand.Intn(90000000))

		// nt_type: 0-9 random
		ntType := rand.Intn(10)

		// Use data from arrays
		title := titles[i]
		message := messages[i]
		nickname := nicknames[i]
		url := urls[i]

		// at_noti: random time within last 30 days
		now := time.Now()
		randomDays := rand.Intn(30)
		randomHours := rand.Intn(24)
		randomMinutes := rand.Intn(60)
		atNoti := now.AddDate(0, 0, -randomDays).Add(-time.Duration(randomHours)*time.Hour - time.Duration(randomMinutes)*time.Minute)

		// stat: 0 or 1 (read/unread)
		stat := rand.Intn(2)

		// frm_uid: random 8-digit number
		frmUid := uint64(10000000 + rand.Intn(90000000))

		_, err := hdb.Exec(query, uid, ntType, title, message, atNoti, stat, frmUid, url, nickname)
		if err != nil {
			fmt.Printf("알림 %d 삽입 실패: %v\n", i+1, err)
			continue
		}

		fmt.Printf("알림 %d 삽입 성공: uid=%d, title=%s\n", i+1, uid, title)
	}

	fmt.Println("10개의 테스트 알림 데이터 삽입 완료")
}

func Test_GetNotiAllList(t *testing.T) {
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	query := "SELECT idx, nt_type, nt_title, nt_msg, at_noti, stat, frm_uid, frm_url, frm_nick FROM noti_his WHERE uid = ?"
	rows, err := hdb.Query(query, 47743344)
	if err != nil {
		fmt.Println("조회 실패:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var noti ptl.Noti
		err := rows.Scan(&noti.Idx, &noti.Ntype, &noti.Title, &noti.Msg, &noti.AtMsg, &noti.Stat, &noti.FromUid, &noti.FromUrl, &noti.FromNick)
		if err != nil {
			fmt.Println("조회 실패:", err)
			return
		}
		fmt.Println(noti)
	}

}

func Test_GetNotiCount(t *testing.T) {
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	query := "SELECT COUNT(*) FROM noti_his WHERE uid = ? AND stat = 0"
	rows, err := hdb.Query(query, 47743344)
	if err != nil {
		fmt.Println("조회 실패:", err)
		return
	}
	defer rows.Close()

	var count int
	err = rows.Scan(&count)
	if err != nil {
		fmt.Println("조회 실패:", err)
		return
	}
	fmt.Println(count)
}

func Test_GetNotiDetail(t *testing.T) {
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	query := "SELECT idx, nt_type, nt_title, nt_msg, at_noti, stat, frm_uid, frm_url, frm_nick FROM noti_his WHERE idx = ?"
	row := hdb.QueryRow(query, 1)

	var noti ptl.Noti
	err := row.Scan(&noti.Idx, &noti.Ntype, &noti.Title, &noti.Msg, &noti.AtMsg, &noti.Stat, &noti.FromUid, &noti.FromUrl, &noti.FromNick)
	if err != nil {
		fmt.Println("조회 실패:", err)
		return
	}
	fmt.Println(noti)

}

func Test_SetAnnouncement(t *testing.T) {
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	query := "INSERT INTO anuc_his (an_title, an_body, an_url, at_msg) VALUES (?, ?, ?, ?)"

	for i := 0; i < 10; i++ {
		title := fmt.Sprintf("test %d", i+1)
		body := fmt.Sprintf("test %d", i+1)
		url := fmt.Sprintf("https://example.com/announcement/%d", i+1)
		// atMsg := time.Now().AddDate(0, 0, -rand.Intn(30))
		now := time.Now()
		randomDays := rand.Intn(30)
		randomHours := rand.Intn(24)
		randomMinutes := rand.Intn(60)
		atMsg := now.AddDate(0, 0, -randomDays).Add(-time.Duration(randomHours)*time.Hour - time.Duration(randomMinutes)*time.Minute)

		_, err := hdb.Exec(query, title, body, url, atMsg)
		if err != nil {
			fmt.Println("조회 실패:", err)
			return
		}
	}

	fmt.Println("10개의 테스트 공지사항 데이터 삽입 완료")
}

func Test_GetAnnouncementList(t *testing.T) {
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	query := "SELECT idx, an_title, an_body, an_url, at_msg FROM anuc_his ORDER BY at_msg DESC LIMIT 3"
	rows, err := hdb.Query(query)
	if err != nil {
		fmt.Println("조회 실패:", err)
		return
	}
	defer rows.Close()

	announcementList := []ptl.Announcement{}
	for rows.Next() {
		var a ptl.Announcement
		err := rows.Scan(&a.Idx, &a.Title, &a.Body, &a.Url, &a.AtMsg)
		if err != nil {
			fmt.Println("조회 실패:", err)
			return
		}
		// AtMsg가 현재시간보다 7일전이면 stat을 0으로 설정
		atMsgTime, err := time.Parse("2006-01-02 15:04:05", a.AtMsg)
		if err == nil {
			sevenDaysAgo := time.Now().AddDate(0, 0, -7)
			if atMsgTime.After(sevenDaysAgo) {
				a.Stat = 1 // new
			} else {
				a.Stat = 0 // old
			}
		}
		announcementList = append(announcementList, a)
	}

	fmt.Println(announcementList)
	fmt.Println(len(announcementList))
}

func Test_GetAnnouncementDetail(t *testing.T) {
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	query := "SELECT idx, an_title, an_body, an_url, at_msg FROM anuc_his WHERE idx = ?"
	row := hdb.QueryRow(query, 1)

	var a ptl.Announcement
	if err := row.Scan(&a.Idx, &a.Title, &a.Body, &a.Url, &a.AtMsg); err != nil {
		fmt.Println("조회 실패:", err)
		return
	} else if err == sql.ErrNoRows {
		fmt.Println("공지사항 상세 조회 실패: 공지사항이 없습니다")
		return
	}
	fmt.Println(a)
	fmt.Println("공지사항 상세 조회 성공")
}

// ===============DM Room db test===============================================
func Test_CreateDMRoom(t *testing.T) {
	// HistoryDB의 CreateDMRoom 메서드 테스트 코드 구현 (chat_his 테이블 사용)
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	// 임시 ptl.UserInfoResp 생성
	tUser := &ptl.UserInfoResp{
		Uid:      123,
		Nick:     "상대방닉네임",
		Area:     "서울",
		Age:      "25",
		Gender:   "1",
		ThumbPic: "https://example.com/thumb.jpg",
	}

	uid := uint64(234) // 내 uid

	// HistoryDB 초기화 (mock root, conf nil 허용)
	historyDB := &HistoryDB{
		conndb: hdb,
	}

	roomID, err := historyDB.CreateDMRoom(uid, tUser)
	if err != nil {
		fmt.Println(err)
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			fmt.Println("already room", roomID)
			return
		}
		fmt.Println("CreateDMRoom 실패:", err)
		return
	}
	fmt.Printf("DM Room 생성 성공 (chat_his idx): %d\n", roomID)
}

func Test_GetDMRoom(t *testing.T) {
	// HistoryDB의 GetDMRoom 메서드 테스트 코드 구현 (chat_his 테이블 사용)
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	roomID := int64(1) // 조회할 DM Room ID

	// HistoryDB 초기화 (mock root, conf nil 허용)
	historyDB := &HistoryDB{
		conndb: hdb,
	}

	room, err := historyDB.GetDMRoom(roomID)
	if err != nil {
		fmt.Println("CreateDMRoom 실패:", err)
		return
	}
	fmt.Printf("DM Room 생성 성공 (chat_his idx): %+v\n", room)
}

func Test_GetDMRoomByUser(t *testing.T) {
	hdb := ConnectDB("hdb")
	if hdb == nil {
		fmt.Println("데이터베이스 연결 실패")
		return
	}
	defer hdb.Close()

	uid := uint64(123) // 조회할 DM Room ID

	// HistoryDB 초기화 (mock root, conf nil 허용)
	historyDB := &HistoryDB{
		conndb: hdb,
	}

	rooms, err := historyDB.GetDMRoomsByUser(uid)
	if err != nil {
		fmt.Println("CreateDMRoom 실패:", err)
		return
	}
	fmt.Printf("DM Room 생성 성공 (chat_his idx): %+v\n", rooms)

}
