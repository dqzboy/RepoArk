#!/usr/bin/env bash
#
# 一键构建：前端 -> 复制产物到 server/webroot/dist -> 编译内嵌前端的后端二进制
# 用法：./scripts/build.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> [1/3] 构建前端 (web)"
cd web
npm ci
npm run build
cd "$ROOT"

echo "==> [2/3] 复制前端产物到 server/webroot/dist"
rm -rf server/webroot/dist
mkdir -p server/webroot/dist
cp -r web/dist/. server/webroot/dist/

echo "==> [3/3] 编译后端二进制（内嵌前端）"
cd server
go build -trimpath -ldflags="-s -w" -o ../repoark-server .
cd "$ROOT"

echo ""
echo "✅ 构建完成：生成 ./repoark-server 二进制文件"
echo "   运行 ./repoark-server 后访问 http://localhost:8080 即可使用前端页面"
