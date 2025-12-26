package migrate

import "github.com/jmoiron/sqlx"

func ensure(db *sqlx.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func applied(db *sqlx.DB) ([]int, error) {
	var versions []int
	err := db.Select(&versions, `SELECT version FROM schema_migrations ORDER BY version`)
	return versions, err
}

func last(db *sqlx.DB) (int, error) {
	var v int
	err := db.Get(&v, `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`)
	return v, err
}
