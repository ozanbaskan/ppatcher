FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY webapp/go.mod webapp/go.sum ./webapp/
RUN cd webapp && go mod download

COPY webapp/ ./webapp/

# Also copy full project source for wails builds and server binary builds
COPY server/ ./server/
COPY go.mod go.sum ./
COPY build-client.sh ./
COPY app.go config.go main.go ./
COPY config.json ./
COPY wails.json ./
COPY frontend/ ./frontend/
COPY build/darwin/ ./build/darwin/
COPY build/windows/ ./build/windows/

RUN cd webapp && CGO_ENABLED=0 GOOS=linux go build -o /app/webapp-bin .

# Pre-download main module dependencies
RUN go mod download

FROM golang:1.25-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata openssh-client sshpass bash git curl jq \
    nodejs npm pkg-config unrar-free \
    libgtk-3-dev libwebkit2gtk-4.1-dev \
    && rm -rf /var/lib/apt/lists/*

# Install wails CLI
RUN go install github.com/wailsapp/wails/v2/cmd/wails@latest

WORKDIR /ppatcher

# Copy the full project source (needed for wails build, go build ./server/, and build-client.sh)
COPY --from=builder /app/server/ ./server/
COPY --from=builder /app/go.mod ./go.mod
COPY --from=builder /app/go.sum ./go.sum
COPY --from=builder /app/build-client.sh ./build-client.sh
COPY --from=builder /app/app.go ./app.go
COPY --from=builder /app/config.go ./config.go
COPY --from=builder /app/config.json ./config.json
COPY --from=builder /app/main.go ./main.go
COPY --from=builder /app/wails.json ./wails.json
COPY --from=builder /app/frontend/ ./frontend/
COPY --from=builder /app/build/ ./build/
RUN chmod +x build-client.sh

# Pre-download dependencies and install frontend deps
RUN go mod download
RUN cd frontend && npm install

# Copy the webapp binary
COPY --from=builder /app/webapp-bin /ppatcher/webapp

ENV PPATCHER_ROOT=/ppatcher

EXPOSE 8080

CMD ["/ppatcher/webapp"]
