# Scheduler Package Function Feature Summary

## scheduler.go - Scheduling System Functions

### Core Data Structures
- `item struct`: Individual scheduled task configuration containing name, description, arguments, delay, ticker, and quit channel
  - **주의**: `ticker`는 반드시 `*time.Ticker` 포인터로 보관. 값 복사(`*time.NewTicker(...)`) 시 Go 1.23+에서 GC가 원본의 내부 타이머를 회수해 채널 수신 중 SIGSEGV(fatal error) 발생
- `Schedule struct`: Main scheduler management structure containing configuration, account database, and task items

### Time Calculation Functions
- `getDuration()`: Calculates the duration until the next scheduled execution based on start time configuration (daily, minute, 5-minute intervals, or custom seconds)

### Scheduler Management Functions
- `NewScheduler()`: Constructor function that initializes the scheduler, loads configuration, connects to database, and sets up scheduled tasks
- `Scheduler()`: Goroutine-based task scheduler that handles periodic execution with initial delay and recurring intervals
- `close()`: Cleanup function that stops the ticker for a specific scheduled task item
- `task()`: Task dispatcher that executes specific functions based on task name (depositcheck, makeexceldaily, 1min, etc.)
- `tmp()`: Placeholder function for temporary task execution (currently empty implementation)
- `MsgWorker()` / `MsgDeleter()`: `msg_remove`/`msg_delete` 작업에서 Redis DM 메시지 정리(`ExpiredMsg(3일)`) 호출
- `Stop()`: `sync.Once` + `WaitGroup` 기반 idempotent 종료 — `context cancel` → job goroutine 종료 대기 → 각 job `ticker/quit` 1회만 `close` (`runJob` defer에서 중복 close 하지 않음)

---

## mailing.go - Email Service Functions

### Email Service Management
- `NewMailing()`: Constructor function that initializes the mailing service and logs the initialization
- `SendMail()`: Email sending function that logs the mail sending action (basic implementation)

---

## scd_test.go - Testing Functions

### Email Testing Functions
- `TestSendMail()`: Comprehensive test function that sends actual emails via SMTP server with TLS encryption using mailplug.co.kr
- `TestDuration()`: Test function that validates time duration calculations for different scheduling scenarios

### Scheduler Testing Functions
- `TestFstCheck()`: Test function for first-time deposit checking functionality (currently commented out/disabled)

---

## Overall Structure Summary

### scheduler.go (7 functions)
- **Core Management**: Constructor, scheduler runner, task dispatcher (3 functions)
- **Time Utilities**: Duration calculation (1 function)
- **Resource Management**: Task cleanup (1 function)
- **Task Execution**: Actual task runner, placeholder function (2 functions)

### mailing.go (2 functions)
- **Service Management**: Constructor, mail sending (2 functions)

### scd_test.go (3 functions)
- **Email Testing**: SMTP mail sending test (1 function)
- **Time Testing**: Duration calculation test (1 function)
- **Feature Testing**: Deposit check test (1 function)

---

## Key Features
- **Flexible Scheduling**: Supports daily, hourly, 5-minute interval, and custom second-based scheduling
- **Interval Rule**: `start`는 첫 실행 시점 정렬용, `duration`(초)는 그 이후 반복 주기. 첫 tick 후 `ticker.Reset(delay)`로 주기만 변경 (티커 재생성 금지 — 채널 교체/GC 회수 방지)
- **Concurrent Execution**: Uses goroutines for non-blocking task execution
- **Configuration-Driven**: Tasks are loaded from configuration files with execute flags (run/exe/o for immediate, cron for cron-style)
- **Database Integration**: Connected to AccountDB for data operations
- **Email Capability**: SMTP-based email sending with TLS encryption support
- **Graceful Shutdown**: Proper cleanup with quit channels and ticker stopping

---

## Task Types Supported
- **depositcheck**: Deposit verification and checking functionality
- **makeexceldaily**: Daily Excel report generation
- **1min**: One-minute interval tasks
- **Custom tasks**: Flexible task naming and execution
- **msg_remove** / **msg_delete**: `chat:rooms:*:msg` 키를 스캔하여 **3일** 초과 메시지 삭제 (`ExpiredMsg`)

---
*Created: 2025-09-15*
*Last Updated: 2025-11-05*
*File Location: /home/jino/go/src/ms-gateway/scheduler/*
