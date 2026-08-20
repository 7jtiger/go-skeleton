# Redis chat history key policy

## 대상
- `models/redis_db.go`

## 정책
- DM 메시지 이력은 `chat:rooms:{roomID}` LIST 키를 사용한다.
- `SaveChatMessage`는 `RPUSH`로 저장한다.
- `GetChatMessages(offset, limit, fromIndex)`는 offset 기반 (내부·목록 last_msg용). `fromIndex`는 나가기/재입장 사용자별 컷오프.
- `GetChatMessagesByCursor(roomID, cursor, limit, fromIndex)`는 DM 히스토리 API용 커서 페이지네이션.
- `DM:HIST:FROM:{uid}:{roomId}`: LeaveDMRoom·재입장(`ensureDMRoom`) 시 현재 LIST 길이로 설정. GetChatList는 viewer의 fromIndex 이후만 반환. **나간/재입장 사용자만** 컷오프, 남은 사용자는 전체 내역 유지.
- 방 상태(`st_chat` / API `partner_left`)는 **MySQL `chat_his` SSOT**. Redis는 메시지·unread·히스토리 컷오프만 담당.
  - `cursor` 없음: 최신 `limit`건
  - `cursor`: 해당 메시지 id보다 오래된 `limit`건
  - 반환: `next_cursor`(배치 내 가장 오래된 id), `has_more`
  - 실시간 신규 메시지는 WebSocket; HTTP는 과거 구간만
