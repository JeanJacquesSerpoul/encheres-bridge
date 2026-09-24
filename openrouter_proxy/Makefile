BINARY  = server_ai
LDFLAGS = -ldflags "-s -w"

ifeq ($(OS),Windows_NT)
    SHELL := powershell.exe
    .SHELLFLAGS := -NoProfile -Command
    MKDIR_DIST = New-Item -ItemType Directory -Force -Path dist | Out-Null
    RM_DIST    = if (Test-Path dist) { Remove-Item -Recurse -Force dist }
    RM_COV     = if (Test-Path coverage.out) { Remove-Item coverage.out }
    SET_LINUX  = $$env:GOOS = 'linux'; $$env:GOARCH = 'amd64'; $$env:CGO_ENABLED = '0';
    SET_WIN    = $$env:GOOS = 'windows'; $$env:GOARCH = 'amd64';
else
    MKDIR_DIST = mkdir -p dist
    RM_DIST    = rm -rf dist
    RM_COV     = rm -f coverage.out
    SET_LINUX  = CGO_ENABLED=0 GOOS=linux GOARCH=amd64
    SET_WIN    = GOOS=windows GOARCH=amd64
endif

.PHONY: all build tidy test clean docker-build docker-up docker-down

all: tidy build

## Build Linux and Windows AMD64 binaries into dist/
build:
	$(MKDIR_DIST)
	$(SET_LINUX) go build $(LDFLAGS) -o dist/$(BINARY) .
	$(SET_WIN) go build $(LDFLAGS) -o dist/$(BINARY).exe .

tidy:
	go mod tidy

test:
	go test ./... -race -coverprofile=coverage.out

clean:
	$(RM_DIST)
	$(RM_COV)

## Docker
docker-build:
	docker build -t $(BINARY):latest .

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
