package migrate

import (
	"testing"
)

func TestUp(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	// Test: Aplicar todas las migraciones
	t.Run("apply all migrations", func(t *testing.T) {
		err := Up(db, dir, false)
		if err != nil {
			t.Fatalf("Up failed: %v", err)
		}
		
		// Verificar que todas las migraciones fueron aplicadas
		AssertMigrationsApplied(t, db, []int{1, 2, 3})
		
		// Verificar que las tablas existen
		if !TableExists(t, db, "users") {
			t.Error("tabla users no fue creada")
		}
		if !TableExists(t, db, "posts") {
			t.Error("tabla posts no fue creada")
		}
		if !IndexExists(t, db, "idx_posts_user_id") {
			t.Error("índice idx_posts_user_id no fue creado")
		}
	})
	
	// Test: Ejecutar Up cuando ya están aplicadas (idempotente)
	t.Run("idempotent up", func(t *testing.T) {
		err := Up(db, dir, false)
		if err != nil {
			t.Fatalf("segundo Up failed: %v", err)
		}
		
		// Debe seguir con las mismas 3 migraciones
		AssertMigrationsApplied(t, db, []int{1, 2, 3})
	})
}

func TestUpDryRun(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	t.Run("dry run does not apply migrations", func(t *testing.T) {
		err := Up(db, dir, true)
		if err != nil {
			t.Fatalf("Up dry-run failed: %v", err)
		}
		
		// No debe haber aplicado ninguna migración
		migrations := GetAppliedMigrations(t, db)
		if len(migrations) != 0 {
			t.Errorf("dry-run aplicó migraciones: %v", migrations)
		}
		
		// Las tablas no deben existir
		if TableExists(t, db, "users") {
			t.Error("dry-run creó la tabla users")
		}
	})
}

func TestUpWithError(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	// Agregar una migración inválida
	CreateInvalidMigration(t, dir)
	
	t.Run("rollback on error", func(t *testing.T) {
		err := Up(db, dir, false)
		if err == nil {
			t.Fatal("esperaba error con SQL inválido")
		}
		
		// Las primeras 3 deben estar aplicadas
		// La 99 (inválida) no debe estar
		migrations := GetAppliedMigrations(t, db)
		if len(migrations) != 3 {
			t.Errorf("expected 3 migrations applied, got %d", len(migrations))
		}
	})
}

func TestDown(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	// Primero aplicar todas
	err := Up(db, dir, false)
	if err != nil {
		t.Fatalf("setup Up failed: %v", err)
	}
	
	t.Run("revert last migration", func(t *testing.T) {
		err := Down(db, dir, false)
		if err != nil {
			t.Fatalf("Down failed: %v", err)
		}
		
		// Debe quedar solo 1 y 2
		AssertMigrationsApplied(t, db, []int{1, 2})
		
		// El índice no debe existir
		if IndexExists(t, db, "idx_posts_user_id") {
			t.Error("índice no fue eliminado")
		}
	})
}

func TestDownDryRun(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	// Aplicar todas
	err := Up(db, dir, false)
	if err != nil {
		t.Fatalf("setup Up failed: %v", err)
	}
	
	t.Run("dry run does not revert", func(t *testing.T) {
		err := Down(db, dir, true)
		if err != nil {
			t.Fatalf("Down dry-run failed: %v", err)
		}
		
		// Todas las migraciones deben seguir aplicadas
		AssertMigrationsApplied(t, db, []int{1, 2, 3})
		
		// El índice debe seguir existiendo
		if !IndexExists(t, db, "idx_posts_user_id") {
			t.Error("dry-run eliminó el índice")
		}
	})
}

func TestDownN(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	// Aplicar todas (3 migraciones)
	err := Up(db, dir, false)
	if err != nil {
		t.Fatalf("setup Up failed: %v", err)
	}
	
	t.Run("revert multiple migrations", func(t *testing.T) {
		err := DownN(db, dir, 2, false)
		if err != nil {
			t.Fatalf("DownN failed: %v", err)
		}
		
		// Debe quedar solo la migración 1
		AssertMigrationsApplied(t, db, []int{1})
		
		// Tabla users debe existir
		if !TableExists(t, db, "users") {
			t.Error("tabla users no existe")
		}
		
		// Tabla posts no debe existir
		if TableExists(t, db, "posts") {
			t.Error("tabla posts no fue eliminada")
		}
	})
	
	t.Run("revert all migrations", func(t *testing.T) {
		// Revertir la que queda
		err := DownN(db, dir, 1, false)
		if err != nil {
			t.Fatalf("DownN failed: %v", err)
		}
		
		// No debe haber migraciones
		migrations := GetAppliedMigrations(t, db)
		if len(migrations) != 0 {
			t.Errorf("expected 0 migrations, got %d", len(migrations))
		}
		
		// Tabla users no debe existir
		if TableExists(t, db, "users") {
			t.Error("tabla users no fue eliminada")
		}
	})
}

func TestDownNDryRun(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	// Aplicar todas
	err := Up(db, dir, false)
	if err != nil {
		t.Fatalf("setup Up failed: %v", err)
	}
	
	t.Run("dry run multiple does not revert", func(t *testing.T) {
		err := DownN(db, dir, 2, true)
		if err != nil {
			t.Fatalf("DownN dry-run failed: %v", err)
		}
		
		// Todas las migraciones deben seguir aplicadas
		AssertMigrationsApplied(t, db, []int{1, 2, 3})
		
		// Todas las tablas deben existir
		if !TableExists(t, db, "users") {
			t.Error("dry-run eliminó tabla users")
		}
		if !TableExists(t, db, "posts") {
			t.Error("dry-run eliminó tabla posts")
		}
	})
}

func TestDownNErrors(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	// Aplicar todas
	err := Up(db, dir, false)
	if err != nil {
		t.Fatalf("setup Up failed: %v", err)
	}
	
	t.Run("error with invalid steps", func(t *testing.T) {
		err := DownN(db, dir, 0, false)
		if err == nil {
			t.Error("esperaba error con steps=0")
		}
		
		err = DownN(db, dir, -1, false)
		if err == nil {
			t.Error("esperaba error con steps negativos")
		}
	})
	
	t.Run("error when reverting more than applied", func(t *testing.T) {
		err := DownN(db, dir, 10, false)
		if err == nil {
			t.Error("esperaba error al intentar revertir más migraciones de las aplicadas")
		}
		
		// No debe haber cambiado nada
		AssertMigrationsApplied(t, db, []int{1, 2, 3})
	})
}

func TestDownWithNoMigrations(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := SetupTestMigrations(t)
	
	t.Run("error when no migrations applied", func(t *testing.T) {
		err := Down(db, dir, false)
		if err == nil {
			t.Error("esperaba error cuando no hay migraciones aplicadas")
		}
	})
}

func TestTransactionalRollback(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	dir := t.TempDir()
	
	// Migración que falla a mitad
	CreateMigrationFile(t, dir, "1_test.up.sql", `
		CREATE TABLE test1 (id INTEGER);
		CREATE TABLE test2 (id INTEGER);
		INVALID SQL HERE;
	`)
	CreateMigrationFile(t, dir, "1_test.down.sql", `DROP TABLE IF EXISTS test1; DROP TABLE IF EXISTS test2;`)
	
	t.Run("rollback on error in migration", func(t *testing.T) {
		err := Up(db, dir, false)
		if err == nil {
			t.Fatal("esperaba error con SQL inválido")
		}
		
		// Ninguna tabla debe existir (rollback completo)
		if TableExists(t, db, "test1") {
			t.Error("tabla test1 existe después de rollback")
		}
		if TableExists(t, db, "test2") {
			t.Error("tabla test2 existe después de rollback")
		}
		
		// La migración no debe estar registrada
		migrations := GetAppliedMigrations(t, db)
		if len(migrations) != 0 {
			t.Errorf("migración fue registrada después de error: %v", migrations)
		}
	})
}
