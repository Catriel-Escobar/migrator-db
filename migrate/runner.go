package migrate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func Up(db *sqlx.DB, dir string, dryRun bool) error {
	if dryRun {
		fmt.Println("\n=== MODO DRY-RUN ACTIVADO ===")
		fmt.Println("No se realizarán cambios en la base de datos")
	}

	// En dry-run no necesitamos lock
	if !dryRun {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		locker, err := NewLocker(db)
		if err != nil {
			return fmt.Errorf("error creando locker: %w", err)
		}

		if err := locker.Lock(ctx); err != nil {
			return fmt.Errorf("no se pudo adquirir lock: %w", err)
		}
		defer locker.Unlock()
	}

	if err := ensure(db); err != nil {
		return err
	}

	migrations, err := Load(dir)
	if err != nil {
		return err
	}
	appliedVersions, err := applied(db)
	if err != nil {
		return err
	}
	done := map[int]bool{}
	for _, v := range appliedVersions {
		done[v] = true
	}

	pendingCount := 0
	for _, m := range migrations {
		if done[m.Version] {
			continue
		}

		pendingCount++

		if dryRun {
			fmt.Printf("[DRY-RUN] Se aplicaría migración %d: %s\n", m.Version, m.Name)
			fmt.Println("\nContenido SQL:")
			fmt.Println("---")
			fmt.Println(m.UpSQL)
			fmt.Println("---")
			continue
		}

		fmt.Printf("Aplicando migración %d: %s\n", m.Version, m.Name)

		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("error iniciando transacción: %w", err)
		}

		if _, err := tx.Exec(m.UpSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("up %d failed: %w", m.Version, err)
		}

		// Usar Rebind para hacer la query portable entre drivers
		query := db.Rebind(`INSERT INTO schema_migrations(version) VALUES(?)`)
		if _, err := tx.Exec(query, m.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("error registrando migración %d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		fmt.Printf("✓ Migración %d aplicada\n", m.Version)
	}

	if dryRun {
		if pendingCount == 0 {
			fmt.Println("[DRY-RUN] No hay migraciones pendientes")
		} else {
			fmt.Printf("\n[DRY-RUN] Total: %d migración(es) pendiente(s)\n", pendingCount)
			fmt.Println("[DRY-RUN] Ningún cambio fue aplicado a la base de datos")
		}
	}

	return nil
}

func Down(db *sqlx.DB, dir string, dryRun bool) error {
	if dryRun {
		fmt.Println("\n=== MODO DRY-RUN ACTIVADO ===")
		fmt.Println("No se realizarán cambios en la base de datos")
	}

	// En dry-run no necesitamos lock
	if !dryRun {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		locker, err := NewLocker(db)
		if err != nil {
			return fmt.Errorf("error creando locker: %w", err)
		}

		if err := locker.Lock(ctx); err != nil {
			return fmt.Errorf("no se pudo adquirir lock: %w", err)
		}
		defer locker.Unlock()
	}

	migrations, err := Load(dir)
	if err != nil {
		return errors.New("no migration to rollback")
	}
	v, err := last(db)
	if err != nil {
		return errors.New("no migration to rollback")
	}

	var target *Migration
	for _, m := range migrations {
		if m.Version == v {
			target = &m
			break
		}
	}

	if target == nil || target.DownSQL == "" {
		return errors.New("no down migration found")
	}

	if dryRun {
		fmt.Printf("[DRY-RUN] Se revertiría migración %d: %s\n", target.Version, target.Name)
		fmt.Println("\nContenido SQL:")
		fmt.Println("---")
		fmt.Println(target.DownSQL)
		fmt.Println("---")
		fmt.Println("\n[DRY-RUN] Ningún cambio fue aplicado a la base de datos")
		return nil
	}

	fmt.Printf("Revirtiendo migración %d: %s\n", target.Version, target.Name)

	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}

	if _, err := tx.Exec(target.DownSQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("error ejecutando down migration: %w", err)
	}

	// Usar Rebind para hacer la query portable entre drivers
	query := db.Rebind(`DELETE FROM schema_migrations WHERE version = ?`)
	if _, err := tx.Exec(query, target.Version); err != nil {
		tx.Rollback()
		return fmt.Errorf("error eliminando registro de migración: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("✓ Migración %d revertida\n", target.Version)
	return nil
}

// DownN revierte N migraciones
func DownN(db *sqlx.DB, dir string, steps int, dryRun bool) error {
	if dryRun {
		fmt.Println("\n=== MODO DRY-RUN ACTIVADO ===")
		fmt.Println("No se realizarán cambios en la base de datos")
	}

	if steps < 1 {
		return errors.New("steps debe ser mayor a 0")
	}

	// En dry-run no necesitamos lock
	if !dryRun {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		locker, err := NewLocker(db)
		if err != nil {
			return fmt.Errorf("error creando locker: %w", err)
		}

		if err := locker.Lock(ctx); err != nil {
			return fmt.Errorf("no se pudo adquirir lock: %w", err)
		}
		defer locker.Unlock()
	}

	migrations, err := Load(dir)
	if err != nil {
		return fmt.Errorf("error cargando migraciones: %w", err)
	}

	appliedVersions, err := applied(db)
	if err != nil {
		return fmt.Errorf("error obteniendo migraciones aplicadas: %w", err)
	}

	if len(appliedVersions) == 0 {
		return errors.New("no hay migraciones para revertir")
	}

	// Validar que no se intente revertir más de las aplicadas
	if steps > len(appliedVersions) {
		return fmt.Errorf("solo hay %d migraci(ones) aplicada(s), no se pueden revertir %d", len(appliedVersions), steps)
	}

	// Crear mapa de migraciones por versión
	migrationMap := make(map[int]*Migration)
	for i := range migrations {
		migrationMap[migrations[i].Version] = &migrations[i]
	}

	// Obtener las últimas N versiones aplicadas (en orden descendente)
	toRevert := appliedVersions[len(appliedVersions)-steps:]
	// Invertir para revertir desde la más reciente
	for i := len(toRevert) - 1; i >= 0; i-- {
		version := toRevert[i]
		target, exists := migrationMap[version]

		if !exists {
			return fmt.Errorf("archivo de migración %d no encontrado", version)
		}

		if target.DownSQL == "" {
			return fmt.Errorf("migración %d no tiene script down", version)
		}

		if dryRun {
			fmt.Printf("[DRY-RUN] Se revertiría migración %d: %s\n", target.Version, target.Name)
			fmt.Println("\nContenido SQL:")
			fmt.Println("---")
			fmt.Println(target.DownSQL)
			fmt.Println("---")
			continue
		}

		fmt.Printf("Revirtiendo migración %d: %s\n", target.Version, target.Name)

		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("error iniciando transacción: %w", err)
		}

		if _, err := tx.Exec(target.DownSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("error ejecutando down migration %d: %w", target.Version, err)
		}

		query := db.Rebind(`DELETE FROM schema_migrations WHERE version = ?`)
		if _, err := tx.Exec(query, target.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("error eliminando registro de migración %d: %w", target.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error en commit: %w", err)
		}

		fmt.Printf("✓ Migración %d revertida\n", target.Version)
	}

	if dryRun {
		fmt.Printf("\n[DRY-RUN] Total: %d migración(es) se revertirían\n", steps)
		fmt.Println("[DRY-RUN] Ningún cambio fue aplicado a la base de datos")
	}

	return nil
}
