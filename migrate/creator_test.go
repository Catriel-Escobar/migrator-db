package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewMigration(t *testing.T) {
	t.Run("create migration files", func(t *testing.T) {
		dir := t.TempDir()
		
		err := NewMigration(dir, "create_users_table")
		if err != nil {
			t.Fatalf("NewMigration failed: %v", err)
		}
		
		// Verificar que se crearon los archivos
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("error reading directory: %v", err)
		}
		
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}
		
		// Verificar nombres de archivos
		var upFile, downFile string
		for _, f := range files {
			name := f.Name()
			if strings.HasSuffix(name, ".up.sql") {
				upFile = name
			}
			if strings.HasSuffix(name, ".down.sql") {
				downFile = name
			}
		}
		
		if upFile == "" {
			t.Error("archivo .up.sql no fue creado")
		}
		if downFile == "" {
			t.Error("archivo .down.sql no fue creado")
		}
		
		// Verificar que contienen el nombre
		if !strings.Contains(upFile, "create_users_table") {
			t.Errorf("nombre incorrecto en up file: %s", upFile)
		}
		if !strings.Contains(downFile, "create_users_table") {
			t.Errorf("nombre incorrecto en down file: %s", downFile)
		}
	})
	
	t.Run("files have correct content", func(t *testing.T) {
		dir := t.TempDir()
		
		err := NewMigration(dir, "test_migration")
		if err != nil {
			t.Fatalf("NewMigration failed: %v", err)
		}
		
		files, _ := os.ReadDir(dir)
		
		for _, f := range files {
			content, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				t.Fatalf("error reading file: %v", err)
			}
			
			if strings.HasSuffix(f.Name(), ".up.sql") {
				if string(content) != "-- UP\n" {
					t.Errorf("up file content incorrect: %s", string(content))
				}
			}
			if strings.HasSuffix(f.Name(), ".down.sql") {
				if string(content) != "-- DOWN\n" {
					t.Errorf("down file content incorrect: %s", string(content))
				}
			}
		}
	})
	
	t.Run("timestamp is unique", func(t *testing.T) {
		dir := t.TempDir()
		
		// Crear primera migración
		err := NewMigration(dir, "first")
		if err != nil {
			t.Fatalf("first NewMigration failed: %v", err)
		}
		
		// Esperar un poco
		time.Sleep(1 * time.Second)
		
		// Crear segunda migración
		err = NewMigration(dir, "second")
		if err != nil {
			t.Fatalf("second NewMigration failed: %v", err)
		}
		
		files, _ := os.ReadDir(dir)
		
		if len(files) != 4 {
			t.Fatalf("expected 4 files, got %d", len(files))
		}
		
		// Extraer timestamps
		timestamps := make(map[string]bool)
		for _, f := range files {
			parts := strings.Split(f.Name(), "_")
			if len(parts) > 0 {
				timestamps[parts[0]] = true
			}
		}
		
		if len(timestamps) != 2 {
			t.Error("timestamps no son únicos")
		}
	})
	
	t.Run("create directory if not exists", func(t *testing.T) {
		baseDir := t.TempDir()
		dir := filepath.Join(baseDir, "nested", "migrations")
		
		// El directorio no existe
		if _, err := os.Stat(dir); err == nil {
			t.Fatal("directory should not exist yet")
		}
		
		err := NewMigration(dir, "test")
		if err != nil {
			t.Fatalf("NewMigration failed: %v", err)
		}
		
		// Ahora debe existir
		if _, err := os.Stat(dir); err != nil {
			t.Error("directory was not created")
		}
		
		// Verificar que se crearon los archivos
		files, _ := os.ReadDir(dir)
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d", len(files))
		}
	})
	
	t.Run("empty name error", func(t *testing.T) {
		dir := t.TempDir()
		
		err := NewMigration(dir, "")
		if err == nil {
			t.Error("expected error with empty name")
		}
		
		if !strings.Contains(err.Error(), "vacío") {
			t.Errorf("error message should mention empty name: %v", err)
		}
	})
	
	t.Run("name with spaces", func(t *testing.T) {
		dir := t.TempDir()
		
		err := NewMigration(dir, "add user table")
		if err != nil {
			t.Fatalf("NewMigration failed: %v", err)
		}
		
		files, _ := os.ReadDir(dir)
		
		// Verificar que el nombre contiene los espacios
		found := false
		for _, f := range files {
			if strings.Contains(f.Name(), "add user table") {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("migration name with spaces not preserved")
		}
	})
	
	t.Run("name with special characters", func(t *testing.T) {
		dir := t.TempDir()
		
		err := NewMigration(dir, "add-user_table")
		if err != nil {
			t.Fatalf("NewMigration failed: %v", err)
		}
		
		files, _ := os.ReadDir(dir)
		if len(files) != 2 {
			t.Error("files were not created with special characters")
		}
	})
}

func TestNewMigrationIntegration(t *testing.T) {
	t.Run("created migrations can be loaded", func(t *testing.T) {
		dir := t.TempDir()
		
		// Crear algunas migraciones
		names := []string{"create_users", "add_posts", "add_comments"}
		for _, name := range names {
			err := NewMigration(dir, name)
			if err != nil {
				t.Fatalf("NewMigration(%s) failed: %v", name, err)
			}
			time.Sleep(10 * time.Millisecond) // evitar colisión de timestamps
		}
		
		// Cargar migraciones
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 3 {
			t.Fatalf("expected 3 migrations, got %d", len(migrations))
		}
		
		// Verificar que los nombres se cargaron correctamente
		for i, expected := range names {
			if migrations[i].Name != expected {
				t.Errorf("migration %d: expected name '%s', got '%s'", 
					i, expected, migrations[i].Name)
			}
		}
	})
	
	t.Run("created migrations can be applied", func(t *testing.T) {
		db := SetupTestDB(t)
		defer db.Close()
		
		dir := t.TempDir()
		
		// Crear migración
		err := NewMigration(dir, "create_test_table")
		if err != nil {
			t.Fatalf("NewMigration failed: %v", err)
		}
		
		// Editar el archivo up para agregar SQL real
		files, _ := os.ReadDir(dir)
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".up.sql") {
				path := filepath.Join(dir, f.Name())
				err := os.WriteFile(path, []byte("CREATE TABLE test (id INTEGER);"), 0644)
				if err != nil {
					t.Fatalf("error writing SQL: %v", err)
				}
			}
			if strings.HasSuffix(f.Name(), ".down.sql") {
				path := filepath.Join(dir, f.Name())
				err := os.WriteFile(path, []byte("DROP TABLE test;"), 0644)
				if err != nil {
					t.Fatalf("error writing SQL: %v", err)
				}
			}
		}
		
		// Aplicar migración
		err = Up(db, dir, false)
		if err != nil {
			t.Fatalf("Up failed: %v", err)
		}
		
		// Verificar que la tabla existe
		if !TableExists(t, db, "test") {
			t.Error("table was not created")
		}
	})
}

func TestNewMigrationConcurrent(t *testing.T) {
	t.Run("concurrent creation", func(t *testing.T) {
		dir := t.TempDir()
		
		done := make(chan error, 3)
		
		// Crear 3 migraciones concurrentemente
		for i := 0; i < 3; i++ {
			i := i
			go func() {
				time.Sleep(time.Duration(i*10) * time.Millisecond)
				done <- NewMigration(dir, "concurrent_test")
			}()
		}
		
		// Esperar a que todas terminen
		for i := 0; i < 3; i++ {
			if err := <-done; err != nil {
				t.Errorf("concurrent NewMigration failed: %v", err)
			}
		}
		
		// Verificar que se crearon 6 archivos (3 migraciones x 2 archivos)
		files, _ := os.ReadDir(dir)
		if len(files) != 6 {
			t.Errorf("expected 6 files, got %d", len(files))
		}
		
		// Todas deberían ser cargables
		migrations, err := Load(dir)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		
		if len(migrations) != 3 {
			t.Errorf("expected 3 migrations, got %d", len(migrations))
		}
	})
}

func TestNewMigrationErrors(t *testing.T) {
	t.Run("invalid directory path", func(t *testing.T) {
		// Intentar crear en un path con caracteres inválidos (Windows)
		invalidPath := string([]byte{0x00})
		err := NewMigration(invalidPath, "test")
		if err == nil {
			t.Error("expected error with invalid path")
		}
	})
	
	t.Run("read-only directory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping test when running as root")
		}
		
		dir := t.TempDir()
		
		// Hacer el directorio de solo lectura
		err := os.Chmod(dir, 0444)
		if err != nil {
			t.Fatalf("error changing permissions: %v", err)
		}
		defer os.Chmod(dir, 0755) // restaurar
		
		err = NewMigration(dir, "test")
		if err == nil {
			t.Error("expected error with read-only directory")
		}
	})
}
