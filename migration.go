package migrate

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
)

// Migration represents a single migration, most importantly containing its
// version number and all the Parts contained within it.
type Migration struct {
	Name    string
	Path    string
	Version int
	Parts   []*Part
}

// NewMigration takes a directory path and parses the version number contained
// within the directory name component. It loops through this directory
// checking for files with the .sql extension, parsing them into Parts.
// NewMigration returns a pointer to a Migration if successful and an error if
// anything goes wrong.
func NewMigration(root string) (*Migration, error) {
	_, name := filepath.Split(root)
	if len(name) < 9 || name[:8] != "version_" {
		return nil, fmt.Errorf("NewMigration: expected migration directory name to be formatted as "+
			"'version_<number>', got '%s'", name)
	}

	// Parse the name component of the directory for the migration version
	// number, ignoring `version_` prefix in the first eight characters
	version, err := strconv.Atoi(name[8:])
	if err != nil {
		return nil, err
	}

	if version == 0 {
		return nil, fmt.Errorf("NewMigration: got disallowed migration version '0', reserved to represent " +
			"the initial state of the database")
	}

	root = filepath.Clean(root)
	migration := &Migration{Name: name, Path: root, Version: version}

	files, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// if the file has a .sql extension, add it to the Migration
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			filePath := path.Join(root, file.Name())

			part, err := NewPart(filePath)
			if err != nil {
				return nil, err
			}

			migration.Parts = append(migration.Parts, part)
		}
	}

	// if no parts were added, return an error
	if len(migration.Parts) == 0 {
		return nil, fmt.Errorf("NewMigration: no migration parts found in '%s'", root)
	}

	return migration, nil
}
