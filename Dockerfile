# --- Stage 1: Build ---
FROM golang:1.21-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-w -s" -o /app/emby-302 .

# --- Stage 2: Final Image ---
FROM alpine:3.18

RUN apk add --no-cache tzdata
WORKDIR /app
COPY --from=builder /app/emby-302 .
EXPOSE 8091
CMD ["/app/emby-302"]