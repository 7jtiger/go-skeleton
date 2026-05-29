# Redis chat history key policy

## 대상
- `models/redis_db.go`

## 정책
- DM 메시지 이력은 `chat:rooms:{roomID}` LIST 키를 사용한다.
- `SaveChatMessage`는 `RPUSH`로 저장한다.
- `GetChatMessages(offset, limit)`는 최신 메시지 기준으로 페이징한다.
  - offset=0: 최신부터
  - limit: 페이지 크기 (`<=0`이면 메시지는 비우고, 반환되는 전체 건수는 그대로 `LLen`)
  - 반환값 첫 번째 `int64`는 **항상** 해당 방 메시지 전체 개수(`LLen`). offset이 전체를 넘거나 방이 비어 있어도 동일.
