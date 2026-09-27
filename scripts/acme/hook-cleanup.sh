#!/bin/sh
# certbot --manual-cleanup-hook：删除挑战文件。
#
# certbot 会带 CERTBOT_DOMAIN / CERTBOT_TOKEN 调用本脚本。
# 删除失败不阻断流程：token 是一次性的，残留对象既不会被复用也不会造成风险，
# 真正的危害是让一次成功的续签因为清理失败而被判定为失败。
set -eu

ROOT="${XQECZ_ROOT:-/opt/xqecz}"
BIN="${XQECZ_BIN:-$ROOT/xqecz-server}"

[ -n "${CERTBOT_TOKEN:-}" ] || exit 0
cd "$ROOT"

"$BIN" acme clean "$CERTBOT_TOKEN" || echo "清理挑战文件失败（不影响本次续签）" >&2
