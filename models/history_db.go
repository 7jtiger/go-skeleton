package models

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"database/sql"

	"github.com/go-sql-driver/mysql"

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

/*
CREATE TABLE `chat_his` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `uid` bigint unsigned NOT NULL,
  `tid` bigint unsigned NOT NULL,
  `room_id` varchar(45) NOT NULL,
  `tnick` varchar(45) DEFAULT NULL,
  `tarea` varchar(45) DEFAULT NULL,
  `tage` int DEFAULT NULL,
  `tgender` tinyint(1) DEFAULT NULL,
  `tthumb_url` varchar(300) DEFAULT NULL,
  `st_chat` tinyint(1) DEFAULT NULL,
  `at_crtchat` datetime DEFAULT NULL,
  `paid_point` double DEFAULT NULL,
  `at_update` datetime DEFAULT NULL,
  PRIMARY KEY (`idx`),
  UNIQUE KEY `room_id_UNIQUE` (`room_id`),
  KEY `idx_chat_his_uid` (`uid`),
  KEY `idx_chat_his_at_update` (`at_update`),
  KEY `idx_chat_his_st_chat_uid_at_update` (`st_chat`,`uid`,`at_update`),
  KEY `idx_chat_his_st_chat_tid_at_update` (`st_chat`,`tid`,`at_update`),
  KEY `idx_chat_his_room_id_st_chat_at_update` (`room_id`,`st_chat`,`at_update`)
) ENGINE=InnoDB AUTO_INCREMENT=15 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
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

	// 풀 과다 방지: DB 4개 합산이 MySQL max_connections를 넘지 않도록 제한
	r.conndb.SetMaxIdleConns(25)
	r.conndb.SetMaxOpenConns(50)
	r.conndb.SetConnMaxLifetime(30 * time.Minute)
	r.conndb.SetConnMaxIdleTime(5 * time.Minute)

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

// CreateDMRoom dm_room 생성 (room_id = FormatDMPairRoomID(uid, tid))
func (p *HistoryDB) CreateDMRoom(uid uint64, tUser *ptl.UserInfoResp) (int64, error) {
	now := time.Now()
	pairRoomID := FormatDMPairRoomID(uid, tUser.Uid)
	res, err := p.conndb.Exec(
		"INSERT INTO chat_his (uid, tid, room_id, st_chat, at_crtchat, at_update, paid_point) VALUES (?, ?, ?, ?, ?, ?, ?)",
		uid, tUser.Uid, pairRoomID, STChatBothIn, now, now, 0,
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

// GetDMRoom idx 기준 조회
func (p *HistoryDB) GetDMRoom(ridx int64) (*DMRoomRow, error) {
	row := p.conndb.QueryRow(
		"SELECT "+dmRoomSelectCols+" FROM chat_his WHERE idx = ? LIMIT 1", ridx,
	)
	return scanDMRoomRow(row)
}

// GetDMRoomByPair uid/tid 쌍 양방향 조회 (없으면 room_id=min_max 폴백)
func (p *HistoryDB) GetDMRoomByPair(uid, tid uint64) (*DMRoomRow, error) {
	row := p.conndb.QueryRow(
		"SELECT "+dmRoomSelectCols+" FROM chat_his WHERE (uid = ? AND tid = ?) OR (uid = ? AND tid = ?) LIMIT 1",
		uid, tid, tid, uid,
	)
	dm, err := scanDMRoomRow(row)
	if err == nil {
		return dm, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return p.GetDMRoomByPairRoomID(FormatDMPairRoomID(uid, tid))
}

// GetDMRoomByPairRoomID chat_his.room_id (minUid_maxUid) 로 조회
func (p *HistoryDB) GetDMRoomByPairRoomID(pairRoomID string) (*DMRoomRow, error) {
	row := p.conndb.QueryRow(
		"SELECT "+dmRoomSelectCols+" FROM chat_his WHERE room_id = ? LIMIT 1",
		pairRoomID,
	)
	return scanDMRoomRow(row)
}

// GetDMRoomByRid idx 문자열 기준 조회 (API room_id)
func (p *HistoryDB) GetDMRoomByRid(rid string) (*DMRoomRow, error) {
	row := p.conndb.QueryRow(
		"SELECT "+dmRoomSelectCols+" FROM chat_his WHERE idx = ? LIMIT 1", rid,
	)
	return scanDMRoomRow(row)
}

// TouchDMRoomActivity 마지막 채팅 활동 시각 갱신 (st_chat 변경 없음)
func (p *HistoryDB) TouchDMRoomActivity(ridx int64, at time.Time) error {
	_, err := p.conndb.Exec("UPDATE chat_his SET at_update = ? WHERE idx = ?", at, ridx)
	return err
}

// UpdateDMRoomStatus st_chat 갱신 (0=BothLeft, 1=BothIn, 2=UIDLeft, 3=TIDLeft)
func (p *HistoryDB) UpdateDMRoomStatus(ridx int64, st int) error {
	_, err := p.conndb.Exec("UPDATE chat_his SET st_chat = ?, at_update = ? WHERE idx = ?", st, time.Now(), ridx)
	return err
}

// IsMySQLDuplicate MySQL unique/duplicate key (1062)
func IsMySQLDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// LeaveDMRoom 사용자 1명 나가기 (st_chat 전이)
func (p *HistoryDB) LeaveDMRoom(ridx int64, leaverUID uint64) (*DMRoomRow, error) {
	room, err := p.GetDMRoom(ridx)
	if err != nil {
		return nil, err
	}
	newSt, err := nextSTChatOnLeave(room.STChat, leaverUID, room)
	if err != nil {
		return nil, err
	}
	if newSt != room.STChat {
		if err := p.UpdateDMRoomStatus(ridx, newSt); err != nil {
			return nil, err
		}
		room.STChat = newSt
		room.AtUpdate = time.Now()
	}
	return room, nil
}

// ActivateDMRoomSide joiner만 재참여 (mkroom/ensureDMRoom)
func (p *HistoryDB) ActivateDMRoomSide(room *DMRoomRow, joinerUID uint64) (*DMRoomRow, error) {
	newSt := nextSTChatOnRejoin(room.STChat, joinerUID, room)
	if newSt == room.STChat {
		return room, nil
	}
	if err := p.UpdateDMRoomStatus(room.Idx, newSt); err != nil {
		return nil, err
	}
	room.STChat = newSt
	room.AtUpdate = time.Now()
	return room, nil
}

// SoftDeleteDMRoom 방 전체 비활성 (양쪽 나감)
func (p *HistoryDB) SoftDeleteDMRoom(ridx int64) error {
	return p.UpdateDMRoomStatus(ridx, STChatBothLeft)
}

// SoftDeleteDMRoomsByUser 사용자 쌍 기준 양쪽 나감
func (p *HistoryDB) SoftDeleteDMRoomsByUser(uid, tid uint64) error {
	room, err := p.GetDMRoomByPair(uid, tid)
	if err != nil {
		return err
	}
	return p.UpdateDMRoomStatus(room.Idx, STChatBothLeft)
}

// ActivateDMRoomByPair joiner(uid)만 재참여 — ensureDMRoom 호환
func (p *HistoryDB) ActivateDMRoomByPair(uid, tid uint64) error {
	room, err := p.GetDMRoomByPair(uid, tid)
	if err != nil {
		return err
	}
	_, err = p.ActivateDMRoomSide(room, uid)
	return err
}

func (p *HistoryDB) UdtDMPaid(ridx int64, point float64) error {
	_, err := p.conndb.Exec(
		"UPDATE chat_his SET paid_point = paid_point + ? WHERE idx = ?",
		point, ridx,
	)
	return err
}

const DMRoomListPageSize = 10

func (p *HistoryDB) countDMRoomsByUser(uid uint64) (int, error) {
	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT 1 FROM chat_his WHERE uid = ? AND st_chat IN (1, 3)
			UNION
			SELECT 1 FROM chat_his WHERE tid = ? AND st_chat IN (1, 2)
		) AS T`
	var totalCount int
	err := p.conndb.QueryRow(countQuery, uid, uid).Scan(&totalCount)
	return totalCount, err
}

func (p *HistoryDB) scanDMRoomRows(rows *sql.Rows) ([]DMRoomRow, error) {
	rooms := make([]DMRoomRow, 0)
	for rows.Next() {
		dm, err := scanDMRoomFields(rows)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, *dm)
	}
	return rooms, rows.Err()
}

// GetAllDMRoomsByUser 사용자 참여 DM 방 전체 조회 (정렬·페이징은 호출측)
func (p *HistoryDB) GetAllDMRoomsByUser(uid uint64) ([]DMRoomRow, int, error) {
	totalCount, err := p.countDMRoomsByUser(uid)
	if err != nil {
		return nil, 0, err
	}
	if totalCount == 0 {
		return []DMRoomRow{}, 0, nil
	}

	query := `
		SELECT * FROM (
			SELECT idx, uid, tid, room_id, st_chat, at_crtchat, at_update, paid_point
			FROM chat_his WHERE uid = ? AND st_chat IN (1, 3)
			UNION
			SELECT idx, uid, tid, room_id, st_chat, at_crtchat, at_update, paid_point
			FROM chat_his WHERE tid = ? AND st_chat IN (1, 2)
		) AS T`

	rows, err := p.conndb.Query(query, uid, uid)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rooms, err := p.scanDMRoomRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return rooms, totalCount, nil
}

// GetDMRoomsByUser 사용자 기준 DM 방 목록 조회 (at_update DESC, DB 페이징 — 레거시/테스트용)
func (p *HistoryDB) GetDMRoomsByUser(uid uint64, page int) (*[]DMRoomRow, int, error) {
	offset := (page - 1) * DMRoomListPageSize
	if offset < 0 {
		offset = 0
	}

	totalCount, err := p.countDMRoomsByUser(uid)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT * FROM (
			SELECT idx, uid, tid, room_id, st_chat, at_crtchat, at_update, paid_point
			FROM chat_his WHERE uid = ? AND st_chat IN (1, 3)
			UNION
			SELECT idx, uid, tid, room_id, st_chat, at_crtchat, at_update, paid_point
			FROM chat_his WHERE tid = ? AND st_chat IN (1, 2)
		) AS T
		ORDER BY at_update DESC
		LIMIT ? OFFSET ?`

	rows, err := p.conndb.Query(query, uid, uid, DMRoomListPageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rooms, err := p.scanDMRoomRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return &rooms, totalCount, nil
}
