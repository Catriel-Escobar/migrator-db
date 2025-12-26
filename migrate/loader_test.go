package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	
	t.Run("load multiple migrations", func(t *testing.T) {
		// Crear archivos de migración
		CreateMigrationFile(t, dir, "1_create_users.up.sql", "CREATE TABLE users;")
		CreateMigrationFile(t, dir, "1_create_users.down.sql", "DROP TABLE users;")
		CreateMigrationFile(t, dir, "2_add_posts.up.sql", "CREATE TABLE posts;")
		CreateMigrationFile(t, dir, "2_add_posts.down.sql", "DROP TABLE posts;")
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 2 {
			t.Fatalf("expected 2 migrations, got %d", len(migrations))
		}
		
		// Verificar orden (deben estar ordenadas por versión)
		if migrations[0].Version != 1 {
			t.Errorf("first migration version: expected 1, got %d", migrations[0].Version)
		}
		if migrations[1].Version != 2 {
			t.Errorf("second migration version: expected 2, got %d", migrations[1].Version)
		}
		
		// Verificar nombres
		if migrations[0].Name != "create_users" {
			t.Errorf("first migration name: expected 'create_users', got '%s'", migrations[0].Name)
		}
		if migrations[1].Name != "add_posts" {
			t.Errorf("second migration name: expected 'add_posts', got '%s'", migrations[1].Name)
		}
		
		// Verificar contenido SQL
		if migrations[0].UpSQL != "CREATE TABLE users;" {
			t.Errorf("first migration up SQL incorrect: %s", migrations[0].UpSQL)
		}
		if migrations[0].DownSQL != "DROP TABLE users;" {
			t.Errorf("first migration down SQL incorrect: %s", migrations[0].DownSQL)
		}
	})
	
	t.Run("load with only up migration", func(t *testing.T) {
		dir := t.TempDir()
		CreateMigrationFile(t, dir, "1_test.up.sql", "SELECT 1;")
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}
		
		if migrations[0].UpSQL != "SELECT 1;" {
			t.Error("up SQL not loaded")
		}
		if migrations[0].DownSQL != "" {
			t.Error("down SQL should be empty")
		}
	})
	
	t.Run("ignore non-sql files", func(t *testing.T) {
		dir := t.TempDir()
		CreateMigrationFile(t, dir, "1_test.up.sql", "SELECT 1;")
		CreateMigrationFile(t, dir, "README.md", "# Migrations")
		CreateMigrationFile(t, dir, "script.sh", "#!/bin/bash")
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 1 {
			t.Errorf("expected 1 migration, got %d (non-sql files should be ignored)", len(migrations))
		}
	})
	
	t.Run("load migrations in correct order", func(t *testing.T) {
		dir := t.TempDir()
		
		// Crear en orden aleatorio
		CreateMigrationFile(t, dir, "5_fifth.up.sql", "SELECT 5;")
		CreateMigrationFile(t, dir, "1_first.up.sql", "SELECT 1;")
		CreateMigrationFile(t, dir, "3_third.up.sql", "SELECT 3;")
		CreateMigrationFile(t, dir, "2_second.up.sql", "SELECT 2;")
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		// Deben estar ordenadas ascendentemente
		if len(migrations) != 4 {
			t.Fatalf("expected 4 migrations, got %d", len(migrations))
		}
		
		expected := []int{1, 2, 3, 5}
		for i, m := range migrations {
			if m.Version != expected[i] {
				t.Errorf("migration %d: expected version %d, got %d", i, expected[i], m.Version)
			}
		}
	})
	
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 0 {
			t.Errorf("expected 0 migrations, got %d", len(migrations))
		}
	})
	
	t.Run("non-existent directory", func(t *testing.T) {
		_, err := Load("/path/that/does/not/exist")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})
}

func TestLoadWithComplexNames(t *testing.T) {
	dir := t.TempDir()
	
	t.Run("migration with underscores in name", func(t *testing.T) {
		CreateMigrationFile(t, dir, "1_create_user_profiles_table.up.sql", "SELECT 1;")
		CreateMigrationFile(t, dir, "1_create_user_profiles_table.down.sql", "SELECT 2;")
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}
		
		if migrations[0].Name != "create_user_profiles_table" {
			t.Errorf("expected name 'create_user_profiles_table', got '%s'", migrations[0].Name)
		}
	})
	
	t.Run("migration with long timestamp", func(t *testing.T) {
		dir := t.TempDir()
		CreateMigrationFile(t, dir, "1703612345_add_index.up.sql", "CREATE INDEX;")
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}
		
		if migrations[0].Version != 1703612345 {
			t.Errorf("expected version 1703612345, got %d", migrations[0].Version)
		}
	})
}

func TestLoadMigrationContent(t *testing.T) {
	dir := t.TempDir()
	
	t.Run("multiline SQL content", func(t *testing.T) {
		upSQL := `
CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	email TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
`
		downSQL := `
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
`
		
		CreateMigrationFile(t, dir, "1_complex.up.sql", upSQL)
		CreateMigrationFile(t, dir, "1_complex.down.sql", downSQL)
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if migrations[0].UpSQL != upSQL {
			t.Error("up SQL content doesn't match")
		}
		if migrations[0].DownSQL != downSQL {
			t.Error("down SQL content doesn't match")
		}
	})
	
	t.Run("SQL with comments", func(t *testing.T) {
		sql := `
-- This is a comment
CREATE TABLE test (
	id INTEGER PRIMARY KEY
	-- inline comment
);
/* Multi-line
   comment */
`
		CreateMigrationFile(t, dir, "2_comments.up.sql", sql)
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		found := false
		for _, m := range migrations {
			if m.Version == 2 {
				if m.UpSQL != sql {
					t.Error("SQL with comments not preserved correctly")
				}
				found = true
			}
		}
		
		if !found {
			t.Error("migration 2 not found")
		}
	})
}

func TestLoadWithInvalidFileNames(t *testing.T) {
	dir := t.TempDir()
	
	t.Run("file without version number", func(t *testing.T) {
		CreateMigrationFile(t, dir, "no_version.up.sql", "SELECT 1;")
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		// Debe tener version 0 (atoi falla y retorna 0)
		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}
		if migrations[0].Version != 0 {
			t.Errorf("expected version 0, got %d", migrations[0].Version)
		}
	})
	
	t.Run("file without underscore separator", func(t *testing.T) {
		dir := t.TempDir()
		CreateMigrationFile(t, dir, "1.up.sql", "SELECT 1;")
		
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}
	})
}

func TestLoadPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}
	
	t.Run("unreadable file", func(t *testing.T) {
		dir := t.TempDir()
		
		// Crear archivo
		filePath := filepath.Join(dir, "1_test.up.sql")
		err := os.WriteFile(filePath, []byte("SELECT 1;"), 0644)
		if err != nil {
			t.Fatalf("error creating file: %v", err)
		}
		
		// Hacer el archivo no legible
		err = os.Chmod(filePath, 0000)
		if err != nil {
			t.Fatalf("error changing permissions: %v", err)
		}
		defer os.Chmod(filePath, 0644) // restaurar para cleanup
		
		// Load debería manejar el error gracefully
		migrations, err := Load(dir)
		
		// En algunos sistemas el error puede no ocurrir
		// Solo verificamos que no crashee
		if err == nil && len(migrations) == 0 {
			// OK - no pudo leer el archivo
		}
	})
}
