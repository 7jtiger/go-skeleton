# Scheduler msg worker policy

## 대상
- `scheduler/msg_worker.go`
- `models/redis_db.go`

## 동작
- `msg_remove` / `msg_delete` 스케줄 작업은 `MsgDeleter()`를 통해 실행한다.
- 실제 Redis 삭제 로직은 `RedisDB.ExpiredMsg(ttl)`에서 수행한다 (기본 3일).

## 키 규칙
- 메시지 키 패턴: `chat:rooms:*:msg`
- 메시지 보관 기준: **3일** 초과 메시지 삭제
- 읽음 포인터: `DM:READ:{uid}:{roomID}` TTL 3일
- 미읽음: `DM:UNREAD:{uid}:{roomID}` TTL 3일
