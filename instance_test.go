package migrate

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

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
