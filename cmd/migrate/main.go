package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

func main() {
	var (
		direction = flag.String("direction", "up", "migration direction: up or down")
		steps     = flag.Int("steps", 0, "number of migrations to apply (0 = all possible)")
	)
	flag.Parse()

	if flag.NArg() > 0 {
		*direction = flag.Arg(0)
	}

	if *direction != "up" && *direction != "down" {
		log.Fatalf("invalid direction %q (expected up or down)", *direction)
	}

	databaseURL := os.Getenv("POSTGRES_URL")
	if databaseURL == "" {
		log.Fatal("POSTGRES_URL is required")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer conn.Close(ctx)

	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		log.Fatalf("ensure schema_migrations table: %v", err)
	}

	migrations, err := loadMigrationFiles("deploy/sql/migrations")
	if err != nil {
		log.Fatalf("load migrations: %v", err)
	}

	if len(migrations) == 0 {
		log.Println("no migration files found")
		return
	}

	applied, err := loadAppliedMigrations(ctx, conn)
	if err != nil {
		log.Fatalf("load applied migrations: %v", err)
	}

	var appliedCount int
	for _, migration := range selectMigrations(*direction, *steps, migrations, applied) {
		if err := runMigration(ctx, conn, *direction, migration); err != nil {
			log.Fatalf("apply migration %s %s: %v", migration.Version, *direction, err)
		}
		appliedCount++
	}

	log.Printf("completed %d %s migration(s)", appliedCount, *direction)
}

type migration struct {
	Version  string
	UpPath   string
	DownPath string
}

func ensureSchemaMigrationsTable(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func loadMigrationFiles(root string) ([]migration, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	versions := map[string]*migration{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			version := strings.TrimSuffix(name, ".up.sql")
			m := versions[version]
			if m == nil {
				m = &migration{Version: version}
				versions[version] = m
			}
			m.UpPath = filepath.Join(root, name)
		}
		if strings.HasSuffix(name, ".down.sql") {
			version := strings.TrimSuffix(name, ".down.sql")
			m := versions[version]
			if m == nil {
				m = &migration{Version: version}
				versions[version] = m
			}
			m.DownPath = filepath.Join(root, name)
		}
	}

	result := make([]migration, 0, len(versions))
	for _, m := range versions {
		if m.UpPath == "" || m.DownPath == "" {
			return nil, fmt.Errorf("migration %q missing up or down file", m.Version)
		}
		result = append(result, *m)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

func loadAppliedMigrations(ctx context.Context, conn *pgx.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return applied, nil
}

func selectMigrations(direction string, steps int, all []migration, applied map[string]bool) []migration {
	selected := make([]migration, 0)

	switch direction {
	case "up":
		for _, m := range all {
			if !applied[m.Version] {
				selected = append(selected, m)
			}
		}
	case "down":
		for i := len(all) - 1; i >= 0; i-- {
			if applied[all[i].Version] {
				selected = append(selected, all[i])
			}
		}
	}

	if steps > 0 && steps < len(selected) {
		selected = selected[:steps]
	}

	return selected
}

func runMigration(ctx context.Context, conn *pgx.Conn, direction string, m migration) error {
	path := m.UpPath
	if direction == "down" {
		path = m.DownPath
	}

	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("migration file %q not found", path)
		}
		return err
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return err
	}

	if direction == "up" {
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, m.Version); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, m.Version); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	log.Printf("applied %s migration %s", direction, m.Version)
	return nil
}
