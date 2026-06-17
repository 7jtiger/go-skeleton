# Livein 회사 홈페이지

웹 솔루션·실시간 채팅 서비스 업체 **Livein**의 공개 홈페이지와 관리자 CMS입니다.

- **프론트엔드**: Svelte 5 SPA (`site/`)
- **백엔드**: 독립 Gin 서버 (`server/`, 포트 3000)
- **콘텐츠**: JSON 파일 기반 (`server/data/content.json`)

ms-gateway API(8080)와 **분리**된 경량 서버입니다.

## 디렉터리 구조

```
web/
├── site/          # Svelte + Vite 소스
├── server/        # Gin API + 정적 파일 서빙
├── dist/          # Vite 빌드 산출물
├── Makefile
└── README.md
```

## 빠른 시작

### 개발 모드 (권장)

**한 번에 실행** — API + 프론트 동시 기동:

```bash
cd web
make dev
```

브라우저에서 **http://localhost:5173** 접속

> 개발 모드는 **두 프로세스**가 필요합니다.
> - Vite (`:5173`) — 화면 + HMR
> - Gin (`:3000`) — `/api` 데이터 (Vite가 프록시)
>
> `make dev-site`만 실행하면 API가 없어 홈페이지가 "콘텐츠를 불러올 수 없습니다"로 보일 수 있습니다.

**터미널 2개로 실행**할 경우:

```bash
# 터미널 1
make dev-server

# 터미널 2
make dev-site
```

### 프로덕션 빌드 및 실행

```bash
make build    # site 빌드 → dist 복사 → Go 바이너리 생성
make run      # embed 모드로 단일 바이너리 실행
```

서버 주소: http://localhost:3000

## 관리자

| 항목 | 값 |
|------|-----|
| URL | http://localhost:3000/admin/login |
| 기본 ID | `admin` |
| 기본 PW | `changeme` |

`server/conf/config.toml`의 `[admin]` 섹션에서 계정·JWT secret을 변경하세요.

### 관리 기능

- `/admin` — 대시보드
- `/admin/content` — 홈페이지 콘텐츠 편집
- `/admin/inquiries` — 문의 목록

## API

| Method | Path | 설명 |
|--------|------|------|
| GET | `/api/health` | 헬스체크 |
| GET | `/api/content` | 공개 콘텐츠 조회 |
| POST | `/api/contact` | 문의 접수 |
| POST | `/api/admin/login` | 관리자 로그인 |
| PUT | `/api/admin/content` | 콘텐츠 수정 (JWT) |
| GET | `/api/admin/inquiries` | 문의 목록 (JWT) |

## 페이지

| 경로 | 설명 |
|------|------|
| `/` | 홈 (Hero, 서비스, 솔루션, 채팅, 신뢰, 프로세스) |
| `/solutions` | 솔루션 상세 |
| `/chat` | 채팅 서비스 소개 |
| `/about` | 회사 소개 |
| `/contact` | 문의 폼 |

## 설정

`server/conf/config.toml`:

```toml
[server]
port = ":3000"
mode = "dev"          # prod 시 release 모드
staticDir = "../dist" # dev 파일시스템 서빙 경로

[admin]
username = "admin"
password = "changeme"
jwtSecret = "..."
jwtExpireMin = 480

[data]
contentPath = "./data/content.json"
inquiryPath = "./data/inquiries.json"
```

## 배포 참고

- `make run`은 `-embed` 플래그로 `server/dist/`를 바이너리에 포함합니다.
- 배포 전 `make build`로 최신 프론트엔드를 반영하세요.
- 프로덕션에서는 `mode = "prod"`, 강력한 `jwtSecret`, bcrypt 해시 비밀번호 사용을 권장합니다.
