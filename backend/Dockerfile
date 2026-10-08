# 阶段 1：构建前端
FROM node:22-alpine AS frontend
WORKDIR /build
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --registry=https://registry.npmmirror.com || npm install --registry=https://registry.npmmirror.com
COPY frontend/ ./
RUN npm run build

# 阶段 2：构建后端（前端产物打包进 static/）
FROM golang:1.26-alpine AS backend
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /build/dist ./static
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/card-server .

# 阶段 3：运行
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1000 app
ENV TZ=Asia/Shanghai CARD_ENV=prod
WORKDIR /app
COPY --from=backend /out/card-server /app/card-server
COPY --from=backend /src/static /app/static
USER app
VOLUME /app/data
EXPOSE 8080
ENTRYPOINT ["/app/card-server"]
