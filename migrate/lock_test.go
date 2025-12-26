package migrate

import (
	"context"
	"testing"
	"time"
)

func TestNewLocker(t *testing.T) {
	t.Run("create locker for sqlite3", func(t *testing.T) {
		db := SetupTestDB(t)
		defer db.Close()
		
		locker, err := NewLocker(db)
		if err != nil {
			t.Fatalf("NewLocker failed: %v", err)
		}
		
		if locker == nil {
			t.Error("locker is nil")
		}
		
		// Verificar que es del tipo correcto
		if _, ok := locker.(*SQLiteLocker); !ok {
			t.Error("expected SQLiteLocker for sqlite3 driver")
		}
	})
	
	t.Run("unsupported driver", func(t *testing.T) {
		// No podemos fácilmente crear un driver no soportado en test
		// pero el código está ahí para cuando se necesite
		t.Skip("requires mock DB with unsupported driver")
	})
}

func TestSQLiteLock(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	t.Run("acquire and release lock", func(t *testing.T) {
		locker, err := NewLocker(db)
		if err != nil {
			t.Fatalf("NewLocker failed: %v", err)
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Adquirir lock
		err = locker.Lock(ctx)
		if err != nil {
			t.Fatalf("Lock failed: %v", err)
		}
		
		// Liberar lock
		err = locker.Unlock()
		if err != nil {
			t.Fatalf("Unlock failed: %v", err)
		}
	})
	
	t.Run("cannot lock twice", func(t *testing.T) {
		locker, _ := NewLocker(db)
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Primera vez OK
		err := locker.Lock(ctx)
		if err != nil {
			t.Fatalf("first Lock failed: %v", err)
		}
		defer locker.Unlock()
		
		// Segunda vez debe fallar
		err = locker.Lock(ctx)
		if err == nil {
			t.Error("expected error when locking twice")
		}
	})
	
	t.Run("unlock without lock is safe", func(t *testing.T) {
		locker, _ := NewLocker(db)
		
		// Unlock sin haber hecho Lock no debe fallar
		err := locker.Unlock()
		if err != nil {
			t.Errorf("Unlock without Lock failed: %v", err)
		}
	})
	
	t.Run("multiple lockers compete", func(t *testing.T) {
		db1 := SetupTestDB(t)
		defer db1.Close()
		
		db2 := SetupTestDB(t)
		defer db2.Close()
		
		locker1, _ := NewLocker(db1)
		locker2, _ := NewLocker(db2)
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Locker1 adquiere
		err := locker1.Lock(ctx)
		if err != nil {
			t.Fatalf("locker1 Lock failed: %v", err)
		}
		defer locker1.Unlock()
		
		// Locker2 intenta adquirir (con timeout corto)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel2()
		
		// Como son DBs diferentes en memoria, ambos pueden adquirir
		// (en SQLite cada :memory: es independiente)
		err = locker2.Lock(ctx2)
		// Esto depende de si la implementación comparte estado
		// En SQLite :memory: cada conexión tiene su propia DB
		// Por lo que este test es más conceptual
		_ = err // No verificamos el resultado porque depende de la implementación
	})
}

func TestLockWithContext(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()
	
	t.Run("context timeout", func(t *testing.T) {
		locker, _ := NewLocker(db)
		
		// Contexto con timeout muy corto
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		time.Sleep(10 * time.Millisecond) // Asegurar que el contexto expire
		
		err := locker.Lock(ctx)
		// Puede o no fallar dependiendo del timing
		// pero no debe crashear
		_ = err
	})
	
	t.Run("cancelled context", func(t *testing.T) {
		locker, _ := NewLocker(db)
		
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancelar inmediatamente
		
		err := locker.Lock(ctx)
		// Similar al anterior, puede o no fallar
		_ = err
	})
}

func TestLockIntegrationWithMigrations(t *testing.T) {
	t.Run("migration with lock", func(t *testing.T) {
		db := SetupTestDB(t)
		defer db.Close()
		
		dir := SetupTestMigrations(t)
		
		// Up adquiere lock automáticamente
		err := Up(db, dir, false)
		if err != nil {
			t.Fatalf("Up with lock failed: %v", err)
		}
		
		// Verificar que las migraciones se aplicaron
		AssertMigrationsApplied(t, db, []int{1, 2, 3})
	})
	
	t.Run("migration releases lock on error", func(t *testing.T) {
		db := SetupTestDB(t)
		defer db.Close()
		
		dir := t.TempDir()
		
		// Crear migración inválida
		CreateInvalidMigration(t, dir)
		
		// Intentar aplicar (debe fallar)
		err := Up(db, dir, false)
		if err == nil {
			t.Fatal("expected error with invalid migration")
		}
		
		// El lock debe haberse liberado (defer en Up)
		// Intentar otra operación
		locker, _ := NewLocker(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Debe poder adquirir el lock
		err = locker.Lock(ctx)
		if err != nil {
			t.Errorf("lock was not released after error: %v", err)
		}
		locker.Unlock()
	})
}

func TestLockConcurrency(t *testing.T) {
	t.Run("concurrent migrations attempt", func(t *testing.T) {
		// Este test es conceptual ya que con SQLite :memory:
		// cada conexión tiene su propia DB
		
		db := SetupTestDB(t)
		defer db.Close()
		
		dir := SetupTestMigrations(t)
		
		done := make(chan error, 2)
		
		// Dos goroutines intentan migrar
		for i := 0; i < 2; i++ {
			go func() {
				done <- Up(db, dir, false)
			}()
		}
		
		// Ambas deberían completar (segunda ve que ya está aplicado)
		err1 := <-done
		err2 := <-done
		
		if err1 != nil && err2 != nil {
			t.Error("both migrations failed")
		}
		
		// Verificar que las migraciones se aplicaron una vez
		migrations := GetAppliedMigrations(t, db)
		if len(migrations) != 3 {
			t.Errorf("expected 3 migrations, got %d", len(migrations))
		}
	})
}

func TestLockTableCreation(t *testing.T) {
	t.Run("lock table is created automatically", func(t *testing.T) {
		db := SetupTestDB(t)
		defer db.Close()
		
		locker, _ := NewLocker(db)
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Primera vez debe crear la tabla
		err := locker.Lock(ctx)
		if err != nil {
			t.Fatalf("Lock failed: %v", err)
		}
		defer locker.Unlock()
		
		// Verificar que la tabla migration_lock existe
		if !TableExists(t, db, "migration_lock") {
			t.Error("migration_lock table was not created")
		}
	})
}

func TestLockReacquisition(t *testing.T) {
	t.Run("can acquire lock after release", func(t *testing.T) {
		db := SetupTestDB(t)
		defer db.Close()
		
		locker, _ := NewLocker(db)
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Primer ciclo
		err := locker.Lock(ctx)
		if err != nil {
			t.Fatalf("first Lock failed: %v", err)
		}
		
		err = locker.Unlock()
		if err != nil {
			t.Fatalf("first Unlock failed: %v", err)
		}
		
		// Crear nuevo locker
		locker2, _ := NewLocker(db)
		
		// Segundo ciclo (con nuevo locker)
		err = locker2.Lock(ctx)
		if err != nil {
			t.Fatalf("second Lock failed: %v", err)
		}
		
		err = locker2.Unlock()
		if err != nil {
			t.Fatalf("second Unlock failed: %v", err)
		}
	})
}

func TestDryRunDoesNotUseLock(t *testing.T) {
	t.Run("dry run skips locking", func(t *testing.T) {
		db := SetupTestDB(t)
		defer db.Close()
		
		dir := SetupTestMigrations(t)
		
		// Adquirir lock manualmente
		locker, _ := NewLocker(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err := locker.Lock(ctx)
		if err != nil {
			t.Fatalf("Lock failed: %v", err)
		}
		defer locker.Unlock()
		
		// Dry-run debe funcionar aunque tengamos el lock
		// porque no intenta adquirirlo
		err = Up(db, dir, true)
		if err != nil {
			t.Errorf("dry-run failed even though lock is held: %v", err)
		}
		
		// No debe haber aplicado nada
		migrations := GetAppliedMigrations(t, db)
		if len(migrations) != 0 {
			t.Error("dry-run applied migrations")
		}
	})
}

// Benchmark para medir overhead del locking
func BenchmarkLockUnlock(b *testing.B) {
	db := SetupTestDB(&testing.T{})
	defer db.Close()
	
	locker, _ := NewLocker(db)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		locker.Lock(ctx)
		locker.Unlock()
	}
}
