.PHONY: all build build-site build-server dev dev-site dev-server run run-dev tidy clean help

SITE_DIR := site
SERVER_DIR := server
DIST_DIR := dist
BINARY := server/livein-web

all: build

help:
	@echo "Livein Web — 실행 방법"
	@echo ""
	@echo "  [개발]  make dev          API + 프론트 동시 실행 (권장)"
	@echo "          make dev-server   터미널 1 — Gin API  :3000"
	@echo "          make dev-site     터미널 2 — Vite     :5173"
	@echo "          브라우저: http://localhost:5173"
	@echo ""
	@echo "  [배포]  make build && make run"
	@echo "          브라우저: http://localhost:3000"
	@echo ""
	@echo "  관리자: http://localhost:5173/admin/login (admin / changeme)"

# 개발: API 서버 + Vite 동시 실행 (Ctrl+C로 모두 종료)
dev:
	@echo "Starting API (:3000) + Vite (:5173)..."
	@echo "Open http://localhost:5173 in your browser"
	@trap 'kill 0' INT TERM; \
	cd $(SERVER_DIR) && go run main.go -config ./conf/config.toml & \
	sleep 1; \
	cd $(SITE_DIR) && pnpm dev; \
	wait

dev-site:
	cd $(SITE_DIR) && pnpm dev

dev-server:
	cd $(SERVER_DIR) && go run main.go -config ./conf/config.toml

build-site:
	cd $(SITE_DIR) && CI=true pnpm install && pnpm build

build-server: build-site
	@mkdir -p $(SERVER_DIR)/dist
	@rm -rf $(SERVER_DIR)/dist/*
	@cp -r $(DIST_DIR)/* $(SERVER_DIR)/dist/
	cd $(SERVER_DIR) && go mod tidy && go build -ldflags="-s -w" -o livein-web .

build: build-server

run: build
	cd $(SERVER_DIR) && ./livein-web -config ./conf/config.toml -embed

run-dev:
	cd $(SERVER_DIR) && go run main.go -config ./conf/config.toml

tidy:
	cd $(SERVER_DIR) && go mod tidy

clean:
	rm -rf $(DIST_DIR) $(SERVER_DIR)/dist $(SERVER_DIR)/livein-web
