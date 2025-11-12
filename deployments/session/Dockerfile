# 构建阶段
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /build

# 安装必要的构建工具
RUN apk add --no-cache git make

# 复制 go mod 文件（利用 Docker 缓存层）
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建 session 二进制文件
# CGO_ENABLED=0 禁用 CGO，生成静态链接的二进制文件
# -ldflags '-w -s' 去除调试信息，减小二进制文件大小
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags '-w -s' \
    -o session \
    ./cmd/session/main.go

# 运行阶段
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata netcat-openbsd

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/session /app/session

# 从构建阶段复制配置文件
COPY --from=builder /build/config/session.yaml /app/config/session.yaml

# 创建日志目录
RUN mkdir -p /home/www/logs/applogs && \
    chown -R appuser:appuser /app /home/www/logs/applogs

# 切换到非 root 用户
USER appuser

# 暴露端口
# 9001: gRPC 端口（用于接收 Gateway 等服务调用）
EXPOSE 9001

# 设置健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z localhost 9001 || exit 1

# 启动命令
ENTRYPOINT ["/app/session"]
CMD ["--config", "/app/config/session.yaml"]

