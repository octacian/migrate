package migrate

import (
	"database/sql"
	"os"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const TestDBPath = "./test.sqlite"

var newExpectError = func(t *testing.T, fn func(...interface{}) error) func(msg, errContains string, args ...interface{}) {
	return func(msg, errContains string, args ...interface{}) {
		if err := fn(args...); err == nil {
			t.Errorf("NewMigration: expected error with %s", msg)
		} else if !strings.Contains(err.Error(), errContains) {
			t.Errorf("NewMigration: got unexpected error message with %s:\n%s", msg, err)
		}
	}
}

var version1UpSQL = `CREATE TABLE IF NOT EXISTS test(ID INT PRIMARY KEY,first_name VARCHAR(255),last_name VARCHAR(255));`
var version1DownSQL = `DROP TABLE IF EXISTS test;`

// RunWithDB runs a closure passing it a prepared database handle and disposing
// of it afterward.
func RunWithDB(fn func(*sql.DB)) {
	db, err := sql.Open("sqlite3", TestDBPath)
	if err != nil {
		panic(err)
	}

	fn(db)

	err = db.Close()
	if err != nil {
		panic(err)
	}

	if err := os.Remove(TestDBPath); err != nil {
		panic(err)
	}
}

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
	expectError := newExpectError(t, func(args ...interface{}) error {
		_, err := NewMigration(args[0].(string))
		return err
	})

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

	expectError("migration version '0'", "disallowed migration version", "testing/zero/version_0")
}

// TestBadMigrationData ensures that NewMigration returns an appropriate error
// message when provided with invalid migration parts.
func TestBadMigrationData(t *testing.T) {
	expectError := newExpectError(t, func(args ...interface{}) error {
		_, err := NewMigration(args[0].(string))
		return err
	})

	expectError("blank migration files", "to begin with a comment denoting", "testing/blank/version_1")
	expectError("migration files containing no direction markers",
		"to begin with a comment denoting", "testing/no_marker/version_1")
	expectError("no upward migration SQL", "no upward migration data", "testing/no_upward/version_1")
	expectError("no downward migration SQL", "no downward migration data", "testing/no_downward/version_1")
	expectError("empty migration directories", "no migration parts", "testing/empty/version_1")
}

// TestNewInstance ensures that an error is returned with a nil database
// handle, when a non-existant directory path is provided, when NewMigration
// fails, when the directory provided is completely empty, when there is a gap
// between the migration versions within the provided directory, and when the
// migration part SQL provided is invalid. Besides this, TestNewInstance also
// ensures that no error is returned when everything is valid.
func TestNewInstance(t *testing.T) {
	if _, err := NewInstance(nil, ""); err == nil {
		t.Error("NewInstance: expected error with nil database handle")
	} else if !strings.Contains(err.Error(), "nil database handle") {
		t.Error("NewInstance: got unexpected error message with nil database handle:\n", err)
	}

	RunWithDB(func(db *sql.DB) {
		if _, err := NewInstance(db, "nothing"); err == nil {
			t.Error("NewInstance: expected error with non-existent instance directory path")
		} else if _, ok := err.(*os.PathError); !ok {
			t.Error("NewInstance: expected error of type *os.PathError with non-existent instance directory path")
		}

		if _, err := NewInstance(db, "testing/blank"); err == nil {
			t.Error("NewInstance: expected error with NewMigration failure")
		} else if !strings.Contains(err.Error(), "to begin with a comment denoting") {
			t.Error("NewInstance: got unexpected error message with NewMigration failure:\n", err)
		}

		if _, err := NewInstance(db, "testing/nothing"); err == nil {
			t.Error("NewInstance: expected error with no migrations")
		} else if !strings.Contains(err.Error(), "no migrations found") {
			t.Error("NewInstance: got unexpected error message with no migrations:\n", err)
		}

		if _, err := NewInstance(db, "testing/gap"); err == nil {
			t.Error("NewInstance: expected error with migration version gap")
		} else if !strings.Contains(err.Error(), "found gap between") {
			t.Error("NewInstance: got unexpected error message with migration version gap:\n", err)
		}

		if instance, err := NewInstance(db, "testing/bad"); err != nil {
			t.Error("NewInstance: got error:\n", err)
		} else {
			instance.Output = &strings.Builder{}

			if err := instance.Latest(); err == nil {
				t.Error("NewInstance.Latest: expected error with invalid migration SQL")
			} else if !strings.Contains(err.Error(), "error while applying migration") {
				t.Error("NewInstance.Latest: got unexpected error message with invalid migration SQL")
			}
		}
	})
}

// TestWorkingInstance ensures that no errors occur with a working instance.
func TestWorkingInstance(t *testing.T) {
	RunWithDB(func(db *sql.DB) {
		if instance, err := NewInstance(db, "testing/working"); err != nil {
			t.Fatal("NewInstance: got error:\n", err)
		} else {
			instance.Output = &strings.Builder{}

			/*if res, ok := instance.Output.(*strings.Builder); !ok {
				t.Error("Instance.Output: got unknown type expected *strings.Builder")
			} else {
				if res != &builder {
					t.Error("Instance.Output: got unknown output")
				}
			}*/

			if version := instance.Version(); version != 0 {
				t.Errorf("Instance.Version: got '%d' expected '0'", version)
			}

			if err := instance.Latest(); err != nil {
				t.Fatal("Instance.Latest: got error:\n", err)
			} else if version := instance.Version(); version != 3 {
				t.Errorf("Instance.Version: got '%d' expected '3' after `Instance.Latest()`", version)
			}

			if err := instance.Latest(); err == nil {
				t.Error("Instance.Latest: expected error with database already on latest migration")
			} else if !strings.Contains(err.Error(), "is the latest") {
				t.Error("Instance.Latest: got unexpected error message with database already on latest migration:\n", err)
			}

			if err := instance.Goto(100); err == nil {
				t.Error("Instance.Goto: expected error with invalid database version '100'")
			} else if !strings.Contains(err.Error(), "does not exist") {
				t.Error("Instance.Goto: got unexpected error message with invalid database version '100':\n", err)
			}

			if err := instance.Goto(-1); err == nil {
				t.Error("Instance.Goto: expected error with invalid database version '-1'")
			} else if !strings.Contains(err.Error(), "does not exist") {
				t.Error("Instance.Goto: got unexpected error message with invalid database version '-1'")
			}

			if err := instance.Goto(3); err == nil {
				t.Error("Instance.Goto: expected error with current database version")
			} else if !strings.Contains(err.Error(), "is already on version") {
				t.Error("Instance.Goto: got unexpected error message with current database version:\n", err)
			}

			if err := instance.Goto(2); err != nil {
				t.Error("Instance.Goto: got error:\n", err)
			} else if version := instance.Version(); version != 2 {
				t.Errorf("Instance.Version: got '%d' expected '2' after `Instance.Goto(2)`", version)
			}

			if err := instance.Goto(0); err != nil {
				t.Error("Instance.Goto: got error:\n", err)
			}
		}
	})
}
