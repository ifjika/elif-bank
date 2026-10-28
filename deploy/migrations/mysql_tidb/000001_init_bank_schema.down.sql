-- ==============================================================================
-- Migration Rollback: 000001_init_bank_schema.down.sql (MySQL & TiDB)
-- Related Documents: docs/TSD.md
-- ==============================================================================

SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS app_users;
DROP TABLE IF EXISTS bank_branches;

SET FOREIGN_KEY_CHECKS = 1;
