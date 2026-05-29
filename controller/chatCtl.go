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
	// userID   string
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
	clients   map[uint64]*ChatClient // uid -> ChatClient
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader
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

// @Summary      Connect DM WebSocket
// @Description  Upgrades HTTP connection to WebSocket for DM real-time messaging. This endpoint requires JWT authentication and uses the authenticated user's UID from context.
// @Description  Request example: GET /dm/v01/ws HTTP/1.1, Host: api.example.com, Upgrade: websocket, Connection: Upgrade.
// @Description  Response example: 101 Switching Protocols (WebSocket handshake success; binary/text frames exchanged).
// @Description  This endpoint is for authenticated sessions, and user identity is derived from JWT context.
// @Description  Legacy note: some clients may still include userId query for compatibility, but auth source is the access token.
// @Description  WebSocket message request examples:
// @Description  1) text-message: {"type":"text-message","to":"456","content":"Hello, how are you?","callMode":"txt|img|...","msgId":"1234567890"}
// @Description  1-1) text-message-ack: {"type":"msg-ack","from":"123","to":"456","roomId":"1234567890","msgId":"1234567890","unread":1,"timestamp":1718851200}
// @Description  2) typing(optional): {"type":"typing","to":"456","roomId":"1234567890"}
// @Description  3) read-receipt: {"type":"read-receipt","to":"456","roomId":"1234567890"}
// @Description  DM Client Work flow:
// @Description  1) connection : 로그인시 ws 연결
// @Description  1-1) {domain}/dm/v01/ws GET 로그인후
// @Description  2) request : 신규 파트너 dm 요청시 방 생성 및 채팅 시작
// @Description  2-1) {domain}/dm/v01/mkroom/:pid POST 신규 파트너 dm 요청시 방 생성 및 채팅 시작
// @Description  3) chatlist : 기존 채팅방 리스트 요청
// @Description  3-1) {domain}/dm/v01/list/:page GET 기존 채팅방 리스트 요청
// @Description  4) totalunread : 총 미읽음 메시지 수 조회
// @Description  4-1) {domain}/dm/v01/total/unread GET 총 미읽음 메시지 수 조회
// @Description  5) read-receipt : 특정 채팅방 입장시 미읽음 초기화
// @Description  5-1) Type : "read-receipt", To : 수신자 UID, RoomID : 채팅방 ID
// @Description  6) text-message : 텍스트 메시지 전송
// @Description  6-1) Type : "text-message", To : 수신자 UID, Content : 텍스트 메시지, MsgID : 메시지 식별자(m-{timestamp(unixtime)})
// @Description  6-2) msg-ack : 메시지 전송 확인 서버에서 보낸사람에게 전송, 수신자에게는 전송하지 않음.
// @Description  7) 로그아웃 : 로그아웃시 ws disconnect
// @Description  7-1) 상대방에게 로그아웃은 전송하지 않음. 로그아웃시에도 전송가능
// @Tags         chat
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer access token. Example: Bearer {access_token}"
// @Success      101  {string}  string  "Switching Protocols (WebSocket handshake success)"
// @Failure      401  {object}  protocol.RespHeader  "Unauthorized"
// @Failure      500  {object}  protocol.RespHeader  "Internal Server Error"
// @Router       /dm/v01/ws [get]
func (cc *ChatController) HandleWebSocket(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid
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
		uid:      uid64,
		lastSeen: time.Now(),
	}

	cc.registerClient(client)

	log.Info(fmt.Sprintf("New chat WebSocket connection: %d", uid64))

	// 고루틴 시작
	go cc.writePump(client)
	go cc.readPump(client)
}

// registerClient 클라이언트 등록
func (cc *ChatController) registerClient(client *ChatClient) {
	cc.clientsMu.Lock()
	defer cc.clientsMu.Unlock()

	// 기존 연결이 있으면 종료
	if oldClient, exists := cc.clients[client.uid]; exists {
		close(oldClient.send)
		oldClient.conn.Close()
	}

	cc.clients[client.uid] = client
	if err := cc.rdb.SetOnline(client.uid); err != nil {
		log.Warn("failed to set online state for %d: %v", client.uid, err)
	}
	log.Info(fmt.Sprintf("Chat client registered: %d", client.uid))
}

// unregisterClient 클라이언트 등록 해제
func (cc *ChatController) unregisterClient(client *ChatClient) {
	cc.clientsMu.Lock()
	defer cc.clientsMu.Unlock()

	if _, exists := cc.clients[client.uid]; exists {
		delete(cc.clients, client.uid)
		close(client.send)
		if err := cc.rdb.DeleteOnline(client.uid); err != nil {
			log.Warn("failed to delete online state for %d: %v", client.uid, err)
		}
		log.Info(fmt.Sprintf("Chat client unregistered: %d", client.uid))
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
	case "call-request":
		cc.handleCallRequestFromDM(client, &msg)
	case "call-accept":
		cc.handleCallAcceptFromDM(client, &msg)
	case "call-reject":
		cc.sendToUser(msg.To, &msg)
	case "call-cancel":
		cc.handleCallCancel(client, &msg)
	default:
		log.Warn(fmt.Sprintf("Unknown chat message type: %s from %d", msg.Type, client.uid))
	}
}

// handleTextMessage 텍스트 메시지 처리
func (cc *ChatController) handleTextMessage(client *ChatClient, msg *ptc.ChatMessage) {
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
			cc.ctl.FCMPusher.SendDMPush(strconv.FormatUint(client.uid, 10), msg.Content, tUser.Did)
		}
	}

	ack := &ptc.ChatMessage{
		Type:      "msg-ack",
		From:      msg.To,
		To:        msg.From,
		RoomID:    msg.RoomID,
		MsgID:     msg.MsgID,
		Unread:    int(unreadCnt),
		Timestamp: time.Now().Unix(),
	}
	cc.sendToUser(strconv.FormatUint(client.uid, 10), ack)

	if err := cc.rdb.SaveChatMessage(msg.RoomID, msg.From, msg.Content, msg.CallMode); err != nil {
		log.Error("Failed to save chat message:", err)
		return
	}

	log.Info(fmt.Sprintf("Text message sent: %s -> %s (room=%s, delivered=%v)", msg.From, msg.To, msg.RoomID, delivered))
}

// handleTyping 타이핑 상태 처리
func (cc *ChatController) handleTyping(client *ChatClient, msg *ptc.ChatMessage) {
	// 수신자에게 타이핑 상태 전송
	cc.sendToUser(msg.To, msg)
}

// handleReadReceipt 읽음 확인 처리
func (cc *ChatController) handleReadReceipt(client *ChatClient, msg *ptc.ChatMessage) {
	if msg.RoomID != "" {
		roomID, err := strconv.ParseInt(msg.RoomID, 10, 64)
		if err != nil {
			log.Warn("invalid roomId in read-receipt: %s", msg.RoomID)
			return
		}
		if err := cc.rdb.ResetUnread(client.uid, roomID); err != nil {
			log.Warn("failed to reset unread for %d room %s: %v", client.uid, msg.RoomID, err)
		}
	}
	cc.sendToUser(msg.To, msg)
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

func (cc *ChatController) trySend(client *ChatClient, data []byte) (sent bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("chat safe send recovered (%d): %v", client.uid, r)
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
func (cc *ChatController) ensureDMRoom(uid uint64, tUser *ptc.UserInfoResp) (*models.DMRoomRow, error) {
	room, err := cc.hdb.GetDMRoomByPair(uid, tUser.Uid)
	if err == nil {
		//TODO : 차단 리스트
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

	//TODO : 과금
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
		Age:      strconv.Itoa(utils.CalcBirth2Age(info.Birth)),
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
// @Summary DM create chat room
// @Description 1:1 채팅방(DM)을 생성. 생성된 방의 정보(방 아이디, 파트너 정보, 미확인 메시지 수, 생성/갱신 시각)를 반환
// @Tags chat
// @Accept json
// @Produce json
// @Param request body object{uid=uint64,tid=uint64} true "pid  partner UID"
// @Success 200 {object} protocol.DMRoomResp "채팅방 생성 성공 및 방 정보"
// @Failure 400 {object} protocol.RespHeader "잘못된 요청"
// @Failure 401 {object} protocol.RespHeader "인증 실패"
// @Failure 500 {object} protocol.RespHeader "서버 내부 오류"
// @Router /dm/v01/mkroom/{pid} [post]
//
// @Example request
// POST /dm/v01/mkroom/{pid} HTTP/1.1
// Content-Type: application/json
//
// @Example success response
//
//	{
//	 "roomId": 123234,
//	 "partner": {
//	   "pid": 234,
//	   "nick": "testuser",
//	   "thumbPic": "https://cdn.example.com/avatar.jpg",
//	   "gender": "1",
//	   "age": "22",
//	   "area": "Seoul"
//	 },
//	 "unread": 0,
//	 "atCreate": "2024-06-11T12:00:00Z",
//	 "atUpdate": "2024-06-12T08:00:00Z"
//	}
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

	partner := &ptc.PartnerInfo{
		PID:      rm.TID,
		Nick:     rm.TNick,
		ThumbPic: rm.TThumbUrl,
		Gender:   strconv.Itoa(rm.TGender),
		Age:      strconv.Itoa(rm.TAge),
		Area:     rm.TArea,
	}

	unread, _ := cc.rdb.GetUnread(uid64, rm.Idx)
	resp := ptc.DMRoomResp{
		RoomID:   rm.Idx,
		Partner:  partner,
		Unread:   int(unread),
		AtCreate: rm.AtCrtCHAT.Format(time.RFC3339),
		AtUpdate: rm.AtUpdate.Format(time.RFC3339),
	}

	cc.ctl.SendDataResponse(c, http.StatusOK, resp)
}

// GetChatRooms godoc
// @Summary      Get User DM Chat Rooms
// @Description  Retrieves the list of direct message chat rooms that the currently authenticated user is participating in. Note: 'userId' query parameter is no longer required or used. Room list is based on the authenticated session.
// @Description  request : GET /dm/v01/list/1 HTTP/1.1
// @Description  response : {"result":0,"resultString":"Success","data":{"rooms":[{"rid":10,"partner":{"pid":4033287471439576593,"nick":"푸른 아름다운 양","thumb_pic":"https://i.ibb.co/99MhfMXt/icon-male-03.webp","gender":"1","age":"34","area":"서울"},"total":0,"unread":133,"at_crtchat":"2026-04-28T11:43:25Z","at_update":"2026-04-28T11:43:25Z"},{"rid":9,"partner":{"pid":5709326013809104361,"nick":"평화로운 아기 바나나","thumb_pic":"https://i.ibb.co/99MhfMXt/icon-male-03.webp","gender":"1","age":"34","area":"서울"},"total":0,"unread":0,"at_crtchat":"2026-04-20T13:27:53Z","at_update":"2026-04-20T13:27:53Z"}],"total_count":2}}
// @Description  pagesize = 10
// @Tags         chat
// @Accept       json
// @Produce      json
// @Success      200 {array}   protocol.DMRoomResp
// @Failure      400 {object} map[string]string "Bad Request"
// @Failure      401 {object} map[string]string "Unauthorized"
// @Failure      500 {object} map[string]string "Internal Server Error"
// @Router       /dm/v01/list/{page} [get]
//
// Host: localhost:8080
// Authorization: Bearer {access_token}
//
// @Example Resp:
//
// {"result":0,"resultString":"Success","data":{"rooms":[{"rid":10,"partner":{"pid":4033287471439576593,"nick":"푸른 아름다운 양","thumbPic":"https://i.ibb.co/99MhfMXt/icon-male-03.webp","gender":"1","age":"34","area":"서울"},"total":0,"unread":7,"at_crtchat":"2026-04-28T11:43:25Z","at_update":"2026-04-28T11:43:25Z"},{"rid":9,"partner":{"pid":5709326013809104361,"nick":"평화로운 아기 바나나","thumbPic":"https://i.ibb.co/99MhfMXt/icon-male-03.webp","gender":"1","age":"34","area":"서울"},"total":0,"unread":0,"at_crtchat":"2026-04-20T13:27:53Z","at_update":"2026-04-20T13:27:53Z"}],"totalcount":1}}
//
// @Example Success Response:
//
//	HTTP/1.1 200 OK
//
// Content-Type: application/json
// {"result":0,"resultString":"Success","data":{"rooms":[{"rid":10,"partner":{"pid":4033287471439576593,"nick":"푸른 아름다운 양","thumb_pic":"https://i.ibb.co/99MhfMXt/icon-male-03.webp","gender":"1","age":"34","area":"서울"},"total":0,"unread":30,"at_crtchat":"2026-04-28T11:43:25Z","at_update":"2026-04-28T11:43:25Z"},{"rid":9,"partner":{"pid":5709326013809104361,"nick":"평화로운 아기 바나나","thumb_pic":"https://i.ibb.co/99MhfMXt/icon-male-03.webp","gender":"1","age":"34","area":"서울"},"total":0,"unread":0,"at_crtchat":"2026-04-20T13:27:53Z","at_update":"2026-04-20T13:27:53Z"}],"total_count":1}}
func (cc *ChatController) GetChatRooms(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		cc.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	page := c.Param("page")
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		pageInt = 1
	}

	rooms, totalCount, err := cc.hdb.GetDMRoomsByUser(uid64, pageInt)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get dm rooms", err)
		return
	}

	resp := make([]ptc.DMRoomResp, 0, len(*rooms))
	for _, rm := range *rooms {
		partner := &ptc.PartnerInfo{
			PID:      rm.TID,
			Nick:     rm.TNick,
			ThumbPic: rm.TThumbUrl,
			Gender:   strconv.Itoa(rm.TGender),
			Age:      strconv.Itoa(rm.TAge),
			Area:     rm.TArea,
		}

		tcnt, messages, err := cc.rdb.GetChatMessages(strconv.FormatInt(rm.Idx, 10), 0, 1)
		if err != nil {
			if tcnt != 1 {
				messages = &[]models.ChatMessageData{}
			}
		}

		unread, _ := cc.rdb.GetUnread(uid64, rm.Idx)
		resp = append(resp, ptc.DMRoomResp{
			RoomID:   rm.Idx,
			Partner:  partner,
			LastMsg:  (*messages)[0].Content,
			Unread:   int(unread),
			AtCreate: rm.AtCrtCHAT.Format(time.RFC3339),
			AtUpdate: rm.AtUpdate.Format(time.RFC3339),
		})
	}

	// cc.ctl.SendDataResponse(c, http.StatusOK, ptc.NewRespDataHeader(ptc.Success, gin.H{"total": totalCount, "rooms": resp}))
	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{"total_count": totalCount, "rooms": resp})
}

// GetTotalUnread 사용자 전체 unread 합계 조회
// GetTotalUnread godoc
// @Summary     Get total unread message count for user
// @Description Returns the sum of unread messages across all chat rooms for the user, as well as per-room counts. (변경사항 반영됨: userId 파라미터는 더 이상 필요하지 않음, 인증된 사용자 기준)
// @Description  request : GET /dm/v01/total/unread HTTP/1.1
// @Description  response : {"result":0,"resultString":"Success","data":{"rooms":{"10":133},"total_count":133}}
// @Tags        chat
// @Accept      json
// @Produce     json
// @Success     200 {object} map[string]interface{} "total: 전체 unread 개수, rooms: 각 방별 unread 개수"
// @Failure     401 {object} map[string]string "인증 실패"
// @Failure     500 {object} map[string]string "Internal server error"
// @Router      /dm/v01/total/unread [get]
//
//	GET /dm/v01/total/unread HTTP/1.1
//	Host: localhost:8080
//	Authorization: Bearer {access_token}
//
// @Example Resp:
//
// {"result":0,"resultString":"Success","data":{"rooms":{"10":30},"total_count":30}}
//
// @Example Success Response:
//
//	HTTP/1.1 200 OK
//	Content-Type: application/json
//	{
//	  "total_count": 7,
//	  "rooms": {
//	    "101": 5,
//	    "202": 2
//	  }
//	}
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
// @Summary      채팅 기록 조회
// @Description  특정 채팅방의 메시지 기록을 조회합니다.
// @Description  request : POST /dm/v01/history/10/1/20 HTTP/1.1
// @Description  response : {"result":0,"resultString":"Success","data":{"messages":[{"id":"527073","roomId":"10","userId":"4033287471439576593","content":"asf","type":"","timestamp":"2026-05-14T22:29:06.189288106+09:00"},{"id":"086572","roomId":"10","userId":"4033287471439576593","content":"gfv","type":"","timestamp":"2026-05-14T22:29:00.694804181+09:00"}],"total_count":109}}
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        roomId  path      string  true  "Chat Room ID"
// @Param        page    path      int     true  "Page number (1-based); path 우선, 없으면 query page (default 1)"
// @Param        limit   path      int     true  "Items per page; path 우선, 없으면 query limit (default 20), pgSize로 상한"
// @Param        pgSize  query     int     false "페이지당 최대 개수 상한 (default 50, min 1)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /dm/v01/history/{roomId}/{page}/{limit} [post]
//
// @example request
//
//	POST /dm/v01/history/10/1/20 HTTP/1.1
//	Host: localhost:8080
//	Authorization: Bearer {access_token}
//
// @example success response
//
//	HTTP/1.1 200 OK
//	Content-Type: application/json
//	{
//	  "result": 0,
//	  "resultString": "Success",
//	  "data": {
//	    "messages": [
//	      {"id":"527073","roomId":"10","userId":"4033287471439576593","content":"asf","type":"","timestamp":"2026-05-14T22:29:06.189288106+09:00"},
//	      {"id":"086572","roomId":"10","userId":"4033287471439576593","content":"gfv","type":"","timestamp":"2026-05-14T22:29:00.694804181+09:00"}
//	    ],
//	    "total_count": 109
//	  }
//	}
func (cc *ChatController) GetChatList(c *gin.Context) {
	roomID := c.Param("roomId")
	if roomID == "" {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "roomId required")
		return
	}

	// 페이지네이션: 라우트 path(/history/:roomId/:page/:limit) 우선, 없으면 query
	page := c.Param("page")
	if page == "" {
		page = c.DefaultQuery("page", "1")
	}
	limit := c.Param("limit")
	if limit == "" {
		limit = c.DefaultQuery("limit", "20")
	}
	pgSize := c.DefaultQuery("pgSize", "50")

	pageNum, err := strconv.Atoi(page)
	if err != nil {
		pageNum = 1
	}
	if pageNum < 1 {
		pageNum = 1
	}

	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		limitNum = 20
	}

	pgSizeNum, err := strconv.Atoi(pgSize)
	if err != nil {
		pgSizeNum = 50
	}
	if pgSizeNum < 1 {
		pgSizeNum = 50
	}
	if limitNum < 1 {
		limitNum = 20
	}
	if limitNum > pgSizeNum {
		limitNum = pgSizeNum
	}

	offset := (pageNum - 1) * limitNum

	// Redis에서 메시지 조회 (total_count = 방 전체 메시지 수)
	tcnt, messages, err := cc.rdb.GetChatMessages(roomID, offset, limitNum)
	if err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get chat history", err)
		return
	}

	var msgSlice []models.ChatMessageData
	if messages != nil {
		msgSlice = *messages
	}
	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{"total_count": tcnt, "messages": msgSlice})
}

// SendCallNotification 통화 알림 전송 (SignalingController에서 호출)
func (cc *ChatController) SendCallNotification(msg *ptc.ChatMessage) {
	cc.sendToUser(msg.To, msg)
	log.Info(fmt.Sprintf("Call notification sent: %s -> %s (type: %s)", msg.From, msg.To, msg.Type))
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
