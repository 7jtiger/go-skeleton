# Scheduler graceful shutdown policy

## 목표
- 스케줄러 종료는 panic 없이 여러 번 호출되어도 안전해야 한다.

## 규칙
- `Schedule.Stop()`는 `sync.Once`로 1회만 실제 종료 동작을 수행한다.
- 각 job 종료는 item별 `sync.Once`로 보호한다.
- job 종료 시 `ticker.Stop()`과 `quit` 채널 close를 함께 처리한다.
- 스케줄러는 `context.WithCancel`을 사용하고, `Stop()`에서 cancel을 호출한다.
