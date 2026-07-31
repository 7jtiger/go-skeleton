package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ──────────────────────────────────────────────
// Protocol (서버 protocol/types.go · models/redis_db.go 와 정합)
// ──────────────────────────────────────────────

type ChatMessage struct {
	Type      string       `json:"type"`
	From      string       `json:"from,omitempty"`
	To        string       `json:"to,omitempty"`
	RoomID    string       `json:"roomId,omitempty"`
	Content   string       `json:"content,omitempty"`
	Timestamp int64        `json:"timestamp,omitempty"`
	CallMode  string       `json:"callMode,omitempty"`
	MsgID     string       `json:"msgId,omitempty"`
	Partner   *PartnerInfo `json:"partner,omitempty"`
	Unread    int          `json:"unread,omitempty"`
}

type PartnerInfo struct {
	PID      uint64 `json:"pid"`
	Nick     string `json:"nick"`
	ThumbPic string `json:"thumb_pic"`
	Gender   string `json:"gender"`
	Age      string `json:"age"`
	Area     string `json:"area"`
}

type DMRoomResp struct {
	RoomID      int64        `json:"rid"`
	Partner     *PartnerInfo `json:"partner"`
	LastMsg     string       `json:"last_msg"`
	PartnerLeft bool         `json:"partner_left"`
	Total       int          `json:"total"`
	Unread      int          `json:"unread"`
	AtCreate    string       `json:"at_crtchat"`
	AtUpdate    string       `json:"at_update"`
}

type APIResp struct {
	Result       int             `json:"result"`
	ResultString string          `json:"resultString"`
	Data         json.RawMessage `json:"data"`
}

type RoomListData struct {
	TotalCount int          `json:"total_count"`
	Rooms      []DMRoomResp `json:"rooms"`
}

type UnreadData struct {
	TotalCount int64            `json:"total_count"`
	Rooms      map[string]int64 `json:"rooms"`
}

// StoredMessage Redis chat:rooms:{id}:msg LIST 항목 (GetChatList 응답)
type StoredMessage struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

type ChatHistoryData struct {
	TotalCount     int64           `json:"total_count"`
	Messages       []StoredMessage `json:"messages"`
	NextCursor     string          `json:"next_cursor"`
	HasMore        bool            `json:"has_more"`
	MyLastMid      string          `json:"my_last_mid"`
	PartnerLastMid string          `json:"partner_last_mid"`
}

type RoomInfoData struct {
	RoomID         string     `json:"room_id"`
	Room           DMRoomResp `json:"room"`
	MyLastMid      string     `json:"my_last_mid"`
	PartnerLastMid string     `json:"partner_last_mid"`
}

// GiftPreset DM 선물 테스트용 프리셋 (서버 handleSendGift 연동)
type GiftPreset struct {
	Code  string // 클라이언트 식별자
	Label string // 표시명
}

// giftCatalog — chatCtl handleSendGift는 현재 content=수량(문자열)만 사용, 아이템명은 서버에서 "화살" 고정
var giftCatalog = []GiftPreset{
	{Code: "arrow", Label: "화살"},
	{Code: "rose", Label: "장미"},
	{Code: "cake", Label: "케이크"},
	{Code: "diamond", Label: "다이아몬드"},
	{Code: "heart", Label: "하트"},
}

var giftAliases = map[string]string{
	"arrow": "arrow", "화살": "arrow",
	"rose": "rose", "장미": "rose",
	"cake": "cake", "케이크": "cake",
	"diamond": "diamond", "다이아": "diamond", "다이아몬드": "diamond",
	"heart": "heart", "하트": "heart",
}

// ──────────────────────────────────────────────
// 세션 상태
// ──────────────────────────────────────────────

type Session struct {
	server     string
	token      string
	ws         *websocket.Conn
	roomID     string
	partnerID  string
	lastCursor string // chatlist 페이지네이션
	myLastMid  string
	done       chan struct{}
	mu         sync.Mutex
	connected  bool
}

var sess = &Session{}

// ──────────────────────────────────────────────
// 출력
// ──────────────────────────────────────────────

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func info(format string, a ...interface{}) {
	fmt.Printf(colorCyan+"[INFO] "+colorReset+format+"\n", a...)
}
func warn(format string, a ...interface{}) {
	fmt.Printf(colorYellow+"[WARN] "+colorReset+format+"\n", a...)
}
func errMsg(format string, a ...interface{}) {
	fmt.Printf(colorRed+"[ERR]  "+colorReset+format+"\n", a...)
}
func recv(format string, a ...interface{}) {
	fmt.Printf(colorGreen+"[RECV] "+colorReset+format+"\n", a...)
}
func ts() string { return time.Now().Format("15:04:05") }

// ──────────────────────────────────────────────
// HTTP
// ──────────────────────────────────────────────

func apiBase() string {
	s := sess.server
	if strings.HasPrefix(s, "http") {
		return s
	}
	return "http://" + s
}

func doReq(method, rawURL string, body string) (int, []byte, error) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if sess.token != "" {
		req.Header.Set("Authorization", "Bearer "+sess.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data, nil
}

func parseAPI(raw []byte) (*APIResp, error) {
	var r APIResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func requireToken() bool {
	if sess.token == "" {
		errMsg("토큰이 필요합니다. /token <jwt>")
		return false
	}
	return true
}

// ──────────────────────────────────────────────
// WebSocket
// ──────────────────────────────────────────────

func wsConnect() error {
	if sess.token == "" {
		return fmt.Errorf("토큰이 설정되지 않았습니다. /token <jwt> 로 설정하세요")
	}
	if sess.connected {
		wsDisconnect()
	}

	scheme := "ws"
	host := sess.server
	if strings.HasPrefix(host, "https") {
		scheme = "wss"
		host = strings.TrimPrefix(host, "https://")
	} else {
		host = strings.TrimPrefix(host, "http://")
	}
	if !strings.Contains(host, "localhost") && !strings.Contains(host, "127.0.0.1") {
		scheme = "wss"
	}
	wsURL := scheme + "://" + host + "/dm/v01/ws"

	header := http.Header{}
	header.Set("Authorization", "Bearer "+sess.token)

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("status=%d body=%s err=%v", resp.StatusCode, string(body), err)
		}
		return err
	}

	sess.mu.Lock()
	sess.ws = conn
	sess.done = make(chan struct{})
	sess.connected = true
	sess.mu.Unlock()

	go readLoop()
	return nil
}

func wsDisconnect() {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if sess.ws != nil {
		_ = sess.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = sess.ws.Close()
		sess.ws = nil
	}
	sess.connected = false
}

func readLoop() {
	defer func() {
		sess.mu.Lock()
		sess.connected = false
		sess.mu.Unlock()
	}()
	for {
		_, raw, err := sess.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				errMsg("WS 읽기 오류: %v", err)
			} else {
				warn("WS 연결 종료")
			}
			return
		}
		lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var msg ChatMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				errMsg("JSON 파싱 실패: %s", line)
				continue
			}
			handleIncoming(&msg)
		}
	}
}

func handleIncoming(msg *ChatMessage) {
	switch msg.Type {
	case "text-message":
		recv("%s [%s] %s  (room=%s msgId=%s)", ts(), msg.From, msg.Content, msg.RoomID, msg.MsgID)
	case "msg-ack":
		recv("%s ACK roomId=%s msgId=%s unread=%d", ts(), msg.RoomID, msg.MsgID, msg.Unread)
		sess.mu.Lock()
		if msg.RoomID != "" {
			sess.roomID = msg.RoomID
		}
		sess.mu.Unlock()
	case "read-receipt":
		recv("%s 읽음 처리 from=%s room=%s msgId=%s", ts(), msg.From, msg.RoomID, msg.MsgID)
	case "typing":
		recv("%s 입력 중... from=%s", ts(), msg.From)
	case "call-incoming":
		recv("%s 통화 요청 수신 from=%s mode=%s room=%s", ts(), msg.From, msg.CallMode, msg.RoomID)
	case "call-accept":
		nick := ""
		if msg.Partner != nil {
			nick = msg.Partner.Nick
		}
		recv("%s 통화 수락 from=%s partner=%s", ts(), msg.From, nick)
	case "call-reject":
		recv("%s 통화 거절 from=%s", ts(), msg.From)
	case "call-cancel":
		recv("%s 통화 취소 from=%s", ts(), msg.From)
	case "call-info":
		recv("%s 통화 안내: %s", ts(), msg.Content)
	case "partner-left", "system-message":
		recv("%s [%s] %s (room=%s)", ts(), msg.Type, msg.Content, msg.RoomID)
	case "dm-incoming":
		recv("%s DM 알림 from=%s content=%s", ts(), msg.From, msg.Content)
	case "put-gift":
		recv("%s 🎁 선물 [%s] from=%s → to=%s room=%s msgId=%s",
			ts(), msg.Content, msg.From, msg.To, msg.RoomID, msg.MsgID)
	default:
		recv("%s [%s] %s", ts(), msg.Type, prettyJSON(msg))
	}
}

func wsSend(msg *ChatMessage) error {
	sess.mu.Lock()
	conn := sess.ws
	ok := sess.connected
	sess.mu.Unlock()
	if !ok || conn == nil {
		return fmt.Errorf("WS 미연결. /connect 먼저 실행하세요")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	info("전송 -> %s", string(data))
	return conn.WriteMessage(websocket.TextMessage, data)
}

func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// ──────────────────────────────────────────────
// REST 핸들러
// ──────────────────────────────────────────────

func applyRoomFromResp(room *DMRoomResp) {
	if room == nil {
		return
	}
	sess.roomID = strconv.FormatInt(room.RoomID, 10)
	if room.Partner != nil {
		sess.partnerID = strconv.FormatUint(room.Partner.PID, 10)
	}
}

func printRoomBrief(r *DMRoomResp) {
	nick := partnerNick(r)
	left := ""
	if r.PartnerLeft {
		left = colorYellow + " [상대 나감]" + colorReset
	}
	fmt.Printf("  rid=%-6d partner=%-14s total=%-4d unread=%-4d%s\n",
		r.RoomID, nick, r.Total, r.Unread, left)
	if r.LastMsg != "" {
		fmt.Printf("    last_msg: %s\n", truncate(r.LastMsg, 60))
	}
}

func cmdMkroom(pid string) {
	if !requireToken() {
		return
	}
	if pid == "" {
		errMsg("사용법: /mkroom <pid> 또는 /target 설정 후 /mkroom")
		return
	}
	code, body, err := doReq("POST", apiBase()+"/dm/v01/mkroom/"+pid, "")
	if err != nil {
		errMsg("요청 실패: %v", err)
		return
	}
	info("mkroom status=%d", code)
	if code != http.StatusOK {
		fmt.Println(string(body))
		return
	}
	api, _ := parseAPI(body)
	if api == nil || api.Data == nil {
		fmt.Println(string(body))
		return
	}
	var room DMRoomResp
	if json.Unmarshal(api.Data, &room) != nil {
		fmt.Println(string(body))
		return
	}
	applyRoomFromResp(&room)
	info("방 준비/재입장 완료 rid=%d partner=%s unread=%d total=%d",
		room.RoomID, partnerNick(&room), room.Unread, room.Total)
	printRoomBrief(&room)
	info("나간 뒤 재입장이면 /chatlist 로 내역이 비어 있는지 확인하세요")
}

func cmdLeave(roomID string) {
	if !requireToken() {
		return
	}
	if roomID == "" {
		roomID = sess.roomID
	}
	if roomID == "" {
		errMsg("사용법: /leave [roomId] 또는 /room 설정 후 /leave")
		return
	}
	code, body, err := doReq("POST", apiBase()+"/dm/v01/rmroom/"+roomID, "")
	if err != nil {
		errMsg("요청 실패: %v", err)
		return
	}
	info("leave status=%d room=%s", code, roomID)
	if code != http.StatusOK {
		fmt.Println(string(body))
		return
	}
	info("방 나가기 완료 — 이 사용자는 /chatlist 시 403 또는 재입장 후 빈 내역")
	info("상대방은 기존 채팅 내역 유지 (다른 터미널에서 /chatlist 확인)")
}

func cmdRinfo(roomID, mid string) {
	if !requireToken() {
		return
	}
	if roomID == "" {
		roomID = sess.roomID
	}
	if roomID == "" {
		errMsg("사용법: /rinfo [roomId] [mid]")
		return
	}
	rawURL := apiBase() + "/dm/v01/rinfo/" + roomID
	if mid != "" {
		rawURL += "?mid=" + url.QueryEscape(mid)
	}
	code, body, err := doReq("GET", rawURL, "")
	if err != nil {
		errMsg("요청 실패: %v", err)
		return
	}
	info("rinfo status=%d", code)
	if code != http.StatusOK {
		fmt.Println(string(body))
		return
	}
	api, _ := parseAPI(body)
	if api == nil || api.Data == nil {
		fmt.Println(string(body))
		return
	}
	var data RoomInfoData
	if json.Unmarshal(api.Data, &data) != nil {
		fmt.Println(string(body))
		return
	}
	sess.roomID = data.RoomID
	applyRoomFromResp(&data.Room)
	sess.myLastMid = data.MyLastMid
	info("방 입장 정보 (unread 초기화됨)")
	fmt.Printf("  room_id=%s my_last_mid=%s partner_last_mid=%s\n",
		data.RoomID, data.MyLastMid, data.PartnerLastMid)
	printRoomBrief(&data.Room)
}

func cmdExists(tid string) {
	if !requireToken() {
		return
	}
	if tid == "" {
		tid = sess.partnerID
	}
	if tid == "" {
		errMsg("사용법: /exists <tid>")
		return
	}
	code, body, err := doReq("GET", apiBase()+"/dm/v01/room/exists/"+tid, "")
	if err != nil {
		errMsg("요청 실패: %v", err)
		return
	}
	info("exists status=%d", code)
	fmt.Println(string(body))
}

func cmdRooms(page string) {
	if !requireToken() {
		return
	}
	if page == "" {
		page = "1"
	}
	code, body, err := doReq("GET", apiBase()+"/dm/v01/list/"+page, "")
	if err != nil {
		errMsg("요청 실패: %v", err)
		return
	}
	info("rooms status=%d page=%s", code, page)
	if code != http.StatusOK {
		fmt.Println(string(body))
		return
	}
	api, _ := parseAPI(body)
	if api == nil || api.Data == nil {
		fmt.Println(string(body))
		return
	}
	var list RoomListData
	if json.Unmarshal(api.Data, &list) != nil {
		fmt.Println(string(body))
		return
	}
	info("총 %d개 방", list.TotalCount)
	for i, r := range list.Rooms {
		fmt.Printf("%s[%d]%s ", colorGray, i+1, colorReset)
		printRoomBrief(&r)
	}
}

func cmdUnread() {
	if !requireToken() {
		return
	}
	code, body, err := doReq("GET", apiBase()+"/dm/v01/total/unread", "")
	if err != nil {
		errMsg("요청 실패: %v", err)
		return
	}
	info("unread status=%d", code)
	if code != http.StatusOK {
		fmt.Println(string(body))
		return
	}
	api, _ := parseAPI(body)
	if api == nil || api.Data == nil {
		fmt.Println(string(body))
		return
	}
	var u UnreadData
	if json.Unmarshal(api.Data, &u) != nil {
		fmt.Println(string(body))
		return
	}
	info("총 미읽음: %d", u.TotalCount)
	for rid, cnt := range u.Rooms {
		fmt.Printf("  room %-8s : %d\n", rid, cnt)
	}
}

func cmdChatlist(roomID, limit, cursor string, saveCursor bool) {
	if !requireToken() {
		return
	}
	if roomID == "" {
		roomID = sess.roomID
	}
	if limit == "" {
		limit = "20"
	}
	if roomID == "" {
		errMsg("사용법: /chatlist [roomId] [limit] [cursor]")
		return
	}
	if cursor == "" && saveCursor {
		cursor = sess.lastCursor
		if cursor != "" {
			info("이전 cursor 사용: %s", cursor)
		}
	}

	qs := fmt.Sprintf("?limit=%s", limit)
	if cursor != "" {
		qs += "&cursor=" + url.QueryEscape(cursor)
	}
	rawURL := apiBase() + "/dm/v01/history/" + roomID + qs
	code, body, err := doReq("GET", rawURL, "")
	if err != nil {
		errMsg("요청 실패: %v", err)
		return
	}
	info("chatlist status=%d room=%s", code, roomID)
	if code != http.StatusOK {
		fmt.Println(string(body))
		if code == http.StatusForbidden {
			warn("방 참여자가 아닙니다. /mkroom 으로 재입장하거나 상대가 나간 상태일 수 있습니다")
		}
		return
	}
	api, _ := parseAPI(body)
	if api == nil || api.Data == nil {
		fmt.Println(string(body))
		return
	}
	var hist ChatHistoryData
	if json.Unmarshal(api.Data, &hist) != nil {
		fmt.Println(string(body))
		return
	}

	sess.myLastMid = hist.MyLastMid
	if hist.NextCursor != "" {
		sess.lastCursor = hist.NextCursor
	} else if cursor == "" {
		sess.lastCursor = ""
	}

	fmt.Printf("%s── 채팅 내역 (viewer 기준 visible=%d) ──%s\n",
		colorBold, hist.TotalCount, colorReset)
	fmt.Printf("  my_last_mid=%s  partner_last_mid=%s\n", hist.MyLastMid, hist.PartnerLastMid)
	if hist.HasMore {
		fmt.Printf("  has_more=true  next_cursor=%s  (/chatmore 로 이전 메시지)\n", hist.NextCursor)
	} else {
		fmt.Printf("  has_more=false\n")
	}

	if len(hist.Messages) == 0 {
		info("메시지 없음 (나간 뒤 재입장 시 정상 — 이전 내역은 DM:HIST:FROM 컷오프로 숨김)")
		return
	}

	for i, m := range hist.Messages {
		at := m.Timestamp.Format("2006-01-02 15:04:05")
		if m.Timestamp.IsZero() {
			at = "-"
		}
		typeTag := m.Type
		if typeTag == "" {
			typeTag = "text-message"
		}
		giftMark := ""
		if typeTag == "put-gift" {
			giftMark = " 🎁"
		}
		fmt.Printf("  %s[%d]%s %s [%s]%s from=%s id=%s\n",
			colorGray, i+1, colorReset, at, typeTag, giftMark, m.UserID, m.ID)
		fmt.Printf("      %s\n", m.Content)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func giftLabel(code string) string {
	for _, g := range giftCatalog {
		if g.Code == code {
			return g.Label
		}
	}
	return code
}

func cmdGifts() {
	info("선물 프리셋 (WS type=put-gift, content=수량)")
	fmt.Printf("  %s참고:%s 서버는 현재 수신 메시지에 '화살' 고정 표기 (handleSendGift TODO)\n",
		colorGray, colorReset)
	for i, g := range giftCatalog {
		fmt.Printf("  %s[%d]%s /gift %s [수량]  — %s\n", colorGray, i+1, colorReset, g.Code, g.Label)
	}
	fmt.Println("  예: /gift 3  |  /gift rose 1  |  /gift 화살 5")
}

// sendGift DM 선물 전송 (chatCtl handleSendGift)
func sendGift(itemCode string, amount int) {
	if sess.partnerID == "" {
		errMsg("/target 으로 상대를 지정하세요")
		return
	}
	if amount < 1 {
		amount = 1
	}
	msgID := fmt.Sprintf("g-%d", time.Now().UnixMilli())
	content := strconv.Itoa(amount)

	if err := wsSend(&ChatMessage{
		Type:    "put-gift",
		To:      sess.partnerID,
		RoomID:  sess.roomID,
		Content: content,
		MsgID:   msgID,
	}); err != nil {
		errMsg("%v", err)
		return
	}
	info("선물 전송 → %s (%s x%d) msgId=%s room=%s",
		sess.partnerID, giftLabel(itemCode), amount, msgID, orDefault(sess.roomID, "(ack 후 설정)"))
}

// parseGiftArgs "/gift rose 3" | "/gift 3" | "/gift 화살"
func parseGiftArgs(arg string) (itemCode string, amount int, err error) {
	itemCode = "arrow"
	amount = 1
	if arg == "" {
		return itemCode, amount, nil
	}
	tokens := strings.Fields(arg)
	if len(tokens) == 0 {
		return itemCode, amount, nil
	}

	// 숫자만: /gift 5
	if n, e := strconv.Atoi(tokens[0]); e == nil {
		return itemCode, n, nil
	}

	// /gift rose [n]
	code, ok := giftAliases[strings.ToLower(tokens[0])]
	if !ok {
		return "", 0, fmt.Errorf("알 수 없는 선물: %s (/gifts 로 목록 확인)", tokens[0])
	}
	itemCode = code
	if len(tokens) >= 2 {
		n, e := strconv.Atoi(tokens[1])
		if e != nil || n < 1 {
			return "", 0, fmt.Errorf("수량은 1 이상의 정수여야 합니다: %s", tokens[1])
		}
		amount = n
	}
	return itemCode, amount, nil
}

func cmdGift(arg string) {
	item, amount, err := parseGiftArgs(arg)
	if err != nil {
		errMsg("%v", err)
		return
	}
	sendGift(item, amount)
}

// ──────────────────────────────────────────────
// 명령어 처리
// ──────────────────────────────────────────────

func printHelp() {
	help := `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  DM Gateway 콘솔 클라이언트 (chatCtl 테스트용)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  설정
    /server <addr>          서버 주소 (기본: localhost:8080)
    /token  <jwt>           JWT 토큰 설정
    /target <uid>           상대 UID 설정
    /room   <roomId>        방 ID 수동 설정
    /status                 현재 세션 상태 확인

  WebSocket
    /connect                WS 연결 (JWT 필요)
    /disconnect             WS 연결 종료

  메시지 (WS 연결 필요)
    /send <메시지>          텍스트 메시지 전송
    (텍스트만 입력)         -> 자동 전송
    /typing                 타이핑 알림
    /read [msgId]           읽음 처리 (msgId 생략 가능)

  선물 (WS — chatCtl put-gift)
    /gifts                  선물 프리셋 목록
    /gift [item] [수량]     선물 전송 (예: /gift 3, /gift rose 1)
    /present                /gift 와 동일 (별칭)

  통화 (WS)
    /call <video|audio|text>
    /accept  /reject  /cancel

  REST — 방
    /mkroom [pid]           방 생성 / 재입장 (나간 사용자 컷오프 갱신)
    /rejoin [pid]           /mkroom 과 동일
    /leave [roomId]         방 나가기 (POST /dm/v01/rmroom/:id)
    /exists <tid>           상대와 방 존재 여부
    /rinfo [roomId] [mid]   방 상세 + 입장(unread 초기화)
    /rooms [page]           인박스 목록

  REST — 메시지
    /chatlist [roomId] [limit] [cursor]   채팅 내역 (커서 페이지네이션)
    /chatmore [limit]                     이전 cursor로 추가 조회
    /unread                               전체 미읽음

  나가기/재입장 테스트 시나리오
    터미널 A,B 각각 /token /target /connect 설정
    1) A: /mkroom <B_uid>  →  /send hello
    2) B: /rooms → /chatlist <rid>  (내역 확인)
    3) A: /leave  →  B: /chatlist (B는 내역 유지)
    4) A: /mkroom <B_uid> → /chatlist (A는 빈 내역)
    5) A: /send new msg  →  양쪽 /chatlist 비교

  선물 테스트 시나리오
    1) A,B /connect 후 A: /mkroom <B>
    2) A: /gift rose 1  →  B: WS 🎁 수신 + msg-ack
    3) B: /chatlist <rid>  →  put-gift 타입 메시지 확인

  /help  /quit
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`
	fmt.Println(help)
}

func printStatus() {
	connStr := colorRed + "미연결" + colorReset
	if sess.connected {
		connStr = colorGreen + "연결됨" + colorReset
	}
	tokenStr := colorRed + "(없음)" + colorReset
	if sess.token != "" {
		t := sess.token
		if len(t) > 20 {
			t = t[:10] + "..." + t[len(t)-10:]
		}
		tokenStr = t
	}
	fmt.Printf(`
  서버:       %s
  토큰:       %s
  WS:         %s
  상대:       %s
  방 ID:      %s
  lastCursor: %s
  myLastMid:  %s
`, sess.server, tokenStr, connStr,
		orDefault(sess.partnerID, "(없음)"),
		orDefault(sess.roomID, "(없음)"),
		orDefault(sess.lastCursor, "(없음)"),
		orDefault(sess.myLastMid, "(없음)"))
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func handleCommand(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	parts := strings.SplitN(line, " ", 2)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "/server":
		if arg == "" {
			errMsg("사용법: /server <addr>")
			return
		}
		sess.server = arg
		info("서버 설정: %s", sess.server)

	case "/token":
		if arg == "" {
			errMsg("사용법: /token <jwt>")
			return
		}
		sess.token = arg
		info("토큰 설정 완료 (len=%d)", len(arg))

	case "/target":
		if arg == "" {
			errMsg("사용법: /target <uid>")
			return
		}
		sess.partnerID = arg
		info("상대 UID 설정: %s", sess.partnerID)

	case "/room":
		if arg == "" {
			errMsg("사용법: /room <roomId>")
			return
		}
		sess.roomID = arg
		sess.lastCursor = ""
		info("방 ID 설정: %s", sess.roomID)

	case "/status":
		printStatus()

	case "/connect":
		if err := wsConnect(); err != nil {
			errMsg("WS 연결 실패: %v", err)
		} else {
			info("WS 연결 성공 (%s)", sess.server)
		}

	case "/disconnect":
		wsDisconnect()
		info("WS 연결 종료")

	case "/send":
		if arg == "" {
			errMsg("사용법: /send <메시지>")
			return
		}
		sendTextMessage(arg)

	case "/typing":
		if sess.partnerID == "" {
			errMsg("/target 으로 상대를 지정하세요")
			return
		}
		if err := wsSend(&ChatMessage{
			Type:   "typing",
			To:     sess.partnerID,
			RoomID: sess.roomID,
		}); err != nil {
			errMsg("%v", err)
		}

	case "/read":
		if sess.partnerID == "" || sess.roomID == "" {
			errMsg("/target 과 /room (또는 ack 후 room) 이 필요합니다")
			return
		}
		msg := &ChatMessage{
			Type:   "read-receipt",
			To:     sess.partnerID,
			RoomID: sess.roomID,
		}
		if arg != "" {
			msg.MsgID = arg
		} else if sess.myLastMid != "" {
			msg.MsgID = sess.myLastMid
		}
		if err := wsSend(msg); err != nil {
			errMsg("%v", err)
		} else {
			info("읽음 처리 전송 room=%s msgId=%s", sess.roomID, msg.MsgID)
		}

	case "/gifts":
		cmdGifts()

	case "/gift", "/present":
		cmdGift(arg)

	case "/call":
		if sess.partnerID == "" {
			errMsg("/target 으로 상대를 지정하세요")
			return
		}
		mode := "video"
		if arg != "" {
			mode = arg
		}
		if err := wsSend(&ChatMessage{
			Type:     "call-request",
			To:       sess.partnerID,
			RoomID:   sess.roomID,
			CallMode: mode,
		}); err != nil {
			errMsg("%v", err)
		} else {
			info("통화 요청 전송 (mode=%s)", mode)
		}

	case "/accept":
		if sess.partnerID == "" {
			errMsg("/target 으로 상대를 지정하세요")
			return
		}
		if err := wsSend(&ChatMessage{Type: "call-accept", To: sess.partnerID, RoomID: sess.roomID}); err != nil {
			errMsg("%v", err)
		} else {
			info("통화 수락 전송")
		}

	case "/reject":
		if sess.partnerID == "" {
			errMsg("/target 으로 상대를 지정하세요")
			return
		}
		if err := wsSend(&ChatMessage{Type: "call-reject", To: sess.partnerID, RoomID: sess.roomID}); err != nil {
			errMsg("%v", err)
		} else {
			info("통화 거절 전송")
		}

	case "/cancel":
		if sess.partnerID == "" {
			errMsg("/target 으로 상대를 지정하세요")
			return
		}
		if err := wsSend(&ChatMessage{
			Type: "call-cancel", To: sess.partnerID, RoomID: sess.roomID, CallMode: "video",
		}); err != nil {
			errMsg("%v", err)
		} else {
			info("통화 취소 전송")
		}

	case "/mkroom", "/rejoin":
		pid := sess.partnerID
		if arg != "" {
			pid = arg
		}
		cmdMkroom(pid)

	case "/leave", "/rmroom":
		cmdLeave(arg)

	case "/rinfo":
		args := strings.Fields(arg)
		roomID, mid := "", ""
		if len(args) >= 1 {
			roomID = args[0]
		}
		if len(args) >= 2 {
			mid = args[1]
		}
		cmdRinfo(roomID, mid)

	case "/exists":
		cmdExists(arg)

	case "/rooms":
		cmdRooms(arg)

	case "/unread":
		cmdUnread()

	case "/chatlist":
		roomID, limit, cursor := "", "20", ""
		if arg != "" {
			args := strings.Fields(arg)
			if len(args) >= 1 {
				roomID = args[0]
			}
			if len(args) >= 2 {
				limit = args[1]
			}
			if len(args) >= 3 {
				cursor = args[2]
			}
		}
		sess.lastCursor = ""
		cmdChatlist(roomID, limit, cursor, false)

	case "/chatmore":
		limit := "20"
		if arg != "" {
			limit = arg
		}
		if sess.lastCursor == "" {
			errMsg("먼저 /chatlist 로 조회하세요 (has_more=true 일 때 사용)")
			return
		}
		cmdChatlist(sess.roomID, limit, sess.lastCursor, false)

	case "/help":
		printHelp()

	case "/quit", "/exit", "/q":
		wsDisconnect()
		info("종료합니다.")
		os.Exit(0)

	default:
		if strings.HasPrefix(cmd, "/") {
			errMsg("알 수 없는 명령어: %s (/help 로 도움말 확인)", cmd)
		} else {
			sendTextMessage(line)
		}
	}
}

func sendTextMessage(content string) {
	if sess.partnerID == "" {
		errMsg("/target 으로 상대를 지정하세요")
		return
	}
	msgID := fmt.Sprintf("m-%d", time.Now().UnixMilli())
	if err := wsSend(&ChatMessage{
		Type:    "text-message",
		To:      sess.partnerID,
		RoomID:  sess.roomID,
		Content: content,
		MsgID:   msgID,
	}); err != nil {
		errMsg("%v", err)
	}
}

func partnerNick(r *DMRoomResp) string {
	if r.Partner != nil && r.Partner.Nick != "" {
		return r.Partner.Nick
	}
	if r.Partner != nil {
		return strconv.FormatUint(r.Partner.PID, 10)
	}
	return "?"
}

func main() {
	sess.server = "localhost:8080"

	fmt.Println(colorCyan + `
  ┌──────────────────────────────────────┐
  │   DM Gateway Console Client          │
  │   /help 로 명령어 · 테스트 시나리오  │
  └──────────────────────────────────────┘` + colorReset)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		fmt.Println()
		wsDisconnect()
		info("Ctrl+C 종료")
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for {
		prompt := colorGray + "dm> " + colorReset
		if sess.connected {
			prompt = colorGreen + "dm" + colorReset
			if sess.partnerID != "" {
				prompt += colorYellow + ":" + sess.partnerID + colorReset
			}
			if sess.roomID != "" {
				prompt += colorGray + "#" + sess.roomID + colorReset
			}
			prompt += colorGreen + "> " + colorReset
		}
		fmt.Print(prompt)

		if !scanner.Scan() {
			break
		}
		handleCommand(scanner.Text())
	}
}
