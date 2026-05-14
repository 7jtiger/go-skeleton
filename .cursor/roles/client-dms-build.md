# client/dms build rule

## 배경
- `go build`는 `_test.go` 파일을 컴파일 대상에 포함하지 않는다.

## 체크 항목
- `client/dms` 폴더에 일반 Go 파일(예: `main.go`)이 최소 1개 있어야 한다.
- CLI 실행 코드는 `_test.go`가 아니라 일반 `.go` 파일로 분리해야 한다.
- 채팅 히스토리 조회는 `/chatlist [roomId] [page] [limit]` 명령으로 사용한다.
