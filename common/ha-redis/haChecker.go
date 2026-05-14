package haredis

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	log "ms-gateway/common/logger"
	conf "ms-gateway/conf"
	"ms-gateway/models"

	"github.com/google/uuid"
)

const (
	// maxFailuresBeforeDemote: 연속 N회 실패 후에만 리더 상실 처리
	// leaderTTL이 아직 남아있으므로 일시적 네트워크 지연은 허용
	maxFailuresBeforeDemote = 3

	// renewTimeout: 리더 갱신은 heartbeat보다 중요하므로 더 넉넉한 타임아웃
	renewTimeout = 5 * time.Second

	// heartbeatTimeout: heartbeat 쓰기 타임아웃
	heartbeatTimeout = 3 * time.Second

	HeartbeatTTL  = 10 * time.Second
	LeaderTTL     = 20 * time.Second
	CheckInterval = 5 * time.Second
)

type HAChecker struct {
	mu            sync.RWMutex
	MyID          string
	rdb           *models.RedisDB
	isLeader      bool
	heartbeatTTL  time.Duration
	leaderTTL     time.Duration
	checkInterval time.Duration
	enabled       bool
	quit          chan struct{}
	wg            sync.WaitGroup

	// exponential backoff
	consecutiveFailures int
	maxBackoff          time.Duration
}

func genUuid() string {
	u := uuid.New()
	return hex.EncodeToString(u[:])
}

func NewHAChecker(cfg *conf.Config, mod *models.Repositories) (*HAChecker, error) {
	if !cfg.HAChecker.Checker {
		return &HAChecker{
			enabled:  false,
			isLeader: true, // HA 비활성 시 항상 리더로 동작
			quit:     make(chan struct{}),
		}, nil
	}

	var rdb *models.RedisDB
	if err := mod.Get(&rdb); err != nil {
		log.Error("HAChecker: Redis connection failed: %v\n", err)
		return nil, err
	}

	myID := genUuid()

	// heartbeatTTL := HeartbeatTTL * time.Second
	// leaderTTL := time.Duration(cfg.HAChecker.LeaderTTL) * time.Second
	// checkInterval := time.Duration(cfg.HAChecker.CheckInterval) * time.Second

	h := &HAChecker{
		MyID:          myID,
		rdb:           rdb,
		isLeader:      false,
		heartbeatTTL:  HeartbeatTTL,
		leaderTTL:     LeaderTTL,
		checkInterval: CheckInterval,
		enabled:       true,
		quit:          make(chan struct{}),
		maxBackoff:    30 * time.Second,
	}

	h.wg.Add(2) // heartbeat + leader
	go h.heartbeatLoop()
	go h.leaderLoop()

	log.Info("HAChecker: instanceID=%s, heartbeatTTL=%v, leaderTTL=%v, interval=%v\n", myID, HeartbeatTTL, LeaderTTL, CheckInterval)

	return h, nil
}

func (h *HAChecker) IsLeader() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isLeader
}

func (h *HAChecker) GetMyID() string {
	return h.MyID
}

// backoffDuration은 연속 실패 횟수에 따른 대기시간을 계산한다.
func (h *HAChecker) backoffDuration() time.Duration {
	if h.consecutiveFailures <= 0 {
		return h.checkInterval
	}
	shift := h.consecutiveFailures
	if shift > 4 {
		shift = 4
	}
	d := h.checkInterval * time.Duration(1<<shift)
	if d > h.maxBackoff {
		d = h.maxBackoff
	}
	return d
}

func (h *HAChecker) heartbeatLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	h.writeHeartbeat()

	for {
		select {
		case <-h.quit:
			return
		case <-ticker.C:
			h.writeHeartbeat()
		}
	}
}

func (h *HAChecker) writeHeartbeat() {
	// Redis 비정상이면 불필요한 타임아웃 대기 방지
	if !h.rdb.IsHealthy() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), heartbeatTimeout)
	defer cancel()

	if err := h.rdb.SetHeartbeat(ctx, h.MyID, h.heartbeatTTL); err != nil {
		log.Error("HAChecker: heartbeat write failed: %v\n", err)
	}
}

// leaderLoop은 주기적 폴링으로 리더 선출/갱신을 수행한다.
// 실패 시 backoff를 적용하여 ticker 간격을 동적으로 조절한다.
func (h *HAChecker) leaderLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	h.evaluateLeader()

	for {
		select {
		case <-h.quit:
			return
		case <-ticker.C:
			h.evaluateLeader()

			// backoff 적용: 실패 시 다음 체크 간격을 늘림
			newInterval := h.backoffDuration()
			ticker.Reset(newInterval)
		}
	}
}

func (h *HAChecker) evaluateLeader() {
	// Redis 비정상이면 스킵 (타임아웃 낭비 방지, 현 상태 유지)
	if !h.rdb.IsHealthy() {
		return
	}

	h.mu.RLock()
	wasLeader := h.isLeader
	h.mu.RUnlock()

	if wasLeader {
		h.renewLeaderWithRetry()
		return
	}

	// 팔로워: TrySetLeader로 리더 선출 시도
	ctx, cancel := context.WithTimeout(context.Background(), renewTimeout)
	defer cancel()

	ok, err := h.rdb.TrySetLeader(ctx, h.MyID, h.leaderTTL)
	if err != nil {
		h.consecutiveFailures++
		log.Error("HAChecker: leader election failed (backoff=%v): %v\n", h.backoffDuration(), err)
		return
	}
	if ok {
		h.mu.Lock()
		h.isLeader = true
		h.mu.Unlock()
		h.consecutiveFailures = 0
		log.Info("HAChecker: leader elected (MyID=%s)\n", h.MyID)
	} else {
		h.consecutiveFailures = 0
	}
}

// renewLeaderWithRetry는 리더 갱신을 최대 2회(1차 + 재시도) 시도한다.
// 일시적 context deadline exceeded를 허용하며,
// maxFailuresBeforeDemote 회 연속 실패 시에만 리더를 포기한다.
func (h *HAChecker) renewLeaderWithRetry() {
	ok, err := h.tryRenew()

	// 1차 성공
	if err == nil && ok {
		h.consecutiveFailures = 0
		return
	}

	// 갱신 성공했지만 본인이 아닌 경우 (다른 인스턴스가 리더를 탈취)
	if err == nil && !ok {
		h.mu.Lock()
		h.isLeader = false
		h.mu.Unlock()
		h.consecutiveFailures = 0
		log.Error("HAChecker: leader lost, another instance took over (MyID=%s)\n", h.MyID)
		return
	}

	// 1차 에러 시 짧은 대기 후 재시도
	time.Sleep(1e9)
	ok, err = h.tryRenew()

	if err == nil && ok {
		h.consecutiveFailures = 0
		log.Info("HAChecker: leader renewal recovered on retry\n")
		return
	}

	// 재시도에서도 다른 인스턴스가 리더인 경우
	if err == nil && !ok {
		h.mu.Lock()
		h.isLeader = false
		h.mu.Unlock()
		h.consecutiveFailures = 0
		log.Error("HAChecker: leader lost after retry (MyID=%s)\n", h.MyID)
		return
	}

	// 2회 연속 에러 (timeout 등)
	h.consecutiveFailures++
	log.Error("HAChecker: leader renewal failed (%d/%d, backoff=%v): %v\n", h.consecutiveFailures, maxFailuresBeforeDemote, h.backoffDuration(), err)

	// grace period: N회 연속 실패해야 리더 상실
	if h.consecutiveFailures >= maxFailuresBeforeDemote {
		h.mu.Lock()
		h.isLeader = false
		h.mu.Unlock()
		log.Error("HAChecker: %d consecutive failures, leader demoted (MyID=%s)\n", h.consecutiveFailures, h.MyID)
	}
}

// tryRenew은 단일 RenewLeader 시도를 수행한다.
func (h *HAChecker) tryRenew() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), renewTimeout)
	defer cancel()
	return h.rdb.RenewLeader(ctx, h.MyID, h.leaderTTL)
}

// Shutdown은 graceful 종료 시 호출. 리더 키를 즉시 삭제하여 빠른 failover를 지원한다.
func (h *HAChecker) Shutdown() {
	if !h.enabled {
		return
	}

	h.mu.RLock()
	wasLeader := h.isLeader
	h.mu.RUnlock()

	if wasLeader {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := h.rdb.DeleteLeader(ctx, h.MyID); err != nil {
			log.Error("HAChecker: leader key delete failed: %v\n", err)
		} else {
			log.Info("HAChecker: leader key deleted\n")
		}
		cancel()
	}

	close(h.quit)
	h.wg.Wait()

	h.mu.Lock()
	h.isLeader = false
	h.mu.Unlock()

	log.Info("HAChecker: shutdown completed\n")
}
