# go-skeleton-servern (server using mvc model with go-lang)

Server-client 아키텍처 패턴에서 server 파트를 표현한 예제 코드와 폴더 구조 입니다.
<br/>예시 코드는 Mongo DB 사용으로작성되어 있습니다.

## 폴더 구조
폴더 구조는 아래와 같습니다.
```
./go-server-skeleton
├── LICENSE
├── Makefile
├── README.md
├── build
├── conf
│   ├── config.go
│   └── config.toml
├── common
├── controller
├── Docs
├── go.mod
├── go.sum
├── logger
│   └── logger.go
├── logs
│   └── go_logger_2022-12-19.log
├── main.go
├── protocol
├── model
└── router
│   └── router.go
└── vendor

11 directories
```

주요 파일 및 디렉터리 설명은 다음과 같습니다.
* main.go : 서버의 엔트리 포인트 역할
* build : make 빌드시 배포 경로, 배포시 필요한 파일 수록
* conf: 서버의 설정 파일을 포함
* common : 공통 라이브러리 및 유틸
* controller: model과 view를 컨트롤 하는 구성으로 api 입출력의 시작점
* Docs : swagger 관련 파일
* logger: 로그작성에 대한 정의
* model : db의 출력 형태를 정의하고, 데이터를 핸들링
* log : 로그 파일 저장 
* protocol : api 응답 정의
* router : 서버의 라우트를 정의
* vendor : 라이브러리 파일 미러



**프로세스의 흐름**
request → router → controller → model → controller → protocol -> response

## 사용 방법
1. Go 설치 및 GOPATH 설정 후 go 폴더 아래 bin, pkg, src 폴더를 생성
2. src 밑으로 해당 레파지토리를 clone함
3. 패키지 초기화 및 실행

패키지 모듈 초기화
```
go mod init
```
패키지 모듈 정렬, 설정 및 go.sum 내용의 패키지 재설치
```
go mod tidy
```

swagger 초기화
```
swag init
```

VSCode에서 정상적인 링크임에도 빨간줄이 쳐지는경우
```
go mod vendor
```


실행
```
go run main.go
```

컴파일 및 실행
```
go bulid -o ready main.go
./ready
```
