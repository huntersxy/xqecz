package migrate

import (
	"context"
	"embed"
	"path"
	"sort"
	"strings"

	"gorm.io/gorm"
)

//go:embed sql/*.sql
var embedded embed.FS

var files = load()

const (
	lockName           = "xqecz:schema_migrations"
	lockTimeoutSeconds = 20
	// baselineSentinel is a table created before embedded migration tracking.
	baselineSentinel  = "contents"
	emptyDatabaseHint = "缺少基础表结构，无法确定迁移基线；请导入生产库快照或先执行建库脚本"
)

// baseline lists migrations that shipped before embedded tracking. Every existing
// database already ran them by hand, so an empty tracking table records them as
// applied instead of replaying destructive ALTERs. They are recorded as assumed,
// never verified: drift introduced by a hand-run that was skipped stays invisible.
var baseline = []string{
	"2026-08-05-unify-content-schema",
	"2026-09-13-add-hot-path-indexes",
	"2026-09-14-add-compressed-at-and-clean-thumb-path",
	"2026-09-14-drop-soft-delete-columns",
	"2026-10-18-add-comment-list-index",
	"2026-10-18-add-post-soft-delete-hot-path-indexes",
	"2026-10-19-add-ai-asset-liked",
	"2026-10-19-add-content-ai-enhanced",
	"2026-10-20-add-ai-task-attempts",
}

func isBaseline(version string) bool {
	for _, candidate := range baseline {
		if candidate == version {
			return true
		}
	}
	return false
}

func load() map[string]string {
	entries, err := embedded.ReadDir("sql")
	if err != nil {
		panic("migrate: embedded sql missing: " + err.Error())
	}
	out := make(map[string]string, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		raw, err := embedded.ReadFile(path.Join("sql", name))
		if err != nil {
			panic("migrate: read " + name + ": " + err.Error())
		}
		out[strings.TrimSuffix(name, ".sql")] = string(raw)
	}
	if len(out) == 0 {
		panic("migrate: no embedded migrations")
	}
	return out
}

// Versions returns embedded migration versions in ascending name order.
func Versions() []string {
	out := make([]string, 0, len(files))
	for version := range files {
		out = append(out, version)
	}
	sort.Strings(out)
	return out
}

// Status reports applied and pending versions without changing the schema.
func Status(ctx context.Context, db *gorm.DB) (applied, pending []string, err error) {
	r := newRunner(db)
	if !r.db.WithContext(ctx).Migrator().HasTable("schema_migrations") {
		return nil, Versions(), nil
	}
	seen, err := r.applied(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, version := range Versions() {
		if seen[version] {
			applied = append(applied, version)
		} else {
			pending = append(pending, version)
		}
	}
	return applied, pending, nil
}
