package models

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	ptl "ms-gateway/protocol"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	log "ms-gateway/common/logger"

	//must be update v8
	"github.com/redis/go-redis/v9"
)

// RedisDB Redis 데이터베이스 작업을 위한 구조체
type RedisDB struct {
	client      *redis.Client
	cfg         *conf.Config
	ctx         context.Context
	monitorQuit chan struct{}
	modName     string
	healthy     atomic.Bool // 리더 상태 체크
}

// ChatRoomData Redis에 저장되는 채팅방 데이터 구조체
type ChatRoomData struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OwnerID      string    `json:"ownerId"`
	IsPrivate    bool      `json:"isPrivate"`
	CreatedAt    time.Time `json:"createdAt"`
	IsActive     bool      `json:"isActive"`
	LastActive   time.Time `json:"lastActive"`
	Participants []string  `json:"participants"`
}

// ChatMessageData Redis에 저장되는 채팅 메시지 데이터 구조체
type ChatMessageData struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatPruneResult 채팅 메시지 정리 결과
type ChatPruneResult struct {
	ScannedRooms int `json:"scannedRooms"`
	DeletedMsgs  int `json:"deletedMsgs"`
}

/* // JWT 토큰 세션 정보 구조체
type JWTSession struct {
	LoginTime time.Time `json:"at_login"`
	ExpireAt  time.Time `json:"at_expire"`
	SID       string    `json:"sid"`
	UID       string    `json:"uid"`
	Nick      string    `json:"nick"`
	Gender    string    `json:"gender"`
	Age       int       `json:"age"`
	Area      string    `json:"area"`
	Email     string    `json:"email"`
}
*/
// ChatRoomListItem 채팅룸 리스트 아이템 구조체
type ChatRoomListItem struct {
	RoomID       string    `json:"roomId"`
	RoomName     string    `json:"roomName"`
	LastActivity time.Time `json:"lastActivity"`
	UnreadCount  int       `json:"unreadCount"`
	LastMessage  string    `json:"lastMessage"`
	Priority     int       `json:"priority"` // 0: 일반, 1: 중요, 2: 긴급
}

// UserWebRTCSession 사용자별 WebRTC 세션 정보
type UserWebRTCSession struct {
	UserID          string    `json:"userId"`
	SessionID       string    `json:"sessionId"`
	IsInCall        bool      `json:"isInCall"`
	CallWith        string    `json:"callWith,omitempty"`
	CallStartTime   time.Time `json:"callStartTime,omitempty"`
	ConnectionState string    `json:"connectionState"` // connecting, connected, disconnected
	LastHeartbeat   time.Time `json:"lastHeartbeat"`
	DeviceInfo      string    `json:"deviceInfo"`
	NetworkInfo     string    `json:"networkInfo"`
}

// NewRedisDB : RedisDB 객체 할당 및 반환
func NewRedisDB(cf *conf.Config, root *Repositories) (IRepository, error) {
	redisOption := redis.Options{
		Addr:         cf.DB["rdb"]["host"].(string),
		DB:           0,
		PoolSize:     30,              // 연결 풀 크기 추가
		MinIdleConns: 2,               // 최소 유휴 연결
		MaxRetries:   3,               // 재시도 횟수
		DialTimeout:  5 * time.Second, // 연결 타임아웃
		ReadTimeout:  3 * time.Second, // 읽기 타임아웃
		WriteTimeout: 3 * time.Second, // 쓰기 타임아웃
	}

	if strings.EqualFold(cf.Server.Mode, "prod") {
		redisOption = redis.Options{
			Addr:     cf.DB["rdb"]["host"].(string),
			Username: cf.DB["rdb"]["user"].(string),
			Password: cf.DB["rdb"]["pass"].(string), // no password set
			// DB:        0,                             // use default DB
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		redisOption = redis.Options{
			Addr:     cf.DB["rdb"]["host"].(string),
			Password: "qwer", // no password set
			// DB:   0, // use default DB
		}
		redisOption.TLSConfig = nil
	}

	client := redis.NewClient(&redisOption)

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	r := &RedisDB{
		client:      client,
		cfg:         cf,
		modName:     cf.Server.Name,
		ctx:         context.Background(),
		monitorQuit: make(chan struct{}),
	}
	r.healthy.Store(true)

	if strings.EqualFold(r.cfg.Server.HCheck, "redis") {
		r.monitorQuit = make(chan struct{})
		go r.healthMonitor()
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
	return p.client.Ping(context.Background()).Err()
}

func (r *RedisDB) SetCache(key, data string) error {

	if err := r.client.Set(context.Background(), key, data, time.Duration(30)*time.Minute).Err(); err != nil {
		return err
	}
	return nil
}

func (r *RedisDB) SetCacheMintime(key, data string, mintime int64) error {
	if err := r.client.Set(context.Background(), key, data, time.Duration(mintime)*time.Minute).Err(); err != nil {
		return err
	}
	return nil
}

func (r *RedisDB) SetBytes(key string, data []byte) error {
	// if err := r.client.Set(context.Background(), key, hexutil.Encode(data), 3600e9).Err(); err != nil {
	// return err
	// }
	return nil
}

func (r *RedisDB) GetCache(key string) (string, error) {
	// redisKey := `auth_` + platform + `_email_` + openID
	email, err := r.client.Get(context.Background(), key).Result()
	if err != nil {
		return "", err
	}
	return email, nil
}

func (r *RedisDB) DeleteCache(key string) error {

	if err := r.client.Del(context.Background(), key).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) IncCount(key string) error {
	if err := r.client.Incr(context.Background(), key).Err(); err != nil && err != redis.Nil {
		return err
	}

	if err := r.client.Expire(context.Background(), key, time.Duration(30)*time.Minute).Err(); err != nil {
		return err
	}
	return nil
}

func (r *RedisDB) SetNXCache(key, data string) error {
	if err := r.client.SetNX(context.Background(), key, data, 3600e9).Err(); err != nil {
		return err
	}
	return nil
}

/*
func (r *RedisDB) HSetMember(arPair []string) error {
	if err := r.client.HSet(context.Background(), "member", arPair).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) HGetMember(key string) string {
	res := r.client.HGet(context.Background(), "member", key)
	return res.Val()
} */

/*
func (r *RedisDB) HSetAccess(key, sid, uid string) error {
	now := time.Now()
	session := JWTSession{
		LoginTime: now,
		ExpireAt:  now.Add(24 * time.Hour),
		SID:       sid,
		UID:       uid,
		Nick:      "test",
		Gender:    "M",
		Age:       20,
		Area:      "Seoul",
		Email:     "test@test.com",
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return err
	}

	if err := r.client.HSet(context.Background(), "access", []string{key, string(sessionJSON)}).Err(); err != nil {
		return err
	}

	return nil
} */
/*
func (r *RedisDB) HGetAccess(key string) *JWTSession {
	res := r.client.HGet(context.Background(), "access", key)

	var session JWTSession
	err := json.Unmarshal([]byte(res.Val()), &session)
	if err != nil {
		return nil
	}
	return &session
}
*/

/*
func (r *RedisDB) HGetAllAddress() ([]string, error) {
	res, err := r.client.HKeys(context.Background(), "gember").Result()
	return res, err
}

func (r *RedisDB) HDelMember(key string) error {
	if err := r.client.HDel(context.Background(), "gember", key).Err(); err != nil {
		return err
	}
	return nil
}
*/
// SetJWTToken JWT 토큰과 세션 정보를 HSET에 저장

func (r *RedisDB) HSetJWTAccess(token string, userInfo *ptl.UserInfoResp) error {
	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	options := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  86400 * 7, //sec //1day
	}

	if err := r.client.HSetEXWithArgs(r.ctx, "AUTH:ACCESS", options, token, string(userInfoJSON)).Err(); err != nil {
		return err
	}

	return nil
}

// ValidateJWTToken JWT 토큰 유효성 검증 (간단 버전)
func (r *RedisDB) HGetJWTAccess(token string) (*ptl.UserInfoResp, error) {
	session, err := r.client.HGet(r.ctx, "AUTH:ACCESS", token).Result()
	if err != nil {
		return nil, err
	}

	var userInfo ptl.UserInfoResp
	err = json.Unmarshal([]byte(session), &userInfo)
	if err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// SetJWTToken JWT 토큰과 세션 정보를 HSET에 저장
func (r *RedisDB) HSetJWTRefresh(token string, userInfo *ptl.UserInfoResp) error {
	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	options := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  86400 * 14, //sec //14day
	}

	if err := r.client.HSetEXWithArgs(r.ctx, "AUTH:REFRESH", options, token, string(userInfoJSON)).Err(); err != nil {
		return err
	}

	return nil
}

// ValidateJWTToken JWT 토큰 유효성 검증 (간단 버전)
func (r *RedisDB) HGetJWTRefresh(token string) (*ptl.UserInfoResp, error) {
	session, err := r.client.HGet(r.ctx, "AUTH:REFRESH", token).Result()
	if err != nil {
		return nil, err
	}

	var userInfo ptl.UserInfoResp
	err = json.Unmarshal([]byte(session), &userInfo)
	if err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (r *RedisDB) DeleteJWTRefreshToken(token string) error {
	if token == "" {
		return nil
	}
	return r.client.HDel(r.ctx, "AUTH:REFRESH", token).Err()
}

func (r *RedisDB) RotateJWTToken(oldRefreshToken, newAccessToken, newRefreshToken string, userInfo *ptl.UserInfoResp) error {
	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	pipe := r.client.TxPipeline()
	if oldRefreshToken != "" {
		pipe.HDel(r.ctx, "AUTH:REFRESH", oldRefreshToken)
	}

	accessOptions := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  86400 * 7,
	}
	pipe.HSetEXWithArgs(r.ctx, "AUTH:ACCESS", accessOptions, newAccessToken, string(userInfoJSON))

	refreshOptions := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  86400 * 14,
	}
	pipe.HSetEXWithArgs(r.ctx, "AUTH:REFRESH", refreshOptions, newRefreshToken, string(userInfoJSON))

	_, err = pipe.Exec(r.ctx)
	return err
}

// SetJWTToken JWT 토큰과 세션 정보를 HSET에 저장
func (r *RedisDB) HSetOTP(email, otp string) error {
	options := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  600, //sec //1day
	}

	if err := r.client.HSetEXWithArgs(r.ctx, "AUTH:OTP", options, email, otp).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) HGetOTP(email string) (string, error) {
	otp, err := r.client.HGet(r.ctx, "AUTH:OTP", email).Result()
	if err != nil {
		return "", err
	}
	return otp, nil
}

// DeleteJWTToken JWT 토큰 삭제 (로그아웃 시)
func (r *RedisDB) DeleteJWTToken(token string) error {
	// 먼저 세션 정보를 가져와서 사용자 ID 확인
	session, err := r.HGetJWTAccess(token)
	if err != nil {
		// 토큰이 없어도 삭제 성공으로 처리
		return nil
	}

	pipe := r.client.Pipeline()

	// 1. JWT 세션 삭제
	pipe.HDel(r.ctx, "AUTH:ACCESS", token)

	// 2. 사용자 활성 토큰 목록에서 제거
	// pipe.SRem(r.ctx, fmt.Sprintf("user:%s:active_tokens", session), token)
	pipe.SRem(r.ctx, fmt.Sprintf("user:%d:active_tokens", session.Uid), token)
	_, err = pipe.Exec(r.ctx)
	return err
}

func (r *RedisDB) HSetUserInfo(user *ptl.UserInfoResp) error {
	/*
		now := time.Now()
		session := JWTSession{
			LoginTime: now,
			ExpireAt:  now.Add(24 * time.Hour),
			SID:       user.ID,
			UID:       user.Uid,
			Nick:      user.Nick,
			Gender:    user.Gender,
			Age:       user.Age,
			Area:      user.Area,
			Email:     "test@test.com",
		}
	*/

	sessionJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("Cupitok-%d-Gateway", user.Uid)
	encSessionJSON, err := utils.EncryptChaCha20(string(sessionJSON), key)
	if err != nil {
		return fmt.Errorf("error decrypting email: %v", err)
	}

	userIDStr := strconv.FormatUint(user.Uid, 10)
	if err := r.client.HSet(context.Background(), "USER:INFO", []string{userIDStr, encSessionJSON}).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) HSetJoinWTRoom(userID uint64, user *ptl.WTRoomUser) error {
	options := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  86400 * 1, //sec //1day
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}

	userIDStr := strconv.FormatUint(userID, 10)
	if err := r.client.HSetEXWithArgs(r.ctx, "WTRoom:User", options, userIDStr, string(userJSON)).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) HRefreshJoinWTRoom(userID uint64) error {
	userIDStr := strconv.FormatUint(userID, 10)

	// Check if the user exists in the hash
	exists, err := r.client.HExists(r.ctx, "WTRoom:User", userIDStr).Result()
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("user %d not found in WTRoom", userID)
	}

	// Get the current user data
	userJSON, err := r.client.HGet(r.ctx, "WTRoom:User", userIDStr).Result()
	if err != nil {
		return err
	}

	// Re-set the field with new expiration time
	options := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  86400 * 1, //sec //1day
	}

	if err := r.client.HSetEXWithArgs(r.ctx, "WTRoom:User", options, userIDStr, userJSON).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) HGetWTRoomUser(userID uint64) (*ptl.WTRoomUser, error) {
	userIDStr := strconv.FormatUint(userID, 10)
	res, err := r.client.HGet(r.ctx, "WTRoom:User", userIDStr).Result()
	if err != nil {
		return nil, err
	}

	var user ptl.WTRoomUser
	err = json.Unmarshal([]byte(res), &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *RedisDB) HGetJoinWTRoomList() (*[]ptl.WTRoomUser, error) {
	res, err := r.client.HGetAll(r.ctx, "WTRoom:User").Result()
	if err != nil {
		return nil, err
	}

	users := make([]ptl.WTRoomUser, 0, len(res))
	for _, userJSON := range res {
		var user ptl.WTRoomUser
		err = json.Unmarshal([]byte(userJSON), &user)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return &users, nil
}

func (r *RedisDB) HGetJoinWTRoomPre7List() (*[]ptl.WTRoomUser, error) {
	res, err := r.client.HGetAll(r.ctx, "WTRoom:User").Result()
	if err != nil {
		return nil, err
	}

	i := 0
	users := make([]ptl.WTRoomUser, 0, 7)
	for _, userJSON := range res {
		if i >= 7 {
			break
		}
		var user ptl.WTRoomUser
		err = json.Unmarshal([]byte(userJSON), &user)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
		i += 1
	}

	return &users, nil
}

func (r *RedisDB) HDeleteJoinWTRoom(userID uint64) error {
	userIDStr := strconv.FormatUint(userID, 10)
	if err := r.client.HDel(r.ctx, "WTRoom:User", userIDStr).Err(); err != nil {
		return err
	}
	return nil
}

// User 사용자 정보 구조체
type User struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Area   string `json:"area"`
	Gender string `json:"gender"`
}

// GetUser 사용자 정보 조회
func (r *RedisDB) GetUser(userID string) (*User, error) {
	userJSON, err := r.client.Get(r.ctx, fmt.Sprintf("user:%s:info", userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	var user User
	err = json.Unmarshal([]byte(userJSON), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUserConnectionStatus 사용자 연결 상태 업데이트
func (db *RedisDB) UpdateUserConnectionStatus(userID string, isConnected bool) error {
	key := fmt.Sprintf("user:%s:connected", userID)
	value := strconv.FormatBool(isConnected)

	return db.client.Set(db.ctx, key, value, 0).Err()
}

// IsUserConnected 사용자 연결 상태 확인
func (db *RedisDB) IsUserConnected(userID string) (bool, error) {
	key := fmt.Sprintf("user:%s:connected", userID)
	val, err := db.client.Get(db.ctx, key).Result()

	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return val == "true", nil
}

// GetJWTSession JWT 토큰으로 세션 정보 조회
func (r *RedisDB) GetUserInfo(uid string) (*ptl.UserInfoResp, error) {
	sessionJSON, err := r.client.HGet(r.ctx, "USER:INFO", uid).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("jwt token not found")
		}
		return nil, err
	}

	var session ptl.UserInfoResp
	key := fmt.Sprintf("Cupitok-%s-Gateway", uid)
	decSessionJSON, err := utils.DecryptChaCha20(sessionJSON, key)
	if err != nil {
		return nil, fmt.Errorf("error decrypting session: %v", err)
	}

	err = json.Unmarshal([]byte(decSessionJSON), &session)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// DeleteAllUserTokens 특정 사용자의 모든 토큰 삭제
func (r *RedisDB) DeleteAllUserTokens(userID string) error {
	// 사용자의 모든 활성 토큰 조회
	tokens, err := r.client.SMembers(r.ctx, fmt.Sprintf("user:%s:active_tokens", userID)).Result()
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	// 각 토큰 삭제
	for _, token := range tokens {
		pipe.HDel(r.ctx, "jwt:sessions", token)
	}

	// 사용자 활성 토큰 목록 삭제
	pipe.Del(r.ctx, fmt.Sprintf("user:%s:active_tokens", userID))

	_, err = pipe.Exec(r.ctx)
	return err
}

// GetUnreadTotalCount 사용자의 전체 읽지 않은 메시지 수 조회
func (r *RedisDB) GetUnreadTotalCount(userID string) (int, error) {
	// 모든 룸 정보 조회
	roomsInfo, err := r.client.HGetAll(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID)).Result()
	if err != nil {
		return 0, err
	}

	totalUnread := 0
	for _, roomJSON := range roomsInfo {
		var roomItem ChatRoomListItem
		if err := json.Unmarshal([]byte(roomJSON), &roomItem); err != nil {
			continue
		}
		totalUnread += roomItem.UnreadCount
	}

	return totalUnread, nil
}

// incrUnreadLua DM 미읽음 INCR + companion 시각 SET + 양 키 EXPIRE (원자 실행)
var incrUnreadLua = redis.NewScript(`
local ttl = tonumber(ARGV[1])
local n = redis.call('INCR', KEYS[1])
redis.call('SET', KEYS[2], ARGV[2])
redis.call('EXPIRE', KEYS[1], ttl)
redis.call('EXPIRE', KEYS[2], ttl)
return n
`)

func dmUnreadKey(uid uint64, roomID int64) string {
	return fmt.Sprintf("DM:UNREAD:%d:%d", uid, roomID)
}

// dmUnreadTSKey IncrUnread 시각(Unix 초) 저장용 companion — 카운터와 동일 TTL
func dmUnreadTSKey(uid uint64, roomID int64) string {
	return dmUnreadKey(uid, roomID) + ":ts"
}

const dmUnreadKeyTTL = 30 * 24 * time.Hour

func dmOnlineKey(uid uint64) string {
	return fmt.Sprintf("DM:ONLINE:%d", uid)
}

// IncrUnread 읽지 않은 메시지 수 증가 (Lua: 카운터·companion 30일 TTL 매 호출 갱신)
func (r *RedisDB) IncrUnread(uid uint64, roomID int64) (int64, error) {
	counterKey := dmUnreadKey(uid, roomID)
	tsKey := dmUnreadTSKey(uid, roomID)
	ttlSec := int64(dmUnreadKeyTTL.Seconds())
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	return incrUnreadLua.Run(r.ctx, r.client, []string{counterKey, tsKey}, ttlSec, ts).Int64()
}

// GetUnread 읽지 않은 메시지 수 조회
func (r *RedisDB) GetUnread(uid uint64, roomID int64) (int64, error) {
	val, err := r.client.Get(r.ctx, dmUnreadKey(uid, roomID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// ResetUnread 읽지 않은 메시지 수 초기화 (카운터·companion 동시 삭제)
func (r *RedisDB) ResetUnread(uid uint64, roomID int64) error {
	return r.client.Del(r.ctx, dmUnreadKey(uid, roomID), dmUnreadTSKey(uid, roomID)).Err()
}

// GetAllUnreadForUser 사용자의 전체 unread 맵 조회
func (r *RedisDB) GetAllUnreadForUser(uid uint64) (map[string]int64, error) {
	pattern := fmt.Sprintf("DM:UNREAD:%d:*", uid)
	cursor := uint64(0)
	result := make(map[string]int64)

	for {
		keys, nextCursor, err := r.client.Scan(r.ctx, cursor, pattern, 200).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			if strings.HasSuffix(key, ":ts") {
				continue
			}
			val, getErr := r.client.Get(r.ctx, key).Result()
			if getErr != nil {
				if errors.Is(getErr, redis.Nil) {
					continue
				}
				return nil, getErr
			}

			cnt, parseErr := strconv.ParseInt(val, 10, 64)
			if parseErr != nil {
				continue
			}

			prefix := fmt.Sprintf("DM:UNREAD:%d:", uid)
			roomID := strings.TrimPrefix(key, prefix)
			if roomID != "" {
				result[roomID] = cnt
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return result, nil
}

// SetOnline DM 온라인 상태 설정
func (r *RedisDB) SetOnline(uid uint64) error {
	return r.client.Set(r.ctx, dmOnlineKey(uid), "1", 90*time.Second).Err()
}

// IsOnline DM 온라인 상태 조회
func (r *RedisDB) IsOnline(uid uint64) (bool, error) {
	exists, err := r.client.Exists(r.ctx, dmOnlineKey(uid)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// DeleteOnline DM 온라인 상태 삭제
func (r *RedisDB) DeleteOnline(uid uint64) error {
	return r.client.Del(r.ctx, dmOnlineKey(uid)).Err()
}

// GetWebRTCConfig 사용자 로그인 시 WebRTC 설정 정보 반환
func (r *RedisDB) GetWebRTCConfig(userID uint64) (*ptl.WebRTCConfig, error) {
	// TODO: 실제 구현에서는 conf.Config를 받아와야 함
	// 현재는 하드코딩된 기본값 사용

	// 기본 STUN 서버 구성 (Google STUN 서버들)
	baseStunServers := []string{
		"stun:stun.l.google.com:19302",
		"stun:stun1.l.google.com:19302",
		"stun:stun2.l.google.com:19302",
		"stun:stun3.l.google.com:19302",
		"stun:stun4.l.google.com:19302",
	}

	// ICE 서버 구성
	iceServers := make([]ptl.ICEServer, 0, len(baseStunServers)+1)

	// STUN 서버들 추가
	for _, stunUrl := range baseStunServers {
		iceServers = append(iceServers, ptl.ICEServer{
			URLs: []string{stunUrl},
			Type: "stun",
		})
	}

	config := &ptl.WebRTCConfig{
		ICEServers:      iceServers,
		SignalingServer: "ws://localhost:8080/ws", // TODO: 환경별 동적 설정 필요
		StunServers:     baseStunServers,
		MediaSettings: ptl.MediaConfig{
			Video: ptl.VideoConfig{
				Enabled: true,
				// Width:   1920,
				// Height:  1080,
				Width:      1280,
				Height:     720,
				FrameRate:  30,
				MaxBitrate: 2000000, // 2Mbps
				// MaxBitrate: 8000000, // 2Mbps
			},
			Audio: ptl.AudioConfig{
				Enabled:          true,
				EchoCancellation: true,
				NoiseSuppression: true,
				AutoGainControl:  true,
				MaxBitrate:       96000, // 96kbps
				// MaxBitrate:       128000, // 128kbps
			},
		},
	}

	/*
		MediaSettings: MediaConfig{
			Video: VideoConfig{
				Enabled:    true,
				Width:      854,    // 480p 해상도
				Height:	    480,     // 480p 해상도
				FrameRate:  24,     // 24fps (영화 표준, 데이터 절약)
				MaxBitrate: 1500000, // 1.5Mbps
			},
			Audio: AudioConfig{
				Enabled:          true,
				EchoCancellation: true,
				NoiseSuppression: true,
				AutoGainControl:  true,
				MaxBitrate:       64000,  // 64kbps (전화 품질 수준)
			},
		}
	*/
	return config, nil
}

/* SetChatRoom 채팅방 정보 저장
// SetChatRoom HSet을 사용하여 채팅방 정보 저장
func (db *RedisDB) SetChatRoom(roomID, roomName, ownerID string, isPrivate bool) error {
	room := ChatRoomData{
		ID:           roomID,
		Name:         roomName,
		OwnerID:      ownerID,
		IsPrivate:    isPrivate,
		CreatedAt:    time.Now(),
		IsActive:     true,
		LastActive:   time.Now(),
		Participants: []string{ownerID},
	}

	roomJSON, err := json.Marshal(room)
	if err != nil {
		return err
	}

	// HSet을 사용하여 채팅방 정보 저장
	err = db.client.HSet(db.ctx, "chat:rooms", roomID, roomJSON).Err()
	if err != nil {
		return err
	}

	// 사용자-채팅방 매핑 저장
	err = db.client.Set(db.ctx, fmt.Sprintf("user:%s:room", ownerID), roomID, 0).Err()
	if err != nil {
		return err
	}

	// 활성 채팅방 목록에 추가
	err = db.client.SAdd(db.ctx, "chat:active_rooms", roomID).Err()
	if err != nil {
		return err
	}

	log.Info("채팅방 생성: ", roomID, " (", roomName, ")")
	return nil
}
*/

/* GetChatRoom 채팅방 정보 조회
// GetChatRoom HGet을 사용하여 채팅방 정보 조회
func (db *RedisDB) GetChatRoom(roomID string) (*ChatRoomData, error) {
	roomJSON, err := db.client.HGet(db.ctx, "chat:rooms", roomID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("chat room not found")
		}
		return nil, err
	}

	var room ChatRoomData
	err = json.Unmarshal([]byte(roomJSON), &room)
	if err != nil {
		return nil, err
	}

	return &room, nil
}
*/

/* GetChatRooms 모든 채팅방 정보 조회
// GetChatRooms HGetAll을 사용하여 모든 채팅방 정보 조회
func (db *RedisDB) GetChatRooms() ([]ChatRoomData, error) {
	roomsMap, err := db.client.HGetAll(db.ctx, "chat:rooms").Result()
	if err != nil {
		return nil, err
	}

	rooms := make([]ChatRoomData, 0, len(roomsMap))
	for _, roomJSON := range roomsMap {
		var room ChatRoomData
		err = json.Unmarshal([]byte(roomJSON), &room)
		if err != nil {
			continue
		}

		// 활성 상태인 채팅방만 반환
		if room.IsActive {
			rooms = append(rooms, room)
		}
	}

	// 생성 시간 순으로 정렬
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].CreatedAt.After(rooms[j].CreatedAt)
	})

	return rooms, nil
}
*/

// SaveChatMessage 채팅 메시지 저장
// SaveChatMessage HSet을 사용하여 채팅 메시지 저장
func (r *RedisDB) SaveChatMessage(msg *ptl.ChatMessage) error {
	// messageID := utils.GenUuid()
	// messageID, err := utils.Gen6DigitCode()
	// if err != nil {
	// 	return err
	// }
	// now := time.Now()

	message := ChatMessageData{
		ID:        msg.MsgID,
		RoomID:    msg.RoomID,
		UserID:    msg.From,
		Content:   msg.Content,
		Type:      msg.Type,
		Timestamp: time.Unix(msg.Timestamp, 0),
	}

	// 메시지 JSON으로 변환
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Redis key 구성: 채팅방별 메시지 리스트
	// 요청 정책: chat:rooms:{roomID} 키에 메시지 내역 저장
	listKey := fmt.Sprintf("chat:rooms:%s:msg", msg.RoomID)

	// 트랜잭션 사용하여 저장 및 TTL 설정
	pipe := r.client.TxPipeline()

	// 1. 리스트에 시간 순서대로 메시지 추가 (최신 메시지가 뒤로)
	pipe.RPush(r.ctx, listKey, messageJSON)

	// 2. 리스트에 TTL 24시간(86400초) 설정 (기존에 값이 있으면 갱신만)
	pipe.Expire(r.ctx, listKey, 24*time.Hour)

	_, err = pipe.Exec(r.ctx)
	return err
}

// GetChatMessages 특정 채팅방의 메시지 조회
// 첫 반환값은 항상 해당 방 메시지 전체 개수(LLen). limit<=0·offset 범위 밖·메시지 없음이어도 동일.
func (r *RedisDB) GetChatMessages(roomID string, offset, limit int) (int64, *[]ChatMessageData, error) {
	listKey := fmt.Sprintf("chat:rooms:%s:msg", roomID)
	total, err := r.client.LLen(r.ctx, listKey).Result()
	if err != nil {
		return 0, nil, err
	}

	empty := []ChatMessageData{}
	emptyPtr := &empty

	if limit <= 0 {
		return total, emptyPtr, nil
	}
	if total == 0 || int64(offset) >= total {
		return total, emptyPtr, nil
	}

	// 최신 메시지 기준(offset=0) 페이징 유지
	start := total - int64(offset) - int64(limit)
	if start < 0 {
		start = 0
	}
	end := total - int64(offset) - 1

	rawMessages, err := r.client.LRange(r.ctx, listKey, start, end).Result()
	if err != nil {
		return total, nil, err
	}
	if len(rawMessages) == 0 {
		return total, emptyPtr, nil
	}

	// LRange 결과는 오래된 순이므로 최신순으로 뒤집는다.
	tCnt := int64(len(rawMessages))
	messages := make([]ChatMessageData, 0, tCnt)
	for i := tCnt - 1; i >= 0; i-- {
		messageJSON := rawMessages[i]

		var message ChatMessageData
		if err := json.Unmarshal([]byte(messageJSON), &message); err != nil {
			continue
		}

		messages = append(messages, message)
	}

	return total, &messages, nil
}

// findMessageListIndex returns the Redis LIST index (0=oldest) for message id.
func (r *RedisDB) findMessageListIndex(listKey, messageID string, total int64) (int64, bool, error) {
	if messageID == "" || total == 0 {
		return -1, false, nil
	}
	rawMessages, err := r.client.LRange(r.ctx, listKey, 0, total-1).Result()
	if err != nil {
		return -1, false, err
	}
	for i, messageJSON := range rawMessages {
		var message ChatMessageData
		if err := json.Unmarshal([]byte(messageJSON), &message); err != nil {
			continue
		}
		if message.ID == messageID {
			return int64(i), true, nil
		}
	}
	return -1, false, nil
}

// parseChatMessagesRaw parses LRange result (oldest→newest) into newest-first slice.
func parseChatMessagesRaw(rawMessages []string) []ChatMessageData {
	tCnt := int64(len(rawMessages))
	messages := make([]ChatMessageData, 0, tCnt)
	for i := tCnt - 1; i >= 0; i-- {
		var message ChatMessageData
		if err := json.Unmarshal([]byte(rawMessages[i]), &message); err != nil {
			continue
		}
		messages = append(messages, message)
	}
	return messages
}

// GetChatMessagesByCursor cursor 기반 메시지 조회 (최신순 반환).
// cursor == "" → 최신 limit건. cursor != "" → 해당 메시지보다 오래된 limit건.
// nextCursor는 이번 배치에서 가장 오래된 메시지 id (다음 요청 cursor로 사용).
func (r *RedisDB) GetChatMessagesByCursor(roomID, cursor string, limit int) (int64, []ChatMessageData, string, bool, error) {
	listKey := fmt.Sprintf("chat:rooms:%s:msg", roomID)
	total, err := r.client.LLen(r.ctx, listKey).Result()
	if err != nil {
		return 0, nil, "", false, err
	}

	empty := []ChatMessageData{}
	if limit <= 0 {
		return total, empty, "", false, nil
	}
	if total == 0 {
		return total, empty, "", false, nil
	}

	var endIdx int64 = total - 1
	if cursor != "" {
		cursorIdx, found, err := r.findMessageListIndex(listKey, cursor, total)
		if err != nil {
			return total, nil, "", false, err
		}
		if !found {
			return total, nil, "", false, fmt.Errorf("invalid cursor: %s", cursor)
		}
		endIdx = cursorIdx - 1
		if endIdx < 0 {
			return total, empty, "", false, nil
		}
	}

	startIdx := endIdx - int64(limit) + 1
	if startIdx < 0 {
		startIdx = 0
	}

	rawMessages, err := r.client.LRange(r.ctx, listKey, startIdx, endIdx).Result()
	if err != nil {
		return total, nil, "", false, err
	}
	if len(rawMessages) == 0 {
		return total, empty, "", false, nil
	}

	messages := parseChatMessagesRaw(rawMessages)
	nextCursor := messages[len(messages)-1].ID
	hasMore := startIdx > 0
	return total, messages, nextCursor, hasMore, nil
}

// PruneExpiredRoomMessages 채팅방별 만료 메시지 정리
// 정책: chat:rooms:{roomID}:msg LIST에서 ttl 기준 이전 메시지 삭제
func (r *RedisDB) Expired24hMsg(ttl time.Duration) (*ChatPruneResult, error) {
	if ttl <= 0 {
		return nil, fmt.Errorf("ttl must be greater than zero")
	}

	cutoff := time.Now().Add(-ttl)
	pattern := "chat:rooms:*:msg"
	var cursor uint64
	result := &ChatPruneResult{}

	for {
		keys, nextCursor, err := r.client.Scan(r.ctx, cursor, pattern, 200).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			result.ScannedRooms++

			rawMessages, lErr := r.client.LRange(r.ctx, key, 0, -1).Result()
			if lErr != nil || len(rawMessages) == 0 {
				continue
			}

			kept := make([]interface{}, 0, len(rawMessages))
			deletedInRoom := 0

			for _, raw := range rawMessages {
				var msg ChatMessageData
				if uErr := json.Unmarshal([]byte(raw), &msg); uErr != nil {
					// 파싱 실패 데이터는 유실 방지를 위해 보존
					kept = append(kept, raw)
					continue
				}

				if msg.Timestamp.Before(cutoff) {
					deletedInRoom++
					continue
				}
				kept = append(kept, raw)
			}

			if deletedInRoom == 0 {
				continue
			}

			pipe := r.client.TxPipeline()
			pipe.Del(r.ctx, key)
			if len(kept) > 0 {
				pipe.RPush(r.ctx, key, kept...)
				pipe.Expire(r.ctx, key, ttl)
			}

			if _, pErr := pipe.Exec(r.ctx); pErr != nil {
				log.Warn("failed to prune key %s: %v", key, pErr)
				continue
			}

			result.DeletedMsgs += deletedInRoom
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return result, nil
}

/* ActivateChatRoom 채팅방 활성화
// ActivateChatRoom 채팅방 활성화
func (r *RedisDB) ActivateChatRoom(roomID string) error {
	roomData, err := r.GetChatRoom(roomID)
	if err != nil {
		return err
	}

	// 활성화 상태로 변경
	roomData.IsActive = true
	roomData.LastActive = time.Now()

	roomJSON, err := json.Marshal(roomData)
	if err != nil {
		return err
	}

	// HSet으로 업데이트
	pipe := r.client.Pipeline()
	pipe.HSet(r.ctx, "chat:rooms", roomID, roomJSON)
	pipe.SAdd(r.ctx, "chat:active_rooms", roomID)
	_, err = pipe.Exec(r.ctx)

	return err
}

// DeactivateChatRoom 채팅방 비활성화
func (r *RedisDB) DeactivateChatRoom(roomID string) error {
	roomData, err := r.GetChatRoom(roomID)
	if err != nil {
		return err
	}

	// 비활성화 상태로 변경
	roomData.IsActive = false

	roomJSON, err := json.Marshal(roomData)
	if err != nil {
		return err
	}

	// HSet으로 업데이트
	pipe := r.client.Pipeline()
	pipe.HSet(r.ctx, "chat:rooms", roomID, roomJSON)
	pipe.SRem(r.ctx, "chat:active_rooms", roomID)
	_, err = pipe.Exec(r.ctx)

	return err
}

// GetUserChatRoom 사용자의 채팅방 ID 조회
func (r *RedisDB) GetUserChatRoom(userID string) (string, error) {
	roomID, err := r.client.Get(r.ctx, fmt.Sprintf("user:%s:room", userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.New("user has no chat room")
		}
		return "", err
	}

	return roomID, nil
}

// DeleteChatRoom 채팅방 삭제 (사용자가 소유자인 경우에만)
func (r *RedisDB) DeleteChatRoom(roomID, userID string) error {
	roomData, err := db.GetChatRoom(roomID)
	if err != nil {
		return err
	}

	// 사용자가 채팅방 소유자인지 확인
	if roomData.OwnerID != userID {
		return errors.New("user is not the owner of the chat room")
	}

	// 트랜잭션으로 채팅방 관련 데이터 삭제
	pipe := db.client.Pipeline()

	// 1. 채팅방 정보 삭제
	pipe.HDel(db.ctx, "chat:rooms", roomID)

	// 2. 활성 채팅방 목록에서 제거
	pipe.SRem(db.ctx, "chat:active_rooms", roomID)

	// 3. 채팅방 메시지 및 메시지 타임스탬프 삭제
	pipe.Del(db.ctx, fmt.Sprintf("chat:room:%s:messages", roomID))
	pipe.Del(db.ctx, fmt.Sprintf("chat:room:%s:message_times", roomID))

	// 4. 사용자-채팅방 매핑 삭제
	pipe.Del(db.ctx, fmt.Sprintf("user:%s:room", userID))

	// 5. 채팅방 참여자 목록 삭제
	for _, participant := range roomData.Participants {
		pipe.Del(db.ctx, fmt.Sprintf("user:%s:room", participant))
	}

	_, err = pipe.Exec(db.ctx)
	return err
}

// ListActiveChatRooms 활성화된 채팅방 목록 조회
func (r *RedisDB) ListActiveChatRooms() ([]ChatRoomData, error) {
	// 활성 채팅방 ID 목록 가져오기
	roomIDs, err := db.client.SMembers(db.ctx, "chat:active_rooms").Result()
	if err != nil {
		return nil, err
	}

	if len(roomIDs) == 0 {
		return []ChatRoomData{}, nil
	}

	// 각 채팅방 정보 조회
	pipe := db.client.Pipeline()
	for _, roomID := range roomIDs {
		pipe.HGet(db.ctx, "chat:rooms", roomID)
	}

	cmds, err := pipe.Exec(db.ctx)
	if err != nil {
		return nil, err
	}

	// 결과 처리
	rooms := make([]ChatRoomData, 0, len(cmds))
	for _, cmd := range cmds {
		hgetCmd := cmd.(*redis.StringCmd)
		roomJSON, err := hgetCmd.Result()
		if err != nil {
			continue
		}

		var room ChatRoomData
		if err := json.Unmarshal([]byte(roomJSON), &room); err != nil {
			continue
		}

		if room.IsActive {
			rooms = append(rooms, room)
		}
	}

	// 최근 활동 순으로 정렬
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].LastActive.After(rooms[j].LastActive)
	})

	return rooms, nil
}
*/

/* AddUserToChatRoom 사용자를 채팅방에 추가
// AddUserToChatRoom 사용자를 채팅방에 추가
func (r *RedisDB) AddUserToChatRoom(roomID, userID string) error {
	roomData, err := db.GetChatRoom(roomID)
	if err != nil {
		return err
	}

	// 이미 참여 중인지 확인
	for _, participant := range roomData.Participants {
		if participant == userID {
			return nil // 이미 참여 중이면 성공으로 처리
		}
	}

	// 참여자 목록에 추가
	roomData.Participants = append(roomData.Participants, userID)
	roomData.LastActive = time.Now()

	roomJSON, err := json.Marshal(roomData)
	if err != nil {
		return err
	}

	// 채팅방 정보 업데이트
	return db.client.HSet(db.ctx, "chat:rooms", roomID, roomJSON).Err()
}
/*

/*
// RemoveUserFromChatRoom 사용자를 채팅방에서 제거
func (r *RedisDB) RemoveUserFromChatRoom(roomID, userID string) error {
	roomData, err := db.GetChatRoom(roomID)
	if err != nil {
		return err
	}

	// 참여자 목록에서 제거
	isFound := false
	newParticipants := make([]string, 0, len(roomData.Participants))

	for _, participant := range roomData.Participants {
		if participant != userID {
			newParticipants = append(newParticipants, participant)
		} else {
			isFound = true
		}
	}

	if !isFound {
		return errors.New("user is not in the chat room")
	}

	roomData.Participants = newParticipants
	roomData.LastActive = time.Now()

	roomJSON, err := json.Marshal(roomData)
	if err != nil {
		return err
	}

	// 채팅방 정보 업데이트
	return db.client.HSet(db.ctx, "chat:rooms", roomID, roomJSON).Err()
}
*/

/* ScanKeys Redis의 SCAN 명령어를 사용하여 지정된 패턴의 키를 검색합니다.
// ScanKeys Redis의 SCAN 명령어를 사용하여 지정된 패턴의 키를 검색합니다.
func (r *RedisDB) ScanKeys(cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	cmd := r.client.Scan(r.ctx, cursor, pattern, count)
	keys, cursor, err := cmd.Result()
	return keys, cursor, err
}
*/

/* SaveUser 사용자 정보 저장
// SaveUser 사용자 정보 저장
func (r *RedisDB) SaveUser(user User) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, fmt.Sprintf("user:%s:info", user.ID), userJSON, 0).Err()
}
*/

/* GetUserActiveTokens 사용자의 활성 토큰 목록 조회
func (r *RedisDB) GetUserActiveTokens(userID string) ([]string, error) {
	return r.client.SMembers(r.ctx, fmt.Sprintf("user:%s:active_tokens", userID)).Result()
}
*/

/* // CleanupExpiredTokens 만료된 토큰 정리 (스케줄러에서 호출)
func (r *RedisDB) CleanupExpiredTokens() error {
	// 모든 JWT 세션 조회
	sessions, err := r.client.HGetAll(r.ctx, "jwt:sessions").Result()
	if err != nil {
		return err
	}

	now := time.Now()
	pipe := r.client.Pipeline()

	for token, sessionJSON := range sessions {
		var session JWTSession
		if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
			continue
		}

		// 만료된 토큰 삭제
		if now.After(session.ExpireAt) {
			pipe.HDel(r.ctx, "jwt:sessions", token)
			pipe.SRem(r.ctx, fmt.Sprintf("user:%s:active_tokens", session.UID), token)
		}
	}

	_, err = pipe.Exec(r.ctx)
	return err
}
*/

/* AddUserToChatRoomList 사용자의 채팅룸 리스트에 룸 추가 (로그인/접속 시)
// AddUserToChatRoomList 사용자의 채팅룸 리스트에 룸 추가 (로그인/접속 시)
func (r *RedisDB) AddUserToChatRoomList(userID, roomID string, roomName string) error {
	now := time.Now()

	// 채팅룸 정보 생성
	roomItem := ChatRoomListItem{
		RoomID:       roomID,
		RoomName:     roomName,
		LastActivity: now,
		UnreadCount:  0,
		LastMessage:  "",
		Priority:     0,
	}

	roomJSON, err := json.Marshal(roomItem)
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()

	// 1. 사용자의 채팅룸 리스트에 추가 (ZSET 사용, 점수는 타임스탬프)
	pipe.ZAdd(r.ctx, fmt.Sprintf("user:%s:chatroom_list", userID), redis.Z{
		Score:  float64(now.Unix()),
		Member: roomID,
	})

	// 2. 채팅룸 상세 정보 저장 (HSET 사용)
	pipe.HSet(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID), roomID, roomJSON)

	_, err = pipe.Exec(r.ctx)
	return err
}
*/
/*
// MoveRoomToTop 특정 이벤트로 인해 채팅룸을 리스트 최상단으로 이동
func (r *RedisDB) MoveRoomToTop(userID, roomID string, eventType string, lastMessage string) error {
	now := time.Now()

	// 기존 룸 정보 조회
	roomJSON, err := r.client.HGet(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID), roomID).Result()
	if err != nil {
		if err == redis.Nil {
			// 룸이 없으면 새로 추가
			return r.AddUserToChatRoomList(userID, roomID, "Unknown Room")
		}
		return err
	}

	var roomItem ChatRoomListItem
	if err := json.Unmarshal([]byte(roomJSON), &roomItem); err != nil {
		return err
	}

	// 룸 정보 업데이트
	roomItem.LastActivity = now
	roomItem.LastMessage = lastMessage

	// 이벤트 타입에 따른 처리
	switch eventType {
	case "message":
		roomItem.UnreadCount++
		roomItem.Priority = 0
	case "mention":
		roomItem.UnreadCount++
		roomItem.Priority = 1
	case "urgent":
		roomItem.UnreadCount++
		roomItem.Priority = 2
	case "read":
		roomItem.UnreadCount = 0
		roomItem.Priority = 0
	}

	updatedRoomJSON, err := json.Marshal(roomItem)
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()

	// 1. ZSET에서 점수 업데이트 (현재 시간 + 우선순위 가중치)
	priorityBonus := float64(roomItem.Priority * 1000000000) // 우선순위 가중치
	newScore := float64(now.Unix()) + priorityBonus

	pipe.ZAdd(r.ctx, fmt.Sprintf("user:%s:chatroom_list", userID), redis.Z{
		Score:  newScore,
		Member: roomID,
	})

	// 2. 룸 정보 업데이트
	pipe.HSet(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID), roomID, updatedRoomJSON)

	_, err = pipe.Exec(r.ctx)
	return err
}
*/
/*
// GetUserChatRoomList 사용자의 채팅룸 리스트 조회 (최신 순으로 정렬)
func (r *RedisDB) GetUserChatRoomList(userID string, offset, limit int) ([]ChatRoomListItem, error) {
	// ZSET에서 점수가 높은 순(최신순)으로 룸 ID 조회
	roomIDs, err := r.client.ZRevRange(
		r.ctx,
		fmt.Sprintf("user:%s:chatroom_list", userID),
		int64(offset),
		int64(offset+limit-1),
	).Result()

	if err != nil {
		return nil, err
	}

	if len(roomIDs) == 0 {
		return []ChatRoomListItem{}, nil
	}

	// 각 룸의 상세 정보 조회
	pipe := r.client.Pipeline()
	for _, roomID := range roomIDs {
		pipe.HGet(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID), roomID)
	}

	cmds, err := pipe.Exec(r.ctx)
	if err != nil {
		return nil, err
	}

	// 결과 처리
	rooms := make([]ChatRoomListItem, 0, len(cmds))
	for _, cmd := range cmds {
		hgetCmd := cmd.(*redis.StringCmd)
		roomJSON, err := hgetCmd.Result()
		if err != nil {
			continue
		}

		var roomItem ChatRoomListItem
		if err := json.Unmarshal([]byte(roomJSON), &roomItem); err != nil {
			continue
		}

		rooms = append(rooms, roomItem)
	}

	return rooms, nil
}
*/
/*
// GetUserChatRoomListWithScores 점수와 함께 채팅룸 리스트 조회 (디버깅용)
func (r *RedisDB) GetUserChatRoomListWithScores(userID string) (map[string]float64, error) {
	result, err := r.client.ZRevRangeWithScores(
		r.ctx,
		fmt.Sprintf("user:%s:chatroom_list", userID),
		0, -1,
	).Result()

	if err != nil {
		return nil, err
	}

	roomScores := make(map[string]float64)
	for _, z := range result {
		roomScores[z.Member.(string)] = z.Score
	}

	return roomScores, nil
}
*/
/*
// RemoveUserFromChatRoomList 사용자의 채팅룸 리스트에서 룸 제거
func (r *RedisDB) RemoveUserFromChatRoomList(userID, roomID string) error {
	pipe := r.client.Pipeline()

	// 1. ZSET에서 룸 제거
	pipe.ZRem(r.ctx, fmt.Sprintf("user:%s:chatroom_list", userID), roomID)

	// 2. 룸 상세 정보 삭제
	pipe.HDel(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID), roomID)

	_, err := pipe.Exec(r.ctx)
	return err
}
*/
/*
// UpdateRoomUnreadCount 채팅룸의 읽지 않은 메시지 수 업데이트
func (r *RedisDB) UpdateRoomUnreadCount(userID, roomID string, unreadCount int) error {
	// 기존 룸 정보 조회
	roomJSON, err := r.client.HGet(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID), roomID).Result()
	if err != nil {
		return err
	}

	var roomItem ChatRoomListItem
	if err := json.Unmarshal([]byte(roomJSON), &roomItem); err != nil {
		return err
	}

	// 읽지 않은 메시지 수 업데이트
	roomItem.UnreadCount = unreadCount

	updatedRoomJSON, err := json.Marshal(roomItem)
	if err != nil {
		return err
	}

	// 룸 정보 업데이트
	return r.client.HSet(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID), roomID, updatedRoomJSON).Err()
}
*/
/*
// GetChatRoomListCount 사용자의 채팅룸 리스트 총 개수 조회
func (r *RedisDB) GetChatRoomListCount(userID string) (int64, error) {
	return r.client.ZCard(r.ctx, fmt.Sprintf("user:%s:chatroom_list", userID)).Result()
}
*/

/* CleanupInactiveRooms 비활성 채팅룸 정리 (일정 기간 이상 활동이 없는 룸)
// CleanupInactiveRooms 비활성 채팅룸 정리 (일정 기간 이상 활동이 없는 룸)
func (r *RedisDB) CleanupInactiveRooms(userID string, inactiveDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -inactiveDays)
	cutoffScore := float64(cutoffTime.Unix())

	// 비활성 룸들 조회
	inactiveRooms, err := r.client.ZRangeByScore(
		r.ctx,
		fmt.Sprintf("user:%s:chatroom_list", userID),
		&redis.ZRangeBy{
			Min: "0",
			Max: fmt.Sprintf("%f", cutoffScore),
		},
	).Result()

	if err != nil || len(inactiveRooms) == 0 {
		return err
	}

	pipe := r.client.Pipeline()

	// 비활성 룸들 삭제
	for _, roomID := range inactiveRooms {
		pipe.ZRem(r.ctx, fmt.Sprintf("user:%s:chatroom_list", userID), roomID)
		pipe.HDel(r.ctx, fmt.Sprintf("user:%s:chatroom_info", userID), roomID)
	}

	_, err = pipe.Exec(r.ctx)
	return err
}
*/

/* SetUserWebRTCSession 사용자 WebRTC 세션 정보 저장
// SetUserWebRTCSession 사용자 WebRTC 세션 정보 저장
func (r *RedisDB) SetUserWebRTCSession(session UserWebRTCSession) error {
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()

	// 1. 사용자 WebRTC 세션 정보 저장
	pipe.HSet(r.ctx, "webrtc:sessions", session.UserID, sessionJSON)

	// 2. 세션 만료 시간 설정 (4시간)
	pipe.Expire(r.ctx, fmt.Sprintf("webrtc:session:%s", session.UserID), 4*time.Hour)

	// 3. 활성 사용자 목록 업데이트 (통화 가능한 사용자)
	if session.ConnectionState == "connected" && !session.IsInCall {
		pipe.SAdd(r.ctx, "webrtc:available_users", session.UserID)
	} else {
		pipe.SRem(r.ctx, "webrtc:available_users", session.UserID)
	}

	_, err = pipe.Exec(r.ctx)
	return err
}
*/

/* GetUserWebRTCSession 사용자 WebRTC 세션 정보 조회 (통화 가능한 사용자 목록 조회)
// GetUserWebRTCSession 사용자 WebRTC 세션 정보 조회
func (r *RedisDB) GetUserWebRTCSession(userID string) (*UserWebRTCSession, error) {
	sessionJSON, err := r.client.HGet(r.ctx, "webrtc:sessions", userID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("webrtc session not found")
		}
		return nil, err
	}

	var session UserWebRTCSession
	err = json.Unmarshal([]byte(sessionJSON), &session)
	if err != nil {
		return nil, err
	}

	return &session, nil
}
*/

/* GetAvailableUsersForCall 통화 가능한 사용자 목록 조회 (자신을 제외한 사용자 목록 반환)
// GetAvailableUsersForCall 통화 가능한 사용자 목록 조회
func (r *RedisDB) GetAvailableUsersForCall(excludeUserID string) ([]string, error) {
	allUsers, err := r.client.SMembers(r.ctx, "webrtc:available_users").Result()
	if err != nil {
		return nil, err
	}

	// 자신을 제외한 사용자 목록 반환
	availableUsers := make([]string, 0, len(allUsers))
	for _, userID := range allUsers {
		if userID != excludeUserID {
			availableUsers = append(availableUsers, userID)
		}
	}

	return availableUsers, nil
}
*/

/* UpdateUserCallStatus 사용자 통화 상태 업데이트
// UpdateUserCallStatus 사용자 통화 상태 업데이트
func (r *RedisDB) UpdateUserCallStatus(userID string, isInCall bool, callWith string) error {
	session, err := r.GetUserWebRTCSession(userID)
	if err != nil {
		// 세션이 없으면 새로 생성
		session = &UserWebRTCSession{
			UserID:          userID,
			SessionID:       utils.GenUuid(),
			ConnectionState: "connected",
			LastHeartbeat:   time.Now(),
		}
	}

	session.IsInCall = isInCall
	session.CallWith = callWith
	if isInCall {
		session.CallStartTime = time.Now()
	}

	return r.SetUserWebRTCSession(*session)
}
*/

/* CleanupInactiveSessions 비활성 WebRTC 세션 정리
// CleanupInactiveSessions 비활성 WebRTC 세션 정리
func (r *RedisDB) CleanupInactiveSessions() error {
	sessions, err := r.client.HGetAll(r.ctx, "webrtc:sessions").Result()
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()
	cutoffTime := time.Now().Add(-30 * time.Minute) // 30분 이상 비활성

	for userID, sessionJSON := range sessions {
		var session UserWebRTCSession
		if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
			continue
		}

		// 비활성 세션 정리
		if session.LastHeartbeat.Before(cutoffTime) {
			pipe.HDel(r.ctx, "webrtc:sessions", userID)
			pipe.SRem(r.ctx, "webrtc:available_users", userID)
		}
	}

	_, err = pipe.Exec(r.ctx)
	return err
}
*/

/* GetWebRTCStats WebRTC 관련 통계 조회
// GetWebRTCStats WebRTC 관련 통계 조회
func (r *RedisDB) GetWebRTCStats() (map[string]interface{}, error) {
	pipe := r.client.Pipeline()

	// 활성 세션 수
	pipe.HLen(r.ctx, "webrtc:sessions")
	// 통화 가능한 사용자 수
	pipe.SCard(r.ctx, "webrtc:available_users")

	cmds, err := pipe.Exec(r.ctx)
	if err != nil {
		return nil, err
	}

	totalSessions := cmds[0].(*redis.IntCmd).Val()
	availableUsers := cmds[1].(*redis.IntCmd).Val()

	// 진행 중인 통화 수 계산
	sessions, err := r.client.HGetAll(r.ctx, "webrtc:sessions").Result()
	if err != nil {
		return nil, err
	}

	activeCallsCount := int64(0)
	for _, sessionJSON := range sessions {
		var session UserWebRTCSession
		if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
			continue
		}
		if session.IsInCall {
			activeCallsCount++
		}
	}

	stats := map[string]interface{}{
		"totalSessions":  totalSessions,
		"availableUsers": availableUsers,
		"activeCalls":    activeCallsCount / 2, // 2명이 1통화이므로 나누기 2
		"lastUpdated":    time.Now(),
	}

	return stats, nil
}
*/
