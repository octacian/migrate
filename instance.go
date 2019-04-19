package migrate

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"time"

	"github.com/octacian/metadb"
)

// ErrNoVersion is returned by Goto when the requested version does not exist.
type ErrNoVersion struct {
	Message string
}

// Error implements the error interface for ErrNoVersion.
func (err *ErrNoVersion) Error() string {
	return err.Message
}

// ErrNoMigrations is returned by Goto and Latest when there are no more
// migrations to apply.
type ErrNoMigrations struct {
	Message string
}

// Error implements the error interface for ErrNoMigrations.
func (err *ErrNoMigrations) Error() string {
	return err.Message
}

// Instance represents a single collective set of migrations. With the
// exception of the Output field, instance is not intended to be directly
// created and manipulated, but rather managed by NewInstance and a variety of
// methods.
type Instance struct {
	db         *sql.DB
	meta       *metadb.Instance
	migrations map[int]*Migration

	// Output controls the destination for messages emitted by the Instance.
	Output io.Writer
}

// NewInstance takes a pointer to a database object and a directory path. It
// loops through this directory, attempting to interpret each sub-directory
// as an individual Migration. Within these sub-directories can be any number
// of files, each representing a single Part. NewInstance returns a pointer to
// an Instance if successful. NewInstance returns an error if there is a gap
// between two migration versions or if any other error occurs.
func NewInstance(db *sql.DB, root string) (*Instance, error) {
	if db == nil {
		return nil, fmt.Errorf("NewInstance: got nil database handle")
	}

	meta, err := metadb.NewInstance(db)
	if err != nil {
		return nil, fmt.Errorf("NewInstance: got error while creating metadb instance:\n%s", err)
	}

	instance := &Instance{db: db, meta: meta, migrations: make(map[int]*Migration, 0), Output: os.Stdout}

	directories, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, directory := range directories {
		if !directory.IsDir() {
			continue
		}

		migration, err := NewMigration(path.Join(root, directory.Name()))
		if err != nil {
			return nil, err
		}

		instance.migrations[migration.Version] = migration
	}

	// if no migrations were added, return an error
	if len(instance.migrations) == 0 {
		return nil, fmt.Errorf("NewInstance: no migrations found in '%s'", root)
	}

	keys := make([]int, 0)
	for key := range instance.migrations {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	lastVersion := 0
	// Check for gaps in migration version
	for _, key := range keys {
		if key != lastVersion+1 {
			return nil, fmt.Errorf("NewInstance: found gap between migration version %d and %d", lastVersion, key)
		}
		lastVersion++
	}

	return instance, nil
}

// Version returns an integer representing which Migration the database is
// currently on. Version panics if the metadata entry in which the version is
// stored exists but cannot be fetched for some reason.
func (instance *Instance) Version() int {
	res, err := instance.meta.Get("migrateVersion")
	if err != nil {
		if _, ok := err.(*metadb.ErrNoEntry); ok {
			return 0
		}

		panic(fmt.Sprint("Instance.Version: got error:\n", err))
	}

	return res.(int)
}

// Goto applies any migrations necessary to bring the database schema to the
// state defined by the migration version specified. Goto employs transactions,
// ensuring that if anything fails, the database is automatically reverted to
// how it was before Goto was called.
func (instance *Instance) Goto(version int) error {
	currentVersion := instance.Version()
	todo := make([]*Migration, 0)
	direction := "up"
	jump := 1
	start := time.Now()

	addToTodo := func(i int) error {
		midway, ok := instance.migrations[i]
		if !ok {
			return &ErrNoVersion{fmt.Sprintf("Instance.Goto: migration for version '%d', on the way to version "+
				"'%d', does not exist", i, version)}
		}
		todo = append(todo, midway)
		return nil
	}

	// if requested version is greater than the current version, migrate up
	if version > currentVersion {
		for i := currentVersion + 1; i <= version; i++ {
			if err := addToTodo(i); err != nil {
				return err
			}
		}

		jump = version - currentVersion
	} else if version < currentVersion { // else if requested version is less than the current version, migrate down
		for i := currentVersion - 1; i > version; i-- {
			if err := addToTodo(i); err != nil {
				return err
			}
		}

		direction = "down"
		jump = currentVersion - version
	} else { // else, specified version is the same as the current version, return an error
		return &ErrNoMigrations{fmt.Sprintf("Instance.Goto: no migrations to apply, database is already on version '%d'",
			version)}
	}

	if jump > 1 {
		fmt.Fprintf(instance.Output, "\033[1mmigrate: Preparing to migrate over %d version(s)...\033[0m\n", jump)
	}

	transaction, err := instance.db.Begin()
	if err != nil {
		return fmt.Errorf("Instance.Goto: got error while starting a transaction:\n%s", err)
	}

	// Loop through and apply migrations
	for key, migration := range todo {
		fmt.Fprintf(instance.Output, "\033[1mmigrate: Beginning migration %s from version %d to %d...\033[0m\n",
			direction, currentVersion+key, migration.Version)

		applied := make([]int, 0)
		failed := make([]int, 0)
		// Apply all migration parts as per direction
		for key, part := range migration.Parts {
			var err error
			if direction == "up" {
				_, err = transaction.Exec(part.Up)
			} else {
				_, err = transaction.Exec(part.Down)
			}

			// if an error was returned, application of the part failed
			if err != nil {
				fmt.Fprintf(instance.Output, "\033[31;1m- Failed to apply '%s': %s\033[0m\n", part.Name, err)
				failed = append(failed, key)
				continue
			}

			applied = append(applied, key)
			fmt.Fprintf(instance.Output, "- Applied '%s'\n", part.Name)
		}

		// if any migration parts failed, cancel transaction and exit
		if len(failed) > 0 {
			fmt.Fprintf(instance.Output, "\n\033[1mmigrate: %d parts failed to apply, reverting %d successfully "+
				"applied parts...\033[0m\n", len(failed), len(applied))

			transaction.Rollback()
			return fmt.Errorf("Instance.Goto: got error while applying migrations")
		}

		fmt.Fprintf(instance.Output, "\033[1mmigrate: Successfully applied %d migration part(s)\n", len(applied))
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("Instance.Goto: got error while committing transaction:\n%s", err)
	}

	if err := instance.meta.Set("migrateVersion", version); err != nil {
		return fmt.Errorf("Instance.Goto: got error while updating migrate version:\n%s", err)
	}

	fmt.Fprintf(instance.Output, "\n\033[1mmigrate: Successfully applied migrations in %s\033[0m\n", time.Since(start))

	return nil
}

// Latest applies any new migrations available. Transactions are employed,
// ensuring that if anything fails, the database is automatically reverted to
// how it was before Latest was called.
func (instance *Instance) Latest() error {
	currentVersion := instance.Version()
	latestVersion := 0

	// Find highest available version
	for _, migration := range instance.migrations {
		if migration.Version > latestVersion {
			latestVersion = migration.Version
		}
	}

	if latestVersion <= currentVersion {
		return &ErrNoMigrations{fmt.Sprintf("Instance.Latest: no migrations to apply, database version %d is the latest",
			currentVersion)}
	}

	return instance.Goto(latestVersion)
}
