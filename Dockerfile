############# builder
FROM golang:1.13.4 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-shoot-auditlog-service
COPY . .
RUN make install-requirements && make VERIFY=true all

############# gardener-extension-shoot-auditlog-service
FROM alpine:3.11.3 AS gardener-extension-shoot-auditlog-service

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-shoot-auditlog-service /gardener-extension-shoot-auditlog-service
ENTRYPOINT ["/gardener-extension-shoot-auditlog-service"]

############# shoot-auditlog-proxy
FROM alpine:3.11.3 AS shoot-auditlog-proxy

COPY charts /charts
COPY --from=builder /go/bin/shoot-auditlog-proxy /shoot-auditlog-proxy
ENTRYPOINT ["/shoot-auditlog-proxy"]
