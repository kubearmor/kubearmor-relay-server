### Builder

FROM golang:1.15.2-alpine3.12 as builder

RUN apk update
RUN apk add --no-cache bash git wget python3 linux-headers build-base clang clang-dev libc-dev bcc-dev librdkafka

WORKDIR /usr/src/kubearmor-relay-server

COPY ./common ./common
COPY ./log ./log
COPY ./server ./server
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY ./main.go ./main.go

RUN GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o kubearmor-relay-server main.go

### Make executable image

FROM alpine:3.14

COPY --from=builder /usr/src/kubearmor-relay-server/kubearmor-relay-server /KubeArmor/kubearmor-relay-server

ENTRYPOINT ["/KubeArmor/kubearmor-relay-server"]
