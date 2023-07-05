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
	@echo "Building app for Windows (AMD64), Linux (AMD64) & MacOS (ARM64)..."
	@go mod tidy
	@GOOS=windows GOARCH=amd64 go build -ldflags="-X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=$(BUILD_MODE)'" -o release/steamquery_windows_amd64/ ./...
	@GOOS=linux GOARCH=amd64 go build -ldflags="-X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=$(BUILD_MODE)'" -o release/steamquery_linux_amd64/ ./...
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=$(BUILD_MODE)'" -o release/steamquery_darwin_arm64/ ./...
	@echo "Done building app"

# DO NOT CHANGE.
build-release:
	@[ "${STEAMQUERY_BUILD_VERSION}" ] || ( echo "STEAMQUERY_BUILD_VERSION is not set"; exit 1 )
	@echo "Building release-ready app for Windows (AMD64), Linux (AMD64) & MacOS (ARM64)..."
	@go mod tidy
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -v -trimpath -ldflags="-s -w -X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=release'" -o release/steamquery_windows_amd64/ ./...
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -trimpath -ldflags="-s -w -X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=release'" -o release/steamquery_linux_amd64/ ./...
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -v -trimpath -ldflags="-s -w -X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=$(BUILD_VERSION)' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=${shell date}' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=release'" -o release/steamquery_darwin_arm64/ ./...
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
	@cd ./testing && ./steamquery -c "./files/config.dev.json" -z -d

# DO NOT CHANGE.
clean:
	@clear
	@go mod tidy
	@rm -rf ./logs
	@rm -rf ./release
	@rm -rf ./testing
	@rm -rf ./dist
