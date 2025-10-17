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

	r.conndb.SetMaxIdleConns(30)
	r.conndb.SetMaxOpenConns(300)
	r.conndb.SetConnMaxLifetime(time.Minute * 3)

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
