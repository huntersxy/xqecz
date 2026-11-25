package migrate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migrate.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestVersionsAreEmbeddedAndSorted(t *testing.T) {
	versions := Versions()
	if len(versions) < 2 {
		t.Fatalf("versions = %#v", versions)
	}
	for i := 1; i < len(versions); i++ {
		if versions[i-1] >= versions[i] {
			t.Fatalf("versions not sorted: %q then %q", versions[i-1], versions[i])
		}
	}
	for _, version := range versions {
		if files[version] == "" {
			t.Fatalf("version %q has no script", version)
		}
	}
	if !strings.Contains(files["2026-11-21-add-site-settings"], "site_settings") {
		t.Fatal("site settings migration is not embedded")
	}
}

func TestRunAppliesOnlyNewVersionsAndIsIdempotent(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	if err := db.Exec("CREATE TABLE contents (id BIGINT PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}

	if err := Run(ctx, db); err != nil {
		t.Fatal(err)
	}
	applied, pending, err := Status(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending after run = %#v", pending)
	}
	if len(applied) != len(Versions()) {
		t.Fatalf("applied %d of %d", len(applied), len(Versions()))
	}
	if !db.Migrator().HasTable("site_settings") {
		t.Fatal("site_settings table was not created")
	}

	if err := Run(ctx, db); err != nil {
		t.Fatalf("second run must be a no-op: %v", err)
	}
}

func TestBaselineIsRecordedInsteadOfReplayed(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	// A pre-tracking database already has contents and none of the baseline tables.
	if err := db.Exec("CREATE TABLE contents (id BIGINT PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, db); err != nil {
		t.Fatal(err)
	}

	applied, _, err := Status(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, version := range applied {
		seen[version] = true
	}
	for _, version := range baseline {
		if _, ok := files[version]; !ok {
			continue
		}
		if !seen[version] {
			t.Fatalf("baseline %q was not recorded", version)
		}
	}
	// Baseline scripts reference tables that do not exist here; replaying them
	// would have failed, so reaching this point proves they were skipped.
}

func TestStatusReportsPendingWithoutChangingSchema(t *testing.T) {
	db := testDB(t)
	applied, pending, err := Status(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 0 || len(pending) != len(Versions()) {
		t.Fatalf("applied=%#v pending=%#v", applied, pending)
	}
	if db.Migrator().HasTable("schema_migrations") {
		t.Fatal("Status must not create tables")
	}
}

func TestRunRejectsEmptyDatabase(t *testing.T) {
	db := testDB(t)
	err := Run(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "基础表结构") {
		t.Fatalf("err = %v", err)
	}
}

func TestSplitHandlesCommentsSemicolonsAndQuotes(t *testing.T) {
	script := `
-- leading comment with ; inside
CREATE TABLE a (id INT);
/* block ; comment */
INSERT INTO a (t) VALUES ('semi;colon', "quote;");
# hash comment
UPDATE a SET t = 'it''s' WHERE id = 1;
`
	statements, err := split(script)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 3 {
		t.Fatalf("statements = %#v", statements)
	}
	if !strings.Contains(statements[0], "CREATE TABLE a") {
		t.Fatalf("first = %q", statements[0])
	}
	if !strings.Contains(statements[1], "'semi;colon'") || !strings.Contains(statements[1], `"quote;"`) {
		t.Fatalf("second = %q", statements[1])
	}
	if !strings.Contains(statements[2], "UPDATE a") {
		t.Fatalf("third = %q", statements[2])
	}
}

func TestSplitRejectsUnterminatedInput(t *testing.T) {
	for _, script := range []string{"SELECT 'a", "SELECT 1; /* open"} {
		if _, err := split(script); err == nil {
			t.Fatalf("expected error for %q", script)
		}
	}
}
