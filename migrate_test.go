package migrate

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const TestDBPath = "./test.sqlite"

func newExpectError(fn func(...interface{}) error) func(t *testing.T, msg, errContains string, args ...interface{}) {
	return func(t *testing.T, msg, errContains string, args ...interface{}) {
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
