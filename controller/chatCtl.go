package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	log "ms-gateway/common/logger"
	"ms-gateway/conf"
	"ms-gateway/models"
	ptl "ms-gateway/protocol"

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
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	uid      uint64
	mu       sync.Mutex
	lastSeen time.Time
}

// ChatController 텍스트 채팅 컨트롤러
type ChatController struct {
	ctl       *Controller
	cfg       *conf.Config
	rep       *models.Repositories
	rdb       *models.RedisDB
	adb       *models.AccountDB
	hdb       *models.HistoryDB
	clients   map[string]*ChatClient // userID -> ChatClient
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader
}

// NewChatController ChatController 생성
func NewChatController(ctl *Controller, rep *models.Repositories) (*ChatController, error) {
	r := &ChatController{
		ctl:     ctl,
		rep:     rep,
		cfg:     ctl.cfg,
		clients: make(map[string]*ChatClient),
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

// HandleWebSocket 텍스트 채팅 WebSocket 연결 핸들러
//
// [송신자 정책] 현재: userId 쿼리 파라미터 기반 (테스트용)
// TODO: JWT 적용 시 아래 주석 블록을 활성화하여 토큰에서 uid를 강제 추출하고,
//
//	userId 파라미터는 무시하도록 전환할 것.
//
// [수신자 정책] 수신자는 로그인 여부와 무관하게 메시지를 수신할 수 있어야 함:
//   - Chat WS 연결 중이면 실시간 전달
//   - 미연결(로그아웃/오프라인)이면 FCM Push로 전달
func (cc *ChatController) HandleWebSocket(c *gin.Context) {
	// TODO(JWT): 아래 블록으로 교체 — JWT에서 uid 강제 추출
	// authHeader := c.GetHeader("Authorization")
	// if authHeader == "" { authHeader = "Bearer " + c.Query("token") }
	// userInfo, err := cc.rdb.HGetJWTAccess(extractBearerToken(authHeader))
	// if err != nil { conn upgrade 거절; return }
	// userId := strconv.FormatUint(userInfo.Uid, 10)
	// uid := userInfo.Uid

	userId := c.Query("userId")
	if userId == "" {
		log.Warn("WebSocket connection attempt without userId")
		cc.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "userId required"), http.StatusBadRequest, nil)
		return
	}

	uid, err := strconv.ParseUint(userId, 10, 64)
	if err != nil {
		log.Warn("Invalid userId for chat websocket: %s", userId)
		cc.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "userId must be numeric"), http.StatusBadRequest, nil)
		return
	}

	// WebSocket 업그레이드
	conn, err := cc.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade failed:", err)
		return
	}

	// 클라이언트 생성 및 등록
	client := &ChatClient{
		conn:     conn,
		send:     make(chan []byte, CHAT_MSG_BUFFER_SIZE),
		userID:   userId,
		uid:      uid,
		lastSeen: time.Now(),
	}

	cc.registerClient(client)

	log.Info(fmt.Sprintf("New chat WebSocket connection: %s", userId))

	// 고루틴 시작
	go cc.writePump(client)
	go cc.readPump(client)
}

// registerClient 클라이언트 등록
func (cc *ChatController) registerClient(client *ChatClient) {
	cc.clientsMu.Lock()
	defer cc.clientsMu.Unlock()

	// 기존 연결이 있으면 종료
	if oldClient, exists := cc.clients[client.userID]; exists {
		close(oldClient.send)
		oldClient.conn.Close()
	}

	cc.clients[client.userID] = client
	if err := cc.rdb.SetOnline(client.uid); err != nil {
		log.Warn("failed to set online state for %s: %v", client.userID, err)
	}
	log.Info(fmt.Sprintf("Chat client registered: %s", client.userID))
}

// unregisterClient 클라이언트 등록 해제
func (cc *ChatController) unregisterClient(client *ChatClient) {
	cc.clientsMu.Lock()
	defer cc.clientsMu.Unlock()

	if _, exists := cc.clients[client.userID]; exists {
		delete(cc.clients, client.userID)
		close(client.send)
		if err := cc.rdb.DeleteOnline(client.uid); err != nil {
			log.Warn("failed to delete online state for %s: %v", client.userID, err)
		}
		log.Info(fmt.Sprintf("Chat client unregistered: %s", client.userID))
	}
}

// readPump 메시지 읽기
func (cc *ChatController) readPump(client *ChatClient) {
	defer func() {
		cc.unregisterClient(client)
		client.conn.Close()
	}()

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
				w.Write(<-client.send)
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
	var msg ptl.ChatMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error("Chat message parse error:", err)
		return
	}

	msg.From = client.userID
	msg.Timestamp = time.Now().Unix()

	switch msg.Type {
	case "text-message":
		cc.handleTextMessage(client, &msg)
	case "typing":
		cc.handleTyping(client, &msg)
	case "read-receipt":
		cc.handleReadReceipt(client, &msg)
	case "call-request":
		cc.handleCallRequestFromDM(client, &msg)
	case "call-accept":
		cc.handleCallAcceptFromDM(client, &msg)
	case "call-reject":
		cc.sendToUser(msg.To, &msg)
	case "call-cancel":
		cc.handleCallCancel(client, &msg)
	default:
		log.Warn(fmt.Sprintf("Unknown chat message type: %s from %s", msg.Type, client.userID))
	}
}

// handleTextMessage 텍스트 메시지 처리
func (cc *ChatController) handleTextMessage(client *ChatClient, msg *ptl.ChatMessage) {
	toUID, err := strconv.ParseUint(msg.To, 10, 64)
	if err != nil {
		log.Warn(fmt.Sprintf("invalid target uid in text-message: %s", msg.To))
		return
	}

	tUser, tUserErr := cc.adb.GetUserInfoByUID(toUID)

	var roomID string
	var unreadCnt int64

	if tUserErr == nil {
		room, roomErr := cc.ensureDMRoom(client.uid, &tUser)
		if roomErr != nil {
			log.Error(fmt.Sprintf("ensureDMRoom failed: %v", roomErr))
		} else {
			roomID = strconv.FormatInt(room.Idx, 10)
			unreadCnt, err = cc.rdb.IncrUnread(toUID, room.Idx)
			if err != nil {
				log.Error(fmt.Sprintf("failed to increment unread: %v", err))
			}
		}
	} else {
		log.Warn(fmt.Sprintf("target user %d not found in DB, delivering message without room: %v", toUID, tUserErr))
	}

	if roomID != "" {
		msg.RoomID = roomID
	}

	delivered := cc.sendToUser(msg.To, msg)
	if !delivered {
		if tUserErr == nil && cc.ctl.FCMPusher != nil {
			cc.ctl.FCMPusher.SendDMPush(client.userID, msg.Content, tUser.Did)
		}
	}

	ack := &ptl.ChatMessage{
		Type:      "msg-ack",
		From:      msg.To,
		To:        msg.From,
		RoomID:    msg.RoomID,
		MsgID:     msg.MsgID,
		Unread:    int(unreadCnt),
		Timestamp: time.Now().Unix(),
	}
	cc.sendToUser(client.userID, ack)

	log.Info(fmt.Sprintf("Text message sent: %s -> %s (room=%s, delivered=%v)", msg.From, msg.To, msg.RoomID, delivered))
}

// handleTyping 타이핑 상태 처리
func (cc *ChatController) handleTyping(client *ChatClient, msg *ptl.ChatMessage) {
	// 수신자에게 타이핑 상태 전송
	cc.sendToUser(msg.To, msg)
}

// handleReadReceipt 읽음 확인 처리
func (cc *ChatController) handleReadReceipt(client *ChatClient, msg *ptl.ChatMessage) {
	if msg.RoomID != "" {
		roomID, err := strconv.ParseInt(msg.RoomID, 10, 64)
		if err != nil {
			log.Warn("invalid roomId in read-receipt: %s", msg.RoomID)
			return
		}
		if err := cc.rdb.ResetUnread(client.uid, roomID); err != nil {
			log.Warn("failed to reset unread for %s room %s: %v", client.userID, msg.RoomID, err)
		}
	}
	cc.sendToUser(msg.To, msg)
}

// sendToUser 특정 사용자에게 메시지 전송
func (cc *ChatController) sendToUser(userID string, msg *ptl.ChatMessage) bool {
	cc.clientsMu.RLock()
	client, exists := cc.clients[userID]
	cc.clientsMu.RUnlock()

	if !exists {
		log.Warn(fmt.Sprintf("User %s not connected to chat WebSocket", userID))
		return false
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Error("Message marshal error:", err)
		return false
	}

	if !cc.trySend(client, data) {
		log.Warn(fmt.Sprintf("Client %s send buffer full", userID))
		return false
	}
	return true
}

func (cc *ChatController) trySend(client *ChatClient, data []byte) (sent bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("chat safe send recovered (%s): %v", client.userID, r)
			sent = false
		}
	}()
	select {
	case client.send <- data:
		return true
	default:
		return false
	}
}

// ensureDMRoom DM 룸 조회/생성.
// [룸 정책] DM 룸은 반드시 dm_room 테이블로 관리한다.
//
//	signaling의 waitingRoom(통화 대기 전용, TTL 있음)과 혼용 금지.
func (cc *ChatController) ensureDMRoom(uid uint64, tUser *ptl.UserInfoResp) (*models.DMRoomRow, error) {
	room, err := cc.hdb.GetDMRoomByPair(uid, tUser.Uid)
	if err == nil {
		//차단 리스트
		if room.STChat == 0 {
			if actErr := cc.hdb.ActivateDMRoomByPair(uid, tUser.Uid); actErr != nil {
				return nil, actErr
			}
			return cc.hdb.GetDMRoomByPair(uid, tUser.Uid)
		}
		return room, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	rmid, createErr := cc.hdb.CreateDMRoom(uid, tUser)
	if createErr != nil {
		return nil, createErr
	}

	resRoom, err := cc.hdb.GetDMRoom(rmid)
	if err != nil {
		return nil, err
	}

	return resRoom, nil
}

func (cc *ChatController) buildPartnerInfoByUID(uid uint64) (*ptl.PartnerInfo, error) {
	info, err := cc.adb.GetUserInfoByUID(uid)
	if err != nil {
		return nil, err
	}
	return &ptl.PartnerInfo{
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
func (cc *ChatController) handleCallRequestFromDM(client *ChatClient, msg *ptl.ChatMessage) {
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
	cc.ctl.FCMPusher.SendCallPush(client.userID, "", msg.CallMode, receiver.Did)

	notice := &ptl.ChatMessage{
		Type:      "call-info",
		From:      msg.To,
		To:        msg.From,
		RoomID:    msg.RoomID,
		Content:   "상대방이 오프라인 상태여서 푸시 알림을 전송했습니다.",
		Timestamp: time.Now().Unix(),
	}
	cc.sendToUser(client.userID, notice)
}

func (cc *ChatController) handleCallAcceptFromDM(client *ChatClient, msg *ptl.ChatMessage) {
	partner, err := cc.buildPartnerInfoByUID(client.uid)
	if err == nil {
		msg.Partner = partner
	}
	cc.sendToUser(msg.To, msg)
}

func (cc *ChatController) handleCallCancel(client *ChatClient, msg *ptl.ChatMessage) {
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
		cc.ctl.FCMPusher.SendCallPush(client.userID, "", "cancel", receiver.Did)
	}
}

// CreateChatRoom 채팅방 생성
func (cc *ChatController) CreateChatRoom(c *gin.Context) {
	var req struct {
		UID uint64 `json:"uid"`
		TID uint64 `json:"tid"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		cc.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// myUID, err := cc.resolveUIDFromRequest(c, req.UID)
	// if err != nil {
	// 	cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
	// 	return
	// }
	/*
		user, err := cc.adb.GetUserInfoByUID(req.UID)
		if err != nil {
			cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
			return
		}
	*/
	tUser, err := cc.adb.GetUserInfoByUID(req.TID)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
		return
	}

	rm, err := cc.ensureDMRoom(req.UID, &tUser)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to create dm room", err)
		return
	}

	/* partner, perr := cc.buildPartnerInfoByUID(req.TargetUID)
	if perr != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to load partner info", perr)
		return
	} */

	partner := &ptl.PartnerInfo{
		PID:      rm.TID,
		Nick:     rm.TNick,
		ThumbPic: rm.TThumbUrl,
		Gender:   strconv.Itoa(rm.TGender),
		Age:      strconv.Itoa(rm.TAge),
		Area:     rm.TArea,
	}

	unread, _ := cc.rdb.GetUnread(req.UID, rm.Idx)
	resp := ptl.DMRoomResp{
		RoomID:   rm.Idx,
		Partner:  partner,
		Unread:   int(unread),
		AtCreate: rm.AtCrtCHAT.Format(time.RFC3339),
		AtUpdate: rm.AtUpdate.Format(time.RFC3339),
	}

	cc.ctl.SendDataResponse(c, http.StatusOK, resp)
}

// GetChatRooms retrieves the list of DM chat rooms for a user.
//
// @Summary      Get User DM Chat Rooms
// @Description  Retrieves the list of direct message chat rooms that the specified user is participating in.
// @Description  Todo: JWT Auth
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        userId  query     string  true  "User UID"
// @Success      200     {array}   protocol.DMRoomResp
// @Failure      400     {object}  protocol.SimpleErrorResp "Bad Request"
// @Failure      401     {object}  protocol.SimpleErrorResp "Unauthorized"
// @Failure      500     {object}  protocol.SimpleErrorResp "Internal Server Error"
// @Router       /dm/v01/rooms [get]
//
// @Example
// GET /dm/v01/rooms?userId=10001 HTTP/1.1
// Host: localhost:8080
//
// Response (200 OK)
// [
//   {
//     "roomId": 12345,
//     "partner": {
//       "pid": 20002,
//       "nick": "testuser",
//       "thumbPic": "https://cdn.example.com/avatar.jpg",
//       "gender": "1",
//       "age": "22",
//       "area": "Seoul"
//     },
//     "unread": 3,
//     "atCreate": "2024-06-11T12:00:00Z",
//     "atUpdate": "2024-06-12T08:00:00Z"
//   }
// ]

func (cc *ChatController) GetChatRooms(c *gin.Context) {
	//Todo: JWT Auth 처리
	/*
		user, exists := c.Get("user")
		if !exists {
			cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
			return
		}

		uid := user.(*ptl.UserInfoResp).Uid
	*/
	// userID := c.Query("userId")
	// uid, err := cc.resolveUIDFromRequest(c, userID)
	// if err != nil {
	// 	cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
	// 	return
	// }

	uid := c.Query("userId")
	uid64, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
		return
	}

	rooms, err := cc.hdb.GetDMRoomsByUser(uid64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get dm rooms", err)
		return
	}

	resp := make([]ptl.DMRoomResp, 0, len(*rooms))
	for _, rm := range *rooms {
		partner := &ptl.PartnerInfo{
			PID:      rm.TID,
			Nick:     rm.TNick,
			ThumbPic: rm.TThumbUrl,
			Gender:   strconv.Itoa(rm.TGender),
			Age:      strconv.Itoa(rm.TAge),
			Area:     rm.TArea,
		}

		unread, _ := cc.rdb.GetUnread(uid64, rm.Idx)
		resp = append(resp, ptl.DMRoomResp{
			RoomID:   rm.Idx,
			Partner:  partner,
			Unread:   int(unread),
			AtCreate: rm.AtCrtCHAT.Format(time.RFC3339),
			AtUpdate: rm.AtUpdate.Format(time.RFC3339),
		})
	}

	cc.ctl.SendDataResponse(c, http.StatusOK, resp)
}

// GetTotalUnread 사용자 전체 unread 합계 조회
// GetTotalUnread godoc
// @Summary     Get total unread message count for user
// @Description Returns the sum of unread messages across all chat rooms for the user, as well as per-room counts
// @Tags        chat
// @Accept      json
// @Produce     json
// @Param       userId query string true "User ID"
// @Success     200 {object} map[string]interface{} "uid: user ID, total: total unread count, rooms: per-room unread counts"
// @Failure     400 {object} map[string]string "Bad request"
// @Failure     500 {object} map[string]string "Internal server error"
// @Router      /dm/v01/total/unread [get]
//
// @Example Request:
//
//	GET /dm/v01/total/unread?userId=12345 HTTP/1.1
//	Host: localhost:8080
//
// @Example Success Response:
//
//	HTTP/1.1 200 OK
//	Content-Type: application/json
//	{
//	  "uid": "12345",
//	  "total": 7,
//	  "rooms": {
//	    "101": 5,
//	    "202": 2
//	  }
//	}
func (cc *ChatController) GetTotalUnread(c *gin.Context) {
	userID := c.Query("userId")
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
		return
	}
	unreadMap, err := cc.rdb.GetAllUnreadForUser(uid)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "failed to get unread", err)
		return
	}
	total := int64(0)
	for _, cnt := range unreadMap {
		total += cnt
	}

	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"uid":   userID,
		"total": total,
		"rooms": unreadMap,
	})
}

// GetChatHistory 채팅 기록 조회
func (cc *ChatController) GetChatHistory(c *gin.Context) {
	roomID := c.Param("roomId")
	if roomID == "" {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "roomId required")
		return
	}
	/*
		// 페이지네이션 파라미터
		page := c.DefaultQuery("page", "1")
		limit := c.DefaultQuery("limit", "50")

		pageNum, err := strconv.Atoi(page)
		if err != nil {
			pageNum = 1
		}

		limitNum, err := strconv.Atoi(limit)
		if err != nil {
			limitNum = 50
		}

		offset := (pageNum - 1) * limitNum

		// Redis에서 메시지 조회
		messages, err := cc.rdb.GetChatMessages(roomID, offset, limitNum)
		if err != nil {
			cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get chat history", err)
			return
		}

		cc.ctl.SendDataResponse(c, http.StatusOK, messages)
	*/
}

// SendMessage REST API를 통한 메시지 전송
func (cc *ChatController) SendMessage(c *gin.Context) {
	var req struct {
		RoomID  string `json:"roomId"`
		UserID  string `json:"userId"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	/*
		 	// Redis에 메시지 저장
			if err := cc.rdb.SaveChatMessage(req.RoomID, req.UserID, req.Content); err != nil {
				cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to save message", err)
				return
			}

			cc.ctl.SendResponse(c, http.StatusOK, "Message sent")
	*/
}

// SendCallNotification 통화 알림 전송 (SignalingController에서 호출)
func (cc *ChatController) SendCallNotification(msg *ptl.ChatMessage) {
	cc.sendToUser(msg.To, msg)
	log.Info(fmt.Sprintf("Call notification sent: %s -> %s (type: %s)", msg.From, msg.To, msg.Type))
}

// IsUserOnline 채팅 WS 온라인 여부 반환
func (cc *ChatController) IsUserOnline(userID string) bool {
	cc.clientsMu.RLock()
	_, exists := cc.clients[userID]
	cc.clientsMu.RUnlock()
	if exists {
		return true
	}

	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return false
	}
	online, onlineErr := cc.rdb.IsOnline(uid)
	if onlineErr != nil {
		return false
	}
	return online
}

// SendDMNotification cross-controller DM 알림 전달
func (cc *ChatController) SendDMNotification(toUserID string, msg *ptl.ChatMessage) {
	if msg == nil {
		return
	}
	cp := *msg
	cp.Type = "dm-incoming"
	cp.To = toUserID
	cc.sendToUser(toUserID, &cp)
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
	case *ptl.UserInfoResp:
		return user.Uid, nil
	case ptl.UserInfoResp:
		return user.Uid, nil
	default:
		return 0, fmt.Errorf("invalid auth context")
	}
}
*/
