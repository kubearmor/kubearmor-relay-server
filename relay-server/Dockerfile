### Builder

FROM golang:1.20-alpine3.17 as builder

RUN apk update
RUN apk add --no-cache make

WORKDIR /usr/src/kubearmor-relay-server

COPY . .

RUN cd relay-server && make

### Copy executable image

FROM alpine:3.14

COPY --from=builder /usr/src/kubearmor-relay-server/relay-server/kubearmor-relay-server /KubeArmor/kubearmor-relay-server

ENTRYPOINT ["/KubeArmor/kubearmor-relay-server"]
