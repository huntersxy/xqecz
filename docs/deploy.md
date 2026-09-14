# 部署

## 产物

| 产物 | 路径 | 说明 |
|------|------|------|
| 后端 | `packages/server/xqecz-server` | Go 单二进制，`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`，`-ldflags="-s -w"`（约 31MB，静态链接、无 CGO） |
| 前端 | `packages/frontend/dist` | 静态文件，构建时已做图片压缩与分包 |

构建命令（本地或 CI 均可，产物与源码目录解耦）：

```bash
pnpm --filter ./packages/frontend run build          # 前端产物 → packages/frontend/dist
cd packages/server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -o xqecz-server ./cmd/server
```

## 生产形态：宝塔「Go 项目 + 静态站点」

生产**不使用 pnpm/npm 统一编排**：后端与前端分别作为独立项目发布，服务器上不需要包管理器。

| 侧 | 面板项目类型 | 配置要点 |
|----|--------------|----------|
| 后端 | Go 项目 | 项目目录＝部署目录；启动命令 `./xqecz-server`；运行用户 `www`；端口与 `.env` 的 `PORT` 一致 |
| 前端 | 静态站点 | 站点根目录＝上传上来的 `dist` 内容；`/api/` 反代到后端端口；媒体目录 alias 到 `data/` |

后端只需以下文件即可运行（不依赖任何 Node 生态文件）：

```
<部署目录>/
├── xqecz-server          # 二进制（chmod +x）
├── .env                  # 配置（凭据、PORT、UPLOAD_DIR 等）
└── data/                 # 运行期生成：uploads / thumbs / images
```

`package.json`、`pnpm-workspace.yaml`、`scripts/start-backend.mjs` 只服务本地开发与旧的 Node 项目形态，生产可不传。

### `.env` 定位规则（重要）

进程启动时按以下优先级确定「项目根」，再读取 `项目根/.env`（`.env.local` 可覆盖）：

1. 环境变量 `PROJECT_ROOT`，其次 `XQECZ_ROOT`（面板里配环境变量最省事，绝对路径）
2. 自工作目录向上查找根标记：`.env`、`pnpm-workspace.yaml`、`.git`（命中即止，最多 8 层）

因此：

- 二进制与 `.env` 放在同一目录（或 `.env` 在二进制所在目录的上级）时，无需任何额外配置；
- 若面板工作目录与 `.env` 不在同一条父链上，请显式设 `PROJECT_ROOT`；
- 未命中任何标记时配置全部走默认值，典型症状是 `Access denied for user 'root'@'localhost' (using password: NO)`——看到了就是根目录没找对。

### nginx（前端站点）参考配置

```nginx
server {
    listen 80;
    server_name your.domain;
    root /www/wwwroot/<前端目录>;      # dist 内容所在目录
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:<后端端口>;   # 末尾不要多余斜杠，保持路径原样传递
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        client_max_body_size 25m;                 # 上传接口单文件上限 20MB，留余量
        proxy_read_timeout 300s;
    }

    # 媒体文件由 nginx 直出更快；也可不配，交给后端托管
    location ^~ /uploads/ { alias /www/wwwroot/<部署目录>/data/uploads/; expires 30d; try_files $uri =404; }
    location ^~ /thumbs/  { alias /www/wwwroot/<部署目录>/data/thumbs/;  expires 30d; try_files $uri =404; }
    location ^~ /images/  { alias /www/wwwroot/<部署目录>/data/images/;  expires 30d; try_files $uri =404; }

    location / { try_files $uri $uri/ /index.html; }   # SPA 深链兜底
}
```

`/api/` 与前端同源后无需再配 `CORS_ORIGINS`；前端构建产物使用相对路径 `/api`（`VITE_API_BASE_URL`）。

### 从旧实例迁移时要对齐的三项

1. **Redis 前缀**：新旧实例并存时必须区分（如 `xqeczgo:`），否则缓存键互相覆盖；完全替换旧实例则沿用原前缀。
2. **媒体目录**：`UPLOAD_DIR` / `THUMB_DIR` / `IMAGES_DIR` 决定落盘位置，必须与数据库里存量的 `/uploads`、`/thumbs`、`/images` 指向同一批文件，否则老图全部 404。
3. **端口**：`.env` 的 `PORT` 需与面板项目配置、nginx `proxy_pass` 三处一致。

## CI（`.github/workflows/deploy.yml`）

触发：push 到 `master`，或在 Actions 页手动触发（可取消勾选「部署后重启后端进程」）。

流程（两端独立产物，互不依赖）：

1. **构建**：前端 `vite build` → `dist`；后端 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"`
2. **后端上传**：`curl -T` 单文件直传 `xqecz-server` 到后端 FTP 账号根（＝ Go 项目目录）；只覆盖这一个文件，不触碰同目录的 `.env` 与 `data/`
3. **前端上传**：`FTP-Deploy-Action` 增量同步 `dist` 到静态站点根
4. **收尾**：经宝塔面板文件接口 `ExecShell` 远程执行「赋可执行位 + 重启」，与面板启动方式一致（`www` 用户、日志追加到 `/www/wwwlogs/go/xqeczserver.log`）
5. **验证**：轮询生产 `HEALTH_URL` 直到返回 200，否则作业失败（面板 `ExecShell` 为异步执行，只能事后校验）

需要的仓库 Secrets（Settings → Secrets and variables → Actions）：

| Secret | 用途 |
|--------|------|
| `FTP_SERVER` | FTP 主机地址 |
| `FTP_USERNAME_BACKEND` | 后端 FTP 账号（其根目录＝ Go 项目目录） |
| `FTP_PASSWORD` | 上述账号密码（前端账号共用此变量） |
| `FTP_USERNAME_FRONTEND` | 前端 FTP 账号（其根目录＝静态站点根） |
| `BT_API_TOKEN` | 宝塔面板「API 接口」密钥，用于远程赋权与重启 |

注意：面板地址与健康检查地址直接写在 workflow 的 `env`（非机密，`BT_PANEL_URL` / `HEALTH_URL`），换服务或换域名时改这里。

## 初始化与维护

```bash
# 首次部署：创建管理员（密码仅打印一次）
./xqecz-server admin -username admin -email you@example.com

# 忘记密码：重置
./xqecz-server admin -username admin -reset

# 查看依赖连通性与表规模（开发机，需 Go 工具链）
cd packages/server && go run ./cmd/dbinfo
```

管理员初始化必须由二进制自身提供子命令——部署机没有 Go 工具链，`go run` 形式的命令在服务器上不可用。

## 回滚

- 后端：部署前保留上一版 `xqecz-server`，回滚即覆盖回去并重启面板项目
- 前端：`dist` 为静态文件，保留上一版目录或压缩包，回滚即替换目录内容
