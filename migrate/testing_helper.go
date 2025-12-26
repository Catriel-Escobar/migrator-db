package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// SetupTestDB crea una base de datos SQLite en memoria para testing
func SetupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("error creando DB de prueba: %v", err)
	}
	
	return db
}

// SetupTestMigrations crea un directorio temporal con migraciones de prueba
func SetupTestMigrations(t *testing.T) string {
	t.Helper()
	
	dir := t.TempDir()
	
	// Migración 1: crear tabla users
	createFile(t, dir, "1_create_users.up.sql", `
CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL
);
`)
	createFile(t, dir, "1_create_users.down.sql", `DROP TABLE IF EXISTS users;`)
	
	// Migración 2: crear tabla posts
	createFile(t, dir, "2_create_posts.up.sql", `
CREATE TABLE posts (
	id INTEGER PRIMARY KEY,
	user_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id)
);
`)
	createFile(t, dir, "2_create_posts.down.sql", `DROP TABLE IF EXISTS posts;`)
	
	// Migración 3: agregar índice
	createFile(t, dir, "3_add_index.up.sql", `
CREATE INDEX idx_posts_user_id ON posts(user_id);
`)
	createFile(t, dir, "3_add_index.down.sql", `DROP INDEX IF EXISTS idx_posts_user_id;`)
	
	return dir
}

// CreateMigrationFile crea un archivo de migración específico
func CreateMigrationFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	createFile(t, dir, filename, content)
}

func createFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("error creando archivo %s: %v", filename, err)
	}
}

// TableExists verifica si una tabla existe en la DB
func TableExists(t *testing.T, db *sqlx.DB, tableName string) bool {
	t.Helper()
	
	var count int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	err := db.Get(&count, query, tableName)
	if err != nil {
		t.Fatalf("error verificando tabla: %v", err)
	}
	
	return count > 0
}

// IndexExists verifica si un índice existe en la DB
func IndexExists(t *testing.T, db *sqlx.DB, indexName string) bool {
	t.Helper()
	
	var count int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`
	err := db.Get(&count, query, indexName)
	if err != nil {
		t.Fatalf("error verificando índice: %v", err)
	}
	
	return count > 0
}

// GetAppliedMigrations obtiene las versiones de migraciones aplicadas
func GetAppliedMigrations(t *testing.T, db *sqlx.DB) []int {
	t.Helper()
	
	versions, err := applied(db)
	if err != nil {
		// Si la tabla no existe aún, retornar vacío
		return []int{}
	}
	
	return versions
}

// AssertMigrationsApplied verifica que ciertas migraciones estén aplicadas
func AssertMigrationsApplied(t *testing.T, db *sqlx.DB, expected []int) {
	t.Helper()
	
	actual := GetAppliedMigrations(t, db)
	
	if len(actual) != len(expected) {
		t.Fatalf("expected %d migrations, got %d", len(expected), len(actual))
	}
	
	for i, v := range expected {
		if actual[i] != v {
			t.Fatalf("migration %d: expected %d, got %d", i, v, actual[i])
		}
	}
}

// CreateInvalidMigration crea una migración con SQL inválido
func CreateInvalidMigration(t *testing.T, dir string) {
	t.Helper()
	
	createFile(t, dir, "99_invalid.up.sql", `INVALID SQL SYNTAX HERE;`)
	createFile(t, dir, "99_invalid.down.sql", `DROP TABLE IF EXISTS nothing;`)
}

// PrintMigrationState imprime el estado actual para debugging
func PrintMigrationState(t *testing.T, db *sqlx.DB, dir string) {
	t.Helper()
	
	fmt.Println("=== Migration State ===")
	
	migrations, err := Load(dir)
	if err != nil {
		fmt.Printf("Error loading migrations: %v\n", err)
	} else {
		fmt.Printf("Total migrations found: %d\n", len(migrations))
		for _, m := range migrations {
			fmt.Printf("  - %d: %s\n", m.Version, m.Name)
		}
	}
	
	applied := GetAppliedMigrations(t, db)
	fmt.Printf("Applied migrations: %d\n", len(applied))
	for _, v := range applied {
		fmt.Printf("  - %d\n", v)
	}
	
	fmt.Println("======================")
}
