# 数据库迁移

结构变更以内嵌 SQL 迁移表达，随二进制发布；推送 `master` 后由部署流水线自动应用，不需要在服务器手工执行 SQL。

## 存放位置

`packages/server/internal/migrate/sql/<日期>-<说明>.sql`，通过 `//go:embed sql/*.sql` 编译进 `xqecz-server`。文件名（去掉 `.sql`）就是版本号，按字典序执行。

## 执行时机与入口

- `xqecz-server migrate`：应用所有未记录的版本，可重复执行。
- `xqecz-server migrate -status`：只列出已应用与待应用版本，不改库。
- 服务端启动：`cmd/server` 在连接数据库后先跑一次 `migrate.Run`，失败即退出（`Exit 1`），避免旧 schema 上跑新代码。
- 部署流水线：`.github/workflows/deploy.yml` 上传 `xqecz-server.new` 后先执行 `migrate -status` / `migrate` / 再 `-status`，通过后才替换并重启。迁移失败时旧进程继续服务。

## 记录表

`schema_migrations(version, applied_at)` 记录已应用版本，由迁移器自动创建。并发部署通过 MySQL `GET_LOCK('xqecz:schema_migrations')` 串行化。

## 基线

`schema_migrations` 为空且 `contents` 表已存在时，视为「历史库」：把 `baseline` 中列出的早期脚本记为已应用，只真正执行新增版本。这样才能把历史上需要手工执行的 `ALTER`/`DELETE` 留在仓库里作记录，而不会在新部署里重放。

这层基线的代价是**不校验**：它假定历史库确实逐条手工执行过那些脚本。若某条在当时被跳过，漂移不会在这里报错（例如本地开发库就缺少 `idx_comments_content_parent_visible_v2`）。需要核对时用 `pnpm server:test` 的迁移测试或直接查 `information_schema`，不要指望基线替你发现。

`baseline` 定义在 `packages/server/internal/migrate/embed.go`，新增迁移不要加进去。

## 新增迁移

1. 在 `packages/server/internal/migrate/sql/` 新建文件，日期前缀 + 简短说明，例如 `2026-12-01-add-example-table.sql`。
2. 语句保持幂等（`CREATE TABLE IF NOT EXISTS`、先查后改），因为同一文件可能在恢复/重建库时重跑。
3. 需要新表时同时补 GORM 模型；表结构的权威来源是迁移。历史模块（`creative`、`aigen`、`vision`）仍保留启动期 `AutoMigrate`，新代码不要再增加。
4. 更新 `docs/database-migrations.md` 的基线说明只在版本归档时动，通常不用改。
5. 本地验证：
   ```bash
   go -C packages/server run ./cmd/server migrate -status
   go -C packages/server run ./cmd/server migrate
   go -C packages/server test ./internal/migrate/
   ```

## 注意

- 每条语句单独执行，文件内不要依赖语句间的会话变量。
- 测试用 SQLite 跑同一批文件；迁移器会去掉 MySQL 专有的 `ENGINE=`/`CHARSET`/`COLLATE` 表选项，因此新迁移应把这类选项写在语句末尾。
- 迁移只增不改：已发布的文件不要再改内容，用新文件表达修正。
