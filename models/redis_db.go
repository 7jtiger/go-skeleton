package models

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/common"

	log "go-skeleton/common/logger"
	"go-skeleton/conf"

	//must be update v8
	"github.com/go-redis/redis/v7"
)

type StoredUnsignedTx struct {
	From      common.Address
	To        common.Address
	Tx        []byte
	ChainName string
}

type RedisDB struct {
	client *redis.Client
}

// NewRedisDB : RedisDB 객체 할당 및 반환
func NewRedisDB(cf *conf.Config, root *Repositories) (IRepository, error) {
	redisOption := redis.Options{
		Addr:      cf.DB["redis"]["host"].(string),
		Password:  cf.DB["redis"]["pass"].(string), // no password set
		DB:        0,                               // use default DB
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if strings.EqualFold(cf.Server.Mode, "alpha") {
		redisOption.TLSConfig = nil
	}

	client := redis.NewClient(&redisOption)

	if _, err := client.Ping().Result(); err != nil {
		return nil, err
	}

	r := &RedisDB{
		client: client,
	}

	log.Info("load repository : RedisDB")
	return r, nil
}

func (p *RedisDB) Start() error {
	return nil
}

func (p *RedisDB) Close() error {
	return p.client.Close()
}

func (p *RedisDB) Ping() error {
	return p.client.Ping().Err()
}

func (r *RedisDB) Terminate() {
	log.Info("Terminated Database")
}

func (r *RedisDB) SetCache(key, data string) error {

	if err := r.client.Set(key, data, time.Duration(30)*time.Minute).Err(); err != nil {
		return err
	}
	return nil
}

func (r *RedisDB) SetBytes(key string, data []byte) error {
	if err := r.client.Set(key, hexutil.Encode(data), 3600e9).Err(); err != nil {
		return err
	}
	return nil
}

func (r *RedisDB) GetCache(key string) (string, error) {
	// redisKey := `auth_` + platform + `_email_` + openID

	email, err := r.client.Get(key).Result()
	if err != nil {
		return "", err
	}
	return email, nil
}

func (r *RedisDB) DeleteCache(key string) error {

	if err := r.client.Del(key).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) IncCount(key string) error {
	if err := r.client.Incr(key).Err(); err != nil && err != redis.Nil {
		return err
	}

	if err := r.client.Expire(key, time.Duration(30)*time.Minute).Err(); err != nil {
		return err
	}
	return nil
}

func (r *RedisDB) SetNXCache(key, data string) error {
	if err := r.client.SetNX(key, data, 3600e9).Err(); err != nil {
		return err
	}
	return nil
}

func (r *RedisDB) HSetMember(arPair []string) error {
	if err := r.client.HSet("member", arPair).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) HGetMember(key string) string {
	res := r.client.HGet("member", key)
	return res.Val()
}
