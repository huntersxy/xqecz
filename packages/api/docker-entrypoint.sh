#!/bin/sh
# 启动时将前端构建产物复制到共享 volume，供 nginx 读取
if [ -d /app/frontend/dist ] && [ "$(ls -A /app/frontend/dist)" ]; then
  cp -r /app/frontend/dist/* /frontend-dist/ 2>/dev/null || true
fi
exec node dist/main.js
