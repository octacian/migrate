package migrate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var regexPartDir = regexp.MustCompile(`^--\s?@migrate/(up|down)$`)

// Part is one out of many other pieces that make up a Migration, separating
// migrate up and migrate down SQL as extracted from the file which holds it.
type Part struct {
	Name string
	Path string
	Up   string
	Down string
}

// NewPart takes a file path and parses its contents, separating migrate up and
// migrate down SQL and returning a Part.
func NewPart(path string) (*Part, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := file.Close(); err != nil {
			panic(fmt.Sprint("Migration.AddPart: got error while closing part file:\n", err))
		}
	}()

	errNoMarker := NewFatalf("Migration.AddFile: expected part file '%s' to begin with a comment "+
		"denoting whether the following SQL represents an upward or downward migration "+
		"(for example: '-- @migrate/up' or '@migrate/down')", path)

	upSQL := ""
	downSQL := ""
	which := -1
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		matches := regexPartDir.FindStringSubmatch(text)

		// if matches were found, check them
		if len(matches) > 1 {
			if matches[1] == "up" {
				which = 0
			} else if matches[1] == "down" {
				which = 1
			}

			continue
		}

		if text == "" {
			continue // Ignore blank strings
		}

		switch which {
		case 0: // if 0, append to upSQL
			upSQL += text
		case 1: // if 1, append to downSQL
			downSQL += text
		default: // otherwise, return error
			return nil, errNoMarker
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	if which == -1 {
		return nil, errNoMarker
	}

	if upSQL == "" {
		return nil, NewFatalf("Migration.AddFile: file '%s' contains no upward migration data", path)
	}

	if downSQL == "" {
		return nil, NewFatalf("Migration.AddFile: file '%s' contains no downward migration data", path)
	}

	_, filename := filepath.Split(path)
	return &Part{Name: filename, Path: path, Up: upSQL, Down: downSQL}, nil
}
