#!/bin/sh
set -eu

: "${SRC_HOST:?}" "${SRC_USER:?}" "${SRC_DB:?}"
: "${TGT_HOST:?}" "${TGT_USER:?}" "${TGT_DB:?}"
SRC_SSL="${SRC_SSL:---skip-ssl}"
TGT_SSL="${TGT_SSL:-}"
SRC_PORT="${SRC_PORT:-3306}"
TGT_PORT="${TGT_PORT:-4000}"
RECREATE="${RECREATE:-0}"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

SRC_PASS_ARG=""
if [ -n "${SRC_PASS:-}" ]; then SRC_PASS_ARG="-p${SRC_PASS}"; fi
TGT_PASS_ARG=""
if [ -n "${TGT_PASS:-}" ]; then TGT_PASS_ARG="-p${TGT_PASS}"; fi

src() { mysql $SRC_SSL -h "$SRC_HOST" -P "$SRC_PORT" -u "$SRC_USER" $SRC_PASS_ARG "$@"; }
tgt() { mysql $TGT_SSL -h "$TGT_HOST" -P "$TGT_PORT" -u "$TGT_USER" $TGT_PASS_ARG "$@"; }

TABLES="users contents comments claims polls poll_votes content_likes content_favorites comment_reports api_keys"

echo "== 1/5 导出源库 =="
mysqldump $SRC_SSL -h "$SRC_HOST" -P "$SRC_PORT" -u "$SRC_USER" $SRC_PASS_ARG \
  --skip-add-locks --skip-comments --complete-insert "$SRC_DB" > "$WORK/raw.sql"
echo "   原始 dump: $(wc -c < "$WORK/raw.sql") 字节"

echo "== 2/5 TiDB 结构兼容修正 =="
python3 - "$WORK/raw.sql" "$WORK/fixed.sql" <<'PY'
import re, sys
src, dst = sys.argv[1], sys.argv[2]
text_col = re.compile(r'^(\s*`[^`]+`\s+(?:tinytext|mediumtext|longtext|text|blob|json)\b[^,]*)', re.I)
out = []
for line in open(src, encoding='utf-8', errors='replace'):
    s = line.strip()
    if s.startswith('/*M!') or s.startswith('/*!'):
        continue                      # MariaDB 私有指令，TiDB 不认
    if text_col.match(line):
        line = re.sub(r"\s+DEFAULT\s+'(?:[^'\\]|\\.)*'", '', line, flags=re.I)
    line = re.sub(r'\s*AUTO_INCREMENT=\d+', '', line)
    out.append(line)
open(dst, 'w', encoding='utf-8').write('\n'.join(out))
PY
echo "   修正后 dump: $(wc -c < "$WORK/fixed.sql") 字节"

echo "== 3/5 准备目标库 =="
if [ "$RECREATE" = "1" ]; then
  echo "   RECREATE=1：删除并重建 $TGT_DB"
  tgt -e "DROP DATABASE IF EXISTS \`$TGT_DB\`; CREATE DATABASE \`$TGT_DB\`;"
else
  tgt -e "CREATE DATABASE IF NOT EXISTS \`$TGT_DB\`;"
  existing=$(tgt -N -B -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='$TGT_DB' AND table_name IN ($(echo "$TABLES" | sed "s/[^ ]*/'&'/g" | tr ' ' ','))")
  if [ "$existing" != "0" ]; then
    echo "   目标库 $TGT_DB 已存在 $existing 张同名业务表；如确认重建请加 RECREATE=1 重跑。" >&2
    exit 1
  fi
  echo "   目标库为空，开始导入"
fi

echo "== 4/5 导入 =="
tgt "$TGT_DB" < "$WORK/fixed.sql"
echo "   导入完成"

echo "== 5/5 逐表校验（按主键有序导出后比对 md5）=="
fail=0
for t in $TABLES; do
  mysqldump $SRC_SSL -h "$SRC_HOST" -P "$SRC_PORT" -u "$SRC_USER" $SRC_PASS_ARG \
    --no-create-info --skip-extended-insert --skip-comments --compact --skip-add-locks \
    --complete-insert --order-by-primary "$SRC_DB" "$t" 2>/dev/null | grep '^INSERT' > "$WORK/s_$t.sql" || true
  mysqldump $TGT_SSL -h "$TGT_HOST" -P "$TGT_PORT" -u "$TGT_USER" $TGT_PASS_ARG \
    --no-create-info --skip-extended-insert --skip-comments --compact --skip-add-locks \
    --complete-insert --order-by-primary "$TGT_DB" "$t" 2>/dev/null | grep '^INSERT' > "$WORK/t_$t.sql" || true
  a=$(md5sum < "$WORK/s_$t.sql" | cut -d' ' -f1)
  b=$(md5sum < "$WORK/t_$t.sql" | cut -d' ' -f1)
  if [ "$a" = "$b" ]; then
    printf "   %-20s IDENTICAL (%s 行)\n" "$t" "$(wc -l < "$WORK/s_$t.sql")"
  else
    printf "   %-20s *** DIFFERS *** src=%s tgt=%s\n" "$t" "$a" "$b"
    fail=1
  fi
done

if [ "$fail" -ne 0 ]; then
  echo "迁移校验失败：存在内容不一致的表，请勿切换生产连接。" >&2
  exit 1
fi
echo "全部 $(( $(echo $TABLES | wc -w) )) 张表逐字节一致，可切换生产 .env 指向 $TGT_DB。"
