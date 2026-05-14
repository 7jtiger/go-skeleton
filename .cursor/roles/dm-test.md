# DM Test Role Notes

## 목적
- DM 통합 테스트에서 최소 검증 범위를 고정한다.

## Test_SendDMMessage 체크리스트
- 송신자 WS 연결 성공
- 수신자 WS 연결 성공
- DM 룸 생성 성공 (`POST /dm/v01/mkroom/:pid`)
- 송신자가 `text-message` 전송 성공
- 수신자가 동일 payload(`from`, `to`, `content`) 수신
- 송신자가 `msg-ack` 수신 및 `msgId` 일치 확인

## 실행 파라미터
- `-dm_target`
- `-dm_token`
- `-dm_uid`
- `-dm_peer_uid`
- `-dm_peer_token`
