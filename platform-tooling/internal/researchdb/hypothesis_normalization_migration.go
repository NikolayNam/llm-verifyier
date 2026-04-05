package researchdb

import (
	"context"
	"database/sql"
	"fmt"
)

func normalizeHypothesisStorageSchema(ctx context.Context, db *sql.DB) error {
	if err := ensureColumnExists(ctx, db, "research_theorems", "row_order", `ALTER TABLE research_theorems ADD COLUMN row_order INTEGER NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	if err := ensureColumnExists(ctx, db, "research_hypothesis_sets", "theorem_file_id", `ALTER TABLE research_hypothesis_sets ADD COLUMN theorem_file_id INTEGER REFERENCES research_theorem_files(id) ON DELETE SET NULL`); err != nil {
		return err
	}
	setsExist, setsType, err := lookupDBObject(ctx, db, "research_hypothesis_sets")
	if err != nil {
		return err
	}
	filesExist, filesType, err := lookupDBObject(ctx, db, "research_theorem_files")
	if err != nil {
		return err
	}
	if !setsExist || setsType != "table" || !filesExist || filesType != "table" {
		return nil
	}
	if _, err := db.ExecContext(ctx, `
UPDATE research_hypothesis_sets
SET theorem_file_id = (
    SELECT tf.id
    FROM research_theorem_files tf
    WHERE tf.research_id = research_hypothesis_sets.research_id
      AND tf.theorem_pack_id = research_hypothesis_sets.theorem_pack_id
    ORDER BY tf.is_historical DESC, tf.imported_at_utc DESC, tf.id DESC
    LIMIT 1
)
WHERE theorem_file_id IS NULL
`); err != nil {
		return fmt.Errorf("backfill research_hypothesis_sets.theorem_file_id: %w", err)
	}
	return nil
}

func ensureColumnExists(ctx context.Context, db *sql.DB, tableName, columnName, ddl string) error {
	exists, objectType, err := lookupDBObject(ctx, db, tableName)
	if err != nil {
		return err
	}
	if !exists || objectType != "table" {
		return nil
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, tableName))
	if err != nil {
		return fmt.Errorf("pragma table_info(%s): %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			columnType string
			notNull   int
			defaultValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan table_info(%s): %w", tableName, err)
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table_info(%s): %w", tableName, err)
	}
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("alter table %s add column %s: %w", tableName, columnName, err)
	}
	return nil
}
