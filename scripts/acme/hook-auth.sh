#!/bin/sh
# certbot --manual-auth-hook：把 HTTP-01 挑战推进七牛桶。
#
# 为什么需要它：img.xiey.work 指向七牛 CDN、回源到桶，ACME 的校验请求
# 根本到不了部署机，标准 webroot 模式无从生效。
#
# certbot 会带这些环境变量调用本脚本：
#   CERTBOT_DOMAIN      正在验证的域名
#   CERTBOT_TOKEN       挑战 token（即 URL 最后一段）
#   CERTBOT_VALIDATION  挑战内容（该 URL 应返回的正文）
set -eu

ROOT="${XQECZ_ROOT:-/opt/xqecz}"
BIN="${XQECZ_BIN:-$ROOT/xqecz-server}"

: "${CERTBOT_TOKEN:?缺少 CERTBOT_TOKEN}"
: "${CERTBOT_VALIDATION:?缺少 CERTBOT_VALIDATION}"

# 二进制靠工作目录找项目根（project.FindRoot），必须在项目根下执行。
cd "$ROOT"

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
printf '%s' "$CERTBOT_VALIDATION" >"$TMP"

# plant 内部会把文件写进桶并回读 CDN 校验内容，不一致即非零退出，
# certbot 随即中止本次签发——比让 ACME 报一句含糊的 403 好定位得多。
"$BIN" acme plant "$CERTBOT_TOKEN" "$TMP"
