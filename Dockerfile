# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

COPY go.mod ./
COPY go.sum* ./
COPY . .
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o outpost .

# Stage 3: Final image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates ffmpeg mkvtoolnix chromaprint

WORKDIR /app

COPY --from=backend-builder /app/outpost .
COPY --from=frontend-builder /app/frontend/build ./frontend/build

EXPOSE 8080

CMD ["./outpost"]
