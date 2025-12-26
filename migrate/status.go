package migrate

import "github.com/jmoiron/sqlx"

func Status(db *sqlx.DB) ([]int, error) {
	return applied(db)
}
