package migrate

import "testing"

var pExpectError = newExpectError(func(args ...interface{}) error {
	_, err := NewPart("testing/" + args[0].(string))
	return err
})

// TestBadParts ensures that NewPart returns an appropriate error message with
// invalid part files.
func TestBadParts(t *testing.T) {
	pExpectError(t, "blank migration files", "to begin with a comment denoting", "blank/version_1/test.sql")
	pExpectError(t, "migration files containing no direction markers",
		"to begin with a comment denoting", "bad_parts/no_markers.sql")
	pExpectError(t, "no upward migration SQL", "no upward migration data", "bad_parts/no_upward.sql")
	pExpectError(t, "no downward migration SQL", "no downward migration data", "bad_parts/no_downward.sql")
}
