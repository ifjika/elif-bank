-- ==============================================================================
-- Migration Rollback: 000001_init_ledger_schema.down.sql
-- Dokumen Terkait: docs/TSD.md (Seksi 1.1)
-- Fungsi: Menghapus seluruh tabel Core Ledger secara bersih
-- ==============================================================================

DROP TABLE IF EXISTS outbox_events CASCADE;
DROP TABLE IF EXISTS ledger_entries CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;
