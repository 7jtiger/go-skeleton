# Go parameters
GO=go
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOTEST=$(GO) test
GOGET=$(GO) get
GORUN=$(GO) run

# Binary names
PACKAGE = basesk
# GOPATH  = $(CURDIR)/.gopath
BASE    = $(CURDIR)$(PACKAGE)
FLAG	= GOOS=linux GOARCH=amd64

Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")

# make 명령어를 실행하면 all 타겟이 실행되어 test와 build가 차례대로 실행
all: $(BASE) ; $(info $(M) building executable…) @ ## Build program binary
	@$(GO) mod tidy
	$Q $(FLAG) $(GOBUILD) -o build/$(PACKAGE)
	@mkdir -p $(CURDIR)/build/conf
	@mkdir -p $(CURDIR)/build/data
	@mkdir -p $(CURDIR)/build/logs
	@cp -f $(CURDIR)/conf/config.toml $(CURDIR)/build/conf/config.toml
	@cp -f $(CURDIR)/run $(CURDIR)/build/

build:
	$(GOBUILD) -o $(PACKAGE)

test:
	$(GOTEST) -v ./...

# clean은 빌드한 바이너리 파일을 삭제
clean:
	$(GOCLEAN)
	rm -f $(PACKAGE)
	rm -f build/$(PACKAGE)

run:
	$(GORUN) main.go

# run은 build를 실행하고 바로 프로그램을 실행
exe:
	$(GOBUILD) -o $(PACKAGE)
	./$(PACKAGE)

# Swagger 빌드
swg:
	@echo "Building Swagger..."
	@swag init -g main.go
