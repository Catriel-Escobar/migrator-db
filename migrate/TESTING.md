# Tests de Migrator

Suite completa de tests para la librería de migraciones.

## Cobertura de Tests

### `runner_test.go` - Tests de Migraciones
- ✅ Aplicar todas las migraciones (`Up`)
- ✅ Modo dry-run para `Up`
- ✅ Rollback de última migración (`Down`)
- ✅ Rollback múltiple (`DownN`)
- ✅ Dry-run para rollbacks
- ✅ Manejo de errores y rollback transaccional
- ✅ Idempotencia (ejecutar Up varias veces)
- ✅ Validaciones de límites

### `loader_test.go` - Tests de Carga de Archivos
- ✅ Cargar múltiples migraciones
- ✅ Ordenamiento correcto por versión
- ✅ Parsing de nombres de archivo
- ✅ Ignorar archivos no-SQL
- ✅ Manejo de directorio vacío
- ✅ Errores con directorio inexistente
- ✅ Contenido SQL multilínea y con comentarios

### `creator_test.go` - Tests de Creación
- ✅ Crear archivos de migración
- ✅ Timestamps únicos
- ✅ Crear directorio si no existe
- ✅ Validación de nombre vacío
- ✅ Nombres con espacios y caracteres especiales
- ✅ Integración con Load y Up
- ✅ Creación concurrente

### `lock_test.go` - Tests de Locking
- ✅ Crear locker para SQLite
- ✅ Adquirir y liberar lock
- ✅ Prevenir doble lock
- ✅ Lock con timeout de contexto
- ✅ Integración con migraciones
- ✅ Liberación de lock en caso de error
- ✅ Dry-run sin locking

### `testing_helper.go` - Utilidades de Testing
- ✅ Setup de DB SQLite en memoria
- ✅ Crear migraciones de prueba
- ✅ Verificar tablas e índices
- ✅ Assertions para migraciones aplicadas

## Ejecutar Tests

### Todos los tests
```bash
cd migrate
go test -v
```

### Test específico
```bash
go test -v -run TestUp
go test -v -run TestLoad
go test -v -run TestNewMigration
go test -v -run TestLock
```

### Con cobertura
```bash
go test -v -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Tests en paralelo
```bash
go test -v -parallel 4
```

### Benchmarks
```bash
go test -bench=. -benchmem
```

## Estructura de Tests

Cada archivo de test sigue este patrón:

```go
func TestFeatureName(t *testing.T) {
    t.Run("specific scenario", func(t *testing.T) {
        // Setup
        db := SetupTestDB(t)
        defer db.Close()
        
        // Execute
        err := SomeFunction(db, args)
        
        // Assert
        if err != nil {
            t.Fatalf("expected no error, got: %v", err)
        }
    })
}
```

## Base de Datos de Test

Los tests usan **SQLite en memoria** (`:memory:`):
- Rápido y sin dependencias externas
- Cada test tiene su propia DB aislada
- Limpieza automática al terminar

## Migraciones de Prueba

Los tests crean migraciones temporales:
- `1_create_users` - Tabla users
- `2_create_posts` - Tabla posts con FK
- `3_add_index` - Índice en posts

## Coverage Objetivo

- ✅ Líneas: >80%
- ✅ Funciones: >90%
- ✅ Branches: >75%

## Tests de Integración

Los tests cubren:
1. Flujo completo: crear → aplicar → revertir
2. Dry-run en todo el ciclo
3. Manejo de errores y rollback
4. Concurrencia básica
5. Locking entre operaciones

## Ejecutar en CI/CD

```yaml
# GitHub Actions
- name: Run tests
  run: |
    cd migrate
    go test -v -race -coverprofile=coverage.out
    go tool cover -func=coverage.out
```

```yaml
# GitLab CI
test:
  script:
    - cd migrate
    - go test -v -cover
```

## Tips para Agregar Tests

1. **Usa subtests** (`t.Run`) para organizar casos
2. **Usa helpers** de `testing_helper.go` para setup
3. **Limpia recursos** con `defer`
4. **Nombra tests descriptivamente**: `TestFeature_Scenario`
5. **Un assert por test** cuando sea posible
6. **Usa tabla de tests** para casos similares

## Ejemplo de Tabla de Tests

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int
        wantErr bool
    }{
        {"valid input", "test", 4, false},
        {"empty input", "", 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Debugging Tests

```bash
# Test específico con logs
go test -v -run TestUp/apply_all_migrations

# Con race detector
go test -race

# Con timeout
go test -timeout 30s

# Verbose output
go test -v -x
```

## Próximos Tests a Agregar

- [ ] Tests con PostgreSQL real
- [ ] Tests con MySQL real
- [ ] Tests de performance con muchas migraciones
- [ ] Tests de concurrencia avanzada
- [ ] Tests de recuperación ante fallos
- [ ] Property-based testing
