# server/Dockerfile

# ---- 构建阶段 ----
FROM golang:1.21-alpine AS builder

WORKDIR /build

# 安装必要的构建工具
RUN apk add --no-cache gcc musl-dev

# 先复制依赖文件，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码
COPY . .

# 编译二进制（静态链接，适合 alpine 运行）
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/server ./cmd/main.go

# ---- 运行阶段 ----
FROM alpine:3.19

WORKDIR /app

# 安装 CA 证书（HTTPS 外部请求需要）和时区数据
RUN apk add --no-cache ca-certificates tzdata

# 从构建阶段复制二进制文件
COPY --from=builder /build/server .

# 复制配置文件
COPY config.yaml .

# 创建上传目录
RUN mkdir -p /app/uploads

EXPOSE 8080

CMD ["./server"]
