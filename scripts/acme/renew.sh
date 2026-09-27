#!/bin/sh
# 每日 cron 执行：回收上一轮遗留的旧证书，然后跑 certbot 续签。
#
# 顺序不能反：prune 依赖「当前绑定」的判断，放在续签之前才能把上一轮
# 换绑留下的旧证书清掉；放在之后则会撞上同样的问题（换绑下发期间旧证书
# 仍被标记为已绑定，删除返回 400611）。
#
# 建议 crontab（每日 03:17，避开整点）：
#   17 3 * * * /opt/xqecz/scripts/acme/renew.sh >> /var/log/xqecz-acme.log 2>&1
set -eu

ROOT="${XQECZ_ROOT:-/opt/xqecz}"
BIN="${XQECZ_BIN:-$ROOT/xqecz-server}"

cd "$ROOT"

# 证书攒到上限会以 400500 挡住下一次续签，所以先回收。
# prune 失败不影响续签本身（证书仍在有效期内），只记日志。
"$BIN" acme prune || echo "prune 失败，忽略" >&2

# certbot 会把首次签发时的 --manual-auth-hook / --manual-cleanup-hook /
# --deploy-hook 记在 renewal conf 里，renew 时自动复用，这里无需重复传参。
# 剩余有效期不足 30 天才会真正触发续签，平时是空跑。
certbot renew --quiet --non-interactive
