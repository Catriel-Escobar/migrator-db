package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func NewMigration(dir, name string) error {
	// Validar que el directorio existe, sino crearlo
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creando directorio: %w", err)
	}

	if name == "" {
		return errors.New("el nombre de la migración no puede estar vacío")
	}

	version := time.Now().Unix()

	up := filepath.Join(dir, fmt.Sprintf("%d_%s.up.sql", version, name))
	down := filepath.Join(dir, fmt.Sprintf("%d_%s.down.sql", version, name))

	if err := os.WriteFile(up, []byte("-- UP\n"), 0644); err != nil {
		return fmt.Errorf("error creando archivo up: %w", err)
	}
	if err := os.WriteFile(down, []byte("-- DOWN\n"), 0644); err != nil {
		return fmt.Errorf("error creando archivo down: %w", err)
	}

	fmt.Printf("✓ Migración creada: %d_%s\n", version, name)
	fmt.Printf("  UP:   %s\n", up)
	fmt.Printf("  DOWN: %s\n", down)

	return nil
}
