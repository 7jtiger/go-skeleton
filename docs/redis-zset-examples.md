# Redis ZSET을 이용한 나열 방식 예제

## 개요

Redis ZSET(Sorted Set)은 각 멤버에 점수(score)를 부여하여 자동으로 정렬된 상태를 유지하는 데이터 구조입니다. 점수에 따라 오름차순으로 정렬되며, 다양한 나열 및 랭킹 시스템 구현에 최적화되어 있습니다.

## 1. 기본 ZSET 사용법

### 데이터 추가 및 조회
```redis
# 데이터 추가 (ZADD key score member)
ZADD leaderboard 100 "player1"
ZADD leaderboard 200 "player2" 
ZADD leaderboard 150 "player3"

# 전체 조회 (낮은 점수부터)
ZRANGE leaderboard 0 -1 WITHSCORES
# 결과: 1) "player1" 2) "100" 3) "player3" 4) "150" 5) "player2" 6) "200"

# 전체 조회 (높은 점수부터)
ZREVRANGE leaderboard 0 -1 WITHSCORES
# 결과: 1) "player2" 2) "200" 3) "player3" 4) "150" 5) "player1" 6) "100"
```

## 2. 현재 프로젝트의 채팅룸 리스트 예제

### 구현된 시스템 분석

우리 프로젝트에서는 사용자별 채팅룸 리스트를 ZSET으로 관리합니다:

```go
// 채팅룸 추가 (로그인/접속 시)
func (r *RedisDB) AddUserToChatRoomList(userID, roomID string, roomName string) error {
    now := time.Now()
    
    // 점수는 Unix 타임스탬프 사용
    pipe.ZAdd(r.ctx, fmt.Sprintf("user:%s:chatroom_list", userID), &redis.Z{
        Score:  float64(now.Unix()),
        Member: roomID,
    })
    
    return err
}

// 우선순위 기반 최상단 이동
func (r *RedisDB) MoveRoomToTop(userID, roomID string, eventType string, lastMessage string) error {
    now := time.Now()
    
    // 우선순위별 가중치 계산
    priorityBonus := float64(roomItem.Priority * 1000000000) // 10억 단위 가중치
    newScore := float64(now.Unix()) + priorityBonus
    
    pipe.ZAdd(r.ctx, fmt.Sprintf("user:%s:chatroom_list", userID), &redis.Z{
        Score:  newScore,
        Member: roomID,
    })
    
    return err
}

// 정렬된 리스트 조회 (최신순)
func (r *RedisDB) GetUserChatRoomList(userID string, offset, limit int) ([]ChatRoomListItem, error) {
    roomIDs, err := r.client.ZRevRange(
        r.ctx,
        fmt.Sprintf("user:%s:chatroom_list", userID),
        int64(offset),
        int64(offset+limit-1),
    ).Result()
    
    return rooms, nil
}
```

### Redis 명령어로 보는 예제
```redis
# 사용자 user123의 채팅룸 리스트
ZADD user:user123:chatroom_list 1640995200 "room001"    # 일반 메시지
ZADD user:user123:chatroom_list 2640995200 "room002"    # 멘션 (우선순위 1 + 1000000000)
ZADD user:user123:chatroom_list 3640995200 "room003"    # 긴급 (우선순위 2 + 2000000000)

# 최신순 조회 (높은 점수부터)
ZREVRANGE user:user123:chatroom_list 0 9 WITHSCORES
# 결과: 긴급 > 멘션 > 일반 순서로 나열
```

## 3. 게임 리더보드 예제

### Go 코드 구현
```go
// GameLeaderboard 게임 리더보드 관리
type GameLeaderboard struct {
    client *redis.Client
    ctx    context.Context
}

// AddPlayerScore 플레이어 점수 추가/업데이트
func (g *GameLeaderboard) AddPlayerScore(gameID, playerID string, score int) error {
    key := fmt.Sprintf("game:%s:leaderboard", gameID)
    
    return g.client.ZAdd(g.ctx, key, &redis.Z{
        Score:  float64(score),
        Member: playerID,
    }).Err()
}

// GetTopPlayers 상위 플레이어 조회
func (g *GameLeaderboard) GetTopPlayers(gameID string, limit int) ([]PlayerRank, error) {
    key := fmt.Sprintf("game:%s:leaderboard", gameID)
    
    // 높은 점수부터 조회
    results, err := g.client.ZRevRangeWithScores(g.ctx, key, 0, int64(limit-1)).Result()
    if err != nil {
        return nil, err
    }
    
    players := make([]PlayerRank, len(results))
    for i, result := range results {
        players[i] = PlayerRank{
            Rank:     i + 1,
            PlayerID: result.Member.(string),
            Score:    int(result.Score),
        }
    }
    
    return players, nil
}

// GetPlayerRank 특정 플레이어 순위 조회
func (g *GameLeaderboard) GetPlayerRank(gameID, playerID string) (int, error) {
    key := fmt.Sprintf("game:%s:leaderboard", gameID)
    
    // ZREVRANK: 높은 점수부터의 순위 (0부터 시작)
    rank, err := g.client.ZRevRank(g.ctx, key, playerID).Result()
    if err != nil {
        if err == redis.Nil {
            return -1, errors.New("player not found")
        }
        return -1, err
    }
    
    return int(rank) + 1, nil // 1부터 시작하는 순위로 변환
}

type PlayerRank struct {
    Rank     int    `json:"rank"`
    PlayerID string `json:"playerId"`
    Score    int    `json:"score"`
}
```

### Redis 명령어 예제
```redis
# 플레이어 점수 추가
ZADD game:battle_royale:leaderboard 15000 "player001"
ZADD game:battle_royale:leaderboard 12500 "player002"
ZADD game:battle_royale:leaderboard 18000 "player003"
ZADD game:battle_royale:leaderboard 16200 "player004"

# 상위 3명 조회
ZREVRANGE game:battle_royale:leaderboard 0 2 WITHSCORES
# 결과: 1) "player003" 2) "18000" 3) "player004" 4) "16200" 5) "player001" 6) "15000"

# 특정 플레이어 순위 조회
ZREVRANK game:battle_royale:leaderboard "player002"
# 결과: 3 (4위, 0부터 시작이므로 +1 필요)

# 점수 범위로 조회 (15000점 이상)
ZRANGEBYSCORE game:battle_royale:leaderboard 15000 +inf WITHSCORES
```

## 4. 시간 기반 활동 로그 예제

### Go 코드 구현
```go
// UserActivityLog 사용자 활동 로그 관리
type UserActivityLog struct {
    client *redis.Client
    ctx    context.Context
}

// LogActivity 활동 기록
func (u *UserActivityLog) LogActivity(userID, activityType string) error {
    now := time.Now()
    activityID := fmt.Sprintf("%s:%d", activityType, now.UnixNano())
    
    key := fmt.Sprintf("user:%s:activity_log", userID)
    
    pipe := u.client.Pipeline()
    
    // 활동 추가 (타임스탬프를 점수로 사용)
    pipe.ZAdd(u.ctx, key, &redis.Z{
        Score:  float64(now.Unix()),
        Member: activityID,
    })
    
    // 일주일 이상 된 로그 삭제
    weekAgo := now.AddDate(0, 0, -7).Unix()
    pipe.ZRemRangeByScore(u.ctx, key, "-inf", fmt.Sprintf("%d", weekAgo))
    
    _, err := pipe.Exec(u.ctx)
    return err
}

// GetRecentActivity 최근 활동 조회
func (u *UserActivityLog) GetRecentActivity(userID string, hours int, limit int) ([]Activity, error) {
    key := fmt.Sprintf("user:%s:activity_log", userID)
    
    // N시간 전부터 현재까지
    since := time.Now().Add(-time.Duration(hours) * time.Hour).Unix()
    now := time.Now().Unix()
    
    // 시간 범위로 조회 (최신순)
    results, err := u.client.ZRevRangeByScore(u.ctx, key, &redis.ZRangeBy{
        Min:    fmt.Sprintf("%d", since),
        Max:    fmt.Sprintf("%d", now),
        Offset: 0,
        Count:  int64(limit),
    }).Result()
    
    if err != nil {
        return nil, err
    }
    
    activities := make([]Activity, len(results))
    for i, result := range results {
        parts := strings.Split(result, ":")
        if len(parts) >= 2 {
            timestamp, _ := strconv.ParseInt(parts[1], 10, 64)
            activities[i] = Activity{
                Type:      parts[0],
                Timestamp: time.Unix(0, timestamp),
            }
        }
    }
    
    return activities, nil
}

type Activity struct {
    Type      string    `json:"type"`
    Timestamp time.Time `json:"timestamp"`
}
```

### Redis 명령어 예제
```redis
# 활동 기록 (Unix 타임스탬프를 점수로 사용)
ZADD user:user123:activity_log 1640995200 "login:1640995200000000000"
ZADD user:user123:activity_log 1640995800 "view_post:1640995800000000000"
ZADD user:user123:activity_log 1640996400 "send_message:1640996400000000000"

# 최근 24시간 활동 조회
ZRANGEBYSCORE user:user123:activity_log 1640908800 1640995200

# 오래된 로그 삭제 (일주일 전)
ZREMRANGEBYSCORE user:user123:activity_log -inf 1640390400
```

## 5. 복합 점수 시스템 예제

### Go 코드 구현
```go
// ContentRanking 콘텐츠 랭킹 시스템 (조회수 + 좋아요 + 시간 가중치)
type ContentRanking struct {
    client *redis.Client
    ctx    context.Context
}

// UpdateContentScore 콘텐츠 점수 업데이트
func (c *ContentRanking) UpdateContentScore(contentID string, views, likes int, createdAt time.Time) error {
    // 복합 점수 계산
    // - 조회수: 1점
    // - 좋아요: 10점
    // - 시간 가중치: 최근일수록 높은 점수 (일주일간 유효)
    
    daysSinceCreated := time.Since(createdAt).Hours() / 24
    timeWeight := math.Max(0, 7-daysSinceCreated) // 7일간 감소
    
    score := float64(views) + float64(likes*10) + timeWeight*100
    
    return c.client.ZAdd(c.ctx, "content:trending", &redis.Z{
        Score:  score,
        Member: contentID,
    }).Err()
}

// GetTrendingContent 트렌딩 콘텐츠 조회
func (c *ContentRanking) GetTrendingContent(limit int) ([]string, error) {
    // 높은 점수부터 조회
    return c.client.ZRevRange(c.ctx, "content:trending", 0, int64(limit-1)).Result()
}

// GetContentsByScoreRange 점수 범위로 콘텐츠 조회
func (c *ContentRanking) GetContentsByScoreRange(minScore, maxScore float64, limit int) ([]string, error) {
    return c.client.ZRevRangeByScore(c.ctx, "content:trending", &redis.ZRangeBy{
        Min:   fmt.Sprintf("%f", minScore),
        Max:   fmt.Sprintf("%f", maxScore),
        Count: int64(limit),
    }).Result()
}
```

### Redis 명령어 예제
```redis
# 복합 점수로 콘텐츠 추가
# content001: 조회수 1000, 좋아요 50, 2일 전 작성 (500점 시간가중치)
ZADD content:trending 2000 "content001"  # 1000 + 500 + 500

# content002: 조회수 800, 좋아요 100, 1일 전 작성 (600점 시간가중치)
ZADD content:trending 2400 "content002"  # 800 + 1000 + 600

# 트렌딩 콘텐츠 상위 10개 조회
ZREVRANGE content:trending 0 9 WITHSCORES

# 특정 점수 범위 조회 (2000점 이상)
ZRANGEBYSCORE content:trending 2000 +inf WITHSCORES
```

## 6. 페이지네이션 구현 예제

### Go 코드 구현
```go
// PaginatedZSetQuery 페이지네이션을 지원하는 ZSET 조회
func (r *RedisDB) PaginatedZSetQuery(key string, page, pageSize int, reverse bool) (*PaginationResult, error) {
    offset := (page - 1) * pageSize
    
    var results []redis.Z
    var err error
    
    if reverse {
        // 높은 점수부터 (내림차순)
        results, err = r.client.ZRevRangeWithScores(r.ctx, key, int64(offset), int64(offset+pageSize-1)).Result()
    } else {
        // 낮은 점수부터 (오름차순)
        results, err = r.client.ZRangeWithScores(r.ctx, key, int64(offset), int64(offset+pageSize-1)).Result()
    }
    
    if err != nil {
        return nil, err
    }
    
    // 전체 개수 조회
    total, err := r.client.ZCard(r.ctx, key).Result()
    if err != nil {
        return nil, err
    }
    
    items := make([]ZSetItem, len(results))
    for i, result := range results {
        items[i] = ZSetItem{
            Member: result.Member.(string),
            Score:  result.Score,
            Rank:   offset + i + 1,
        }
    }
    
    return &PaginationResult{
        Items:       items,
        CurrentPage: page,
        PageSize:    pageSize,
        TotalItems:  int(total),
        TotalPages:  int(math.Ceil(float64(total) / float64(pageSize))),
        HasNext:     page*pageSize < int(total),
        HasPrev:     page > 1,
    }, nil
}

type ZSetItem struct {
    Member string  `json:"member"`
    Score  float64 `json:"score"`
    Rank   int     `json:"rank"`
}

type PaginationResult struct {
    Items       []ZSetItem `json:"items"`
    CurrentPage int        `json:"currentPage"`
    PageSize    int        `json:"pageSize"`
    TotalItems  int        `json:"totalItems"`
    TotalPages  int        `json:"totalPages"`
    HasNext     bool       `json:"hasNext"`
    HasPrev     bool       `json:"hasPrev"`
}
```

### Redis 명령어 예제
```redis
# 페이지 1 (0-9번째)
ZREVRANGE leaderboard 0 9 WITHSCORES

# 페이지 2 (10-19번째)  
ZREVRANGE leaderboard 10 19 WITHSCORES

# 페이지 3 (20-29번째)
ZREVRANGE leaderboard 20 29 WITHSCORES

# 전체 개수 확인
ZCARD leaderboard
```

## 7. ZSET 성능 특성

### 시간 복잡도
- **ZADD**: O(log N)
- **ZRANGE/ZREVRANGE**: O(log N + M) (M은 반환되는 요소 수)
- **ZRANK/ZREVRANK**: O(log N)
- **ZREM**: O(log N)
- **ZCARD**: O(1)

### 메모리 최적화 팁
```go
// 1. 오래된 데이터 주기적 정리
func (r *RedisDB) CleanupOldZSetData(key string, maxAge time.Duration) error {
    cutoff := time.Now().Add(-maxAge).Unix()
    return r.client.ZRemRangeByScore(r.ctx, key, "-inf", fmt.Sprintf("%d", cutoff)).Err()
}

// 2. 상위 N개만 유지
func (r *RedisDB) KeepTopNItems(key string, n int) error {
    // 하위 순위 데이터 삭제 (상위 N개만 남김)
    return r.client.ZRemRangeByRank(r.ctx, key, 0, -int64(n+1)).Err()
}

// 3. 배치 처리로 성능 향상
func (r *RedisDB) BatchZSetOperations(operations []ZSetOperation) error {
    pipe := r.client.Pipeline()
    
    for _, op := range operations {
        switch op.Type {
        case "ADD":
            pipe.ZAdd(r.ctx, op.Key, &redis.Z{Score: op.Score, Member: op.Member})
        case "REM":
            pipe.ZRem(r.ctx, op.Key, op.Member)
        }
    }
    
    _, err := pipe.Exec(r.ctx)
    return err
}
```

## 8. 실제 사용 사례 정리

### 1. 채팅룸 리스트 (현재 구현됨)
- **점수**: 시간 + 우선순위 가중치
- **정렬**: 최신순 (ZREVRANGE)
- **용도**: 메시지 도착 시 최상단 재정렬

### 2. 게임 리더보드
- **점수**: 게임 점수
- **정렬**: 높은 점수순 (ZREVRANGE)
- **용도**: 실시간 순위 조회

### 3. 콘텐츠 랭킹
- **점수**: 조회수 + 좋아요 + 시간가중치
- **정렬**: 인기순 (ZREVRANGE)
- **용도**: 트렌딩 콘텐츠 추천

### 4. 활동 로그
- **점수**: Unix 타임스탬프
- **정렬**: 시간순 (ZRANGE/ZREVRANGE)
- **용도**: 최근 활동 내역 조회

이러한 ZSET 활용 방식들을 참고하여 다양한 나열 및 랭킹 시스템을 구현하실 수 있습니다! 🚀 