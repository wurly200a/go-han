#!/bin/bash
set -euo pipefail

cd ~/go-han

echo "[deploy] git pull..."
git fetch origin
git rebase origin/main

echo "[deploy] building & restarting backend/frontend..."
docker compose up -d --build --no-deps backend frontend

echo "[deploy] done."
docker compose ps
