package models

import (
	"fmt"
	"reflect"
	"sync"

	log "go-skeleton/common/logger"
	"go-skeleton/conf"
)

const (
	noDoc = "mongo: no documents in result"
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
		// {NewRedisDB, cf}, //다른 respository로서 제일먼저 추가되어야함.
		// {NewAccountDB, cf},
		NewContractDB,
		NewBankerDB,
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
	p.lock.RLock()
	defer p.lock.RUnlock()

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
