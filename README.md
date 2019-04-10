# migrate

migrate provides a barebones API to manage database schema migration. It is capable of not only upgrading but also downgrading schemas. migrate is intended to be as simple as possible, and thus does not integrate command-line tools into its workflow, instead depending upon a simple, easily replicatable, directory structure:

```
migrate/
├── version_1
│   └── test.sql
├── version_2
│   └── test.sql
└── version_3
	└── test.sql
```

For more information and API documentation see [GoDoc](https://godoc.org/github.com/octacian/migrate).
