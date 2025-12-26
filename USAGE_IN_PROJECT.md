# Uso con CLI desde tu proyecto

Si quieres usar el CLI `go run` directamente, crea este archivo en tu proyecto:

## Opción 1: Script simple

**migrate.go:**
```go
//go:build ignore
// +build ignore

package main

import (
	"os"
	"github.com/catriel-escobar/migrator-db/cmd/main"
	
	// Importa solo el driver que necesites:
	_ "github.com/lib/pq"                 // PostgreSQL
	// _ "github.com/go-sql-driver/mysql" // MySQL
	// _ "github.com/mattn/go-sqlite3"    // SQLite
)

func main() {
	main.Run()
}
```

**Uso:**
```bash
go run migrate.go new create_users
go run migrate.go up
go run migrate.go status
```

## Opción 2: En tu aplicación

**main.go:**
```go
package main

import (
	"log"
	"os"
	
	_ "github.com/lib/pq"  // Tu driver
	"github.com/jmoiron/sqlx"
	"github.com/catriel-escobar/migrator-db/migrate"
)

func main() {
	db := connectDB()
	defer db.Close()
	
	// Auto-migrar al iniciar
	if err := migrate.Up(db, "./migrations", false); err != nil {
		log.Fatal(err)
	}
	
	startServer(db)
}

func connectDB() *sqlx.DB {
	db, _ := sqlx.Connect("postgres", os.Getenv("DB_URL"))
	return db
}
```

## Opción 3: Solo la librería (sin CLI)

```go
package main

import (
	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
	"github.com/catriel-escobar/migrator-db/migrate"
)

func main() {
	db, _ := sqlx.Connect("postgres", "...")
	
	// Crear migración
	migrate.NewMigration("./migrations", "create_users")
	
	// Aplicar
	migrate.Up(db, "./migrations", false)
	
	// Estado
	versions, _ := migrate.Status(db)
	
	// Rollback
	migrate.DownN(db, "./migrations", 2, false)
}
```

## Resumen

**Instalación:**
```bash
go get github.com/catriel-escobar/migrator-db
go get github.com/lib/pq  # tu driver
```

**En tu código, importa:**
```go
import (
    _ "github.com/lib/pq"  // driver
    "github.com/catriel-escobar/migrator-db/migrate"
)
```

¡Eso es todo! No necesitas el CLI si usas directamente las funciones de la librería.
