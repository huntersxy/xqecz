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

**CI 只发后端**。前端由 EdgeOne Makers 自行构建（`xq.xiey.work` 已绑到 Makers）：仓库根 `.env` 被 gitignore，
`VITE_API_BASE_URL` / `VITE_MEDIA_BASE_URL` 只能在 Makers 控制台配置，因此 CI 既不构建也不发布前端，
前端构建失败在 Makers 的构建日志里可见。构建因此只需要 Go，不必再装 Node/pnpm。

流程（目标机没有面板、没有 FTP，只有 SSH 与 OpenRC，故旧的三段式面板调用已删除）：

1. **构建**：`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"`
2. **上传**：`scp` 直传为 `xqecz-server.new`——**不能直接覆盖 `xqecz-server`**，正在运行的二进制被内核拒写（`ETXTBSY`）；只碰这一个文件，不触碰同目录的 `.env` 与 `data/`
3. **收尾**：`doas mv -f xqecz-server.new xqecz-server` 原子替换（不改变运行中进程持有的 inode）→ `doas rc-service xqecz restart`
4. **验证健康**：轮询 `HEALTH_URL` 直到 200
5. **验证产物**：读回**部署目录**与**运行进程**（`/proc/<pid>/exe`）两份 md5，与本次构建比对

第 5 步不能省：`mv` 或 `restart` 静默失败时服务照样 200，实际跑的还是上一版二进制——只有比对「运行中的那个」才暴露。
SSH 输出是同步的，读回即可，不必像旧版（宝塔 `ExecShell` 异步）那样写文件→`GetFileBody` 取回→解析 JSON 包装。
计数用 `pidof` 而非 `pgrep -c`：**busybox 的 `pgrep` 没有 `-c`**，且 `pgrep -f` 会匹配到执行该脚本的 shell 自身。

需要的仓库 Secrets（Settings → Secrets and variables → Actions）：

| Secret | 用途 |
|--------|------|
| `DEPLOY_HOST` | 部署机地址（仓库内不出现服务器 IP） |
| `DEPLOY_SSH_KEY` | 专用部署私钥；对应公钥写进目标机的 `~/.ssh/authorized_keys` |
| `DEPLOY_PORT` | SSH 端口，**可选**，workflow 缺省 22；当前目标机设为 **20222**（见下节） |

用户名不是机密，直接写在 workflow 里：`alpine`（其家目录含 `authorized_keys`）。

注意：`HEALTH_URL` 写在 workflow 的 `env`（非机密），换服务或换域名时改这里。**必须打 API 源站**——
`xq.xiey.work` 是 Makers 静态站且不反代，`/api/*` 在那里返回 404，拿它当探针会「服务明明好的却判失败」。

## 另一台生产机：Alpine + OpenRC（39.101.76.249）

除宝塔形态外，后端也可直接跑在精简 Linux 上。当前 39.101.76.249 即此形态（Alpine 3.20、非 root 用户 `alpine` 配免密 `doas`、无 Docker/宝塔）：

```
/opt/xqecz/
├── xqecz-server          # Linux amd64 二进制（chmod +x）
├── .env                  # 生产配置（600，alpine 属主）
├── migrate-to-tidb.sh    # 库迁移脚本副本
└── data/                 # uploads / thumbs / bin（媒体本地托管，缩略图纯本地）
```

| 项 | 值 |
|----|-----|
| 服务管理 | `/etc/init.d/xqecz`（OpenRC，`command_user=alpine`，日志 `/var/log/xqecz-server.log`） |
| 自启 | `rc-update add xqecz default`，与 `chronyd` 同 runlevel |
| SSH 端口 | **20222**（定义在 `/etc/ssh/sshd_config.d/20-port.conf`）。安全组**未放行 2222**——实测 22/20222/29418/40022 通、2222 不通，改端口别想当然选 2222。原 22 上常态有爆破连接（一排 `sshd [accepted]` 子进程），换掉后降到个位数 |
| 数据库 | TiDB Cloud Serverless（`MYSQL_TLS=true`，独立库 `xqecz`） |
| Redis | 共享实例，**独立前缀 `xqeczgo:`**（与旧实例 `xqecz:` 隔离） |
| 媒体 | 本地 `data/` 托管；`thumbs` 不镜像 R2，必须随库一起迁移 |
| 日志轮转 | `/etc/periodic/daily/xqecz-logrotate`（超 5MB 轮转，保留 7 份，copytruncate 免重启） |

常用命令：

```bash
doas rc-service xqecz status|restart        # 服务状态与控制
doas tail -f /var/log/xqecz-server.log      # 日志（中文正常，勿用 GBK 工具读）
doas chronyc tracking                       # 时钟偏移（R2 403 时先查这里）
```

两个坑：

- **时钟必须先同步**：该机首次启动时钟慢了 13.7 小时，导致 R2 签名全部 403（凭据无误也照拒）。`chronyc makestep` 校正，并在 `chrony.conf` 补 `makestep 1.0 3` 让开机也步进。
- **媒体不会自动出现**：`thumbs` 是纯本地资源且首页瀑布流全靠它，迁库时必须一并搬 `data/uploads` 与 `data/thumbs`（`bin` 是垃圾桶，可不搬）。
- **改 SSH 端口必须双端口过渡**：先 `Port 22` + `Port 20222` 并存 → 从外部用密钥认证新口成功 → 改 `~/.dsh/dsh-ssh.json` 与 `DEPLOY_PORT` secret → 最后才撤 22。安全组在云控制台、这里改不了，一旦新口没放行就是把自己关在外面。验证端口是否放行的廉价办法：在机器上绑几个候选端口、从外部逐个 TCP 连一下（**只读、不碰 sshd**）。

## 从 MySQL/MariaDB 迁移到 TiDB

用 `scripts/migrations/2026-09-26-migrate-to-tidb.sh`（需 `mysql`/`mysqldump` CLI，两端凭据走环境变量）：

```bash
SRC_HOST=<源> SRC_USER=<u> SRC_PASS=<p> SRC_DB=xqv2 \
TGT_HOST=<tidb网关> TGT_PORT=4000 TGT_USER=<u> TGT_PASS=<p> TGT_DB=xqecz \
sh scripts/migrations/2026-09-26-migrate-to-tidb.sh        # 目标库已有同名表会拒绝，加 RECREATE=1 才重建
```

脚本自动处理三处差异并做逐字节校验：去掉 `TEXT`/`BLOB`/`JSON` 的 `DEFAULT`（TiDB 报 1101）、丢弃 MariaDB 私有 `/*M!` 指令、按主键有序导出后比对 md5（TiDB 不支持 `CHECKSUM TABLE`）。任一张表不一致即非零退出。

TiDB 侧另两点：`sys`/`mysql` 等系统库对业务账号**只读**，必须建独立库；连接强制加密，`.env` 需 `MYSQL_TLS=true`。

## Cloudflare R2 媒体镜像（可选）

原图与压缩图各在 R2 留一份（缩略图纯本地），用于给访问 R2 更快的用户做详情页大图加速。

**启用方式**：在部署目录的 `.env`（不是仓库里的 `.env.example`）补上 R2 段并重启后端：

```bash
R2_ACCOUNT_ID=<Cloudflare 账号 ID>
R2_ACCESS_KEY_ID=<R2 令牌的 Access Key ID>
R2_SECRET_ACCESS_KEY=<R2 令牌的 Secret Access Key>
R2_BUCKET=<桶名>
R2_PREFIX=uploads                 # 桶内公共前缀，默认 uploads
R2_PUBLIC_BASE=https://file.example.com   # 对象公开域名（自定义域名或 r2.dev）
```

要点：

- **四项凭据任一为空即整体停用**：任务静默休眠、`mirror_img` 字段为空，前端自动走源站，行为与未接入时完全一致。
- **`R2_PUBLIC_BASE` 决定前端能否测速**：留空 = 只镜像不对外暴露（纯备份）；填了才会开启「详情页大图在源站与 R2 之间择快」。换域名只改这里，**不需要重新构建前端**。
- **R2 是 append-only 的**：内容删除时本地文件进 `data/bin` 保留，R2 对象不删，便于回溯。
- **首次部署会回填存量**：启动后立即扫描全部内容并补齐缺失对象（每批 500 条，经 Redis 锁保证多实例只有一个在跑），之后每 `R2_SYNC_INTERVAL_SECONDS`（默认 300s）扫一轮兜住偶发失败。
- **服务器需能出网到 `*.r2.cloudflarestorage.com`**（实测该服务器到 Cloudflare 通，`api.cloudflare.com` 约 0.8s）。若走不通，镜像会持续失败重试但不影响上传与访问——本地始终是权威副本。
- 对象名与本地同构：`<R2_PREFIX>/<文件名>`，桶内可直接与 `data/uploads` 对账。
- 压缩是**原地改写**（路径不变、内容变），因此 R2 上的对象名不变、内容会随压缩更新；判重只认内容 md5，不认「传过没有」。

排查用命令（部署机上直接跑，无需 Go）：

```bash
grep -iE 'R2_' /www/wwwroot/xqecz-golang/.env          # 确认配置已就位
tail -f /www/wwwlogs/go/xqeczserver.log | grep -E 'r2 '  # 看镜像/回填日志
```

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
