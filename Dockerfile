FROM golang:1.21.1-alpine3.18 AS build-env

WORKDIR /go/src/github.com/nexa/nexa

COPY go.mod go.sum ./

RUN set -eux; apk add --no-cache ca-certificates=20230506-r0 build-base=0.5-r3 git=2.40.1-r0 linux-headers=6.3-r0

RUN --mount=type=bind,target=. --mount=type=secret,id=GITHUB_TOKEN \
    git config --global url."https://$(cat /run/secrets/GITHUB_TOKEN)@github.com/".insteadOf "https://github.com/"; \
    go mod download

COPY . .

RUN make build

RUN go install github.com/MinseokOh/toml-cli@latest

FROM alpine:3.18

WORKDIR /root

COPY --from=build-env /go/src/github.com/nexa/nexa/build/nexad /usr/bin/nexad
COPY --from=build-env /go/bin/toml-cli /usr/bin/toml-cli

RUN apk add --no-cache ca-certificates=20230506-r0 jq=1.6-r3 curl=8.2.1-r0 bash=5.2.15-r5 vim=9.0.1568-r0 lz4=1.9.4-r4 rclone=1.62.2-r4 \
    && addgroup -g 1000 nexa \
    && adduser -S -h /home/nexa -D nexa -u 1000 -G nexa

USER 1000
WORKDIR /home/nexa

EXPOSE 26656 26657 1317 9090 8545 8546

CMD ["nexad"]
