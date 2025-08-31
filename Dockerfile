# 使用 Debian 最新稳定版作为基础镜像
FROM debian:bookworm-slim

# 安装运行时依赖
RUN apt-get update && apt-get install -y \
    ca-certificates \
    sqlite3 \
    libopus0 \
    procps \
    && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 复制 WSL 编译好的二进制文件
COPY _build/server /app/main
COPY config.yaml /app/config.yaml

# 确保可执行权限
RUN chmod +x /app/main

# 启动命令
CMD ["./main"]
