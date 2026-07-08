FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder
ARG TARGETOS TARGETARCH
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod ./
COPY go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -trimpath -o /onvif-relay ./cmd/onvif-relay

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /onvif-relay /usr/bin/onvif-relay
EXPOSE 8080/tcp 3702/udp
VOLUME ["/config"]
ENTRYPOINT ["/usr/bin/onvif-relay"]
CMD ["-config", "/config/config.yaml"]
