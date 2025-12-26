# Ejemplos de Uso

## Ejemplo 1: Flujo Básico

```bash
# 1. Configurar variables de entorno
export DB_DRIVER=postgres
export DB_URL="postgres://user:pass@localhost:5432/myapp?sslmode=disable"

# 2. Crear primera migración
./migrator new create_users_table

# 3. Editar el archivo up
cat > migrations/1735234567_create_users_table.up.sql << 'EOF'
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
EOF

# 4. Editar el archivo down
cat > migrations/1735234567_create_users_table.down.sql << 'EOF'
DROP TABLE IF EXISTS users;
EOF

# 5. Simular la migración primero
./migrator up --dry-run

# 6. Si todo OK, aplicar
./migrator up

# 7. Ver estado
./migrator status
```

## Ejemplo 2: Workflow con Dry-Run en Producción

```bash
# En tu máquina local
./migrator new add_posts_table
# ... editar archivos SQL ...
git add migrations/
git commit -m "Add posts table migration"
git push

# En servidor de staging
git pull
export DB_DRIVER=postgres
export DB_URL="postgres://user:pass@staging-db:5432/myapp"

# Verificar qué se va a aplicar
./migrator up --dry-run

# Si todo OK, aplicar
./migrator up

# En servidor de producción (después de validar en staging)
git pull
export DB_URL="postgres://user:pass@prod-db:5432/myapp"

# SIEMPRE dry-run primero en producción
./migrator up --dry-run

# Revisar la salida cuidadosamente
# Si todo correcto, aplicar
./migrator up
```

## Ejemplo 3: Rollback con Dry-Run

```bash
# Ops! La última migración tiene un problema

# Ver qué está aplicado
./migrator status
# Output: 
# Migraciones aplicadas:
#   - 1735234567
#   - 1735234890
#   - 1735235120  <- problema aquí

# Ver qué pasaría si revertimos
./migrator down --dry-run

# Output muestra el SQL que se ejecutaría
# Si está correcto, revertir
./migrator down

# Verificar
./migrator status
```

## Ejemplo 4: Rollback Múltiple

```bash
# Tienes 5 migraciones aplicadas y quieres volver 3 atrás

./migrator status
# Migraciones aplicadas:
#   - 1735234567
#   - 1735234890
#   - 1735235120
#   - 1735235340
#   - 1735235560

# Simular revertir las últimas 3
./migrator down --steps 3 --dry-run

# Output:
# [DRY-RUN] Se revertiría migración 1735235560: add_comments
# [DRY-RUN] Se revertiría migración 1735235340: add_likes
# [DRY-RUN] Se revertiría migración 1735235120: add_posts

# Si está correcto, ejecutar
./migrator down --steps 3

# Verificar
./migrator status
# Migraciones aplicadas:
#   - 1735234567
#   - 1735234890
```

## Ejemplo 5: Uso en Docker

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o migrator cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/migrator .
COPY --from=builder /app/migrations ./migrations

CMD ["./migrator", "up"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  db:
    image: postgres:15
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - postgres_data:/var/lib/postgresql/data

  migrate:
    build: .
    depends_on:
      - db
    environment:
      DB_DRIVER: postgres
      DB_URL: postgres://user:pass@db:5432/myapp?sslmode=disable
    command: ["./migrator", "up"]

  app:
    build: .
    depends_on:
      - migrate
    environment:
      DB_DRIVER: postgres
      DB_URL: postgres://user:pass@db:5432/myapp?sslmode=disable
    ports:
      - "8080:8080"

volumes:
  postgres_data:
```

**Uso:**

```bash
# Primera vez
docker-compose up migrate

# Verificar
docker-compose run migrate ./migrator status

# Aplicar con dry-run
docker-compose run migrate ./migrator up --dry-run

# Rollback
docker-compose run migrate ./migrator down
```

## Ejemplo 6: Init Container en Kubernetes

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
type: Opaque
stringData:
  url: postgres://user:pass@postgres:5432/myapp?sslmode=disable

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      # Init container ejecuta migraciones antes de que arranque la app
      initContainers:
      - name: migrations
        image: myapp:1.0.0
        command: ["./migrator", "up"]
        env:
        - name: DB_DRIVER
          value: postgres
        - name: DB_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
      
      # Contenedor principal
      containers:
      - name: app
        image: myapp:1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: DB_DRIVER
          value: postgres
        - name: DB_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
```

**Flujo:**
1. Deploy se inicia
2. Todos los pods crean init containers simultáneamente
3. Uno adquiere el lock, los demás esperan
4. El que tiene el lock ejecuta las migraciones
5. Libera el lock
6. Los demás verifican (ya aplicadas), continúan
7. Todos los pods principales inician

## Ejemplo 7: CI/CD con GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build migrator
        run: go build -o migrator cmd/main.go
      
      - name: Validate migrations (dry-run)
        env:
          DB_DRIVER: postgres
          DB_URL: ${{ secrets.STAGING_DB_URL }}
        run: |
          ./migrator up --dry-run
          if [ $? -ne 0 ]; then
            echo "Dry-run failed! Migrations have errors."
            exit 1
          fi
      
      - name: Apply migrations to staging
        env:
          DB_DRIVER: postgres
          DB_URL: ${{ secrets.STAGING_DB_URL }}
        run: ./migrator up
      
      - name: Check migration status
        env:
          DB_DRIVER: postgres
          DB_URL: ${{ secrets.STAGING_DB_URL }}
        run: ./migrator status
      
      # Solo si staging fue exitoso
      - name: Deploy to production
        if: success()
        env:
          DB_DRIVER: postgres
          DB_URL: ${{ secrets.PROD_DB_URL }}
        run: |
          # Validar primero
          ./migrator up --dry-run
          # Aplicar
          ./migrator up
```

## Ejemplo 8: Migraciones Complejas

```sql
-- 1735234567_create_user_system.up.sql

-- Tabla de usuarios
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de perfiles
CREATE TABLE profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url TEXT,
    bio TEXT
);

-- Tabla de roles
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

-- Tabla intermedia usuarios-roles
CREATE TABLE user_roles (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Índices
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_profiles_user_id ON profiles(user_id);
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

-- Datos iniciales
INSERT INTO roles (name) VALUES 
    ('admin'),
    ('user'),
    ('moderator');
```

```sql
-- 1735234567_create_user_system.down.sql

DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;
```

```bash
# Probar
./migrator up --dry-run
./migrator up
./migrator status
./migrator down --dry-run
./migrator down
```

## Ejemplo 9: Migraciones con Datos

```sql
-- 1735235000_seed_initial_data.up.sql

-- Crear admin user
INSERT INTO users (email, password_hash) 
VALUES ('admin@example.com', '$2a$10$...');

-- Asignar rol admin
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id 
FROM users u, roles r 
WHERE u.email = 'admin@example.com' 
  AND r.name = 'admin';

-- Crear perfil
INSERT INTO profiles (user_id, first_name, last_name)
SELECT id, 'System', 'Admin'
FROM users 
WHERE email = 'admin@example.com';
```

```sql
-- 1735235000_seed_initial_data.down.sql

-- Eliminar usuario admin y sus relaciones (CASCADE se encarga)
DELETE FROM users WHERE email = 'admin@example.com';
```

## Ejemplo 10: Manejo de Errores

```bash
# Migración con error SQL
./migrator up

# Output:
# Aplicando migración 1735235678: add_invalid_column
# up 1735235678 failed: pq: column "nonexistent" does not exist

# La transacción hace rollback automático
# Puedes corregir el SQL y reintentar

# Verificar qué quedó aplicado
./migrator status
```

## Tips de Uso

### Tip 1: Alias útiles

```bash
# Agregar a .bashrc o .zshrc
alias mig='./migrator'
alias mig-up='./migrator up'
alias mig-down='./migrator down'
alias mig-status='./migrator status'
alias mig-dry='./migrator up --dry-run'
```

### Tip 2: Script de validación pre-commit

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Validar que todas las migraciones nuevas tengan up y down
for file in migrations/*.up.sql; do
    base="${file%.up.sql}"
    if [ ! -f "${base}.down.sql" ]; then
        echo "Error: ${file} no tiene archivo .down.sql correspondiente"
        exit 1
    fi
done

echo "✓ Todas las migraciones tienen archivos up y down"
```

### Tip 3: Makefile

```makefile
.PHONY: migrate-up migrate-down migrate-status migrate-new migrate-dry

migrate-up:
	./migrator up

migrate-down:
	./migrator down

migrate-status:
	./migrator status

migrate-new:
	@read -p "Nombre de la migración: " name; \
	./migrator new $$name

migrate-dry:
	./migrator up --dry-run

migrate-rollback:
	@read -p "¿Cuántos pasos revertir? " steps; \
	./migrator down --steps $$steps --dry-run
```

**Uso:**

```bash
make migrate-new
make migrate-dry
make migrate-up
make migrate-status
```
