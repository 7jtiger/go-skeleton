package models

import (
	"fmt"
	"sync"
	"time"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	log "ms-gateway/common/logger"
	"ms-gateway/conf"
)

/*
CREATE TABLE `evt_chkin` (
	`idx` int NOT NULL,
	`uid` bigint unsigned NOT NULL,
	`seq_count` int DEFAULT NULL,
	`at_lastupd` int DEFAULT NULL,
	PRIMARY KEY (`idx`),
	UNIQUE KEY `idx_UNIQUE` (`idx`),
	UNIQUE KEY `uid_UNIQUE` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
*/

type ItemDB struct {
	conndb *sql.DB
	cfg    *conf.Config
	root   *Repositories

	quit     chan struct{}
	quitWait sync.WaitGroup
}

func NewItemDB(cf *conf.Config, root *Repositories) (IRepository, error) {
	r := &ItemDB{
		cfg:  cf,
		root: root,
		quit: make(chan struct{}),
	}

	var err error
	c := r.cfg.DB["idb"]
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

	log.Info("load repository : Item db")
	return r, nil
}

func (p *ItemDB) Start() error {
	return nil
}

func (p *ItemDB) Terminate() {
	close(p.quit)
	p.quitWait.Wait()

	log.Info("Terminated Database")
}

func (p *ItemDB) Close() error {
	return p.conndb.Close()
}

func (p *ItemDB) Ping() error {
	return p.conndb.Ping()
}

func (p *ItemDB) heartbeat() {
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

func (p *ItemDB) GetCheckIn(uid uint64) (bool, error) {
	date := time.Now().Format("2006-01-02")
	row := p.conndb.QueryRow("SELECT COUNT(*) FROM evt_chkin WHERE uid = ? AND at_lastupd = ?", uid, date)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	return true, nil
}

//====terms==========================================================================
/*
CREATE TABLE `terms_info` (
  `idx` int NOT NULL AUTO_INCREMENT,
  `privacy_url` varchar(145) DEFAULT NULL,
  `terms_url` varchar(145) DEFAULT NULL,
  PRIMARY KEY (`idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
*/

func (p *ItemDB) GetTermsInfo() (string, string, error) {
	row := p.conndb.QueryRow("SELECT privacy_url, terms_url FROM terms_info")
	var privacyUrl, termsUrl string
	err := row.Scan(&privacyUrl, &termsUrl)
	if err != nil {
		return "", "", err
	}
	return privacyUrl, termsUrl, nil
}
