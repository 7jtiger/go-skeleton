package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	log "banker/common/logger"
	"banker/conf"
	"banker/protocol"
)

// ScopeDB : 유저정보를 제공
type BankerDB struct {
	conndb *sql.DB
	cf     *conf.Config
	root   *Repositories

	quit     chan struct{}
	quitWait sync.WaitGroup
}

// NewBankerDB : 객체 할당 및 반환
func NewBankerDB(cfg *conf.Config, root *Repositories) (IRepository, error) {
	r := &BankerDB{
		cf:   cfg,
		root: root,
		quit: make(chan struct{}),
	}
	var err error
	c := r.cf.DB["banker"]
	uri := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", c["user"], c["pass"], c["host"], c["name"])
	r.conndb, err = sql.Open("mysql", uri)
	if err != nil {
		return nil, fmt.Errorf("database connection error: %v", err)
	}

	r.conndb.SetMaxIdleConns(30)
	r.conndb.SetMaxOpenConns(300)
	r.conndb.SetConnMaxLifetime(time.Minute * 3)

	go r.heartbeat()

	log.Info("load repository : BankerDB")
	return r, nil
}

func (p *BankerDB) Terminate() {
	close(p.quit)
	p.quitWait.Wait()

	log.Info("Terminated Database")
}

func (r *BankerDB) Start() error {
	return nil
}

func (p *BankerDB) Close() error {
	return p.conndb.Close()
}

func (p *BankerDB) Ping() error {
	return p.conndb.Ping()
}

func (p *BankerDB) heartbeat() {
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


