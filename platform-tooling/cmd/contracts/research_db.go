package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

func resolveResearchDBPath(workspaceRoot, dbPath string) string {
	if strings.TrimSpace(dbPath) == "" {
		return filepath.Join(workspaceRoot, filepath.FromSlash(researchdb.DefaultDBRelativePath))
	}
	return resolveBenchmarkArgPath(workspaceRoot, dbPath)
}

func countResearchDBTables(ctx context.Context, dbPath string) (int, error) {
	db, err := researchdb.Open(ctx, dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM sqlite_master
WHERE type = 'table' AND name LIKE 'research_%'
`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count research tables: %w", err)
	}
	return count, nil
}

func quoteSQLiteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func resetResearchDBInPlace(ctx context.Context, dbPath string) error {
	db, err := researchdb.Open(ctx, dbPath)
	if err != nil {
		return err
	}

	rows, err := db.QueryContext(ctx, `
SELECT name
FROM sqlite_master
WHERE type = 'table' AND name LIKE 'research_%'
ORDER BY name DESC
`)
	if err != nil {
		return fmt.Errorf("list research tables: %w", err)
	}
	defer rows.Close()

	tableNames := make([]string, 0, 16)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan research table name: %w", err)
		}
		tableNames = append(tableNames, name)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate research table names: %w", err)
	}

	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("disable foreign keys for reset: %w", err)
	}
	for _, name := range tableNames {
		if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS `+quoteSQLiteIdentifier(name)); err != nil {
			return fmt.Errorf("drop research table %s: %w", name, err)
		}
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("re-enable foreign keys for reset: %w", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close research db after in-place reset: %w", err)
	}

	reopened, err := researchdb.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	return reopened.Close()
}

func runResearchDBInitCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("research-db-init", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dbPath := fs.String("db", firstNonEmptyEnv("RESEARCH_DB_PATH"), "SQLite research db path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	resolvedDBPath := resolveResearchDBPath(workspaceRoot, *dbPath)
	tableCount, err := countResearchDBTables(context.Background(), resolvedDBPath)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "research db initialized\nworkspace_root: %s\ndb_path: %s\nresearch_tables: %d\n",
		filepath.ToSlash(workspaceRoot), filepath.ToSlash(resolvedDBPath), tableCount)
	return nil
}

func runResearchDBResetCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("research-db-reset", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dbPath := fs.String("db", firstNonEmptyEnv("RESEARCH_DB_PATH"), "SQLite research db path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	resolvedDBPath := resolveResearchDBPath(workspaceRoot, *dbPath)
	for _, candidate := range []string{
		resolvedDBPath,
		resolvedDBPath + "-wal",
		resolvedDBPath + "-shm",
		resolvedDBPath + "-journal",
	} {
		if removeErr := os.Remove(candidate); removeErr != nil && !os.IsNotExist(removeErr) {
			if candidate != resolvedDBPath {
				return fmt.Errorf("remove %s: %w", filepath.ToSlash(candidate), removeErr)
			}
			if resetErr := resetResearchDBInPlace(context.Background(), resolvedDBPath); resetErr != nil {
				return fmt.Errorf("remove %s: %w; in-place reset failed: %v", filepath.ToSlash(candidate), removeErr, resetErr)
			}
			tableCount, countErr := countResearchDBTables(context.Background(), resolvedDBPath)
			if countErr != nil {
				return countErr
			}
			_, _ = fmt.Fprintf(stdout, "research db reset completed\nworkspace_root: %s\ndb_path: %s\nreset_mode: in_place\nresearch_tables: %d\n",
				filepath.ToSlash(workspaceRoot), filepath.ToSlash(resolvedDBPath), tableCount)
			return nil
		}
	}

	tableCount, err := countResearchDBTables(context.Background(), resolvedDBPath)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "research db reset completed\nworkspace_root: %s\ndb_path: %s\nresearch_tables: %d\n",
		filepath.ToSlash(workspaceRoot), filepath.ToSlash(resolvedDBPath), tableCount)
	return nil
}

func runResearchDBSyncCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("research-db-sync", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", firstNonEmptyEnv("RESEARCH_DB_PROJECT_FOLDER", "HILBERT_BENCHMARK_PROJECT_FOLDER"), "Research project folder under research/artifacts")
	dbPath := fs.String("db", firstNonEmptyEnv("RESEARCH_DB_PATH"), "SQLite research db path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if strings.TrimSpace(*projectFolder) == "" {
		return fmt.Errorf("research db sync requires -project-folder")
	}
	*dbPath = resolveResearchDBPath(workspaceRoot, *dbPath)

	result, err := researchdb.SyncProject(context.Background(), *dbPath, researchdb.SyncOptions{
		WorkspaceRoot: workspaceRoot,
		ProjectFolder: strings.TrimSpace(*projectFolder),
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "research db sync completed\nworkspace_root: %s\ndb_path: %s\nresearch_slug: %s\ntheorem_files: %d\ntheorems_imported: %d\nresult_files: %d\nresult_rows_imported: %d\n",
		filepath.ToSlash(workspaceRoot), filepath.ToSlash(result.DBPath), result.ResearchSlug, result.TheoremFiles, result.TheoremsImported, result.ResultFiles, result.ResultRowsImported)
	return nil
}

