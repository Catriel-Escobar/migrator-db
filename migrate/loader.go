package migrate

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func Load(dir string) ([]Migration, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	m := map[int]*Migration{}

	for _, f := range files {
		name := f.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		parts := strings.Split(name, "_")
		version, _ := strconv.Atoi(parts[0])
		full := filepath.Join(dir, name)
		sql, err := os.ReadFile(full)
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}
		entry, ok := m[version]
		if !ok {
			// Extraer el nombre de la migración del nombre del archivo
			nameParts := parts[1:]
			migrationName := ""
			if len(nameParts) > 0 {
				migrationName = strings.TrimSuffix(strings.Join(nameParts, "_"), ".up.sql")
				migrationName = strings.TrimSuffix(migrationName, ".down.sql")
			}
			entry = &Migration{Version: version, Name: migrationName}
			m[version] = entry
		}

		if strings.Contains(name, ".up.") {
			entry.UpSQL = string(sql)
		} else if strings.Contains(name, ".down.") {
			entry.DownSQL = string(sql)
		}
	}

	var out []Migration
	for _, v := range m {
		out = append(out, *v)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Version < out[j].Version
	})

	return out, nil
}
