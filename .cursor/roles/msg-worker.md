# Scheduler msg worker policy

## 대상
- `scheduler/msg_worker.go`
- `models/redis_db.go`

## 동작
- `msg_remove` 스케줄 작업은 `MsgWorker()`를 통해 실행한다.
- 실제 Redis 삭제 로직은 `RedisDB.PruneExpiredRoomMessages(ttl)`에서 수행한다.

## 키 규칙
- 메시지 키 패턴: `chat:rooms:*:msg`
- 메시지 보관 기준: 24시간 초과 메시지 삭제
