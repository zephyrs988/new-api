#!/usr/bin/env bash
# 构建 Docker 镜像并推送到 suziheng/new-api:<时间戳>
set -euo pipefail

REPO="suziheng/new-api"
TAG="${REPO}:$(date '+%Y%m%d%H%M%S')"

echo "构建镜像: $TAG"
docker buildx build \
  --platform linux/amd64 \
  -t "$TAG" \
  --push \
  .

echo "推送完成: $TAG"
