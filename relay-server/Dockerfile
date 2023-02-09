### Builder

FROM golang:1.15.2-alpine3.12 as builder

RUN apk update
RUN apk add --no-cache make

WORKDIR /usr/src/kubearmor-relay-server

COPY . .

RUN cd relay-server && make
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.15 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

### Copy executable image

FROM alpine:3.14

COPY --from=builder /usr/src/kubearmor-relay-server/relay-server/kubearmor-relay-server /KubeArmor/kubearmor-relay-server
COPY --from=builder /bin/grpc_health_probe ./grpc_health_probe

ENTRYPOINT ["/KubeArmor/kubearmor-relay-server"]
