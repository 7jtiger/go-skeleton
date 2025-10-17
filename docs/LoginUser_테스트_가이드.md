# LoginUser 함수 테스트 가이드

## 개요
`LoginUser` 함수의 포괄적인 테스트를 위해 작성된 테스트 코드들에 대한 설명과 실행 방법을 안내합니다.

## 테스트 파일 구조

### 1. Controller 레벨 테스트 (`controller/login_test.go`)
HTTP API 엔드포인트 레벨에서의 통합 테스트

### 2. Database 레벨 테스트 (`models/login_db_test.go`)
데이터베이스 접근 로직의 단위 테스트

## 테스트 케이스 분류

### 🟢 **기본 기능 테스트**
- `TestLoginUser`: 정상적인 로그인 시나리오
- `TestLoginUserInvalidCredentials`: 잘못된 패스워드 테스트
- `TestLoginUserNonExistentUser`: 존재하지 않는 사용자 테스트

### 🟡 **입력 검증 테스트**
- `TestLoginUserMissingFields`: 필수 필드 누락 테스트
- `TestLoginUserMalformedJSON`: 잘못된 JSON 형식 테스트
- `TestLoginUserSpecialCharacters`: 특수문자 포함 데이터 테스트

### 🔴 **보안 테스트**
- SQL Injection 시도 테스트
- 패스워드 암호화/복호화 검증
- 특수문자 및 위험한 입력값 처리

### ⚡ **성능 테스트**
- `TestLoginUserConcurrency`: 동시 요청 처리 테스트
- `TestLoginUserStressTest`: 스트레스 테스트
- `BenchmarkLoginUser`: 성능 벤치마크

### 🧪 **엣지 케이스 테스트**
- `TestLoginUserEdgeCases`: 극단적인 입력값 테스트
- 매우 긴 ID/패스워드
- 유니코드 문자 처리
- 공백만으로 이루어진 입력

### 🔧 **시스템 테스트**
- `TestLoginUserTimeout`: 타임아웃 시뮬레이션
- `TestLoginUserRateLimit`: 레이트 리미팅 테스트
- `TestLoginUserResponseValidation`: 응답 구조 검증

## 테스트 실행 방법

### 전체 로그인 테스트 실행
```bash
# Controller 레벨 테스트
cd /home/jino/go/src/ms-gateway
go test ./controller -v -run TestLogin

# Database 레벨 테스트  
go test ./models -v -run TestLogin
```

### 개별 테스트 실행
```bash
# 기본 로그인 테스트만 실행
go test ./controller -v -run TestLoginUser$

# 보안 관련 테스트만 실행
go test ./controller -v -run TestLoginUserSpecialCharacters

# 성능 테스트 실행
go test ./controller -v -run TestLoginUserStressTest
```

### 벤치마크 테스트 실행
```bash
# 성능 벤치마크
go test ./models -bench=BenchmarkLoginUser -benchmem

# 패스워드 복호화 벤치마크
go test ./models -bench=BenchmarkPasswordDecryption -benchmem
```

### 스트레스 테스트 제외하고 실행
```bash
# short 모드로 실행 (스트레스 테스트 제외)
go test ./controller -v -short -run TestLogin
```

## 테스트 환경 설정

### 사전 준비사항
1. **서버 실행**: 테스트 대상 서버가 `http://127.0.0.1:8080`에서 실행 중이어야 함
2. **테스트 데이터**: 테스트용 사용자 계정이 데이터베이스에 존재해야 함
   - ID: `testuser`
   - PW: `1234`

### 테스트 데이터 설정
```sql
-- 테스트용 사용자 데이터 삽입 예시
INSERT INTO user_info (sid, uid, pw_hash, nick, name, email) 
VALUES ('testuser', 'testuid', '[encrypted_password]', '테스트유저', '테스트', 'test@example.com');
```

## 예상 테스트 결과

### ✅ 성공 케이스
```json
{
  "result_code": "100",
  "result_msg": "success",
  "data": {
    "uid": "testuid",
    "nick": "테스트유저",
    "token": "jwt_token_here"
  }
}
```

### ❌ 실패 케이스 (잘못된 패스워드)
```json
{
  "result_code": "104", 
  "result_msg": "failed to login user"
}
```

### ❌ 실패 케이스 (필수 필드 누락)
```json
{
  "result_code": "102",
  "result_msg": "all fields are required"
}
```

## 테스트 커버리지 확인

```bash
# 커버리지 측정
go test ./controller -coverprofile=controller_coverage.out -run TestLogin
go test ./models -coverprofile=models_coverage.out -run TestLogin

# 커버리지 리포트 생성
go tool cover -html=controller_coverage.out -o controller_coverage.html
go tool cover -html=models_coverage.out -o models_coverage.html
```

## Mock 테스트 활용

Database 레벨 테스트에서는 실제 데이터베이스 대신 Mock을 사용하여:
- 빠른 테스트 실행
- 의존성 격리
- 다양한 시나리오 시뮬레이션

## 주의사항

### 🚨 **테스트 실행시 주의점**
1. **서버 실행 필요**: API 테스트는 실제 서버가 실행 중이어야 함
2. **데이터베이스 상태**: 테스트 데이터가 올바르게 설정되어 있어야 함
3. **네트워크 상태**: localhost 연결이 가능해야 함
4. **동시성 테스트**: 시스템 리소스에 영향을 줄 수 있음

### 📝 **테스트 결과 해석**
- **PASS**: 테스트 성공
- **FAIL**: 테스트 실패 (상세 오류 메시지 확인)
- **SKIP**: 조건부 스킵된 테스트 (예: short 모드)

## 문제 해결

### 자주 발생하는 문제
1. **서버 연결 실패**: 서버가 실행 중인지 확인
2. **테스트 데이터 없음**: 데이터베이스에 테스트 사용자 계정 생성
3. **권한 문제**: 포트 8080 사용 권한 확인
4. **타임아웃 오류**: 네트워크 또는 서버 성능 확인

### 디버깅 팁
```bash
# 상세 로그와 함께 테스트 실행
go test ./controller -v -run TestLogin 2>&1 | tee test.log

# 특정 테스트만 실행하여 문제 격리
go test ./controller -v -run TestLoginUser$ -count=1
```

## 확장 가능성

### 추가 테스트 시나리오
- JWT 토큰 검증 테스트
- Redis 세션 관리 테스트
- WebRTC 설정 통합 테스트
- 로그인 시도 제한 테스트
- 다중 디바이스 로그인 테스트

---

*작성일: 2025-09-15*  
*작성자: AI Assistant*  
*파일 위치: `/home/jino/go/src/ms-gateway/docs/LoginUser_테스트_가이드.md`*





