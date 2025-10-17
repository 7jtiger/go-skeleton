package models

import (
	"fmt"
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

func (p *HistoryDB) GetNotiCount(uid uint64) (int, error) {
	rows, err := p.conndb.Query("SELECT COUNT(*) FROM noti_his WHERE uid = ? AND stat = 0", uid)
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
