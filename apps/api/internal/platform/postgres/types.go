package postgres

// MigrationMeta holds metadata for a single migration.
type MigrationMeta struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Safe        bool   `json:"safe"`
}

// migrationEntry represents one discovered migration folder.
type migrationEntry struct {
	Path string
	Meta MigrationMeta
	Num  int
}

// MigrationRow is a migration with its applied status, exposed for CLI use.
type MigrationRow struct {
	migrationEntry
	Applied bool
}

// MigrationInfo is a lightweight migration summary for filesystem-only listing.
type MigrationInfo struct {
	Num         int
	Name        string
	Description string
	Author      string
	Safe        bool
}
