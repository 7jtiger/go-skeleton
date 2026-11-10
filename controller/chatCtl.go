package controller

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	"ms-gateway/models"
	ptc "ms-gateway/protocol"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	CHAT_READ_BUFFER_SIZE  = 4096
	CHAT_WRITE_BUFFER_SIZE = 4096
	CHAT_MAX_MESSAGE_SIZE  = 64 * 1024 // 64KB
	CHAT_MSG_BUFFER_SIZE   = 256
	CHAT_PING_PERIOD       = 54 * time.Second
	CHAT_PONG_WAIT         = 60 * time.Second
	CHAT_WRITE_WAIT        = 10 * time.Second
)

// ChatClient 텍스트 채팅 WebSocket 클라이언트
type ChatClient struct {
	conn *websocket.Conn
	send chan []byte
	uid  uint64
	nick string

	sendMu      sync.Mutex
	sendClosed  bool
	queuedBytes int64
	closeOnce   sync.Once
}

// ChatController 텍스트 채팅 컨트롤러
type ChatController struct {
	ctl           *Controller
	cfg           *conf.Config
	rep           *models.Repositories
	rdb           *models.RedisDB
	adb           *models.AccountDB
	hdb           *models.HistoryDB
	clients       map[uint64]*ChatClient // uid -> ChatClient
	clientsMu     sync.RWMutex
	presenceLocks [256]sync.Mutex // Serialize per-UID presence updates without holding the clients map lock.
	totalConn     int64
	upgrader      websocket.Upgrader
}

// NewChatController ChatController 생성
func NewChatController(ctl *Controller, rep *models.Repositories) (*ChatController, error) {
	r := &ChatController{
		ctl:     ctl,
		rep:     rep,
		cfg:     ctl.cfg,
		clients: make(map[uint64]*ChatClient),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  CHAT_READ_BUFFER_SIZE,
			WriteBufferSize: CHAT_WRITE_BUFFER_SIZE,
			CheckOrigin: func(r *http.Request) bool {
				return true // 개발 중에는 모든 origin 허용
			},
		},
	}

	if err := rep.Get(&r.adb, &r.hdb, &r.rdb); err != nil {
		return nil, err
	}

	return r, nil
}

// HandleWebSocket godoc
// @Summary      DM WebSocket 연결
// @Description  DM WebSocket 연결 API입니다.
// @Description
// @Description  **요청**: Authorization: Bearer access token. HTTP Upgrade 요청.
// @Description
// @Description  **처리**: JWT UID로 연결하며 같은 UID의 기존 연결을 교체합니다. 프로세스 연결 상한 초과 시 503입니다. 수신 프레임 최대 64KiB, 송신 큐 최대 256개/512KiB입니다.
// @Description
// @Description  **응답**: 101 Switching Protocols 이후 WebSocket 메시지로 통신합니다. REST JSON 성공 응답은 없습니다.
// @Tags         DM
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Success      101 {string} string "Switching Protocols (WebSocket)"
// @Failure      400 {string} string "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      503 {object} protocol.RespHeader "서버 동시 처리·연결 한도 초과"
// @Router       /dm/v01/ws [get]
// @Example Request: GET /dm/v01/ws
func (cc *ChatController) HandleWebSocket(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	userInfo, ok := user.(*ptc.UserInfoResp)
	if !ok || userInfo == nil {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid user context")
		return
	}
	uid64 := userInfo.Uid
	if !reserveConnection(&cc.totalConn, MAX_VDWS_CONNECT) {
		cc.ctl.SimpleError(c, http.StatusServiceUnavailable, "Server at capacity")
		return
	}
	// WebSocket 업그레이드
	conn, err := cc.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		atomic.AddInt64(&cc.totalConn, -1)
		log.Error("WebSocket upgrade failed:", err)
		return
	}

	// 클라이언트 생성 및 등록
	client := &ChatClient{
		conn: conn,
		send: make(chan []byte, CHAT_MSG_BUFFER_SIZE),
		uid:  uid64,
		nick: userInfo.Nick,
	}

	cc.registerClient(client)

	log.Info(fmt.Sprintf("New chat WebSocket connection: %d", uid64))

	// 고루틴 시작
	go cc.writePump(client)
	go func() {
		defer atomic.AddInt64(&cc.totalConn, -1)
		cc.readPump(client)
	}()
}

// registerClient 클라이언트 등록
func (cc *ChatController) registerClient(client *ChatClient) {
	lock := &cc.presenceLocks[client.uid%uint64(len(cc.presenceLocks))]
	lock.Lock()
	defer lock.Unlock()
	cc.clientsMu.Lock()
	oldClient := cc.clients[client.uid]
	cc.clients[client.uid] = client
	cc.clientsMu.Unlock()

	// 기존 연결이 있으면 종료 (맵 교체 후 구 연결의 unregister는 no-op)
	if oldClient != nil && oldClient != client {
		cc.closeClientSend(oldClient)
		oldClient.conn.Close()
	}

	if err := cc.rdb.SetOnline(client.uid); err != nil {
		log.Warn("failed to set online state for %d: %v", client.uid, err)
	}
	log.Info(fmt.Sprintf("Chat client registered: %d", client.uid))
}

func (cc *ChatController) closeClientSend(client *ChatClient) {
	if client == nil {
		return
	}
	client.sendMu.Lock()
	defer client.sendMu.Unlock()
	client.closeOnce.Do(func() {
		client.sendClosed = true
		close(client.send)
	})
}

// unregisterClient 클라이언트 등록 해제
func (cc *ChatController) unregisterClient(client *ChatClient) {
	lock := &cc.presenceLocks[client.uid%uint64(len(cc.presenceLocks))]
	lock.Lock()
	defer lock.Unlock()
	cc.clientsMu.Lock()

	current, exists := cc.clients[client.uid]
	if !exists || current != client {
		// 재접속으로 맵이 새 클라이언트로 교체된 경우 구 연결 정리는 스킵
		cc.clientsMu.Unlock()
		return
	}

	delete(cc.clients, client.uid)
	cc.clientsMu.Unlock()
	cc.closeClientSend(client)
	if err := cc.rdb.DeleteOnline(client.uid); err != nil {
		log.Warn("failed to delete online state for %d: %v", client.uid, err)
	}
	log.Info(fmt.Sprintf("Chat client unregistered: %d", client.uid))
}

// readPump 메시지 읽기
func (cc *ChatController) readPump(client *ChatClient) {
	defer func() {
		cc.unregisterClient(client)
		client.conn.Close()
	}()

	client.conn.SetReadLimit(CHAT_MAX_MESSAGE_SIZE)
	client.conn.SetReadDeadline(time.Now().Add(CHAT_PONG_WAIT))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(CHAT_PONG_WAIT))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("Chat WebSocket error:", err)
			}
			break
		}

		// 메시지 처리
		cc.handleMessage(client, message)
	}
}

// writePump 메시지 쓰기
func (cc *ChatController) writePump(client *ChatClient) {
	ticker := time.NewTicker(CHAT_PING_PERIOD)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			if ok {
				atomic.AddInt64(&client.queuedBytes, -int64(len(message)))
			}
			client.conn.SetWriteDeadline(time.Now().Add(CHAT_WRITE_WAIT))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 대기 중인 메시지 일괄 전송
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				next := <-client.send
				atomic.AddInt64(&client.queuedBytes, -int64(len(next)))
				w.Write(next)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(CHAT_WRITE_WAIT))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 메시지 처리
func (cc *ChatController) handleMessage(client *ChatClient, message []byte) {
	var msg ptc.ChatMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error("Chat message parse error:", err)
		return
	}

	msg.From = strconv.FormatUint(client.uid, 10)
	msg.Timestamp = time.Now().Unix()

	switch msg.Type {
	case "text-message":
		cc.handleTextMessage(client, &msg)
	case "typing":
		cc.handleTyping(client, &msg)
	case "read-receipt":
		cc.handleReadReceipt(client, &msg)
	case "put-gift":
		cc.handleSendGift(client, &msg)
		/* 	case "get-gift":
		cc.handleRecvGift(client, &msg) */
	// case "present":
	/* 	case "call-request":
	   		cc.handleCallRequestFromDM(client, &msg)
	   	case "call-accept":
	   		cc.handleCallAcceptFromDM(client, &msg)
	   	case "call-reject":
	   		cc.sendToUser(msg.To, &msg)
	   	case "call-cancel":
	   		cc.handleCallCancel(client, &msg) */
	default:
		log.Warn(fmt.Sprintf("Unknown chat message type: %s from %d", msg.Type, client.uid))
	}
}

// handleTextMessage 텍스트 메시지 처리
func (cc *ChatController) handleTextMessage(client *ChatClient, msg *ptc.ChatMessage) {
	ctx, ok := cc.prepareDMMessage(client, msg)
	if !ok {
		return
	}
	cc.persistAndDeliverDMMessage(client, msg, ctx)
}

// handleTyping 타이핑 상태 처리
func (cc *ChatController) handleTyping(client *ChatClient, msg *ptc.ChatMessage) {
	// 수신자에게 타이핑 상태 전송
	cc.sendToUser(msg.To, msg)
}

type dmMessageContext struct {
	targetUID uint64
	target    ptc.UserInfoResp
	room      *models.DMRoomRow
	now       int64
}

func (cc *ChatController) prepareDMMessage(client *ChatClient, msg *ptc.ChatMessage) (*dmMessageContext, bool) {
	toUID, err := strconv.ParseUint(msg.To, 10, 64)
	if err != nil {
		log.Warn("invalid target uid in %s: %s", msg.Type, msg.To)
		return nil, false
	}
	if toUID == client.uid {
		log.Warn("self dm message rejected: uid=%d type=%s", client.uid, msg.Type)
		return nil, false
	}

	tUser, err := cc.adb.GetUserInfoByUID(toUID)
	if err != nil {
		log.Warn("target user %d not found in DB: %v", toUID, err)
		return nil, false
	}

	room, err := cc.ensureDMRoom(client.uid, &tUser)
	if err != nil {
		log.Error("ensureDMRoom failed uid=%d to=%d: %v", client.uid, toUID, err)
		cc.sendBillingFailedAck(client, msg, err.Error())
		return nil, false
	}
	if room == nil {
		log.Error("ensureDMRoom returned nil uid=%d to=%d", client.uid, toUID)
		return nil, false
	}

	now := time.Now().Unix()
	msg.RoomID = strconv.FormatInt(room.Idx, 10)
	msg.Timestamp = now

	return &dmMessageContext{
		targetUID: toUID,
		target:    tUser,
		room:      room,
		now:       now,
	}, true
}

func (cc *ChatController) persistAndDeliverDMMessage(client *ChatClient, msg *ptc.ChatMessage, ctx *dmMessageContext) {
	if ctx == nil || ctx.room == nil {
		return
	}

	if err := cc.rdb.SaveChatMessage(msg); err != nil {
		log.Error("Failed to save chat message: %v", err)
		return
	}

	unreadCnt, err := cc.rdb.IncrUnread(ctx.targetUID, ctx.room.Idx)
	if err != nil {
		log.Error("failed to increment unread: %v", err)
	}

	delivered := cc.sendToUser(msg.To, msg)
	if !delivered && cc.ctl.FCMPusher != nil {
		cc.ctl.FCMPusher.SendDMPush(cc.senderDisplayName(client), msg.Content, ctx.target.Did)
	}

	ack := &ptc.ChatMessage{
		Type:      "msg-ack",
		From:      msg.To,
		To:        msg.From,
		RoomID:    msg.RoomID,
		MsgID:     msg.MsgID,
		Unread:    int(unreadCnt),
		CallMode:  msg.CallMode,
		Timestamp: ctx.now,
	}
	cc.sendToUser(strconv.FormatUint(client.uid, 10), ack)

	if err := cc.hdb.TouchDMRoomActivity(ctx.room.Idx, time.Unix(msg.Timestamp, 0)); err != nil {
		log.Warn("failed to touch dm room activity: %v", err)
	}
}

func (cc *ChatController) senderDisplayName(client *ChatClient) string {
	if client != nil && client.nick != "" {
		return client.nick
	}
	if client != nil {
		return strconv.FormatUint(client.uid, 10)
	}
	return ""
}

// handleReadReceipt 읽음 확인 처리: unread reset + last_read mid 전진 + 상대 relay
func (cc *ChatController) handleReadReceipt(client *ChatClient, msg *ptc.ChatMessage) {
	if msg.RoomID == "" {
		return
	}
	roomID, err := strconv.ParseInt(msg.RoomID, 10, 64)
	if err != nil {
		log.Warn("invalid roomId in read-receipt: %s", msg.RoomID)
		return
	}

	mid := msg.MsgID
	if mid == "" {
		latest, lErr := cc.rdb.GetLatestMessageID(msg.RoomID)
		if lErr != nil {
			log.Warn("failed to get latest mid for room %s: %v", msg.RoomID, lErr)
		} else {
			mid = latest
			msg.MsgID = mid
		}
	}

	if mid != "" {
		updated, sErr := cc.rdb.SetLastReadMid(client.uid, roomID, mid)
		if sErr != nil {
			log.Warn("failed to set last read mid for %d room %s: %v", client.uid, msg.RoomID, sErr)
		} else if updated {
			log.Info("last read mid updated: uid=%d room=%s mid=%s", client.uid, msg.RoomID, mid)
		}
	}

	if err := cc.rdb.ResetUnread(client.uid, roomID); err != nil {
		log.Warn("failed to reset unread for %d room %s: %v", client.uid, msg.RoomID, err)
	}

	msg.From = strconv.FormatUint(client.uid, 10)
	if msg.To != "" {
		cc.sendToUser(msg.To, msg)
	}
}

func (cc *ChatController) handleSendGift(client *ChatClient, msg *ptc.ChatMessage) {
	giftCount := strings.TrimSpace(msg.Content)
	if giftCount == "" {
		log.Warn("empty gift count from %d to %s", client.uid, msg.To)
		return
	}
	if _, err := strconv.ParseUint(giftCount, 10, 64); err != nil {
		log.Warn("invalid gift count from %d to %s: %s", client.uid, msg.To, giftCount)
		return
	}

	ctx, ok := cc.prepareDMMessage(client, msg)
	if !ok {
		return
	}

	msg.Content = cc.senderDisplayName(client) + "님이(가) 화살 " + giftCount + "개를 선물했습니다."
	msg.CallMode = "gift"
	cc.persistAndDeliverDMMessage(client, msg, ctx)
}

/* func (cc *ChatController) handleRecvGift(client *ChatClient, msg *ptc.ChatMessage) {
	//TODO: implement
}
*/

// updateRoomEntryRead 방 입장 시 last_read mid 갱신 (query mid 또는 최신 메시지)
func (cc *ChatController) updateRoomEntryRead(uid uint64, roomID int64, roomIDStr, mid string) {
	if mid == "" {
		latest, err := cc.rdb.GetLatestMessageID(roomIDStr)
		if err != nil {
			log.Warn("room entry: get latest mid failed room=%s: %v", roomIDStr, err)
			return
		}
		mid = latest
	}
	if mid == "" {
		return
	}
	if _, err := cc.rdb.SetLastReadMid(uid, roomID, mid); err != nil {
		log.Warn("room entry: set last read mid failed uid=%d room=%s: %v", uid, roomIDStr, err)
		return
	}
	if err := cc.rdb.ResetUnread(uid, roomID); err != nil {
		log.Warn("room entry: reset unread failed uid=%d room=%s: %v", uid, roomIDStr, err)
	}
}

// sendToUser 특정 사용자에게 메시지 전송
func (cc *ChatController) sendToUser(uid string, msg *ptc.ChatMessage) bool {
	uid64, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		log.Warn(fmt.Sprintf("invalid uid in sendToUser: %s", uid))
		return false
	}
	cc.clientsMu.RLock()
	client, exists := cc.clients[uid64]
	cc.clientsMu.RUnlock()

	if !exists {
		log.Warn(fmt.Sprintf("User %d not connected to chat WebSocket", uid64))
		return false
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Error("Message marshal error:", err)
		return false
	}

	if !cc.trySend(client, data) {
		log.Warn(fmt.Sprintf("Client %d send buffer full", uid64))
		return false
	}
	return true
}

func (cc *ChatController) trySend(client *ChatClient, data []byte) bool {
	if client == nil {
		return false
	}
	client.sendMu.Lock()
	defer client.sendMu.Unlock()
	if client.sendClosed || atomic.LoadInt64(&client.queuedBytes)+int64(len(data)) > 512<<10 {
		return false
	}
	atomic.AddInt64(&client.queuedBytes, int64(len(data)))
	select {
	case client.send <- data:
		return true
	default:
		atomic.AddInt64(&client.queuedBytes, -int64(len(data)))
		return false
	}
}

// ExistDMRoom godoc
// @Summary      DM 채팅방 존재 확인
// @Description  DM 채팅방 존재 확인 API입니다.
// @Description
// @Description  **요청**: path tid: 상대 UID.
// @Description
// @Description  **처리**: JWT 사용자와 상대방의 방 존재 여부를 조회합니다.
// @Description
// @Description  **응답**: 최상위 exist.
// @Tags         DM
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} object{exist=bool} "최상위 exist."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /dm/v01/room/exists/{tid} [get]
// @Example Response: {"exist":true}
// @x-response-example {"exist":true}
// @Example Request: GET /dm/v01/room/exists/4033287471439576593
func (cc *ChatController) ExistDMRoom(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	tid64, err := strconv.ParseUint(c.Param("tid"), 10, 64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
		return
	}

	_, err = cc.hdb.GetDMRoomByPair(uid64, tid64)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"exist": true})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusOK, gin.H{"exist": false})
		return
	}
	cc.ctl.SimpleError(c, http.StatusInternalServerError, err.Error())
}

// ensureDMRoom DM 룸 강제 생성/갱신 (방 상태 SSOT = MySQL chat_his.st_chat).
// 메시지·컷오프만 Redis. signaling waitingRoom 과 혼용 금지.
func (cc *ChatController) ensureDMRoom(uid uint64, tUser *ptc.UserInfoResp) (*models.DMRoomRow, error) {
	room, err := cc.hdb.GetDMRoomByPair(uid, tUser.Uid)
	if err == nil {
		return cc.openExistingDMRoom(room, uid, tUser.Uid, false)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return cc.createDMRoomOrRecover(uid, tUser)
}

// openExistingDMRoom 기존 방 강제 상태 갱신 (나가기 후 재요청·재입장 포함).
// alreadyBilled=true 이면 create 충돌 복구 경로(과금 이미 수행)에서 재과금 생략.
func (cc *ChatController) openExistingDMRoom(room *models.DMRoomRow, uid, partnerUID uint64, alreadyBilled bool) (*models.DMRoomRow, error) {
	prevST := room.STChat
	wasOut := !models.IsUserInDMRoom(prevST, room.UID, room.TID, uid)
	partnerWasOut := models.IsPartnerLeft(prevST, room.UID, room.TID, uid)
	bothLeft := prevST == models.STChatBothLeft

	// 과금: 신규 생성 / 상대 나감 재요청 / 양쪽 나감 후 재오픈. 나만 나갔다 재입장(상대 유지)은 미과금.
	needBill := !alreadyBilled && (partnerWasOut || bothLeft)
	if needBill {
		if err := cc.checkSettleMent(uid, partnerUID, room.Idx); err != nil {
			return nil, err
		}
	}

	var err error
	if partnerWasOut {
		if err = cc.hdb.UpdateDMRoomStatus(room.Idx, models.STChatBothIn); err != nil {
			return nil, err
		}
		room.STChat = models.STChatBothIn
		room.AtUpdate = time.Now()
	} else {
		room, err = cc.hdb.ActivateDMRoomSide(room, uid)
		if err != nil {
			return nil, err
		}
	}

	if wasOut {
		if err := cc.setDMHistoryCutoff(uid, room.Idx); err != nil {
			log.Warn("failed to set dm history cutoff on rejoin uid=%d room=%d: %v", uid, room.Idx, err)
		}
	}

	log.Info("ensureDMRoom update uid=%d room=%d prev_st=%d new_st=%d wasOut=%v partnerWasOut=%v billed=%v",
		uid, room.Idx, prevST, room.STChat, wasOut, partnerWasOut, needBill || alreadyBilled)
	return room, nil
}

// createDMRoomOrRecover 신규 생성. room_id UNIQUE 충돌 시 기존 방 조회 후 강제 갱신.
func (cc *ChatController) createDMRoomOrRecover(uid uint64, tUser *ptc.UserInfoResp) (*models.DMRoomRow, error) {
	if err := cc.checkSettleMent(uid, tUser.Uid, 0); err != nil {
		return nil, err
	}

	pairRoomID := models.FormatDMPairRoomID(uid, tUser.Uid)
	log.Info("CreateDMRoom uid=%d tid=%d pair_room_id=%s", uid, tUser.Uid, pairRoomID)

	rmid, createErr := cc.hdb.CreateDMRoom(uid, tUser)
	if createErr != nil {
		if models.IsMySQLDuplicate(createErr) {
			room, err := cc.hdb.GetDMRoomByPair(uid, tUser.Uid)
			if err != nil {
				return nil, fmt.Errorf("create duplicate but lookup failed: %w", err)
			}
			log.Info("CreateDMRoom duplicate recovered uid=%d room=%d", uid, room.Idx)
			return cc.openExistingDMRoom(room, uid, tUser.Uid, true)
		}
		return nil, createErr
	}

	return cc.hdb.GetDMRoom(rmid)
}

// checkSettleMent DM 방 생성·재요청 시 과금 (차후 잔액 확인·차감·UdtDMPaid 구현)
func (cc *ChatController) checkSettleMent(requester, partner uint64, roomID int64) error {
	_ = requester
	_ = partner
	_ = roomID
	return nil
}

// resolvePartnerLeft MySQL st_chat 기준 (방 상태 SSOT). Redis 메시지와 분리.
func (cc *ChatController) resolvePartnerLeft(rm *models.DMRoomRow, viewerUID uint64) bool {
	return models.IsPartnerLeft(rm.STChat, rm.UID, rm.TID, viewerUID)
}

func (cc *ChatController) sendBillingFailedAck(client *ChatClient, msg *ptc.ChatMessage, reason string) {
	ack := &ptc.ChatMessage{
		Type:      "billing-failed",
		From:      msg.To,
		To:        msg.From,
		RoomID:    msg.RoomID,
		MsgID:     msg.MsgID,
		Content:   reason,
		CallMode:  msg.CallMode,
		Timestamp: time.Now().Unix(),
	}
	cc.sendToUser(strconv.FormatUint(client.uid, 10), ack)
}

// setDMHistoryCutoff 나가기/재입장 사용자에게 현재 시점 이후 메시지만 보이도록 LIST 인덱스 컷오프 설정
func (cc *ChatController) setDMHistoryCutoff(uid uint64, roomID int64) error {
	roomIDStr := strconv.FormatInt(roomID, 10)
	listLen, err := cc.rdb.GetChatMessageListLen(roomIDStr)
	if err != nil {
		return err
	}
	if err := cc.rdb.SetDMHistoryFromIndex(uid, roomID, listLen); err != nil {
		return err
	}
	_ = cc.rdb.ResetUnread(uid, roomID)
	return nil
}

func (cc *ChatController) dmHistoryFromIndex(uid uint64, roomID int64) int64 {
	fromIdx, err := cc.rdb.GetDMHistoryFromIndex(uid, roomID)
	if err != nil {
		log.Warn("failed to get dm history from index uid=%d room=%d: %v", uid, roomID, err)
		return 0
	}
	return fromIdx
}

func (cc *ChatController) resolveLastMsg(rm *models.DMRoomRow, viewerUID uint64) (string, int) {
	roomIDStr := strconv.FormatInt(rm.Idx, 10)
	fromIdx := cc.dmHistoryFromIndex(viewerUID, rm.Idx)
	tcnt, messages, err := cc.rdb.GetChatMessages(roomIDStr, 0, 1, fromIdx)
	lastMsg := ""
	if err == nil && messages != nil && len(*messages) > 0 {
		switch (*messages)[0].Mode {
		case "img", "image":
			lastMsg = "사진"
		default:
			lastMsg = (*messages)[0].Content
		}
	}
	if lastMsg == "" && cc.resolvePartnerLeft(rm, viewerUID) {
		lastMsg = models.PartnerLeftMsg
	}
	return lastMsg, int(tcnt)
}

func (cc *ChatController) buildDMRoomResp(rm *models.DMRoomRow, viewerUID uint64) (ptc.DMRoomResp, error) {
	return cc.buildDMRoomRespWithUnread(rm, viewerUID, -1)
}

func (cc *ChatController) buildDMRoomRespWithUnread(rm *models.DMRoomRow, viewerUID uint64, unreadHint int64) (ptc.DMRoomResp, error) {
	partnerUID := models.PartnerUIDOfRoom(rm, viewerUID)
	info, err := cc.adb.GetUserInfoByUID(partnerUID)
	if err != nil {
		return ptc.DMRoomResp{}, err
	}
	partner := &ptc.PartnerInfo{
		PID:      info.Uid,
		Nick:     info.Nick,
		ThumbPic: info.ThumbPic,
		Gender:   info.Gender,
		Age:      info.Age,
		Area:     info.Area,
	}

	partnerLeft := cc.resolvePartnerLeft(rm, viewerUID)
	lastMsg, total := cc.resolveLastMsg(rm, viewerUID)

	var unread int64
	if unreadHint >= 0 {
		unread = unreadHint
	} else {
		unread, _ = cc.rdb.GetUnread(viewerUID, rm.Idx)
	}

	return ptc.DMRoomResp{
		RoomID:      rm.Idx,
		Partner:     partner,
		LastMsg:     lastMsg,
		PartnerLeft: partnerLeft,
		Total:       total,
		Unread:      int(unread),
		AtCreate:    rm.AtCrtCHAT.Format(time.RFC3339),
		AtUpdate:    rm.AtUpdate.Format(time.RFC3339),
	}, nil
}

// buildDMRoomRespPrefetched 일괄 조회한 파트너·마지막 메시지로 응답 구성 (인박스 목록 전용, N+1 제거)
func (cc *ChatController) buildDMRoomRespPrefetched(rm *models.DMRoomRow, viewerUID uint64, unread int64, partners map[uint64]ptc.UserInfoResp, last models.LastMsgInfo) (ptc.DMRoomResp, error) {
	partnerUID := models.PartnerUIDOfRoom(rm, viewerUID)
	info, ok := partners[partnerUID]
	if !ok {
		// 배치 결과에 없으면 단건 조회로 폴백
		var err error
		info, err = cc.adb.GetUserInfoByUID(partnerUID)
		if err != nil {
			return ptc.DMRoomResp{}, err
		}
	}

	partner := &ptc.PartnerInfo{
		PID:      info.Uid,
		Nick:     info.Nick,
		ThumbPic: info.ThumbPic,
		Gender:   info.Gender,
		Age:      info.Age,
		Area:     info.Area,
	}

	partnerLeft := cc.resolvePartnerLeft(rm, viewerUID)
	lastMsg := last.Content
	if lastMsg == "" && partnerLeft {
		lastMsg = models.PartnerLeftMsg
	}

	return ptc.DMRoomResp{
		RoomID:      rm.Idx,
		Partner:     partner,
		LastMsg:     lastMsg,
		PartnerLeft: partnerLeft,
		Total:       int(last.Total),
		Unread:      int(unread),
		AtCreate:    rm.AtCrtCHAT.Format(time.RFC3339),
		AtUpdate:    rm.AtUpdate.Format(time.RFC3339),
	}, nil
}

type dmRoomSortItem struct {
	row        models.DMRoomRow
	unread     int64
	lastRecvAt int64
}

// sortDMRoomsForInbox 1) unread>0 우선 2) unread 그룹 내 :ts(최근 수신) 3) at_update 최신
func sortDMRoomsForInbox(rooms []models.DMRoomRow, unreadInfo map[string]models.UnreadInfo) []models.DMRoomRow {
	if len(rooms) == 0 {
		return rooms
	}

	items := make([]dmRoomSortItem, len(rooms))
	for i, rm := range rooms {
		key := strconv.FormatInt(rm.Idx, 10)
		info := unreadInfo[key]
		items[i] = dmRoomSortItem{
			row:        rm,
			unread:     info.Count,
			lastRecvAt: info.LastRecvAt,
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		aHasUnread := a.unread > 0
		bHasUnread := b.unread > 0
		if aHasUnread != bHasUnread {
			return aHasUnread
		}
		if aHasUnread && bHasUnread && a.lastRecvAt != b.lastRecvAt {
			return a.lastRecvAt > b.lastRecvAt
		}
		return a.row.AtUpdate.After(b.row.AtUpdate)
	})

	sorted := make([]models.DMRoomRow, len(items))
	for i, it := range items {
		sorted[i] = it.row
	}
	return sorted
}

func paginateDMRooms(rooms []models.DMRoomRow, page int) []models.DMRoomRow {
	if page < 1 {
		page = 1
	}
	start := (page - 1) * models.DMRoomListPageSize
	if start >= len(rooms) {
		return []models.DMRoomRow{}
	}
	end := start + models.DMRoomListPageSize
	if end > len(rooms) {
		end = len(rooms)
	}
	return rooms[start:end]
}

func (cc *ChatController) notifyPartnerLeft(room *models.DMRoomRow, leaverUID uint64) {
	partnerUID := models.PartnerUIDOfRoom(room, leaverUID)
	roomIDStr := strconv.FormatInt(room.Idx, 10)
	partnerUIDStr := strconv.FormatUint(partnerUID, 10)
	leaverUIDStr := strconv.FormatUint(leaverUID, 10)
	now := time.Now().Unix()

	sysMsg := &ptc.ChatMessage{
		Type:      "system-message",
		From:      "system",
		To:        partnerUIDStr,
		RoomID:    roomIDStr,
		Content:   models.PartnerLeftMsg,
		MsgID:     fmt.Sprintf("sys-leave-%d", now),
		Timestamp: now,
	}
	if err := cc.rdb.SaveChatMessage(sysMsg); err != nil {
		log.Error("Failed to save partner-left system message:", err)
	} else {
		if err := cc.hdb.TouchDMRoomActivity(room.Idx, time.Unix(now, 0)); err != nil {
			log.Warn("failed to touch dm room activity on partner-left: %v", err)
		}
	}

	if _, err := cc.rdb.IncrUnread(partnerUID, room.Idx); err != nil {
		log.Warn("failed to increment unread on partner-left: %v", err)
	}

	wsMsg := &ptc.ChatMessage{
		Type:      "partner-left",
		From:      leaverUIDStr,
		To:        partnerUIDStr,
		RoomID:    roomIDStr,
		Content:   models.PartnerLeftMsg,
		MsgID:     sysMsg.MsgID,
		Timestamp: now,
	}
	cc.sendToUser(partnerUIDStr, wsMsg)
	// cc.sendToUser(partnerUIDStr, sysMsg)
}

func (cc *ChatController) buildPartnerInfoByUID(uid uint64) (*ptc.PartnerInfo, error) {
	info, err := cc.adb.GetUserInfoByUID(uid)
	if err != nil {
		return nil, err
	}
	return &ptc.PartnerInfo{
		PID:      info.Uid,
		Nick:     info.Nick,
		ThumbPic: info.ThumbPic,
		Gender:   info.Gender,
		Age:      info.Age,
		Area:     info.Area,
	}, nil
}

// handleCallRequestFromDM DM 중 통화 요청 처리
// [통화 연계 정책]
//  1. 수신자가 signaling waitingRoom에 있으면 → signaling으로 즉시 전달
//  2. signaling 미접속 + chat WS 온라인이면 → call-incoming DM 알림
//  3. 완전 오프라인이면 → FCM push fallback
func (cc *ChatController) handleCallRequestFromDM(client *ChatClient, msg *ptc.ChatMessage) {
	msg.Type = "call-request"

	// 1) signaling waitingRoom 확인 → 즉시 전달
	if cc.ctl.Signaling != nil && cc.ctl.Signaling.IsUserInWaitingRoom(msg.To) {
		cc.ctl.Signaling.ForwardCallRequest(msg)
		return
	}

	// 2) chat WS 온라인 확인 → call-incoming 알림
	msg.Type = "call-incoming"
	delivered := cc.sendToUser(msg.To, msg)
	if delivered {
		return
	}

	// 3) 오프라인 → FCM push fallback
	toUID, err := strconv.ParseUint(msg.To, 10, 64)
	if err != nil {
		return
	}
	if cc.ctl.FCMPusher == nil {
		return
	}
	receiver, userErr := cc.adb.GetUserInfoByUID(toUID)
	if userErr != nil {
		return
	}

	fromUID := strconv.FormatUint(client.uid, 10)

	cc.ctl.FCMPusher.SendCallPush(fromUID, "", msg.CallMode, receiver.Did)

	notice := &ptc.ChatMessage{
		Type:      "call-info",
		From:      msg.To,
		To:        msg.From,
		RoomID:    msg.RoomID,
		Content:   "상대방이 오프라인 상태여서 푸시 알림을 전송했습니다.",
		Timestamp: time.Now().Unix(),
	}
	cc.sendToUser(fromUID, notice)
}

func (cc *ChatController) handleCallAcceptFromDM(client *ChatClient, msg *ptc.ChatMessage) {
	partner, err := cc.buildPartnerInfoByUID(client.uid)
	if err == nil {
		msg.Partner = partner
	}
	cc.sendToUser(msg.To, msg)
}

func (cc *ChatController) handleCallCancel(client *ChatClient, msg *ptc.ChatMessage) {
	msg.Type = "call-cancel"
	delivered := cc.sendToUser(msg.To, msg)
	if !delivered {
		toUID, err := strconv.ParseUint(msg.To, 10, 64)
		if err != nil {
			return
		}
		if cc.ctl.FCMPusher == nil {
			return
		}
		receiver, userErr := cc.adb.GetUserInfoByUID(toUID)
		if userErr != nil {
			return
		}
		cc.ctl.FCMPusher.SendCallPush(msg.From, "", "cancel", receiver.Did)
	}
}

// CreateChatRoom godoc
// @Summary      DM 채팅방 생성 또는 조회
// @Description  DM 채팅방 생성 또는 조회 API입니다.
// @Description
// @Description  **요청**: path pid: 상대 UID. body 없음.
// @Description
// @Description  **처리**: JWT 본인과 상대 계정으로 기존 방을 확인하거나 생성합니다.
// @Description
// @Description  **응답 data**: DMRoomResp.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         DM
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        pid path string true "상대 UID"
// @Success      200 {object} protocol.RespDataHeader "data: DMRoomResp."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /dm/v01/mkroom/{pid} [post]
// @Example Request: POST /dm/v01/mkroom/4033287471439576593
func (cc *ChatController) CreateChatRoom(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	pid64, err := strconv.ParseUint(c.Param("pid"), 10, 64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
		return
	}

	// if err := c.ShouldBindJSON(&req); err != nil {
	// 	cc.ctl.RespError(c, ptc.NewRespHeader(ptc.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
	// 	return
	// }

	tUser, err := cc.adb.GetUserInfoByUID(pid64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
		return
	}

	rm, err := cc.ensureDMRoom(uid64, &tUser)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to create dm room", err)
		return
	}

	resp, err := cc.buildDMRoomResp(rm, uid64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to build dm room response", err)
		return
	}

	cc.ctl.SendDataResponse(c, http.StatusOK, resp)
}

// DeleteChatRoom godoc
// @Summary      DM 채팅방 나가기
// @Description  DM 채팅방 나가기 API입니다.
// @Description
// @Description  **요청**: path room_id: 방 인덱스.
// @Description
// @Description  **처리**: JWT 사용자가 참여 중인 방에서 나갑니다. 없는 방은 404, 비참여자는 403입니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         DM
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        room_id path string true "채팅방 ID"
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "권한 없음 또는 차단 관계"
// @Failure      404 {object} protocol.RespHeader "대상 없음 또는 조회 가능한 대상 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /dm/v01/rmroom/{room_id} [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /dm/v01/rmroom/1
func (cc *ChatController) DeleteChatRoom(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists || user == nil {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	leaverUID := user.(*ptc.UserInfoResp).Uid

	roomID, err := strconv.ParseInt(c.Param("room_id"), 10, 64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "room_id required")
		return
	}

	room, err := cc.hdb.GetDMRoom(roomID)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusNotFound, "room not found", err)
		return
	}
	if leaverUID != room.UID && leaverUID != room.TID {
		cc.ctl.SimpleError(c, http.StatusForbidden, "not a room participant")
		return
	}
	/* 	if !models.IsUserInDMRoom(room.STChat, room.UID, room.TID, leaverUID) {
		cc.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
		return
	} */

	room, err = cc.hdb.LeaveDMRoom(roomID, leaverUID)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to leave dm room", err)
		return
	}

	// 1명만 나간 상태(2=uid 나감, 3=tid 나감)이면 room_id 기준 상대방에게 알림
	if room.STChat == models.STChatUIDLeft || room.STChat == models.STChatTIDLeft {
		cc.notifyPartnerLeft(room, leaverUID)
	}

	if err := cc.setDMHistoryCutoff(leaverUID, roomID); err != nil {
		log.Warn("failed to set dm history cutoff on leave uid=%d room=%d: %v", leaverUID, roomID, err)
	}

	cc.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

// GetChatRooms godoc
// @Summary      DM 채팅방 목록 조회
// @Description  DM 채팅방 목록 조회 API입니다.
// @Description
// @Description  **요청**: path page는 사용하지 않습니다. query page(기본 1)를 전달합니다.
// @Description
// @Description  **처리**: 방 목록을 미읽음 등 기준으로 정렬하고 페이지당 10개를 반환합니다.
// @Description
// @Description  **응답 data**: total_count, rooms.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         DM
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "경로 호환용이며 무시합니다. query 값을 사용하세요."
// @Param        page query int false "실제 조회 페이지" default(1)
// @Success      200 {object} protocol.RespDataHeader "data: total_count, rooms."
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /dm/v01/list/{page} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"total_count":0,"rooms":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"total_count":0,"rooms":[]}}
// @Example Request: GET /dm/v01/list/1
func (cc *ChatController) GetChatRooms(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	pageInt := ptc.CvtParamAtoi(c.DefaultQuery("page", "1"), 1)

	allRooms, totalCount, err := cc.hdb.GetAllDMRoomsByUser(uid64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get dm rooms", err)
		return
	}
	if len(allRooms) == 0 {
		cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{"total_count": 0, "rooms": []ptc.DMRoomResp{}})
		return
	}

	unreadInfo, err := cc.rdb.GetAllUnreadInfoForUser(uid64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get dm unread info", err)
		return
	}

	sorted := sortDMRoomsForInbox(allRooms, unreadInfo)
	pageRooms := paginateDMRooms(sorted, pageInt)

	// 페이지 내 파트너 프로필(DB IN 쿼리)·마지막 메시지(Redis 파이프라인) 일괄 조회 (N+1 제거)
	partnerUIDs := make([]uint64, 0, len(pageRooms))
	roomIDs := make([]string, 0, len(pageRooms))
	for i := range pageRooms {
		partnerUIDs = append(partnerUIDs, models.PartnerUIDOfRoom(&pageRooms[i], uid64))
		roomIDs = append(roomIDs, strconv.FormatInt(pageRooms[i].Idx, 10))
	}

	partnerInfos, err := cc.adb.GetUserInfosByUIDs(partnerUIDs)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get partner infos", err)
		return
	}

	lastMsgs, err := cc.rdb.GetLastMessagesForRooms(uid64, roomIDs)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get last messages", err)
		return
	}

	resp := make([]ptc.DMRoomResp, 0, len(pageRooms))
	for i := range pageRooms {
		rm := &pageRooms[i]
		key := strconv.FormatInt(rm.Idx, 10)
		unread := unreadInfo[key].Count // 맵에 없으면 unread 0

		item, err := cc.buildDMRoomRespPrefetched(rm, uid64, unread, partnerInfos, lastMsgs[key])
		if err != nil {
			cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to build dm room response", err)
			return
		}
		resp = append(resp, item)
	}

	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{"total_count": totalCount, "rooms": resp})
}

// GetChatRoomByRID godoc
// @Summary      DM 채팅방 상세 조회 및 읽음 갱신
// @Description  DM 채팅방 상세 조회 및 읽음 갱신 API입니다.
// @Description
// @Description  **요청**: path room_id, query mid(선택).
// @Description
// @Description  **처리**: 활성 참여자만 조회할 수 있습니다. mid 또는 최신 메시지로 읽음 위치와 미읽음 수를 갱신합니다.
// @Description
// @Description  **응답 data**: room_id, room, my_last_mid, partner_last_mid.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         DM
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        room_id path string true "채팅방 ID"
// @Param        mid query string false "마지막으로 읽은 메시지 ID (생략 시 최신)"
// @Success      200 {object} protocol.RespDataHeader "data: room_id, room, my_last_mid, partner_last_mid."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "권한 없음 또는 차단 관계"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /dm/v01/rinfo/{room_id} [get]
// @Example Request: GET /dm/v01/rinfo/1
func (cc *ChatController) GetChatRoomByRID(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	roomID := c.Param("room_id")
	if roomID == "" {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "room_id required")
		return
	}

	rm, err := cc.hdb.GetDMRoomByRid(roomID)
	if err != nil || rm == nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get dm rooms", err)
		return
	}
	if !models.IsUserInDMRoom(rm.STChat, rm.UID, rm.TID, uid64) {
		cc.ctl.SimpleError(c, http.StatusForbidden, "not a room participant")
		return
	}

	// 방 입장: last_read mid 갱신 + unread reset
	cc.updateRoomEntryRead(uid64, rm.Idx, roomID, c.Query("mid"))

	resp, err := cc.buildDMRoomResp(rm, uid64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to build dm room response", err)
		return
	}

	myMid, _ := cc.rdb.GetLastReadMid(uid64, rm.Idx)
	partnerUID := models.PartnerUIDOfRoom(rm, uid64)
	partnerMid, _ := cc.rdb.GetLastReadMid(partnerUID, rm.Idx)

	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"room_id":          roomID,
		"room":             resp,
		"my_last_mid":      myMid,
		"partner_last_mid": partnerMid,
	})
}

// GetTotalUnread godoc
// @Summary      DM 미읽음 수 조회
// @Description  DM 미읽음 수 조회 API입니다.
// @Description
// @Description  **요청**: 별도 입력 없음.
// @Description
// @Description  **처리**: JWT 사용자의 전체 및 방별 미읽음 수를 조회합니다.
// @Description
// @Description  **응답 data**: total_count, rooms(방 ID별 카운트).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         DM
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Success      200 {object} protocol.RespDataHeader "data: total_count, rooms(방 ID별 카운트)."
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /dm/v01/total/unread [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"total_count":0,"rooms":{}}}
// @x-response-example {"result":0,"resultString":"Success","data":{"total_count":0,"rooms":{}}}
// @Example Request: GET /dm/v01/total/unread
func (cc *ChatController) GetTotalUnread(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	unreadMap, err := cc.rdb.GetAllUnreadForUser(uid64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "failed to get unread", err)
		return
	}
	total := int64(0)
	for _, cnt := range unreadMap {
		total += cnt
	}

	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"total_count": total,
		"rooms":       unreadMap,
	})
}

// GetChatList godoc
// @Summary      DM 메시지 이력 조회
// @Description  DM 메시지 이력 조회 API입니다.
// @Description
// @Description  **요청**: path room_id, query cursor(선택), limit(기본 2). pgSize는 사용하지 않습니다.
// @Description
// @Description  **처리**: 활성 참여자 여부와 방 이력 시작 위치를 적용합니다. 잘못된 cursor는 400입니다. limit ≤ 0이면 빈 목록이며 현재 명시적인 최대 limit 제한은 없습니다.
// @Description
// @Description  **응답 data**: total_count, messages, next_cursor, has_more, my_last_mid, partner_last_mid.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         DM
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        room_id path string true "채팅방 ID"
// @Param        cursor query string false "이전 응답의 next_cursor"
// @Param        limit query int false "요청 메시지 수 (0 이하: 빈 목록)" default(2)
// @Success      200 {object} protocol.RespDataHeader "data: total_count, messages, next_cursor, has_more, my_last_mid, partner_last_mid."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "권한 없음 또는 차단 관계"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /dm/v01/history/{room_id} [get]
// @Example Request: GET /dm/v01/history/1
func (cc *ChatController) GetChatList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	roomID := c.Param("room_id")
	if roomID == "" {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "room_id required")
		return
	}

	rm, err := cc.hdb.GetDMRoomByRid(roomID)
	if err != nil || rm == nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get dm room", err)
		return
	}
	if !models.IsUserInDMRoom(rm.STChat, rm.UID, rm.TID, uid64) {
		cc.ctl.SimpleError(c, http.StatusForbidden, "not a room participant")
		return
	}

	cursor := c.Query("cursor")
	limitNum := ptc.CvtParamAtoi(c.Query("limit"), 2)

	fromIdx := cc.dmHistoryFromIndex(uid64, rm.Idx)
	tcnt, messages, nextCursor, hasMore, err := cc.rdb.GetChatMessagesByCursor(roomID, cursor, limitNum, fromIdx)
	if err != nil {
		if cursor != "" {
			cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
			return
		}
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get chat history", err)
		return
	}

	myMid, _ := cc.rdb.GetLastReadMid(uid64, rm.Idx)
	partnerUID := models.PartnerUIDOfRoom(rm, uid64)
	partnerMid, _ := cc.rdb.GetLastReadMid(partnerUID, rm.Idx)

	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"total_count":      tcnt,
		"messages":         messages,
		"next_cursor":      nextCursor,
		"has_more":         hasMore,
		"my_last_mid":      myMid,
		"partner_last_mid": partnerMid,
	})
}

// SendCallNotification 통화 알림 전송 (SignalingController에서 호출)
func (cc *ChatController) SendCallNotification(msg *ptc.ChatMessage) {
	cc.sendToUser(msg.To, msg)
	log.Info(fmt.Sprintf("Call notification sent: %s -> %s (type: %s)", msg.From, msg.To, msg.Type))
}

// Shutdown은 graceful 종료 시 모든 DM WebSocket 연결을 닫습니다.
func (cc *ChatController) Shutdown() {
	cc.clientsMu.Lock()
	clients := make([]*ChatClient, 0, len(cc.clients))
	for _, client := range cc.clients {
		clients = append(clients, client)
	}
	cc.clientsMu.Unlock()

	for _, client := range clients {
		client.conn.Close()
	}
}

// IsUserOnline 채팅 WS 온라인 여부 반환
func (cc *ChatController) IsUserOnline(userID string) bool {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return false
	}
	cc.clientsMu.RLock()
	_, exists := cc.clients[uid]
	cc.clientsMu.RUnlock()
	if exists {
		return true
	}

	online, onlineErr := cc.rdb.IsOnline(uid)
	if onlineErr != nil {
		return false
	}
	return online
}

// SendDMNotification cross-controller DM 알림 전달
func (cc *ChatController) SendDMNotification(toUserID string, msg *ptc.ChatMessage) {
	if msg == nil {
		return
	}
	cp := *msg
	cp.Type = "dm-incoming"
	cp.To = toUserID
	cc.sendToUser(toUserID, &cp)
}

// UploadImage godoc
// @Summary      DM 이미지 업로드
// @Description  DM 이미지 업로드 API입니다.
// @Description
// @Description  **요청**: multipart files: 최대 5개, 각 5MiB 이하.
// @Description
// @Description  **처리**: JWT 인증과 simpleImgUploadCF 검증·업로드 후 파일명별 URL을 반환합니다.
// @Description
// @Description  **응답 data**: msg, data(파일명 → URL 맵).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Description  업로드 미들웨어 오류는 JSON 문자열 또는 error 객체일 수 있으며 핸들러의 RespHeader 오류와 형식이 다릅니다.
// @Description  **업로드 공통 제한**: 요청 전체 크기·동시 업로드 제한을 초과하면 413/503입니다. 기본값은 530MiB/8건이며 서버 설정에 따라 달라집니다.
// @Tags         DM
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        files formData file true "업로드 파일 (동일 필드명으로 여러 파일 전송 가능)"
// @Success      200 {object} protocol.RespDataHeader "data: msg, data(파일명 → URL 맵)."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      413 {object} map[string]interface{} "업로드 요청 크기 제한 초과"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Failure      503 {object} map[string]interface{} "서버 동시 처리·연결 한도 초과"
// @Router       /dm/v01/upload/img [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","data":{"photo.jpg":"https://imagedelivery.net/example/public"}}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","data":{"photo.jpg":"https://imagedelivery.net/example/public"}}}
// @Example Request: POST /dm/v01/upload/img
func (cc *ChatController) UploadImage(c *gin.Context) {
	// JWT 인증 검사
	_, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// 파일 파라미터 취득
	files, ok := c.Get("upFiles")
	if !ok {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "file required")
		return
	}

	fps := files.([]*multipart.FileHeader)
	// Cloudflare에 이미지 업로드
	cldFlrInfos, err := utils.UploadCldFlrImg(fps, cc.cfg.Server.CfId, cc.cfg.Server.CfToken)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload image", err)
		return
	}

	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"msg":  "ok",
		"data": cldFlrInfos, // 업로드된 Cloudflare 이미지 URL (파일명: URL 맵)
	})
}

/*
func (cc *ChatController) resolveUIDFromRequest(c *gin.Context, queryUserID string) (uint64, error) {
	if queryUserID != "" {
		uid, err := strconv.ParseUint(queryUserID, 10, 64)
		if err == nil {
			return uid, nil
		}
	}

	userAny, exists := c.Get("user")
	if !exists {
		return 0, fmt.Errorf("userId required")
	}

	switch user := userAny.(type) {
	case *ptc.UserInfoResp:
		return user.Uid, nil
	case ptc.UserInfoResp:
		return user.Uid, nil
	default:
		return 0, fmt.Errorf("invalid auth context")
	}
}
*/
