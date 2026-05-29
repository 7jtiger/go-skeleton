package models

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	log "ms-gateway/common/logger"
	"ms-gateway/conf"
	ptl "ms-gateway/protocol"
)

type HistoryDB struct {
	conndb *sql.DB
	cfg    *conf.Config
	root   *Repositories

	quit     chan struct{}
	quitWait sync.WaitGroup
}

// DMRoomRow dm_room 정보 테이블 로우 구조체
type DMRoomRow struct {
	Idx       int64     `json:"idx"`     //roomID
	RoomID    string    `json:"room_id"` //roomID
	UID       uint64    `json:"uid"`
	TID       uint64    `json:"tid"`
	TNick     string    `json:"tnick"`
	TArea     string    `json:"tarea"`
	TAge      int       `json:"tage"`
	TGender   int       `json:"tgender"`
	TThumbUrl string    `json:"tthumb_url"`
	STChat    int       `json:"st_chat"`
	AtCrtCHAT time.Time `json:"at_crtchat"`
	PaidPoint float64   `json:"paid_point"`
	AtUpdate  time.Time `json:"at_update"`
}

/*
CREATE TABLE `chat_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` bigint unsigned NOT NULL,
  `tid` bigint unsigned NOT NULL,
  `tnick` varchar(45) DEFAULT NULL,
  `tarea` varchar(45) DEFAULT NULL,
  `tage` int DEFAULT NULL,
  `tgender` tinyint(1) DEFAULT NULL,
  `tthumb_url` varchar(300) DEFAULT NULL,
  `st_chat` tinyint(1) DEFAULT NULL,
  `at_crtchat` datetime DEFAULT NULL,
  `paid_point` double DEFAULT NULL,
  `at_update` datetime DEFAULT NULL,
  PRIMARY KEY (`idx`,`uid`,`tid`),
  UNIQUE KEY `idx_UNIQUE` (`idx`),
  KEY `idx_chat_his_uid` (`uid`),
  KEY `idx_chat_his_tid` (`tid`),
  KEY `idx_chat_his_at_update` (`at_update`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
*/

func NewHistoryDB(cf *conf.Config, root *Repositories) (IRepository, error) {
	r := &HistoryDB{
		cfg:  cf,
		root: root,
		quit: make(chan struct{}),
	}

	var err error
	c := r.cfg.DB["hdb"]
	uri := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", c["user"], c["pass"], c["host"], c["name"])
	r.conndb, err = sql.Open("mysql", uri)
	if err != nil {
		return nil, fmt.Errorf("database connection error: %v", err)
	}

	r.conndb.SetMaxIdleConns(30)
	r.conndb.SetMaxOpenConns(300)
	r.conndb.SetConnMaxLifetime(time.Minute * 3)

	go r.heartbeat()

	log.Info("load repository : History db")
	return r, nil
}

func (p *HistoryDB) Start() error {
	return nil
}

func (p *HistoryDB) Terminate() {
	close(p.quit)
	p.quitWait.Wait()

	log.Info("Terminated Database")
}

func (p *HistoryDB) Close() error {
	return p.conndb.Close()
}

func (p *HistoryDB) Ping() error {
	return p.conndb.Ping()
}

func (p *HistoryDB) heartbeat() {
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

func (p *HistoryDB) GetNotiAllList(uid uint64) (*[]ptl.Noti, error) {
	noti, err := p.conndb.Query("SELECT idx, nt_type, nt_title, nt_msg, at_noti, stat, frm_uid, frm_url, frm_nick FROM noti_his WHERE uid = ? ORDER BY idx DESC", uid)
	if err != nil {
		return nil, err
	}
	defer noti.Close()

	notiList := []ptl.Noti{}
	for noti.Next() {
		var n ptl.Noti
		err := noti.Scan(&n.Idx, &n.Ntype, &n.Title, &n.Msg, &n.AtMsg, &n.Stat, &n.FromUid, &n.FromUrl, &n.FromNick)
		if err != nil {
			return nil, err
		}
		notiList = append(notiList, n)
	}

	return &notiList, nil
}

func (p *HistoryDB) GetNewNotiCount(uid uint64) (int, error) {
	//stat = 0 : default, stat = 1 : send, stat = 2 : read, stat = 3 delete
	rows, err := p.conndb.Query("SELECT COUNT(*) FROM noti_his WHERE uid = ? AND stat = 1", uid)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int
	err = rows.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (p *HistoryDB) GetNotiDetail(idx int) (*ptl.Noti, error) {
	row := p.conndb.QueryRow("SELECT idx, nt_type, nt_title, nt_msg, at_noti, stat, frm_uid, frm_url, frm_nick FROM noti_his WHERE idx = ? LIMIT 1", idx)
	var n ptl.Noti
	err := row.Scan(&n.Idx, &n.Ntype, &n.Title, &n.Msg, &n.AtMsg, &n.Stat, &n.FromUid, &n.FromUrl, &n.FromNick)
	if err != nil {
		return nil, err
	} else if err == sql.ErrNoRows {
		return nil, fmt.Errorf("noti detail not found")
	}

	return &n, nil
}

func (p *HistoryDB) SetNotiRead(idx int) error {
	_, err := p.conndb.Exec("UPDATE noti_his SET stat = 1 WHERE idx = ?", idx)
	if err != nil {
		return err
	}

	return nil
}

func (p *HistoryDB) SaveAnnouncement(a *ptl.Announcement) error {
	_, err := p.conndb.Exec("INSERT INTO anuc_his (an_title, an_body, an_url, at_msg) VALUES (?, ?, ?, ?)", a.Title, a.Body, a.Url, time.Now())
	if err != nil {
		return err
	} else if err == sql.ErrNoRows {
		return fmt.Errorf("announcement save error")
	}

	return nil
}

func (p *HistoryDB) GetAnnouncementList() (*[]ptl.Announcement, error) {
	rows, err := p.conndb.Query("SELECT idx, an_title, an_body, an_url, at_msg FROM anuc_his ORDER BY at_msg DESC LIMIT 20")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	announcementList := []ptl.Announcement{}
	for rows.Next() {
		var a ptl.Announcement
		err := rows.Scan(&a.Idx, &a.Title, &a.Body, &a.Url, &a.AtMsg)
		if err != nil {
			return nil, err
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

	return &announcementList, nil
}

func (p *HistoryDB) GetAnnouncementDetail(idx int) (*ptl.Announcement, error) {
	row := p.conndb.QueryRow("SELECT idx, an_title, an_body, an_url, at_msg FROM anuc_his WHERE idx = ? LIMIT 1", idx)
	var a ptl.Announcement
	err := row.Scan(&a.Idx, &a.Title, &a.Body, &a.Url, &a.AtMsg)
	if err != nil {
		return nil, err
	} else if err == sql.ErrNoRows {
		return nil, fmt.Errorf("announcement detail not found")
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

	return &a, nil
}

func (p *HistoryDB) GetNewMsgCount(uid uint64) (int, error) {
	//stat = 0 : default, stat = 1 : send, stat = 2 : read, stat = 3 delete
	rows, err := p.conndb.Query("SELECT COUNT(*) FROM msg_his WHERE read_uid = ? AND stat = 1", uid)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int
	err = rows.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

/*
	Idx       int       `json:"idx"` //roomID
	UID       uint64    `json:"uid"`	// myUID
	TID       uint64    `json:"tid"`	// targetUID
	TNick     string    `json:"tnick"`	// targetNick
	TArea     string    `json:"tarea"`	// targetArea
	TAge    int       `json:"tage"`	// targetAgent
	TGender   int       `json:"tgender"`
	TThumbUrl string    `json:"tthumb_url"`
	STChat    int       `json:"st_chat"`	// status chat(1: active, 0: inactive)
	AtCrtCHAT time.Time `json:"at_crtchat"`
	PaidPoint float64   `json:"paid_point"`	// paid point
	AtUpdate time.Time `json:"at_update"`	// update time

*/

func getRoomID(uid, tid uint64) string {
	if uid < tid {
		return fmt.Sprintf("%d_%d", uid, tid)
	}

	return fmt.Sprintf("%d_%d", tid, uid)
}

func getIdByRoomID(roomID string) (uint64, uint64) {
	parts := strings.Split(roomID, "_")
	if len(parts) != 2 {
		return 0, 0
	}

	uid, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, 0
	}
	tid, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, 0
	}
	return uid, tid
}

// CreateDMRoom dm_room 생성
func (p *HistoryDB) CreateDMRoom(uid uint64, tUser *ptl.UserInfoResp) (int64, error) {
	roomID := getRoomID(uid, tUser.Uid)
	res, err := p.conndb.Exec(
		"INSERT INTO chat_his (uid, tid, room_id, tnick, tarea, tage, tgender, tthumb_url, st_chat, at_crtchat, paid_point, at_update) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		uid, tUser.Uid, roomID, tUser.Nick, tUser.Area, tUser.Age, tUser.Gender, tUser.ThumbPic, 1, time.Now(), 0, time.Now(),
	)
	if err != nil {
		return 0, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return lastID, nil
}

// GetDMRoom room_id 기준 조회
func (p *HistoryDB) GetDMRoom(ridx int64) (*DMRoomRow, error) {
	row := p.conndb.QueryRow(
		"SELECT idx, uid, tid, room_id, tnick, tarea, tage, tgender, tthumb_url, st_chat, at_crtchat, paid_point, at_update FROM chat_his WHERE idx = ? LIMIT 1", ridx,
	)
	var dm DMRoomRow
	if err := row.Scan(&dm.Idx, &dm.UID, &dm.TID, &dm.RoomID, &dm.TNick, &dm.TArea, &dm.TAge, &dm.TGender, &dm.TThumbUrl, &dm.STChat, &dm.AtCrtCHAT, &dm.PaidPoint, &dm.AtUpdate); err != nil {
		return nil, err
	}
	return &dm, nil
}

/*
// GetDMRoom room_id 기준 조회
func (p *HistoryDB) GetDMRoomByRid(rid string) (*DMRoomRow, error) {
	row := p.conndb.QueryRow(
		"SELECT idx, uid, tid, room_id, tnick, tarea, tage, tgender, tthumb_url, st_chat, at_crtchat, paid_point, at_update FROM chat_his WHERE room_id = ? LIMIT 1", rid,
	)
	var dm DMRoomRow
	if err := row.Scan(&dm.Idx, &dm.UID, &dm.TID, &dm.RoomID, &dm.TNick, &dm.TArea, &dm.TAge, &dm.TGender, &dm.TThumbUrl, &dm.STChat, &dm.AtCrtCHAT, &dm.PaidPoint, &dm.AtUpdate); err != nil {
		return nil, err
	}
	return &dm, nil
}
*/
// GetDMRoom By uid Pair 기준 조회
func (p *HistoryDB) GetDMRoomByPair(uid, tid uint64) (*DMRoomRow, error) {
	rid := getRoomID(uid, tid)
	row := p.conndb.QueryRow(
		"SELECT idx, uid, tid, room_id, tnick, tarea, tage, tgender, tthumb_url, st_chat, at_crtchat, paid_point, at_update FROM chat_his WHERE room_id = ? LIMIT 1",
		rid,
	)
	if row.Err() != nil {
		return nil, row.Err()
	}

	var dm DMRoomRow
	if err := row.Scan(&dm.Idx, &dm.UID, &dm.TID, &dm.RoomID, &dm.TNick, &dm.TArea, &dm.TAge, &dm.TGender, &dm.TThumbUrl, &dm.STChat, &dm.AtCrtCHAT, &dm.PaidPoint, &dm.AtUpdate); err != nil {
		return nil, err
	}

	return &dm, nil
}

// GetDMRoom By uid Pair 기준 조회
func (p *HistoryDB) GetDMRoomByRid(rid string) (*DMRoomRow, error) {
	row := p.conndb.QueryRow(
		"SELECT idx, uid, tid, room_id, tnick, tarea, tage, tgender, tthumb_url, st_chat, at_crtchat, paid_point, at_update FROM chat_his WHERE room_id = ? LIMIT 1",
		rid,
	)
	if row.Err() != nil {
		return nil, row.Err()
	}

	var dm DMRoomRow
	if err := row.Scan(&dm.Idx, &dm.UID, &dm.TID, &dm.RoomID, &dm.TNick, &dm.TArea, &dm.TAge, &dm.TGender, &dm.TThumbUrl, &dm.STChat, &dm.AtCrtCHAT, &dm.PaidPoint, &dm.AtUpdate); err != nil {
		return nil, err
	}

	return &dm, nil
}

func (p *HistoryDB) SoftDeleteDMRoom(ridx int64) error {
	row := p.conndb.QueryRow(
		"UPDATE chat_his SET st_chat = 0 WHERE idx = ?", ridx,
	)

	if row.Err() != nil {
		return row.Err()
	}

	return nil
}

// SoftDeleteDMRoomsByUser 사용자 기준 soft delete
func (p *HistoryDB) SoftDeleteDMRoomsByUser(uid, tid uint64) error {
	rid := getRoomID(uid, tid)
	row := p.conndb.QueryRow(
		"UPDATE chat_his SET st_chat = 0 WHERE room_id = ?", rid,
	)

	if row.Err() != nil {
		return row.Err()
	}

	return nil
}

// ActivateDMRoomByPair 사용자 쌍 DM 방 재활성화
func (p *HistoryDB) ActivateDMRoomByPair(uid, tid uint64) error {
	rid := getRoomID(uid, tid)
	_, err := p.conndb.Exec("UPDATE chat_his SET st_chat = 1 WHERE room_id = ?", rid)
	return err
}

func (p *HistoryDB) UdtDMPaid(ridx int64, point float64) error {
	row := p.conndb.QueryRow(
		"UPDATE chat_his SET paid_point = paid_point + ? WHERE idx = ?",
		point, ridx,
	)

	if row.Err() != nil {
		return row.Err()
	}

	return nil
}

/* // GetDMRoomByPair 상대방 uid로 조회
func (p *HistoryDB) GetDMRoomByUID(tid uint64) (*DMRoomRow, error) {
	row := p.conndb.QueryRow(
		"SELECT idx, uid, tid, tnick, tarea, tage, tgender, tthumb_url, st_chat, at_crtchat, paid_point, at_update FROM chat_his WHERE tid = ? LIMIT 1",
		tid,
	)
	var dm DMRoomRow
	if err := row.Scan(&dm.Idx, &dm.UID, &dm.TID, &dm.TNick, &dm.TArea, &dm.TAge, &dm.TGender, &dm.TThumbUrl, &dm.STChat, &dm.AtCrtCHAT, &dm.PaidPoint, &dm.AtUpdate); err != nil {
		return nil, err
	}
	return &dm, nil
}
*/
// GetDMRoomsByUser 사용자 기준 DM 방 목록 조회
// 페이징 기능 추가와 쿼리 오류 수정

// GetDMRoomsByUser retrieves a paginated list of DM rooms for the given user (as sender or receiver)
// Note: union 서브쿼리에서 paging(ORDER BY, LIMIT)이 제대로 동작하지 않아 OUTER 쿼리에서 정렬 및 페이징 적용 필요
func (p *HistoryDB) GetDMRoomsByUser(uid uint64, page int) (*[]DMRoomRow, int, error) {
	const pageSize = 10
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// UNION 시 LIMIT/OFFSET을 각 서브쿼리에 개별 적용하면 전체에 정상 동작하지 않음
	// 따라서 전체 union 한 뒤 OUTER 쿼리에서 order/limit/offset 처리:
	// (SELECT ... WHERE st_chat=1 AND uid=?) UNION (SELECT ... WHERE st_chat=1 AND tid=?) -> as T
	// SELECT * FROM ( ... ) as T ORDER BY at_update DESC LIMIT ? OFFSET ?

	// 먼저 전체 row 갯수를 구한다.
	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT 1
			FROM chat_his WHERE st_chat = 1 AND uid = ?
			UNION
			SELECT 1
			FROM chat_his WHERE st_chat = 1 AND tid = ?
		) AS T`
	var totalCount int
	err := p.conndb.QueryRow(countQuery, uid, uid).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT * FROM (
			SELECT idx, uid, tid, room_id, tnick, tarea, tage, tgender, tthumb_url, st_chat, at_crtchat, paid_point, at_update
			FROM chat_his WHERE st_chat = 1 AND uid = ?
			UNION
			SELECT idx, uid, tid, room_id, tnick, tarea, tage, tgender, tthumb_url, st_chat, at_crtchat, paid_point, at_update
			FROM chat_his WHERE st_chat = 1 AND tid = ?
		) AS T
		ORDER BY at_update DESC
		LIMIT ? OFFSET ?`
	rows, err := p.conndb.Query(query, uid, uid, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rooms := make([]DMRoomRow, 0)
	for rows.Next() {
		var dm DMRoomRow
		if err := rows.Scan(&dm.Idx, &dm.UID, &dm.TID, &dm.RoomID, &dm.TNick, &dm.TArea, &dm.TAge, &dm.TGender, &dm.TThumbUrl, &dm.STChat, &dm.AtCrtCHAT, &dm.PaidPoint, &dm.AtUpdate); err != nil {
			return nil, 0, err
		}
		rooms = append(rooms, dm)
	}
	// totalCount 값을 사용해 반환 타입 확장 필요시 구조체 등으로 반환하거나, caller 측에서 사용할 수 있도록 조치 필요
	return &rooms, len(rooms), nil
}
