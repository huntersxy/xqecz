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

## 生产形态：CI 直传 + OpenRC（后端）/ EdgeOne Makers（前端）

生产**不使用 pnpm/npm 统一编排**：后端与前端分别发布，服务器上不需要包管理器。

| 侧 | 形态 | 配置要点 |
|----|------|----------|
| 后端 | CI 经 SSH 直传 + OpenRC | 部署目录＝`/opt/xqecz`；启动命令 `./xqecz-server`；运行用户 `alpine`；端口与 `.env` 的 `PORT` 一致 |
| 前端 | EdgeOne Makers 自行构建 | Makers 控制台配 `VITE_API_BASE_URL` / `VITE_MEDIA_BASE_URL`（都打后端域名）；站点自身不反代 `/api`，跨域由后端 `CORS_ORIGINS` 白名单放行 |

前端检查由 `.github/workflows/frontend-check.yml` 独立运行，包含类型、单测及非空媒体基址的生产构建；通过检查不会触发它发布站点。Makers 构建和发布状态仍以平台记录为准。媒体地址归属、Canvas/WebGL 跨域及发布后的浏览器验收见 `docs/frontend-media.md`；本地正常或打包成功不能代替实际站点验收。

后端只需以下文件即可运行（不依赖任何 Node 生态文件）：

```
<部署目录>/
├── xqecz-server          # 二进制（chmod +x）
├── .env                  # 配置（凭据、PORT、UPLOAD_DIR 等）
└── data/                 # 运行期生成：uploads / thumbs / images（历史兼容）/ bin
```

`package.json`、`pnpm-workspace.yaml`、`scripts/start-backend.mjs` 只服务本地开发，生产可不传。

### `.env` 定位规则（重要）

进程启动时按以下优先级确定「项目根」，再读取 `项目根/.env`（`.env.local` 可覆盖）：

1. 环境变量 `PROJECT_ROOT`，其次 `XQECZ_ROOT`（在服务配置里设环境变量最省事，绝对路径）
2. 自工作目录向上查找根标记：`.env`、`pnpm-workspace.yaml`、`.git`（命中即止，最多 8 层）

因此：

- 二进制与 `.env` 放在同一目录（或 `.env` 在二进制所在目录的上级）时，无需任何额外配置；
- 若进程工作目录与 `.env` 不在同一条父链上（如经 OpenRC/systemd 拉起、工作目录是 `/`），请显式设 `PROJECT_ROOT`；
- 未命中任何标记时配置全部走默认值，典型症状是 `Access denied for user 'root'@'localhost' (using password: NO)`——看到了就是根目录没找对。

### nginx（自建前端站点）参考配置

> 当前生产前端由 EdgeOne Makers 托管、不经过 nginx；本节只在把 `dist` 放到自有服务器时参考。

```nginx
server {
    listen 80;
    server_name your.domain;
    root /srv/xqecz-web;              # dist 内容所在目录
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
    location ^~ /uploads/ { alias /opt/xqecz/data/uploads/; expires 30d; try_files $uri =404; }
    location ^~ /thumbs/  { alias /opt/xqecz/data/thumbs/;  expires 30d; try_files $uri =404; }
    location ^~ /images/  { alias /opt/xqecz/data/images/;  expires 30d; try_files $uri =404; }

    location / { try_files $uri $uri/ /index.html; }   # SPA 深链兜底
}
```

`/api/` 与前端同源后无需再配 `CORS_ORIGINS`；前端构建产物使用相对路径 `/api`（`VITE_API_BASE_URL`）。

### 从旧实例迁移时要对齐的三项

1. **Redis 前缀**：新旧实例并存时必须区分（如 `xqeczgo:`），否则缓存键互相覆盖；完全替换旧实例则沿用原前缀。
2. **媒体目录**：`UPLOAD_DIR` / `THUMB_DIR` / `IMAGES_DIR` 决定落盘位置，必须与数据库里存量的 `/uploads`、`/thumbs`、`/images` 指向同一批文件，否则老图全部 404。
3. **端口**：`.env` 的 `PORT` 需与服务管理配置（OpenRC 读同一份 `.env`）、nginx `proxy_pass` 三处一致。

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

后端跑在精简 Linux 上，无 Docker、无面板。当前 39.101.76.249 即此形态（Alpine 3.20、非 root 用户 `alpine` 配免密 `doas`）：

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
| 数据库 | TiDB Cloud Serverless **北京实例**（网关 `gateway01.cn-beijing.aliyun.pingkai.cn:4000`，`MYSQL_TLS=true`，独立库 `xqecz`）；2026-10-10 自广州实例等量搬入，见「从 MySQL/MariaDB 迁移到 TiDB」 |
| Redis | 共享实例，**独立前缀 `xqeczgo:`**（与旧实例 `xqecz:` 隔离） |
| 媒体 | 本地 `data/` 托管；`thumbs` 不镜像（纯本地），必须随库一起迁移 |
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

**源端也是托管 TiDB 时要显式开 TLS**：脚本对源端默认 `--skip-ssl`（自建 MySQL/MariaDB 常无证书），托管实例会直接拒连。这种「TiDB → TiDB」的等量搬运加 `SRC_SSL=--ssl`：

```bash
SRC_HOST=<源网关> SRC_USER=<u> SRC_PASS=<p> SRC_DB=xqecz SRC_SSL=--ssl \
TGT_HOST=<目标网关> TGT_PORT=4000 TGT_USER=<u> TGT_PASS=<p> TGT_DB=xqecz \
sh scripts/migrations/2026-09-26-migrate-to-tidb.sh
```

目标库要**先手建**，并显式指定源库的 collation——托管 TiDB 新实例的默认是 `utf8mb4_bin`，照搬会整库 collation 不一致：

```sql
CREATE DATABASE IF NOT EXISTS `xqecz` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
```

校验自增游标**不要看 `information_schema.tables.auto_increment`**：TiDB 对 `CLUSTERED` 自增主键的表（如 `content_likes`）恒返回 0，`ALTER TABLE ... AUTO_INCREMENT=` 也不生效。可靠判据是 `SHOW CREATE TABLE` 的 `AUTO_INCREMENT=` 与 `SHOW TABLE <t> NEXT_ROW_ID`，两端一致即无需手工对齐。

TiDB 侧另两点：`sys`/`mysql` 等系统库对业务账号**只读**，必须建独立库（建到 `sys` 上直接报 `ERROR 1142 ... CREATE command denied`）；连接强制加密，`.env` 需 `MYSQL_TLS=true`。

### 实例搬迁记录

**2026-09-26（首次）**：自建 MySQL → 广州 TiDB Cloud Serverless，脚本首版即为此用途。

**2026-10-10**：广州 TiDB → 北京 TiDB 等量搬迁，在 39 机上用 `/opt/xqecz/migrate-to-tidb.sh` 执行（`SRC_SSL=--ssl`），30 秒完成。

- 十张表逐表 md5 **全部一致**：users 59 / contents 313 / comments 37 / claims 13 / polls 1 / poll_votes 74 / content_likes 5 / content_favorites 1 / comment_reports 2 / api_keys 0；另做四轮结构对账（DDL、表级 collation 与注释、列级 79 行、索引定义）全部 SAME。
- 切换只改 39 机 `/opt/xqecz/.env` 的三个键（`MYSQL_HOST` / `MYSQL_USER` / `MYSQL_PASSWORD`），旧文件备份为 `.env.bak-20261010-134412`；`doas rc-service xqecz restart` 后 `/api/health`、`/api/content/tags`、`/api/content/recommend`、`/api/poll/list` 均 200 且是搬迁后数据。
- 开发机 `.env` 同步切到北京实例；39 机脚本副本已换成带 `SRC_SSL` 的版本（旧副本留作 `migrate-to-tidb.sh.bak-20260926`）。
- **旧广州实例未删，保留作回滚**：把 `.env` 三个键换回旧值并重启即可。注意搬迁后两端会分叉，回滚等于丢掉北京实例上的新写入，只在确认新实例异常时使用。

## Cloudflare R2 媒体镜像（可选，次选）

原图与压缩图各在 R2 留一份（缩略图纯本地）。R2 是镜像链上的**次选**——
首选是自建 OpenList（见下节），只有在它不可达时前端才用 R2；两者也在同一侧被并行推送。

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
- **`R2_PUBLIC_BASE` 决定前端能否用得上 R2**：留空 = 只镜像不对外暴露（纯备份）；填了才会下发 `mirror_img`，前端在 OpenList 不可达时退到这里。换域名只改这里，**不需要重新构建前端**。
- **R2 是 append-only 的**：内容删除时本地文件进 `data/bin` 保留，R2 对象不删，便于回溯。
- **首次部署会回填存量**：启动后立即扫描全部内容并补齐缺失对象（每批 500 条，经 Redis 锁保证多实例只有一个在跑），之后每 `R2_SYNC_INTERVAL_SECONDS`（默认 300s）扫一轮兜住偶发失败。
- **服务器需能出网到 `*.r2.cloudflarestorage.com`**（实测该服务器到 Cloudflare 通，`api.cloudflare.com` 约 0.8s）。若走不通，镜像会持续失败重试但不影响上传与访问——本地始终是权威副本。
- 对象名与本地同构：`<R2_PREFIX>/<文件名>`，桶内可直接与 `data/uploads` 对账。
- 压缩是**原地改写**（路径不变、内容变），因此 R2 上的对象名不变、内容会随压缩更新；判重只认内容 md5，不认「传过没有」。

排查用命令（部署机上直接跑，无需 Go）：

```bash
grep -iE '^(R2|OPENLIST)_' /opt/xqecz/.env   # 确认配置已就位
tail -f /var/log/xqecz-server.log | grep -E 'r2 '  # 看镜像/回填日志
```

### 自建 OpenList 镜像（可选，首选）

与 R2 并行再推一份到自建 OpenList。详情页按 **OpenList → R2 → 源站** 逐级回退，只有前一级不可达才用下一级。
OpenList 在前是因为它跑在自有广州机上、硬盘直出，不受第三方 CDN 计费与欠费停服的牵制；
R2 走 Cloudflare，国内访问常被拖慢甚至不通，故作为次选。基址或 token 任一为空即整体停用，不影响 R2 单独工作。

```bash
OPENLIST_ENDPOINT=https://drive.xiey.work  # OpenList 入口（推送用 /api/fs/put）
OPENLIST_TOKEN=<OpenList 后台的 admin token>
OPENLIST_PREFIX=uploads                     # 与 R2 同名，便于两侧对账
OPENLIST_PUBLIC_BASE=https://drive.xiey.work/d  # 对外下载基址，**必须带 /d**，前端据此取 mirror2_*
OPENLIST_SYNC_INTERVAL_SECONDS=300
OPENLIST_UPLOAD_TIMEOUT_SECONDS=120
```

四个容易踩的点：

- **鉴权头是 `Authorization: <token>`，不带 `Bearer`**。带了必被拒；请求头里不加引号以外的任何包装。
- **对象不存在不是 404**。`POST /api/fs/get` 对缺失对象回 `{"code":500,"message":"failed to get obj: object not found"}`，
  判存在只能匹配 message；免签下载 `/d/*` 对缺失文件同样回 JSON 500。回填逻辑依赖这个判定，改客户端时别按 HTTP 语义想当然。
- **没有远端哈希**。OpenList 的 Local 驱动 `hashinfo` 恒为 `null`，拿不到 md5，故判重退路是「远端字节数 == 本地字节数」
  （见 `internal/mirror` 的 `sameObject()`）。坚持「无哈希就重传」会让每轮回填全量重推。
- **`OPENLIST_PUBLIC_BASE` 必须带直链前缀 `/d`，只写裸域名是错的**。裸域名在 OpenList 侧是 SPA 前端，
  `https://drive.xiey.work/uploads/x.webp` 回的是 `200 text/html`（index.html 首页），而不是图片——
  表现是「换源成功但图全是 HTML」。正确值形如 `https://drive.xiey.work/d`，此时 `Target.ObjectURL()`
  拼出的才是 `https://drive.xiey.work/d/uploads/x.webp`。留空 = 只镜像不对外暴露（纯备份）；填了才会下发 `mirror2_*`。
  换域名只改这里，**不需要重新构建前端**。

#### 实例侧要求：免签直链

公开下载走 `/d/<对象路径>`，要求实例 `sign_all=false`，否则 `/d/`、`/p/` 裸访问一律 401。
置法（后台设置项）：`[{"key":"sign_all","value":"false"}]`。挂载点必须落在根 `/`，
这样 `/d/uploads/<文件名>` 才与生产对象键 `uploads/<文件名>` 对齐，URL 形状与 R2 一致。

#### 实例侧要求：暴露面收敛（可选）

先说明「不做什么」，因为直觉方案是错的：**不能在 nginx 上一刀切挡 `/api/`**。
`drive.xiey.work` 这个域名同时也是 OpenList 面板自身的入口，面板每个页面都要打 `/api/*`
（访问日志里可见 `referer "https://drive.xiey.work/@manage/about"` → `GET /api/public/settings`、
`GET /api/me`）。把 `/api/` 一律 403 会把面板整个打死。

真正值得收的是**绕过域名的那条明文路**：容器以 `0.0.0.0:5244->5244/tcp` 发布
（docker-proxy，`/opt/1panel/apps/openlist/openlist/docker-compose.yml` 里
`ports: - ${HOST_IP}:${PANEL_APP_PORT_HTTP}:5244`，而 `.env` 的 `HOST_IP=''` 为空即等价 `0.0.0.0`），
实测从公网 TCP 直接可达。注意 1Panel 的防火墙链管不到它：已发布端口经 nat PREROUTING DNAT 后
**直接进 FORWARD**，不经过 INPUT 上的 `1PANEL_BASIC_AFTER`（DROP 链），唯一有效的位置是 `DOCKER-USER`。

收掉这条路的办法是把发布地址绑回环（改 `.env` 的 `HOST_IP=127.0.0.1` 后重建容器）；
openresty 是 net=host、经 `127.0.0.1:5244` 反代，不受影响。

> 这一步属于纵深防御，不是堵已知漏洞：guest 账号已禁用（`x_users` 里 `guest` 行 `disabled=1`），
> 不带 `Authorization` 访问 `/api/fs/get`、`/api/fs/put`、`/api/fs/list`、`/api/fs/remove`、
> `/api/admin/*`、`/api/me` 一律回 `{"code":401,"message":"Guest user is disabled, login please"}`，
> 只有 `/api/public/settings` 是匿名的。裸端口与域名路的匿名面**完全一样**，
> 收口减少的是「不经 CDN、无 TLS 观测」的这一层，而非攻击面本身。

#### 存量回填

启动后立即扫描全部内容并补齐缺失对象（每批 500 条，经 Redis 锁 `lock:mirror:openlist` 保证多实例只有一个在跑），
之后每 `OPENLIST_SYNC_INTERVAL_SECONDS`（默认 300s）扫一轮兜住偶发失败。
与 R2 的回填任务各持一把锁、互不阻塞，日志前缀取目标名（`r2 …` / `openlist …`）。

- 对象名与本地同构：`<OPENLIST_PREFIX>/<文件名>`，挂载目录里可直接与 `data/uploads` 对账。
- 压缩是**原地改写**（路径不变、内容变），因此对象名不变、内容随压缩更新。

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

## 镜像端的证书

对外提供下载的镜像端只有一个：广州机的 `drive.xiey.work`（OpenList 反代）。

**证书由 1Panel 托管**：`xiey.work` 的 NS 在阿里云万网（`dns1/dns2.hichina.com`），
1Panel 里已存有可复用的阿里云 DNS 凭据（`website_dns_accounts` 表，type=`AliYun`），
走 DNS-01 签发给 `*.xiey.work` 的通配证书，续期由面板自动完成，不需要部署机参与。

**注意**：仓库里曾有自建的 `xqecz-server acme` 子命令与 `scripts/acme/` 三个 certbot hook
（为七牛 CDN 域名做 HTTP-01 挑战），随七牛下线一并删除。部署机上的残留需要清掉：

```bash
grep -n acme /etc/crontabs/root      # 删掉 `17 3 * * * /opt/xqecz/scripts/acme/renew.sh` 那一行
rm -rf /opt/xqecz/scripts/acme
apk del certbot                      # 若只为该链路装的
```

## 回滚

- 后端：部署前保留上一版 `xqecz-server`，回滚即覆盖回去并执行 `doas rc-service xqecz restart`
- 前端：`dist` 为静态文件，保留上一版目录或压缩包，回滚即替换目录内容
- 数据库：换实例时**旧实例不删**，回滚即把 `/opt/xqecz/.env` 的 `MYSQL_*` 换回旧值再重启（备份文件名带切换时间戳，见「从 MySQL/MariaDB 迁移到 TiDB」的实例搬迁记录）
