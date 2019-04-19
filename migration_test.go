package migrate

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

var mExpectError = newExpectError(func(args ...interface{}) error {
	_, err := NewMigration(args[0].(string))
	return err
})

// TestWorkingMigration ensures that NewMigration performs as expected with a
// valid migration directory path.
func TestWorkingMigration(t *testing.T) {
	if migration, err := NewMigration("testing/working/version_1"); err != nil {
		t.Error("NewMigration: got error:\n", err)
	} else {
		if migration.Name != "version_1" {
			t.Errorf("NewMigration: got name '%s' expected 'version_1'", migration.Name)
		}
		if migration.Path != "testing/working/version_1" {
			t.Errorf("NewMigration: got path '%s' expected 'testing/working/version_1", migration.Path)
		}
		if migration.Version != 1 {
			t.Errorf("NewMigration: got version '%d' expected '1'", migration.Version)
		}
		if len(migration.Parts) != 1 {
			t.Errorf("NewMigration: got %d parts expected 1", len(migration.Parts))
		}
		if migration.Parts[0].Name != "test.sql" {
			t.Errorf("NewMigrations.Parts: got part name '%s' expected 'test.sql'", migration.Parts[0].Name)
		}
		if migration.Parts[0].Path != "testing/working/version_1/test.sql" {
			t.Errorf("NewMigrations.Parts: got part path '%s' expected 'testing/working/version_1/test.sql'", migration.Parts[0].Path)
		}
		if migration.Parts[0].Up != version1UpSQL {
			t.Errorf("NewMigration.Parts: got up part:\n%s\n\nexpected:\n%s", migration.Parts[0].Up, version1UpSQL)
		}
		if migration.Parts[0].Down != version1DownSQL {
			t.Errorf("NewMigration.Parts: got down part:\n%s\n\nexpected:\n%s", migration.Parts[0].Down, version1DownSQL)
		}
	}
}

// TestBadMigrationPath ensures that NewMigration returns an appropriate error
// when the migration directory path provided is in some way invalid.
func TestBadMigrationPath(t *testing.T) {
	if _, err := NewMigration("version_abc"); err == nil {
		t.Error("NewMigration: expected error with invalid migration directory name")
	} else if _, ok := err.(*strconv.NumError); !ok {
		t.Error("NewMigration: expected error of type *strconv.NumError with invalid migration directory name")
	}

	if _, err := NewMigration("v1"); err == nil {
		t.Error("NewMigration: expected error with invalid migration directory name")
	} else if !strings.Contains(err.Error(), "name to be formatted as") {
		t.Error("NewMigration: got unexpected error message with invalid migration directory name, got:\n", err)
	}

	if _, err := NewMigration("version_100"); err == nil {
		t.Error("NewMigration: expected error with non-existent migration directory path")
	} else if _, ok := err.(*os.PathError); !ok {
		t.Error("NewMigration: expected error of type *os.PathError with non-existent migration directory path")
	}

	mExpectError(t, "migration version '0'", "disallowed migration version", "testing/zero/version_0")
}

// TestNoParts ensures that NewMigration returns an appropriate error message
// when no migration parts exist.
func TestNoParts(t *testing.T) {
	mExpectError(t, "empty migration directories", "no migration parts", "testing/empty/version_1")
}
