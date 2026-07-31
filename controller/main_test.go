package controller

import (
	"os"
	"testing"

	"ms-gateway/common/logger"
)

// TestMain 테스트 실행 전 로거를 초기화한다.
// (로거 미초기화 시 trySend 등의 recover 경로에서 nil 포인터 panic 발생)
func TestMain(m *testing.M) {
	_ = os.MkdirAll("./logs", 0o755)
	_ = logger.InitLogger("ctl-test", "prod", 1, 1)

	code := m.Run()

	_ = os.RemoveAll("./logs")
	os.Exit(code)
}
