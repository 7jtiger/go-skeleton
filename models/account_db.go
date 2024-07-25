package models

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	log "go-skeleton/common/logger"
	"go-skeleton/conf"
)

// ScopeDB : 유저정보를 제공
type AccountDB struct {
	client         *mongo.Client
	colUinfo       *mongo.Collection
	cacheChainLock sync.RWMutex
	start          chan struct{}
}

// NewAccountDB : 객체 할당 및 반환
func NewAccountDB(cf *conf.Config, root *Repositories) (IRepository, error) {
	cfg := cf.DB["accountDB"]
	r := &AccountDB{
		start: make(chan struct{}),
	}
	var err error

	credential := options.Credential{
		Username: cfg["user"].(string),
		Password: cfg["pass"].(string),
	}

	if r.client, err = mongo.Connect(context.Background(), options.Client().ApplyURI(cfg["datasource"].(string)).SetAuth(credential)); err != nil {
		return nil, err
	} else if err := r.client.Ping(context.Background(), nil); err != nil {
		return nil, err
	} else {
		db := r.client.Database(cfg["name"].(string)) //accountdb
		r.colUinfo = db.Collection("uinfo")
	}

	log.Info("load repository : AccountDB")
	return r, nil
}

func (p *AccountDB) Start() error {
	return func() (err error) {
		defer func() {
			if v := recover(); v != nil {
				err = v.(error)
			}
		}()
		close(p.start)
		return
	}()
}

func (p *AccountDB) Terminate() {
	log.Info("Terminated Database")
}

func (p *AccountDB) Close() error {
	return p.client.Disconnect(context.TODO())
}

func (p *AccountDB) Ping() error {
	return p.client.Ping(context.TODO(), nil)
}

func (p *AccountDB) GetAccount() {
	// filter := bson.D{}
	// cursor, err := p.colUinfo.Find(context.TODO(), filter)
	// if err != nil {
	// 	panic(err)
	// }

	// var pers []Uinfo
	// for _, result := range ufo {
	// 	cursor.Decode(&result)
	// 	output, err := json.MarshalIndent(result, "", "    ")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Printf("%s\n", output)
	// }
}
