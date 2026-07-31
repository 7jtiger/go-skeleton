# client/dms CLI 기능 요약

## 파일
- `client/dms/main.go` — DM 채팅·나가기·내역 테스트용 콘솔 클라이언트

## 실행
```bash
cd client/dms && go run .
```

## 주요 명령

### 설정
- `/server <addr>`: 서버 주소 (기본 `localhost:8080`)
- `/token <jwt>`: JWT 설정
- `/target <uid>`: 상대 UID
- `/room <roomId>`: 방 ID 수동 설정
- `/status`: 세션 상태 (room, cursor, myLastMid)

### WebSocket
- `/connect`, `/disconnect`
- `/send <text>`, `/typing`, `/read [msgId]`
- `/call`, `/accept`, `/reject`, `/cancel`
- `/gifts`: 선물 프리셋 목록
- `/gift [item] [수량]`, `/present`: DM 선물 전송 (WS `type=put-gift`, `content`=수량 문자열)
  - 예: `/gift 3`, `/gift rose 1`, `/gift 화살 5`
  - 수신: `put-gift` + `msg-ack` (텍스트와 동일 흐름)
  - 서버 `handleSendGift`는 현재 수신 문구에 아이템명 "화살" 고정 (포인트 차감 TODO)

### REST — 방
- `/mkroom [pid]`, `/rejoin [pid]`: 방 생성·재입장 (`POST /dm/v01/mkroom/:pid`) — 재입장 시 `DM:HIST:FROM` 컷오프 갱신
- `/leave [roomId]`, `/rmroom`: 방 나가기 (`POST /dm/v01/rmroom/:room_id`)
- `/rinfo [roomId] [mid]`: 방 상세·입장 (`GET /dm/v01/rinfo/:room_id`) — unread 초기화
- `/exists <tid>`: 방 존재 여부 (`GET /dm/v01/room/exists/:tid`)
- `/rooms [page]`: 인박스 (`GET /dm/v01/list/:page`) — `total_count`, `last_msg`, `partner_left` 표시

### REST — 메시지
- `/chatlist [roomId] [limit] [cursor]`: 채팅 내역 (`GET /dm/v01/history/:room_id`)
  - 응답: `messages` (Redis `StoredMessage`: id, user_id, content, type, timestamp), `total_count` (viewer 기준 visible), `next_cursor`, `has_more`, `my_last_mid`, `partner_last_mid`
  - 나간 뒤 재입장한 사용자는 `total_count=0` / 빈 messages (상대는 기존 내역 유지)
- `/chatmore [limit]`: 이전 `/chatlist`의 `next_cursor`로 추가 조회
- `/unread`: 전체 미읽음 (`total_count`, `rooms` 맵)

## 나가기/재입장 테스트 시나리오
1. 터미널 A,B 각각 `/token`, `/target`, `/connect`
2. A: `/mkroom <B_uid>` → `/send hello`
3. B: `/rooms` → `/chatlist <rid>` (내역 확인)
4. A: `/leave` → B: `/chatlist` (B는 내역 유지)
5. A: `/mkroom <B_uid>` → `/chatlist` (A는 빈 내역)
6. A: `/send new` → 양쪽 `/chatlist` 비교

## 선물 테스트 시나리오
1. A,B `/connect` 후 A: `/mkroom <B>`
2. A: `/gift rose 1` → B: WS `put-gift` 수신 + A: `msg-ack`
3. B: `/chatlist <rid>` → `put-gift` 타입 메시지·선물 문구 확인

## API 응답 필드 정합 (2026-07)
- 방 목록: `total_count` (not `totalcount`)
- 미읽음: `total_count` + `rooms`
- 채팅 내역 메시지: WS `ChatMessage`가 아닌 Redis `StoredMessage` 구조
