# client/dms CLI 기능 요약

## 파일
- `client/dms/main.go`

## 주요 명령
- `/server <addr>`: 서버 주소 설정
- `/token <jwt>`: JWT 설정
- `/target <uid>`: 상대 UID 설정
- `/room <roomId>`: 현재 방 ID 설정
- `/connect`, `/disconnect`: DM WebSocket 연결/종료
- `/mkroom [pid]`: DM 룸 생성
- `/rooms [page]`: DM 룸 목록 조회
- `/unread`: 전체 미읽음 조회
- `/chatlist [roomId] [limit] [cursor]`: 채팅방 과거 메시지 조회 (커서 페이지네이션)
  - `roomId` 생략 시 현재 `/room` 값 사용
  - 기본값: `limit=20`, `cursor` 없음(최신부터)
  - 호출 경로: `GET /dm/v01/history/{roomId}?limit={limit}&cursor={cursor}`
  - 응답: `messages`, `total_count`, `next_cursor`, `has_more` (신규 메시지는 WS)
