# Livein Web Server

## 개요

`ms-gateway/web/` 디렉터리의 독립 Gin 서버입니다. Livein 회사 홈페이지(Svelte SPA) 정적 파일 서빙과 JSON 기반 CMS API를 제공합니다.

ms-gateway 본 서버(8080)와 분리되어 **포트 3000**에서 동작합니다.

## 아키텍처

```
Browser → Gin (:3000)
            ├── /assets/*     정적 파일 (Vite 빌드)
            ├── /api/content  공개 CMS 조회
            ├── /api/contact  문의 접수
            ├── /api/admin/*  관리자 API (JWT)
            └── NoRoute       SPA index.html fallback
```

## 주요 파일

| 파일 | 역할 |
|------|------|
| `server/main.go` | 진입점, go:embed dist |
| `server/router/router.go` | 라우팅, 정적/SPA 서빙 |
| `server/controller/controller.go` | content, contact, auth 핸들러 |
| `server/store/file.go` | JSON 파일 atomic read/write |
| `server/data/content.json` | 홈페이지 콘텐츠 |
| `server/data/inquiries.json` | 문의 내역 |
| `server/conf/config.toml` | 서버·관리자·경로 설정 |

## API

### 공개

- `GET /api/health` — 상태 확인
- `GET /api/content` — CMS 콘텐츠
- `POST /api/contact` — `{ name, email, message }`

### 관리자 (Bearer JWT)

- `POST /api/admin/login` — `{ username, password }`
- `PUT /api/admin/content` — 전체 Content JSON
- `GET /api/admin/inquiries` — 문의 목록

## 실행

```bash
cd web
make dev-server   # 개발 (파일시스템 dist)
make build && make run  # 프로덕션 (embed)
```

## 보안

- 관리자 API는 JWT 미들웨어 적용
- SecurityHeaders: X-Content-Type-Options, X-Frame-Options
- 프로덕션: config.toml의 password를 bcrypt 해시로 교체 가능 (평문 fallback은 개발용)

## 프론트엔드

Svelte 5 SPA는 `web/site/`에 있으며, Vite dev 서버(5173)에서 `/api`를 3000으로 프록시합니다.

자세한 내용은 `web/README.md` 참고.
