FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY orchestrator/go.mod orchestrator/go.sum ./
RUN go mod download
COPY orchestrator/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o orchestrator .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/orchestrator /orchestrator
CMD ["/orchestrator"]
