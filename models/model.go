package models

import (
	"fmt"
	"reflect"
	"sync"

	log "ms-gateway/common/logger"
	"ms-gateway/conf"
)

const (
	noDoc = "mysql: no documents in result"
)

// repository 타입
type IRepository interface {
	Start() error
	Terminate()
	Close() error
	Ping() error
}

// repository의 생성 함수 타입
type RepositoryConstructor func(conf *conf.Config, root *Repositories) (IRepository, error)

// repositories manager
type Repositories struct {
	lock  sync.RWMutex
	lk    sync.RWMutex
	cfg   *conf.Config
	elems map[reflect.Type]reflect.Value
}

// 모든 repository를 생성 및 등록
func NewModel(cf *conf.Config) (*Repositories, error) {
	r := &Repositories{
		cfg:   cf,
		elems: make(map[reflect.Type]reflect.Value),
	}

	constructors := []RepositoryConstructor{
		NewAccountDB,
		NewHistoryDB,
		NewItemDB,
		NewRedisDB,
		NewStoryDB,
		// NewContractDB,
		// NewBankerDB,
		// NewConnectNetwork,
	}

	for _, constructor := range constructors {
		if err := r.Register(constructor, cf); err != nil {
			return nil, err
		}
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	for t, e := range r.elems {
		if err := e.Interface().(IRepository).Start(); err != nil {
			log.Error("NewRepositories", "repository", t, "error", err)
			return nil, err
		}
		log.Info("Repository started", "type", t)
	}

	return r, nil
}

// respository의 constructor를 호출하여 리턴된 instance를 map에 삽입한다.
func (p *Repositories) Register(constructor RepositoryConstructor, config *conf.Config) error {
	r, err := constructor(config, p)
	if err != nil {
		return err
	}
	if r == nil {
		return nil
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	if _, ok := p.elems[reflect.TypeOf(r)]; ok {
		return fmt.Errorf("duplicated instance of %v", reflect.TypeOf(r))
	}
	p.elems[reflect.TypeOf(r)] = reflect.ValueOf(r)
	return nil
}

// 주어진 rs의 타입의 respository를 찾아서 받은 rs에 값을 넣음.
func (p *Repositories) Get(rs ...interface{}) error {
	p.lk.Lock()
	defer p.lk.Unlock()

	var notFounds []reflect.Type

	for _, v := range rs {
		elem := reflect.ValueOf(v).Elem()
		if e, ok := p.elems[elem.Type()]; ok {
			elem.Set(e)
		} else {
			notFounds = append(notFounds, elem.Type())
		}
	}

	if len(notFounds) > 0 {
		err := fmt.Errorf("unknown repository: %v", notFounds[0])
		for _, e := range notFounds[1:] {
			err = fmt.Errorf("%v, %v", err, e)
		}
		return err
	}

	return nil
}

// Shutdown은 등록된 repository heartbeat를 종료하고 DB 연결을 닫습니다.
func (p *Repositories) Shutdown() {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for t, e := range p.elems {
		repo, ok := e.Interface().(IRepository)
		if !ok {
			continue
		}
		repo.Terminate()
		if err := repo.Close(); err != nil {
			log.Error("repository close failed", "type", t, "error", err)
		}
	}
}
