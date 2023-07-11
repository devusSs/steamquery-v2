# Update the version to your needs via env / shell.
BUILD_VERSION=$(STEAMQUERY_BUILD_VERSION)
BUILD_MODE=${STEAMQUERY_BUILD_MODE}

# DO NOT CHANGE.
BUILD_OS 				:=
ifeq ($(OS),Windows_NT)
	BUILD_OS = windows
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		BUILD_OS = linux
	endif
	ifeq ($(UNAME_S),Darwin)
		BUILD_OS = darwin
	endif
endif

# DO NOT CHANGE.
BUILD_ARCH 				:=
ifeq ($(echo %PROCESSOR_ARCHITECTURE%), "AMD64")
	BUILD_ARCH = amd64
else
	UNAME_M := $(shell uname -m)
	ifeq ($(UNAME_M), x86_64)
		BUILD_ARCH = amd64
	endif
	ifeq ($(UNAME_M), arm64)
		BUILD_ARCH = arm64
	endif
endif

# DO NOT CHANGE.
build:
	@[ "${STEAMQUERY_BUILD_VERSION}" ] || ( echo "STEAMQUERY_BUILD_VERSION is not set"; exit 1 )
	@[ "${STEAMQUERY_BUILD_MODE}" ] || ( echo "STEAMQUERY_BUILD_MODE is not set"; exit 1 )
	@echo "Cross-compiling app for Windows (AMD64), Linux (AMD64) & MacOS (ARM64)..."
	@go mod tidy
	@GOOS=windows GOARCH=amd64 go build -v -trimpath -ldflags="-s -w -X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=$(BUILD_MODE)' -X 'github.com/devusSs/steamquery-v2/updater.BuildGitCommit=${shell git rev-parse HEAD}'" -o release/steamquery_windows_amd64/ ./...
	@GOOS=linux GOARCH=amd64 go build -v -trimpath -ldflags="-s -w -X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=$(BUILD_MODE)' -X 'github.com/devusSs/steamquery-v2/updater.BuildGitCommit=${shell git rev-parse HEAD}'" -o release/steamquery_linux_amd64/ ./...
	@GOOS=darwin GOARCH=arm64 go build -v -trimpath -ldflags="-s -w -X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=$(BUILD_MODE)' -X 'github.com/devusSs/steamquery-v2/updater.BuildGitCommit=${shell git rev-parse HEAD}'" -o release/steamquery_darwin_arm64/ ./...
	@echo "Done building app"

# DO NOT CHANGE.
dev: build
	@clear
	@rm -rf ./testing
	@mkdir ./testing
	@mkdir ./testing/files
	@cp -R ./files ./testing
	@cp ./release/steamquery_$(BUILD_OS)_$(BUILD_ARCH)/steamquery ./testing
	@cd ./testing && ./steamquery -c "./files/config.dev.json" -d -du -sc

# DO NOT CHANGE.
run: build
	@clear
	@rm -rf ./testing
	@mkdir ./testing
	@mkdir ./testing/files
	@cp -R ./files ./testing
	@cp ./release/steamquery_$(BUILD_OS)_$(BUILD_ARCH)/steamquery ./testing
	@cd ./testing && ./steamquery -c "./files/config.dev.json" -d -du

# DO NOT CHANGE.
beta: build
	@clear
	@rm -rf ./testing
	@mkdir ./testing
	@mkdir ./testing/files
	@cp -R ./files ./testing
	@cp ./release/steamquery_$(BUILD_OS)_$(BUILD_ARCH)/steamquery ./testing
	@cd ./testing && ./steamquery -c "./files/config.dev.json" -d -du -sc -b

# DO NOT CHANGE.
watchdog: build
	@clear
	@rm -rf ./testing
	@mkdir ./testing
	@mkdir ./testing/files
	@cp -R ./files ./testing
	@cp ./release/steamquery_$(BUILD_OS)_$(BUILD_ARCH)/steamquery ./testing
	@cd ./testing && ./steamquery -c "./files/config.dev.json" -d -du -w

# DO NOT CHANGE.
version: build
	@clear
	@rm -rf ./testing
	@mkdir ./testing
	@cp ./release/steamquery_$(BUILD_OS)_$(BUILD_ARCH)/steamquery ./testing
	@cd ./testing && ./steamquery -v -du -d

# DO NOT CHANGE.
analysis: build
	@clear
	@rm -rf ./testing
	@mkdir ./testing
	@mkdir ./testing/files
	@cp -R ./files ./testing
	@cp ./release/steamquery_$(BUILD_OS)_$(BUILD_ARCH)/steamquery ./testing
	@cd ./testing && ./steamquery -c "./files/config.dev.json" -a -du -d

# DO NOT CHANGE.
stats: build
	@clear
	@rm -rf ./testing
	@mkdir ./testing
	@mkdir ./testing/files
	@cp -R ./files ./testing
	@cp ./release/steamquery_$(BUILD_OS)_$(BUILD_ARCH)/steamquery ./testing
	@cd ./testing && ./steamquery -c "./files/config.dev.json" -z -d -du

# DO NOT CHANGE.
env: build
	@clear
	@rm -rf ./testing
	@mkdir ./testing
	@mkdir ./testing/files
	@cp -R ./files ./testing
	@cp docker.env ./testing/docker.env
	@cp ./release/steamquery_$(BUILD_OS)_$(BUILD_ARCH)/steamquery ./testing
	@cd ./testing && ./steamquery -c "./files/config.dev.json" -e -efile="docker.env" -d -du

# DO NOT CHANGE.
docker-up:
	@clear
	@echo "Checking for docker.env file..."
	@[ -f ./docker.env ] && echo "Found docker.env file, proceeding..." || echo "ERROR: missing docker.env file. Please create a docker.env file and edit it" && exit 0
	@export STEAMQUERY_GIT_COMMIT=${shell git rev-parse HEAD} && docker compose --env-file=docker.env up --build -d
	@echo ""
	@echo "Please use 'make docker-down' to shutdown the app and containers."

# DO NOT CHANGE.
docker-down:
	@echo "This command will NOT delete Docker images or volumes (data will be persistent)."
	@echo "If you need functionality to remove images and volumes please use the Docker GUI / CLI if available."
	@export STEAMQUERY_GIT_COMMIT=${shell git rev-parse HEAD} && docker compose --env-file=docker.env down

# DO NOT CHANGE.
clean:
	@clear
	@go mod tidy
	@rm -rf ./logs
	@rm -rf ./release
	@rm -rf ./testing
	@rm -rf ./dist
	@rm -rf ./tmp
