# Migrator - Librería de Migraciones para Go

Librería simple y robusta para manejar migraciones de base de datos en Go con soporte para PostgreSQL, MySQL y SQLite.

## Características

✅ **Multi-driver**: Soporta PostgreSQL, MySQL y SQLite  
✅ **Locking robusto**: Previene migraciones concurrentes en entornos distribuidos (Kubernetes, Docker)  
✅ **Transaccional**: Cada migración se ejecuta en una transacción  
✅ **CLI simple**: Interfaz de línea de comandos fácil de usar  
✅ **Rollback**: Revertir la última migración aplicada  

## Instalación

```bash
go get github.com/catriel-escobar/migrator
```

## Uso Básico

### 1. Configurar Variables de Entorno

```bash
export DB_DRIVER=postgres  # o mysql, sqlite3
export DB_URL="postgres://user:pass@localhost:5432/dbname?sslmode=disable"
```

**Ejemplos de connection strings:**

```bash
# PostgreSQL
export DB_URL="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

# MySQL
export DB_URL="user:pass@tcp(localhost:3306)/dbname?parseTime=true"

# SQLite
export DB_URL="./database.db"
```

### 2. Crear una Nueva Migración

```bash
./migrator new create_users_table
```

Esto crea dos archivos:
```
migrations/
  1703612345_create_users_table.up.sql
  1703612345_create_users_table.down.sql
```

### 3. Editar los Archivos SQL

**1703612345_create_users_table.up.sql:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
```

**1703612345_create_users_table.down.sql:**
```sql
DROP TABLE IF EXISTS users;
```

### 4. Aplicar Migraciones

```bash
./migrator up
```

Salida:
```
Aplicando migración 1703612345: create_users_table
✓ Migración 1703612345 aplicada
```

### 5. Modo Dry-Run (Simulación)

**Simular sin aplicar cambios:**

```bash
./migrator up --dry-run
```

Salida:
```
=== MODO DRY-RUN ACTIVADO ===
No se realizarán cambios en la base de datos

[DRY-RUN] Se aplicaría migración 1703612345: create_users_table

Contenido SQL:
---
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE
);
---

[DRY-RUN] Total: 1 migración(es) pendiente(s)
[DRY-RUN] Ningún cambio fue aplicado a la base de datos
```

**También funciona con down:**

```bash
./migrator down --dry-run
```

### 6. Revertir Última Migración

```bash
./migrator down
```

### 7. Revertir Múltiples Migraciones

```bash
# Revertir las últimas 3 migraciones
./migrator down --steps 3
```

Salida:
```
Revirtiendo migración 1703612450: add_posts_table
✓ Migración 1703612450 revertida
Revirtiendo migración 1703612380: add_comments_table
✓ Migración 1703612380 revertida
Revirtiendo migración 1703612345: create_users_table
✓ Migración 1703612345 revertida
```

**Con dry-run:**

```bash
./migrator down --steps 3 --dry-run
```

### 8. Ver Estado

```bash
./migrator status
```

## Uso Programático

```go
package main

import (
    "log"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "github.com/catriel-escobar/migrator/migrate"
)

func main() {
    db, err := sqlx.Connect("postgres", "postgres://user:pass@localhost/db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Aplicar todas las migraciones pendientes
    if err := migrate.Up(db, "./migrations", false); err != nil {
        log.Fatal(err)
    }

    // Simular aplicación de migraciones (dry-run)
    if err := migrate.Up(db, "./migrations", true); err != nil {
        log.Fatal(err)
    }

    // Revertir última migración
    if err := migrate.Down(db, "./migrations", false); err != nil {
        log.Fatal(err)
    }

    // Revertir las últimas 3 migraciones
    if err := migrate.DownN(db, "./migrations", 3, false); err != nil {
        log.Fatal(err)
    }

    // Simular revertir 2 migraciones (dry-run)
    if err := migrate.DownN(db, "./migrations", 2, true); err != nil {
        log.Fatal(err)
    }

    // Ver migraciones aplicadas
    versions, err := migrate.Status(db)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("Versiones aplicadas:", versions)
}
```

## Comandos Disponibles

```bash
# Aplicar todas las migraciones pendientes
./migrator up

# Simular aplicación (no hace cambios)
./migrator up --dry-run

# Revertir última migración
./migrator down

# Revertir múltiples migraciones
./migrator down --steps N

# Simular reversión
./migrator down --dry-run
./migrator down --steps 3 --dry-run

# Ver estado de migraciones
./migrator status

# Crear nueva migración
./migrator new <nombre>
```

## Locking en Entornos Distribuidos

El sistema de locking previene que múltiples instancias ejecuten migraciones simultáneamente:

### Kubernetes / Docker Swarm

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3  # Múltiples pods
  template:
    spec:
      initContainers:
      - name: migrations
        image: myapp:latest
        command: ["./migrator", "up"]
        env:
        - name: DB_DRIVER
          value: "postgres"
        - name: DB_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
      containers:
      - name: app
        image: myapp:latest
```

**¿Qué pasa?**
- Los 3 pods inician simultáneamente
- Solo UNO adquiere el lock y ejecuta las migraciones
- Los otros 2 esperan, verifican que ya están aplicadas, y continúan

### Docker Compose

```yaml
version: '3.8'
services:
  app:
    image: myapp:latest
    deploy:
      replicas: 3
    environment:
      DB_DRIVER: postgres
      DB_URL: postgres://user:pass@db:5432/mydb
    depends_on:
      - db
  db:
    image: postgres:15
```

## Cómo Funciona el Locking

### PostgreSQL
Usa **advisory locks** (`pg_advisory_lock`):
- Lock en memoria del servidor
- No bloquea tablas ni filas
- Se libera al cerrar la conexión

### MySQL
Usa **named locks** (`GET_LOCK`/`RELEASE_LOCK`):
- Lock nombrado compartido entre sesiones
- Timeout configurable
- Se libera al cerrar la conexión

### SQLite
Usa **tabla de lock con transacción SERIALIZABLE**:
- Aprovecha el file-level locking de SQLite
- Implementación compatible con el modelo de los otros drivers

## Estructura de Archivos

```
migrations/
  1703612345_create_users.up.sql
  1703612345_create_users.down.sql
  1703612450_add_posts.up.sql
  1703612450_add_posts.down.sql
```

**Convención de nombres:**
- `{timestamp}_{descripcion}.up.sql` - Para aplicar
- `{timestamp}_{descripcion}.down.sql` - Para revertir
- El timestamp es Unix time (segundos desde 1970)

## Tabla de Control

La librería crea automáticamente una tabla `schema_migrations`:

```sql
CREATE TABLE schema_migrations (
    version BIGINT PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Mejores Prácticas

### ✅ DO

- **Usa dry-run antes de producción** para validar cambios
- Siempre crea archivos `.up.sql` y `.down.sql`
- Usa transacciones implícitas (cada migración es una transacción)
- Prueba tus migraciones en desarrollo antes de producción
- Usa `IF NOT EXISTS` / `IF EXISTS` cuando sea apropiado
- Mantén las migraciones pequeñas y enfocadas

### ❌ DON'T

- No modifiques migraciones ya aplicadas en producción
- No uses comandos que no soporten transacciones en MySQL (ALTER TABLE puede ser problemático)
- No ejecutes migraciones manualmente en la base de datos (usa siempre la herramienta)
- No nombres tus migraciones con caracteres especiales
- No reviertas migraciones en producción sin probar primero con --dry-run

## Ejemplo Completo

```go
// cmd/main.go
package main

import (
    "log"
    "os"
    _ "github.com/lib/pq"
    "github.com/jmoiron/sqlx"
    "github.com/catriel-escobar/migrator/migrate"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatal("uso: migrator [up|down|new|status]")
    }

    driver := os.Getenv("DB_DRIVER")
    dsn := os.Getenv("DB_URL")

    db, err := sqlx.Connect(driver, dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    switch os.Args[1] {
    case "up":
        if err := migrate.Up(db, "./migrations", false); err != nil {
            log.Fatal(err)
        }
    case "down":
        steps := 1 // O parsearlo de flags
        if err := migrate.DownN(db, "./migrations", steps, false); err != nil {
            log.Fatal(err)
        }
    case "status":
        versions, err := migrate.Status(db)
        if err != nil {
            log.Fatal(err)
        }
        log.Println("Migraciones aplicadas:", versions)
    case "new":
        if len(os.Args) < 3 {
            log.Fatal("uso: migrator new <nombre>")
        }
        if err := migrate.NewMigration("./migrations", os.Args[2]); err != nil {
            log.Fatal(err)
        }
    default:
        log.Fatal("comando desconocido:", os.Args[1])
    }
}
```

## Compilar

```bash
go build -o migrator cmd/main.go
```

## Testing

```bash
# Crear base de datos de prueba
createdb migrator_test

# Variables de entorno
export DB_DRIVER=postgres
export DB_URL="postgres://localhost/migrator_test?sslmode=disable"

# Crear migración de prueba
./migrator new create_test_table

# Simular primero (dry-run)
./migrator up --dry-run

# Aplicar
./migrator up

# Verificar estado
./migrator status

# Simular reversión
./migrator down --dry-run

# Revertir
./migrator down

# Limpiar
dropdb migrator_test
```

## Roadmap

- [x] Soporte para múltiples rollbacks (`down --steps N`)
- [x] Modo dry-run
- [ ] Validación de secuencia de migraciones
- [ ] Configuración de nombre de tabla de control
- [ ] Tests unitarios
- [ ] Migraciones en código Go (además de SQL)

## Licencia

MIT

## Contribuciones

¡Las contribuciones son bienvenidas! Por favor abre un issue o PR.
