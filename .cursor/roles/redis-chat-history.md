# Redis chat history key policy

## 대상
- `models/redis_db.go`

## 정책
- DM 메시지 이력은 `chat:rooms:{roomID}` LIST 키를 사용한다.
- `SaveChatMessage`는 `RPUSH`로 저장한다.
- `GetChatMessages(offset, limit)`는 최신 메시지 기준으로 페이징한다.
  - offset=0: 최신부터
  - limit: 페이지 크기
