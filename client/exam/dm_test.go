package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ──────────────────────────────────────────────
// Protocol
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
	ThumbPic string `json:"thumbPic"`
	Gender   string `json:"gender"`
	Age      string `json:"age"`
	Area     string `json:"area"`
}

type DMRoomResp struct {
	RoomID   int64        `json:"rid"`
	Partner  *PartnerInfo `json:"partner"`
	Total    int          `json:"total"`
	Unread   int          `json:"unread"`
	AtCreate string       `json:"at_crtchat"`
	AtUpdate string       `json:"at_update"`
}

type APIResp struct {
	Result       int             `json:"result"`
	ResultString string          `json:"resultString"`
	Data         json.RawMessage `json:"data"`
}

type RoomListData struct {
	TotalCount int          `json:"totalcount"`
	Rooms      []DMRoomResp `json:"rooms"`
}

type UnreadData struct {
	Total int64            `json:"total"`
	Rooms map[string]int64 `json:"rooms"`
}

// ──────────────────────────────────────────────
// 세션 상태
// ──────────────────────────────────────────────

type Session struct {
	server    string
	token     string
	ws        *websocket.Conn
	roomID    string
	partnerID string
	done      chan struct{}
	mu        sync.Mutex
	connected bool
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
)

func info(format string, a ...interface{})  { fmt.Printf(colorCyan+"[INFO] "+colorReset+format+"\n", a...) }
func warn(format string, a ...interface{})  { fmt.Printf(colorYellow+"[WARN] "+colorReset+format+"\n", a...) }
func errMsg(format string, a ...interface{}) { fmt.Printf(colorRed+"[ERR]  "+colorReset+format+"\n", a...) }
func recv(format string, a ...interface{})  { fmt.Printf(colorGreen+"[RECV] "+colorReset+format+"\n", a...) }
func ts() string                            { return time.Now().Format("15:04:05") }

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

func doReq(method, url string, body string) (int, []byte, error) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
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

func parseData(raw []byte) (json.RawMessage, error) {
	var r APIResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return r.Data, nil
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
	url := scheme + "://" + host + "/dm/v01/ws"

	header := http.Header{}
	header.Set("Authorization", "Bearer "+sess.token)

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, resp, err := dialer.Dial(url, header)
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
		sess.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		sess.ws.Close()
		sess.ws = nil
	}
	sess.connected = false
}

func readLoop() {
	defer func() {
		sess.mu.Lock()
		sess.connected = false
		close(sess.done)
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
		recv("%s [%s] %s  (room=%s)", ts(), msg.From, msg.Content, msg.RoomID)
	case "msg-ack":
		recv("%s ACK roomId=%s msgId=%s unread=%d", ts(), msg.RoomID, msg.MsgID, msg.Unread)
		sess.mu.Lock()
		if msg.RoomID != "" {
			sess.roomID = msg.RoomID
		}
		sess.mu.Unlock()
	case "read-receipt":
		recv("%s 읽음 처리 from=%s room=%s", ts(), msg.From, msg.RoomID)
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
	case "dm-incoming":
		recv("%s DM 알림 from=%s content=%s", ts(), msg.From, msg.Content)
	default:
		recv("%s [%s] %s", ts(), msg.Type, prettyJSON(msg))
	}
}

func wsSend(msg *ChatMessage) error {
	if !sess.connected || sess.ws == nil {
		return fmt.Errorf("WS 미연결. /connect 먼저 실행하세요")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	info("전송 -> %s", string(data))
	return sess.ws.WriteMessage(websocket.TextMessage, data)
}

func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// ──────────────────────────────────────────────
// 명령어 처리
// ──────────────────────────────────────────────

func printHelp() {
	help := `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  DM Gateway 콘솔 클라이언트
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
    또는 그냥 텍스트 입력   -> 자동으로 텍스트 메시지 전송
    /typing                 타이핑 알림 전송
    /read                   읽음 처리 전송

  통화 (WS 연결 필요)
    /call <video|audio|text>   통화 요청
    /accept                    통화 수락
    /reject                    통화 거절
    /cancel                    통화 취소

  REST API
    /mkroom [pid]           방 생성 (pid 생략 시 /target 값 사용)
    /rooms  [page]          방 목록 조회 (기본 page=1)
    /unread                 전체 미읽음 조회

  기타
    /help                   도움말
    /quit                   종료
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
  서버:    %s
  토큰:    %s
  WS:      %s
  상대:    %s
  방 ID:   %s
`, sess.server, tokenStr, connStr, orDefault(sess.partnerID, "(없음)"), orDefault(sess.roomID, "(없음)"))
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
	// ── 설정 ──
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
		info("방 ID 설정: %s", sess.roomID)

	case "/status":
		printStatus()

	// ── WS 연결 ──
	case "/connect":
		if err := wsConnect(); err != nil {
			errMsg("WS 연결 실패: %v", err)
		} else {
			info("WS 연결 성공 (%s)", sess.server)
		}

	case "/disconnect":
		wsDisconnect()
		info("WS 연결 종료")

	// ── 메시지 ──
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
			errMsg("/target 과 /room (또는 메시지 전송 후 ack) 이 필요합니다")
			return
		}
		if err := wsSend(&ChatMessage{
			Type:   "read-receipt",
			To:     sess.partnerID,
			RoomID: sess.roomID,
		}); err != nil {
			errMsg("%v", err)
		} else {
			info("읽음 처리 전송 (room=%s)", sess.roomID)
		}

	// ── 통화 ──
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
		if err := wsSend(&ChatMessage{
			Type:   "call-accept",
			To:     sess.partnerID,
			RoomID: sess.roomID,
		}); err != nil {
			errMsg("%v", err)
		} else {
			info("통화 수락 전송")
		}

	case "/reject":
		if sess.partnerID == "" {
			errMsg("/target 으로 상대를 지정하세요")
			return
		}
		if err := wsSend(&ChatMessage{
			Type:   "call-reject",
			To:     sess.partnerID,
			RoomID: sess.roomID,
		}); err != nil {
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
			Type:     "call-cancel",
			To:       sess.partnerID,
			RoomID:   sess.roomID,
			CallMode: "video",
		}); err != nil {
			errMsg("%v", err)
		} else {
			info("통화 취소 전송")
		}

	// ── REST API ──
	case "/mkroom":
		pid := sess.partnerID
		if arg != "" {
			pid = arg
		}
		if pid == "" {
			errMsg("사용법: /mkroom <pid> 또는 /target 설정 후 /mkroom")
			return
		}
		if sess.token == "" {
			errMsg("토큰이 필요합니다. /token <jwt>")
			return
		}
		code, body, err := doReq("POST", apiBase()+"/dm/v01/mkroom/"+pid, "")
		if err != nil {
			errMsg("요청 실패: %v", err)
			return
		}
		info("mkroom status=%d", code)
		data, _ := parseData(body)
		if data != nil {
			var room DMRoomResp
			if json.Unmarshal(data, &room) == nil {
				sess.roomID = strconv.FormatInt(room.RoomID, 10)
				if room.Partner != nil {
					sess.partnerID = strconv.FormatUint(room.Partner.PID, 10)
				}
				info("방 생성 완료 rid=%d partner=%s unread=%d",
					room.RoomID,
					orDefault(partnerNick(&room), sess.partnerID),
					room.Unread)
				return
			}
		}
		fmt.Println(string(body))

	case "/rooms":
		if sess.token == "" {
			errMsg("토큰이 필요합니다. /token <jwt>")
			return
		}
		page := "1"
		if arg != "" {
			page = arg
		}
		code, body, err := doReq("GET", apiBase()+"/dm/v01/list/"+page, "")
		if err != nil {
			errMsg("요청 실패: %v", err)
			return
		}
		info("rooms status=%d", code)
		data, _ := parseData(body)
		if data != nil {
			var list RoomListData
			if json.Unmarshal(data, &list) == nil {
				info("총 %d개 (page %s)", list.TotalCount, page)
				for i, r := range list.Rooms {
					nick := partnerNick(&r)
					fmt.Printf("  %s[%d]%s rid=%-8d partner=%-12s unread=%d\n",
						colorGray, i+1, colorReset, r.RoomID, nick, r.Unread)
				}
				return
			}
		}
		fmt.Println(string(body))

	case "/unread":
		if sess.token == "" {
			errMsg("토큰이 필요합니다. /token <jwt>")
			return
		}
		code, body, err := doReq("GET", apiBase()+"/dm/v01/total/unread", "")
		if err != nil {
			errMsg("요청 실패: %v", err)
			return
		}
		info("unread status=%d", code)
		data, _ := parseData(body)
		if data != nil {
			var u UnreadData
			if json.Unmarshal(data, &u) == nil {
				info("총 미읽음: %d", u.Total)
				for rid, cnt := range u.Rooms {
					fmt.Printf("  room %-8s : %d\n", rid, cnt)
				}
				return
			}
		}
		fmt.Println(string(body))

	// ── 기타 ──
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

// ──────────────────────────────────────────────
// main
// ──────────────────────────────────────────────

func main() {
	sess.server = "localhost:8080"

	fmt.Println(colorCyan + `
  ┌──────────────────────────────────────┐
  │   DM Gateway Console Client          │
  │   /help 로 명령어 확인               │
  └──────────────────────────────────────┘` + colorReset)

	// Ctrl+C 처리
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
		// 프롬프트 표시
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
