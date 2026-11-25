package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Migrations must stay additive and idempotent: Run executes on every start.
type runner struct {
	db      *gorm.DB
	dialect string
}

func newRunner(db *gorm.DB) *runner {
	return &runner{db: db, dialect: db.Dialector.Name()}
}

func (r *runner) hasBaseSchema(ctx context.Context) bool {
	return r.db.WithContext(ctx).Migrator().HasTable(baselineSentinel)
}

// Run applies every embedded migration that has not been recorded yet.
func Run(ctx context.Context, db *gorm.DB) error {
	r := newRunner(db)
	if err := r.ensureTable(ctx); err != nil {
		return err
	}

	release, err := r.lock(ctx)
	if err != nil {
		return err
	}
	defer release()

	applied, err := r.applied(ctx)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		if !r.hasBaseSchema(ctx) {
			return fmt.Errorf("数据库为空：%s", emptyDatabaseHint)
		}
		if err := r.seedBaseline(ctx, applied); err != nil {
			return err
		}
	}

	for _, version := range Versions() {
		if applied[version] {
			continue
		}
		if err := r.apply(ctx, version, files[version]); err != nil {
			return err
		}
		slog.Info("migration applied", "version", version)
	}
	return nil
}

func (r *runner) ensureTable(ctx context.Context) error {
	ddl := `CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(191) NOT NULL,
  applied_at DATETIME(3) NOT NULL,
  PRIMARY KEY (version)
)`
	if r.dialect != "mysql" {
		ddl = `CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT NOT NULL PRIMARY KEY,
  applied_at DATETIME NOT NULL
)`
	}
	if err := r.db.WithContext(ctx).Exec(ddl).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

// lock serializes concurrent deploys on MySQL; tests run single-process.
func (r *runner) lock(ctx context.Context) (func(), error) {
	if r.dialect != "mysql" {
		return func() {}, nil
	}
	var acquired sql.NullInt64
	if err := r.db.WithContext(ctx).Raw("SELECT GET_LOCK(?, ?)", lockName, lockTimeoutSeconds).Scan(&acquired).Error; err != nil {
		return nil, fmt.Errorf("acquire migration lock: %w", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return nil, errors.New("另一个部署正在执行迁移，请稍后重试")
	}
	return func() {
		_ = r.db.Exec("SELECT RELEASE_LOCK(?)", lockName).Error
	}, nil
}

func (r *runner) applied(ctx context.Context) (map[string]bool, error) {
	var versions []string
	if err := r.db.WithContext(ctx).Raw("SELECT version FROM schema_migrations").Scan(&versions).Error; err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	out := make(map[string]bool, len(versions))
	for _, version := range versions {
		out[version] = true
	}
	return out, nil
}

// seedBaseline records hand-applied migrations on a database that predates
// embedded tracking, so historical destructive ALTERs are not replayed.
func (r *runner) seedBaseline(ctx context.Context, applied map[string]bool) error {
	for _, version := range baseline {
		if _, ok := files[version]; !ok {
			continue
		}
		if err := r.record(ctx, version); err != nil {
			return err
		}
		applied[version] = true
	}
	slog.Warn("migration baseline assumed applied without verification", "versions", len(applied))
	return nil
}

func (r *runner) apply(ctx context.Context, version, script string) error {
	statements, err := split(script)
	if err != nil {
		return fmt.Errorf("migration %s: %w", version, err)
	}
	for i, statement := range statements {
		if err := r.db.WithContext(ctx).Exec(r.dialectize(statement)).Error; err != nil {
			return fmt.Errorf("migration %s 第 %d 条语句失败: %w", version, i+1, err)
		}
	}
	return r.record(ctx, version)
}

// dialectize drops MySQL-only column and table options so the same embedded
// files can run against the test dialect. Production is MySQL, where nothing is
// rewritten.
func (r *runner) dialectize(statement string) string {
	if r.dialect == "mysql" {
		return statement
	}
	statement = strings.ReplaceAll(statement, "DATETIME(3)", "DATETIME")
	upper := strings.ToUpper(statement)
	for _, marker := range []string{" ENGINE=", " DEFAULT CHARSET=", " COLLATE=", " CHARACTER SET "} {
		if idx := strings.Index(upper, marker); idx > 0 {
			statement = statement[:idx]
			break
		}
	}
	return strings.TrimSpace(statement)
}

func (r *runner) record(ctx context.Context, version string) error {
	if err := r.db.WithContext(ctx).Exec(
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", version, time.Now().UTC(),
	).Error; err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	return nil
}

// split separates SQL statements on semicolons outside quotes and comments.
func split(script string) ([]string, error) {
	runes := []rune(script)
	var (
		out     []string
		current strings.Builder
		quote   rune
		line    bool
		block   bool
		hasCode bool
		last    rune
	)
	flush := func() {
		if hasCode {
			out = append(out, strings.TrimSpace(current.String()))
		}
		current.Reset()
		hasCode = false
	}

	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}

		if line {
			if ch == '\n' {
				line = false
				current.WriteRune(ch)
			}
			continue
		}
		if block {
			if ch == '*' && next == '/' {
				block = false
				i++
			}
			continue
		}
		if quote != 0 {
			current.WriteRune(ch)
			if ch == quote && last != '\\' {
				quote = 0
			}
			last = ch
			continue
		}

		if ch == '-' && next == '-' && (i+2 >= len(runes) || strings.ContainsRune(" \t\r\n", runes[i+2])) {
			line = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			block = true
			i++
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			quote = ch
			hasCode = true
			current.WriteRune(ch)
			last = ch
			continue
		}
		if ch == ';' {
			flush()
			continue
		}
		if !strings.ContainsRune(" \t\r\n", ch) {
			hasCode = true
		}
		current.WriteRune(ch)
		last = ch
	}

	if quote != 0 || block {
		return nil, errors.New("SQL 存在未闭合的引号或注释")
	}
	flush()
	return out, nil
}
