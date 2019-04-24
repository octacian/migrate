package migrate

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func expectError(t *testing.T, name string, msg string, fn func() error, substr ...string) {
	if err := fn(); err == nil {
		t.Errorf("%s: expected error with %s", name, msg)
	} else {
		for _, str := range substr {
			if !strings.Contains(err.Error(), str) {
				t.Errorf("%s: expected substring '%s' in error with %s, got:\n%s",
					name, str, msg, err.Error())
			}
		}
	}
}

// TestNewInstance ensures that an error is returned with a nil database
// handle, when a non-existant directory path is provided, when NewMigration
// fails, when the directory provided is completely empty, when there is a gap
// between the migration versions within the provided directory, and when the
// migration part SQL provided is invalid. Besides this, TestNewInstance also
// ensures that no error is returned when everything is valid.
func TestNewInstance(t *testing.T) {
	expectError(t, "NewInstance", "nil database handle", func() error { _, err := NewInstance(nil, ""); return err },
		"nil database handle")

	RunWithDB(func(db *sql.DB) {
		if _, err := NewInstance(db, "nothing"); err == nil {
			t.Error("NewInstance: expected error with non-existent instance directory path")
		} else if _, ok := err.(*os.PathError); !ok {
			t.Error("NewInstance: expected error of type *os.PathError with non-existent instance directory path")
		}

		expectError(t, "NewInstance", "NewMigration failure",
			func() error { _, err := NewInstance(db, "testing/blank"); return err }, "to begin with a comment denoting")
		expectError(t, "NewInstance", "no migrations",
			func() error { _, e := NewInstance(db, "testing/nothing"); return e }, "no migrations found")
		expectError(t, "NewInstance", "migration version gap",
			func() error { _, e := NewInstance(db, "testing/gap"); return e }, "found gap between")

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
			output := &strings.Builder{}
			instance.Output = output

			withOutput := func(name string, fn func(), substr ...string) {
				output.Reset()
				fn()

				for _, str := range substr {
					if !strings.Contains(strings.ToLower(output.String()), str) {
						t.Errorf("Instance.%s: expected substring '%s' in output, got:\n%s",
							name, str, output.String())
					}
				}
			}

			if version := instance.Version(); version != 0 {
				t.Errorf("Instance.Version: got '%d' expected '0'", version)
			}

			withOutput("Latest", func() {
				if err := instance.Latest(); err != nil {
					t.Fatal("Instance.Latest: got error:\n", err)
				} else if version := instance.Version(); version != 3 {
					t.Errorf("Instance.Version: got '%d' expected '3' after `Instance.Latest()`", version)
				}
			}, "over 3 version(s)", "0 to 1", "1 to 2", "2 to 3")

			expectError(t, "Instance.Latest", "database already on latest migration",
				func() error { return instance.Latest() }, "no migrations to apply")
			expectError(t, "Instance.Goto", "invalid database version '100'",
				func() error { return instance.Goto(100) }, "does not exist")
			expectError(t, "Instance.Goto", "invalid database version '-1'",
				func() error { return instance.Goto(-1) }, "does not exist")
			expectError(t, "Instance.Goto", "current database version",
				func() error { return instance.Goto(3) }, "no migrations to apply")

			withOutput("Goto(2)", func() {
				if err := instance.Goto(2); err != nil {
					t.Error("Instance.Goto: got error:\n", err)
				} else if version := instance.Version(); version != 2 {
					t.Errorf("Instance.Version: got '%d' expected '2' after `Instance.Goto(2)`", version)
				}
			}, "3 to 2", "1 migration part")

			withOutput("Goto(0)", func() {
				if err := instance.Goto(0); err != nil {
					t.Error("Instance.Goto: got error:\n", err)
				} else if version := instance.Version(); version != 0 {
					t.Errorf("Instance.Version: got '%d' expected '0' after `Instance.Goto(0)`", version)
				}
			}, "over 2 version(s)", "2 to 1", "1 to 0")

			if list := instance.List(); len(list) != 3 {
				t.Errorf("Instance.List: got length of %d expected 3", len(list))
			} else {
				for key, value := range []int{1, 2, 3} {
					if list[key] != value {
						t.Errorf("Instance.List: got '%#v' expected '[]int{1, 2, 3}'", list)
						break
					}
				}
			}
		}
	})
}
