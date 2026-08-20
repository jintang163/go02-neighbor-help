# 运行镜像：单阶段 golang:1.22
#
# 说明：本机 Docker Desktop 访问 registry-1.docker.io 不稳定（alpine 拉取超时），
# 故采用单阶段 golang:1.22 直接构建运行，避免多阶段 alpine 镜像拉取失败。
FROM golang:1.22

WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .

RUN go build -o /usr/local/bin/nh-server .

RUN mkdir -p /app/data

ENV APP_ADDR=:8080
ENV APP_DATA_PATH=/app/data/store.json
ENV APP_ADMIN_USERNAME=admin
ENV APP_ADMIN_PASSWORD=admin123

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/bin/bash", "-c", "curl -fsS http://localhost:8080/healthz || exit 1"]

CMD ["nh-server"]
