FROM golang:1.20.5 AS builder
ENV TZ=Europe/Berlin
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
ARG BUILD_VERSION
ARG BUILD_MODE
ARG GIT_COMMIT
RUN mkdir /app
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o steamquery -v -trimpath -ldflags="-s -w -X 'github.com/devusSs/steamquery-v2/updater.BuildVersion=${BUILD_VERSION}' -X 'github.com/devusSs/steamquery-v2/updater.BuildDate=$(date)' -X 'github.com/devusSs/steamquery-v2/updater.BuildMode=${BUILD_MODE}' -X 'github.com/devusSs/steamquery-v2/updater.BuildGitCommit=${GIT_COMMIT}'" ./...

FROM alpine:latest AS production
ENV TZ=Europe/Berlin
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
COPY --from=builder /app/steamquery ./
COPY --from=builder /app/files/gcloud.json ./files/gcloud.json
RUN mv ./steamquery ./steamquery-v2
CMD ["./steamquery-v2", "-d", "-du", "-e", "-w"]
