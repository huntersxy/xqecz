#!/bin/sh
# certbot --deploy-hook：续签成功后把新证书上传七牛并绑定到 CDN 域名。
#
# certbot 会带这些环境变量调用本脚本：
#   RENEWED_LINEAGE  新证书所在目录（含 fullchain.pem 与 privkey.pem）
#   RENEWED_DOMAINS  本次续签覆盖的域名，空格分隔
#
# 只有续签**真的成功**时 certbot 才会调用本 hook，因此这里无需再判断证书是否变化。
set -eu

ROOT="${XQECZ_ROOT:-/opt/xqecz}"
BIN="${XQECZ_BIN:-$ROOT/xqecz-server}"

: "${RENEWED_LINEAGE:?缺少 RENEWED_LINEAGE}"
[ -f "$RENEWED_LINEAGE/fullchain.pem" ] || { echo "找不到 $RENEWED_LINEAGE/fullchain.pem" >&2; exit 1; }
[ -f "$RENEWED_LINEAGE/privkey.pem" ] || { echo "找不到 $RENEWED_LINEAGE/privkey.pem" >&2; exit 1; }

cd "$ROOT"

# fullchain = 服务器证书 + 中间证书，正是七牛要求的 ca 字段；
# 只传叶子证书会以 400323「证书链验证失败」被拒。
"$BIN" acme deploy "$RENEWED_LINEAGE/fullchain.pem" "$RENEWED_LINEAGE/privkey.pem"
