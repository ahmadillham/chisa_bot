package services

import (
	"database/sql"
	"log/slog"
)

// migrateBanTable migrates a ban table from the old schema (jid-only PK)
// to the new schema (jid + group_jid composite PK).
// If the table already has the new schema, this is a no-op.
func migrateBanTable(db *sql.DB, tableName string) {
	// Check if group_jid column exists by querying PRAGMA.
	var hasGroupJID bool
	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}
		if name == "group_jid" {
			hasGroupJID = true
			break
		}
	}

	if hasGroupJID {
		return // Already migrated.
	}

	slog.Info("Migrating ban table to per-group schema", "table", tableName)

	// Old schema only had `jid TEXT PRIMARY KEY`.
	// We need to recreate the table with the new composite PK.
	// Old entries without group_jid context are dropped (admins need to re-ban per group).
	tx, err := db.Begin()
	if err != nil {
		slog.Error("Failed to start migration transaction", "table", tableName, "error", err)
		return
	}

	// Count old entries for logging.
	var count int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM ` + tableName).Scan(&count)

	// Drop old table and recreate with new schema.
	_, err = tx.Exec(`DROP TABLE IF EXISTS ` + tableName)
	if err != nil {
		_ = tx.Rollback()
		slog.Error("Failed to drop old table", "table", tableName, "error", err)
		return
	}

	_, err = tx.Exec(`CREATE TABLE ` + tableName + ` (
		jid TEXT NOT NULL,
		group_jid TEXT NOT NULL,
		PRIMARY KEY (jid, group_jid)
	)`)
	if err != nil {
		_ = tx.Rollback()
		slog.Error("Failed to create new table", "table", tableName, "error", err)
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit migration", "table", tableName, "error", err)
		return
	}

	if count > 0 {
		slog.Warn("Old ban entries dropped during migration (no group context). Admins should re-ban per group.",
			"table", tableName, "dropped_entries", count)
	}
}
