FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY allstar-mock.go go.mod go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o allstar-mock allstar-mock.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/allstar-mock ./

# Create non-root user
RUN addgroup -g 1001 -S appuser && \
    adduser -u 1001 -S appuser -G appuser
USER appuser

EXPOSE 34001/udp

ENTRYPOINT ["./allstar-mock"]