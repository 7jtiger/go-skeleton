package models

import (
	"context"
	"time"

	log "ms-gateway/common/logger"

	"github.com/redis/go-redis/v9"
)

// ----------------------- HA Redis Functions -----------------------
// Lua script: 리더 키의 값이 본인 MyID일 때만 삭제
var deleteLeaderScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

// Lua script: 리더 키의 값이 본인 MyID일 때만 TTL 갱신
var renewLeaderScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("EXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

const (
	RefreshTTL = 15 // deprecated: RenewLeader now uses ttl param
)

func (r *RedisDB) Terminate() {
	if r.monitorQuit != nil {
		select {
		case <-r.monitorQuit:
		default:
			close(r.monitorQuit)
		}
	}
	log.Info("Terminated RedisDB health monitor")
}

// 리더 상태 체크
func (r *RedisDB) IsHealthy() bool {
	return r.healthy.Load()
}

func (r *RedisDB) healthMonitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.monitorQuit:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := r.client.Ping(ctx).Err()
			cancel()

			wasHealthy := r.healthy.Load()
			if err != nil {
				r.healthy.Store(false)
				if wasHealthy {
					log.Error("Redis: failed health check (unhealthy): %v\n", err)
				}
			} else {
				r.healthy.Store(true)
				if !wasHealthy {
					log.Info("Redis: recover health check (healthy)")
				}
			}
		}
	}
}

func (r *RedisDB) GetKeyPath(ty string) string {
	switch ty {
	case "hbeat":
		return "HA:" + r.modName + ":HBeat:"
	case "leader":
		return "HA:" + r.modName + ":Leader"
	}
	return ""
}

func (r *RedisDB) SetHeartbeat(ctx context.Context, MyID string, ttl time.Duration) error {
	key := r.GetKeyPath("hbeat") + MyID
	return r.client.Set(ctx, key, "alive", ttl).Err()
}

func (r *RedisDB) TrySetLeader(ctx context.Context, MyID string, ttl time.Duration) (bool, error) {
	key := r.GetKeyPath("leader")
	result, err := r.client.SetArgs(ctx, key, MyID, redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return result == "OK", nil
}

func (r *RedisDB) RenewLeader(ctx context.Context, MyID string, ttl time.Duration) (bool, error) {
	lkey := r.GetKeyPath("leader")
	ttlSec := int(ttl.Seconds())
	result, err := renewLeaderScript.Run(ctx, r.client, []string{lkey}, MyID, ttlSec).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (r *RedisDB) GetLeader(ctx context.Context) (string, error) {
	lkey := r.GetKeyPath("leader")
	result, err := r.client.Get(ctx, lkey).Result()
	if err == redis.Nil {
		return "", nil
	}
	return result, err
}

func (r *RedisDB) DeleteLeader(ctx context.Context, MyID string) error {
	lkey := r.GetKeyPath("leader")
	_, err := deleteLeaderScript.Run(ctx, r.client, []string{lkey}, MyID).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	return nil
}

// ----------------------- HA Redis Functions -----------------------
