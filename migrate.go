/*
Package migrate provides a barebones API to manage database schema migration.

Directory Structure

migrate has three key concepts: instances, migrations, and parts. Instances are
top-level directories and contain an arbitrary number of migrations. Each
migration contains an arbitrary number of parts.

Generally speaking only a single instance will be necessary, the only
exception to this being if multiple schemas must be managed. migrate places
no limitations on the name of instance directories.

Each migration directory represents a single schema version, and as a result
follows a static naming convention, `version_<number>`, where `<number>` is the
schema version represented by the migration.

An arbitrary number of parts may be placed within a single migration directory.
Unlike instances and migrations, parts are simply SQL files. They follow no
particular naming conventions, the only requirement being that they end with
the `.sql` file extension. Their contents, however, must be organized in a
specific manner, documented in the Part Structure section below.

The lowest allowed schema/migration version is `1`, `0` is reserved to
represent the initial state of the database before any migrations are applied.
Gaps between version numbers are also not allowed and will raise an error.

For example:

	migrate/
	├── version_1
	│   └── test.sql
	├── version_2
	│   └── test.sql
	└── version_3
		└── test.sql

Part Structure

Although part files use the SQL syntax and are simply SQL files, they require
that both upward and downward migration SQL be specified to allow migrate to
not only upgrade the schema but also downgrade it, To show this separation, a
simple convention involving native SQL comments has been employed:

	-- @migrate/up
	CREATE TABLE example(ID INT AUTO_INCREMENT PRIMARY KEY);

	-- @migrate/down
	DROP TABLE example;

The first line of the file must be either `-- @migrate/up` or
`-- @migrate/down`, with the space after `--` being optional. These tags may
occur in any order and more than once.

All that is required is that the first line of each file begin with one of
these tags and that there be at least one of each.

Basics

To get started with migrate, open a database connection and create a new
instance, passing it the database handler and an instance directory path:

	database, _ := sql.Open(...) // Open a database connection
	defer database.Close()

	instance, err := migrate.NewInstance(database, "migrate")
	if err != nil {
		panic(err)
	}

With an instance created, migrate can now manage the database schema:

	// Upgrade the schema to the latest version available
	if err := instance.Latest(); err != nil {
		panic(err)
	}

	// Fetch the new schema version
	fmt.Println(instance.Version()) // Output: 3

	// Downgrade the database to its initial state
	if err := instance.Goto(0); err != nil {
		panic(err)
	}

`Goto` may also be used to migrate the schema to any existing version,
regardless of whether up or down relative to the current.
*/
package migrate
