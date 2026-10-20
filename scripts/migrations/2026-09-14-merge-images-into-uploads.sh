#!/bin/bash
set -euo pipefail

MODE="${1:-dry-run}"
case "$MODE" in dry-run|apply) ;; *) echo "用法: $0 [dry-run|apply]"; exit 2 ;; esac

DATA_DIR=/www/wwwroot/xqecz/data
ENV_FILE=/www/wwwroot/xqecz-golang/.env
[ -f "$ENV_FILE" ] || { echo "缺少 $ENV_FILE"; exit 1; }
set -a; . "$ENV_FILE"; set +a
: "${MYSQL_USER:?MYSQL_USER 未配置}"; : "${MYSQL_PASSWORD:?}"; : "${MYSQL_DATABASE:?}"
DB_HOST="${MYSQL_HOST:-127.0.0.1}"; DB_PORT="${MYSQL_PORT:-3306}"

WORK="/root/xqecz-merge-images-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$WORK"
echo "工作目录: $WORK"
echo "模式: $MODE"

export MYSQL_PWD="$MYSQL_PASSWORD"
mysql_q() { mysql -h "$DB_HOST" -P "$DB_PORT" -u "$MYSQL_USER" "$MYSQL_DATABASE" "$@"; }

mysql_q -N -B -e "SELECT id, file_path, thumb_path FROM contents WHERE file_path LIKE 'images/%' ORDER BY id" > "$WORK/rows.tsv"
TOTAL=$(grep -c . "$WORK/rows.tsv" || true)
echo "待迁移记录: $TOTAL 条"

: > "$WORK/update.sql"; : > "$WORK/missing-src.txt"; : > "$WORK/referenced.txt"

moved=0; reused=0; thumb_renamed=0; thumb_missing=0; processed=0

while IFS=$'\t' read -r id fp tp; do
  [ -z "${id:-}" ] && continue
  printf '%s\n' "$(basename "$fp")" >> "$WORK/referenced.txt"

  src="$DATA_DIR/$fp"
  if [ ! -f "$src" ]; then
    printf '%s\t%s\n' "$id" "$fp" >> "$WORK/missing-src.txt"
    continue
  fi

  md5=$(md5sum "$src" | awk '{print $1}')
  target="$DATA_DIR/uploads/$md5.webp"

  if [ -f "$target" ]; then
    reused=$((reused + 1))
  else
    if [ "$MODE" = apply ]; then
      mv -f "$src" "$target"
    else
      printf '%s -> %s\n' "$src" "$target" >> "$WORK/plan-move.txt"
    fi
    moved=$((moved + 1))
  fi

  new_tp="$tp"
  if [ -n "$tp" ] && [ "$tp" != "NULL" ]; then
    old_thumb="$DATA_DIR/$tp"
    if [ -f "$old_thumb" ]; then
      cand="thumbs/${md5}_thumb.webp"
      if [ "$cand" != "$tp" ]; then
        if [ "$MODE" = apply ]; then
          mv -f "$old_thumb" "$DATA_DIR/$cand"
        else
          printf '%s -> %s\n' "$old_thumb" "$DATA_DIR/$cand" >> "$WORK/plan-thumb.txt"
        fi
      fi
      new_tp="$cand"
      thumb_renamed=$((thumb_renamed + 1))
    else
      thumb_missing=$((thumb_missing + 1))
    fi
  fi

  printf "UPDATE contents SET file_path='%s', thumb_path='%s' WHERE id=%s;\n" \
    "$md5.webp" "$new_tp" "$id" >> "$WORK/update.sql"
  processed=$((processed + 1))
done < "$WORK/rows.tsv"

orphan=0
if [ -d "$DATA_DIR/images" ]; then
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    base=$(basename "$f")
    grep -qxF "$base" "$WORK/referenced.txt" && continue
    orphan=$((orphan + 1))
    md5=$(md5sum "$f" | awk '{print $1}')
    target="$DATA_DIR/uploads/$md5.webp"
    if [ -f "$target" ]; then
      if [ "$MODE" = apply ]; then rm -f "$f"; else printf '%s -> (dup) %s\n' "$f" "$target" >> "$WORK/plan-orphan.txt"; fi
    else
      if [ "$MODE" = apply ]; then mv -f "$f" "$target"; else printf '%s -> %s\n' "$f" "$target" >> "$WORK/plan-orphan.txt"; fi
    fi
  done < <(find "$DATA_DIR/images" -maxdepth 1 -type f)
fi

echo "---- 统计 ----"
echo "成功处理:        $processed 条"
echo "文件移入 uploads: $moved"
echo "uploads 已有同内容(直接复用): $reused"
echo "缩略图改名:      $thumb_renamed"
echo "缩略图缺失(保留原值): $thumb_missing"
echo "images 孤儿文件:  $orphan"
if [ -s "$WORK/missing-src.txt" ]; then
  echo "⚠ 源文件缺失(未迁移): $(grep -c . "$WORK/missing-src.txt") 条 → $WORK/missing-src.txt"
fi

if [ "$MODE" = apply ]; then
  echo "---- 更新数据库 ----"
  { echo "START TRANSACTION;"; cat "$WORK/update.sql"; echo "COMMIT;"; } | mysql_q
  echo "数据库更新完成"

  echo "---- 校验 ----"
  mysql_q -N -B -e "SELECT CONCAT('剩余 images 前缀记录: ', COUNT(*)) FROM contents WHERE file_path LIKE 'images/%'"
  echo "images 剩余文件数: $(find "$DATA_DIR/images" -maxdepth 1 -type f | wc -l)"
  echo "uploads 文件数:    $(find "$DATA_DIR/uploads" -maxdepth 1 -type f | wc -l)"
  echo "thumbs 文件数:     $(find "$DATA_DIR/thumbs" -maxdepth 1 -type f | wc -l)"
else
  echo "---- dry-run 完成，未改动任何文件与数据 ----"
  echo "待执行 SQL: $WORK/update.sql（$(grep -c . "$WORK/update.sql" || true) 条）"
fi
