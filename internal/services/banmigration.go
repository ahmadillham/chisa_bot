package services

import (
	"database/sql"
	"log/slog"
	"strings"
)

// ensureGlobalBanTable migrates a ban table to the global schema (jid-only PK).
// Existing per-group entries are collapsed to distinct JIDs, making prior bans global.
func ensureGlobalBanTable(db *sql.DB, tableName string) error {
	// Check if group_jid column exists by querying PRAGMA.
	var hasGroupJID bool
	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return err
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

	if !hasGroupJID {
		return nil
	}

	slog.Info("Migrating ban table to global schema", "table", tableName)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	tableIdent := quoteSQLiteIdent(tableName)
	tempIdent := quoteSQLiteIdent(tableName + "_global_migration")

	if _, err = tx.Exec(`DROP TABLE IF EXISTS ` + tempIdent); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err = tx.Exec(`CREATE TABLE ` + tempIdent + ` (jid TEXT PRIMARY KEY)`); err != nil {
		_ = tx.Rollback()
		return err
	}

	res, err := tx.Exec(`INSERT OR IGNORE INTO ` + tempIdent + ` (jid)
		SELECT DISTINCT jid FROM ` + tableIdent + ` WHERE jid IS NOT NULL AND jid != ''`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	migrated, _ := res.RowsAffected()

	if _, err = tx.Exec(`DROP TABLE ` + tableIdent); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err = tx.Exec(`ALTER TABLE ` + tempIdent + ` RENAME TO ` + tableIdent); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	slog.Info("Migrated ban table to global schema", "table", tableName, "jids", migrated)
	return nil
}

func quoteSQLiteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
