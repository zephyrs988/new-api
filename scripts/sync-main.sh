#!/usr/bin/env bash
# 拉取 GitHub 最新 main 分支并合并到当前分支
set -euo pipefail

CURRENT=$(git rev-parse --abbrev-ref HEAD)

echo "当前分支: $CURRENT"
echo "正在拉取 origin/main 最新代码..."
git fetch origin main

echo "合并 origin/main 到 $CURRENT ..."
git merge origin/main --no-edit

echo "完成，已将 origin/main 合并到 $CURRENT"
