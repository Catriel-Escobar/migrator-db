package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Locker representa un mecanismo de bloqueo específico para cada base de datos
type Locker interface {
	// Lock adquiere el bloqueo
	Lock(ctx context.Context) error
	// Unlock libera el bloqueo
	Unlock() error
}

// NewLocker crea un locker apropiado según el driver de la base de datos
func NewLocker(db *sqlx.DB) (Locker, error) {
	driver := db.DriverName()

	switch driver {
	case "postgres":
		return &PostgresLocker{db: db}, nil
	case "mysql":
		return &MySQLLocker{db: db}, nil
	case "sqlite3":
		return &SQLiteLocker{db: db}, nil
	default:
		return nil, fmt.Errorf("driver no soportado para locking: %s", driver)
	}
}

// PostgresLocker usa advisory locks de PostgreSQL
type PostgresLocker struct {
	db     *sqlx.DB
	conn   *sql.Conn
	locked bool
}

const pgLockKey = 0x6D696772617465 // "migrate" en hex

func (l *PostgresLocker) Lock(ctx context.Context) error {
	if l.locked {
		return errors.New("ya está bloqueado")
	}

	// Obtener una conexión dedicada
	conn, err := l.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("error obteniendo conexión: %w", err)
	}
	l.conn = conn

	// Intentar adquirir el lock con timeout
	var acquired bool
	query := `SELECT pg_try_advisory_lock($1)`
	
	if err := conn.QueryRowContext(ctx, query, pgLockKey).Scan(&acquired); err != nil {
		conn.Close()
		return fmt.Errorf("error adquiriendo lock: %w", err)
	}

	if !acquired {
		conn.Close()
		return errors.New("no se pudo adquirir el lock, otra migración está en progreso")
	}

	l.locked = true
	return nil
}

func (l *PostgresLocker) Unlock() error {
	if !l.locked {
		return nil
	}

	defer l.conn.Close()

	var released bool
	query := `SELECT pg_advisory_unlock($1)`
	if err := l.conn.QueryRowContext(context.Background(), query, pgLockKey).Scan(&released); err != nil {
		return fmt.Errorf("error liberando lock: %w", err)
	}

	l.locked = false
	return nil
}

// MySQLLocker usa GET_LOCK() de MySQL
type MySQLLocker struct {
	db     *sqlx.DB
	conn   *sql.Conn
	locked bool
}

const mysqlLockName = "migrator_lock"

func (l *MySQLLocker) Lock(ctx context.Context) error {
	if l.locked {
		return errors.New("ya está bloqueado")
	}

	// Obtener una conexión dedicada
	conn, err := l.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("error obteniendo conexión: %w", err)
	}
	l.conn = conn

	// Intentar adquirir el lock con timeout de 10 segundos
	var result sql.NullInt64
	query := `SELECT GET_LOCK(?, 10)`
	
	if err := conn.QueryRowContext(ctx, query, mysqlLockName).Scan(&result); err != nil {
		conn.Close()
		return fmt.Errorf("error adquiriendo lock: %w", err)
	}

	if !result.Valid || result.Int64 != 1 {
		conn.Close()
		return errors.New("no se pudo adquirir el lock, otra migración está en progreso")
	}

	l.locked = true
	return nil
}

func (l *MySQLLocker) Unlock() error {
	if !l.locked {
		return nil
	}

	defer l.conn.Close()

	var result sql.NullInt64
	query := `SELECT RELEASE_LOCK(?)`
	if err := l.conn.QueryRowContext(context.Background(), query, mysqlLockName).Scan(&result); err != nil {
		return fmt.Errorf("error liberando lock: %w", err)
	}

	l.locked = false
	return nil
}

// SQLiteLocker usa una tabla de bloqueo con transacción exclusiva
type SQLiteLocker struct {
	db     *sqlx.DB
	tx     *sql.Tx
	locked bool
}

func (l *SQLiteLocker) Lock(ctx context.Context) error {
	if l.locked {
		return errors.New("ya está bloqueado")
	}

	// Crear tabla de locks si no existe
	_, err := l.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS migration_lock (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			locked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("error creando tabla de lock: %w", err)
	}

	// Iniciar transacción exclusiva
	tx, err := l.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}
	l.tx = tx

	// Intentar insertar el lock
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO migration_lock (id) VALUES (1)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error adquiriendo lock: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error verificando lock: %w", err)
	}

	// Si no se insertó ninguna fila, significa que ya existe el lock
	if rows == 0 {
		// Intentar actualizar con un timeout
		done := make(chan error, 1)
		go func() {
			_, err := tx.ExecContext(ctx, `UPDATE migration_lock SET locked_at = CURRENT_TIMESTAMP WHERE id = 1`)
			done <- err
		}()

		select {
		case err := <-done:
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("no se pudo adquirir el lock: %w", err)
			}
		case <-time.After(10 * time.Second):
			tx.Rollback()
			return errors.New("timeout esperando lock, otra migración está en progreso")
		case <-ctx.Done():
			tx.Rollback()
			return ctx.Err()
		}
	}

	l.locked = true
	return nil
}

func (l *SQLiteLocker) Unlock() error {
	if !l.locked {
		return nil
	}

	// Commit de la transacción libera el lock
	if err := l.tx.Commit(); err != nil {
		return fmt.Errorf("error liberando lock: %w", err)
	}

	// Limpiar el registro de lock
	_, err := l.db.Exec(`DELETE FROM migration_lock WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("error limpiando lock: %w", err)
	}

	l.locked = false
	return nil
}
