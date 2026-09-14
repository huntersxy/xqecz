#!/bin/bash
# 统一 thumbs 命名：把所有缩略图规整为「<主图 md5>_thumb.webp」，并同步库中的 thumb_path。
# 未被任何记录引用的缩略图移入 data/bin（垃圾桶，保留而非删除）。
#
# 背景：主图已在上一步统一为 uploads/<md5>.<ext>，早期缩略图却是 <n>_<时间戳>_thumb.webp，
# 两种规则并存导致「按主图名找缩略图」的调试与迁移都很别扭。
#
# 用法：unify-thumb-names.sh [dry-run|apply]
set -euo pipefail

MODE="${1:-dry-run}"
case "$MODE" in dry-run|apply) ;; *) echo "用法: $0 [dry-run|apply]"; exit 2 ;; esac

DATA_DIR=/www/wwwroot/xqecz/data
BIN_DIR="$DATA_DIR/bin"
ENV_FILE=/www/wwwroot/xqecz-golang/.env
[ -f "$ENV_FILE" ] || { echo "缺少 $ENV_FILE"; exit 1; }
set -a; . "$ENV_FILE"; set +a
: "${MYSQL_USER:?}"; : "${MYSQL_PASSWORD:?}"; : "${MYSQL_DATABASE:?}"
DB_ARGS="-h ${MYSQL_HOST:-127.0.0.1} -P ${MYSQL_PORT:-3306} -u $MYSQL_USER $MYSQL_DATABASE"
export MYSQL_PWD="$MYSQL_PASSWORD"
q() { mysql $DB_ARGS -N -B -e "$1"; }

WORK="/root/xqecz-unify-thumbs-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$WORK"
echo "工作目录: $WORK   模式: $MODE"

# 用哨兵分隔符输出，避免空字段被折叠导致字段错位
q "SELECT CONCAT(id,'|',IFNULL(file_path,''),'|',IFNULL(thumb_path,'')) FROM contents ORDER BY id" > "$WORK/rows.txt"
TOTAL=$(grep -c . "$WORK/rows.txt" || true)
echo "记录总数: $TOTAL"

: > "$WORK/update.sql"; : > "$WORK/referenced.txt"
renamed=0; deduped=0; keep=0; skip_no_file=0; missing=0

while IFS='|' read -r id fp tp; do
  [ -z "${id:-}" ] && continue
  [ -z "$tp" ] && continue

  # 目标名：主图为裸文件名（<md5>.<ext>）时用其 stem；否则沿用原缩略图名
  stem=""
  case "$fp" in
    "" ) ;;
    thumbs/*|images/*) ;;
    *) stem="${fp%.*}" ;;
  esac
  [ -z "$stem" ] && { keep=$((keep+1)); continue; }

  want="thumbs/${stem}_thumb.webp"
  [ "$tp" = "$want" ] && { keep=$((keep+1)); printf '%s\n' "$(basename "$tp")" >> "$WORK/referenced.txt"; continue; }

  src="$DATA_DIR/$tp"
  dst="$DATA_DIR/$want"
  if [ ! -f "$src" ]; then
    missing=$((missing+1))
    echo "  缩略图缺失 id=$id  $tp"
    printf '%s\n' "$(basename "$want")" >> "$WORK/referenced.txt"
    printf "UPDATE contents SET thumb_path='%s' WHERE id=%s;\n" "$want" "$id" >> "$WORK/update.sql"
    continue
  fi

  if [ -f "$dst" ]; then
    # 目标已存在（内容相同即去重）：直接改库引用，源文件入桶
    if [ "$MODE" = apply ]; then rm -f "$src"; else printf '%s -> (dup) %s\n' "$src" "$dst" >> "$WORK/plan.txt"; fi
    deduped=$((deduped+1))
  else
    if [ "$MODE" = apply ]; then mv -f "$src" "$dst"; else printf '%s -> %s\n' "$src" "$dst" >> "$WORK/plan.txt"; fi
    renamed=$((renamed+1))
  fi
  printf '%s\n' "$(basename "$want")" >> "$WORK/referenced.txt"
  printf "UPDATE contents SET thumb_path='%s' WHERE id=%s;\n" "$want" "$id" >> "$WORK/update.sql"
done < "$WORK/rows.txt"

# 未被引用的缩略图入桶
orphan=0
if [ -d "$DATA_DIR/thumbs" ]; then
  while IFS= read -r f; do
    base=$(basename "$f")
    grep -qxF "$base" "$WORK/referenced.txt" && continue
    orphan=$((orphan+1))
    if [ "$MODE" = apply ]; then
      mkdir -p "$BIN_DIR"
      dst="$BIN_DIR/$base"
      [ -f "$dst" ] && dst="$BIN_DIR/$(date +%s%3N)_$base"
      mv -f "$f" "$dst"
    else
      printf '%s -> (bin) %s\n' "$f" "$BIN_DIR/$base" >> "$WORK/plan-orphan.txt"
    fi
  done < <(find "$DATA_DIR/thumbs" -maxdepth 1 -type f)
fi

echo "---- 统计 ----"
echo "改名:            $renamed"
echo "去重(目标已存在): $deduped"
echo "无需改动:        $keep"
echo "库引用源文件缺失: $missing"
echo "孤儿缩略图入桶:   $orphan"

if [ "$MODE" = apply ]; then
  echo "---- 更新数据库 ----"
  { echo "START TRANSACTION;"; cat "$WORK/update.sql"; echo "COMMIT;"; } | mysql $DB_ARGS
  echo "数据库更新完成"
  echo "---- 校验 ----"
  echo "thumbs 剩余文件: $(find "$DATA_DIR/thumbs" -maxdepth 1 -type f | wc -l)"
  echo "新格式文件数:    $(ls "$DATA_DIR/thumbs" | grep -cE '^[a-f0-9]{32}_thumb\.webp$' || true)"
  echo "非新格式:        $(ls "$DATA_DIR/thumbs" | grep -vcE '^[a-f0-9]{32}_thumb\.webp$' || true)"
  echo "bin 目录文件数:  $(find "$BIN_DIR" -maxdepth 1 -type f 2>/dev/null | wc -l)"
else
  echo "---- dry-run 完成，未改动任何文件与数据 ----"
fi
