package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	log "banker/common/logger"
	"banker/conf"
	pt "banker/protocol"
)

// ScopeDB : 유저정보를 제공
type ContractDB struct {
	conndb *sql.DB
	cf     *conf.Config
	root   *Repositories

	quit     chan struct{}
	quitWait sync.WaitGroup
}

// NewContractDB : 객체 할당 및 반환
func NewContractDB(cfg *conf.Config, root *Repositories) (IRepository, error) {
	r := &ContractDB{
		cf:   cfg,
		root: root,
		quit: make(chan struct{}),
	}
	var err error
	c := r.cf.DB["contract"]
	uri := fmt.Sprintf("%s:%s@tcp(%s)/%s", c["user"], c["pass"], c["host"], c["name"])
	r.conndb, err = sql.Open("mysql", uri)
	if err != nil {
		return nil, fmt.Errorf("database connection error: %v", err)
	}

	r.conndb.SetMaxIdleConns(30)
	r.conndb.SetMaxOpenConns(300)
	r.conndb.SetConnMaxLifetime(time.Minute * 3)

	go r.heartbeat()

	log.Info("load repository : ContractDB")
	return r, nil
}

func (p *ContractDB) Terminate() {
	close(p.quit)
	p.quitWait.Wait()

	log.Info("Terminated Database")
}

func (r *ContractDB) Start() error {
	return nil
}

func (p *ContractDB) Close() error {
	return p.conndb.Close()
}

func (p *ContractDB) Ping() error {
	return p.conndb.Ping()
}

func (p *ContractDB) heartbeat() {
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
				if err := e.Interface().(IRepository).Ping(); err != nil {
					log.Error("mysql ping fail.", err)
				}
			}
		}
	}
}

func (p *ContractDB) GetContractABI(db *sql.DB, address string) (*pt.Contract, error) {
	query := "SELECT id, address, abi FROM contracts WHERE address = ?"
	row := db.QueryRow(query, address)

	var cABI pt.Contract
	err := row.Scan(&cABI.CID, &cABI.CAddress, &cABI.CName, &cABI.CABI, &cABI.CType, &cABI.CDate)
	if err != nil {
		return nil, err
	}

	return &cABI, nil
}
