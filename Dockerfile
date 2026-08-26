# --- API Stage ---
FROM golang:alpine AS api-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api ./cmd/api/main.go

FROM alpine:latest AS api
RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 1000
WORKDIR /app
COPY --from=api-builder --chown=appuser:appgroup /app/api .
USER 1000:1000
EXPOSE 8080
CMD ["./api"]

# --- UI Stage ---
FROM node:20 AS ui-builder
WORKDIR /app
COPY ui/package*.json ./
RUN npm install
COPY ui/ .
RUN npx expo export -p web

FROM golang:alpine AS ui-server-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o web ./cmd/web/main.go

FROM alpine:latest AS ui
RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 1000
WORKDIR /app
COPY --from=ui-server-builder --chown=appuser:appgroup /app/web .
COPY --from=ui-builder --chown=appuser:appgroup /app/dist ./ui/dist
USER 1000:1000
EXPOSE 8081
CMD ["./web"]
