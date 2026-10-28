-- ==============================================================================
-- Migration Rollback: 000001_init_auth_schema.down.sql
-- Dokumen Terkait: docs/TSD.md (Seksi 1.2)
-- Fungsi: Menghapus tabel otentikasi & cabang bank di MariaDB
-- ==============================================================================

DROP TABLE IF EXISTS app_users;
DROP TABLE IF EXISTS bank_branches;
