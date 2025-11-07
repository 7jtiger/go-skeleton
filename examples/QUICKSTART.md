# 🚀 빠른 시작 가이드

## Go P2P 화상채팅 클라이언트

5분 안에 P2P 화상채팅을 테스트해보세요!

### 1단계: 서버 시작 ⚙️

```bash
cd /home/jino/go/src/ms-gateway
./ms-gateway
```

### 2단계: 클라이언트 빌드 🔨

```bash
cd /home/jino/go/src/ms-gateway/examples
go build -o video-chat-client video_chat_client.go
```

### 3단계: 첫 번째 사용자 실행 👤

새 터미널:
```bash
cd /home/jino/go/src/ms-gateway/examples
./video-chat-client alice
```

### 4단계: 두 번째 사용자 실행 👤

또 다른 새 터미널:
```bash
cd /home/jino/go/src/ms-gateway/examples
./video-chat-client bob
```

### 5단계: 통화 시작 📞

alice 터미널에서:
```bash
> call bob
```

### 6단계: 메시지 전송 💬

연결 후:
```bash
> send bob Hello Bob!
```

bob 터미널에서 응답:
```bash
> send alice Hi Alice!
```

### 7단계: 통화 종료 📴

```bash
> end bob
```

## 🎉 완료!

P2P 연결과 메시지 전송이 성공했습니다!

## 다음 단계

- `help` 명령으로 더 많은 기능 확인
- `README_GO_CLIENT.md`에서 상세 문서 확인
- 여러 사용자와 동시 통화 테스트

---
**도움이 필요하신가요?** README_GO_CLIENT.md의 "문제 해결" 섹션을 참조하세요.
