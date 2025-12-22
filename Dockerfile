# --- API Stage ---
FROM golang:alpine AS api-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o api ./cmd/api/main.go

FROM alpine:latest AS api
WORKDIR /app
COPY --from=api-builder /app/api .
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
RUN go build -o web ./cmd/web/main.go

FROM alpine:latest AS ui
WORKDIR /app
COPY --from=ui-server-builder /app/web .
COPY --from=ui-builder /app/dist ./ui/dist
EXPOSE 8081
CMD ["./web"]
