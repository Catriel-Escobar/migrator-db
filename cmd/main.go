package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/catriel-escobar/migrator-db/migrate"
	"github.com/jmoiron/sqlx"
	// IMPORTANTE: Importa el driver que necesites comentando/descomentando:
	// _ "github.com/lib/pq"                 // PostgreSQL
	// _ "github.com/go-sql-driver/mysql"    // MySQL
	// _ "github.com/mattn/go-sqlite3"       // SQLite
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: migrator [up|down|new|status] [flags]")
	}

	command := os.Args[1]

	// Configurar flags según el comando
	var dryRun bool
	var steps int

	switch command {
	case "up", "down":
		fs := flag.NewFlagSet(command, flag.ExitOnError)
		fs.BoolVar(&dryRun, "dry-run", false, "Simular la ejecución sin aplicar cambios")
		if command == "down" {
			fs.IntVar(&steps, "steps", 1, "Número de migraciones a revertir")
		}
		fs.Parse(os.Args[2:])
	}

	driver := os.Getenv("DB_DRIVER")
	dsn := os.Getenv("DB_URL")

	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	switch command {
	case "up":
		if err := migrate.Up(db, "./migrations", dryRun); err != nil {
			log.Fatal(err)
		}
	case "down":
		if steps < 1 {
			log.Fatal("steps debe ser mayor a 0")
		}
		if err := migrate.DownN(db, "./migrations", steps, dryRun); err != nil {
			log.Fatal(err)
		}
	case "status":
		versions, err := migrate.Status(db)
		if err != nil {
			log.Fatal(err)
		}
		if len(versions) == 0 {
			fmt.Println("No hay migraciones aplicadas")
		} else {
			fmt.Println("Migraciones aplicadas:")
			for _, v := range versions {
				fmt.Printf("  - %d\n", v)
			}
		}
	case "new":
		if len(os.Args) < 3 {
			log.Fatal("usage: migrator new <nombre>")
		}
		if err := migrate.NewMigration("./migrations", os.Args[2]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("comando desconocido: %s", command)
	}
}
