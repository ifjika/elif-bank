-- ==============================================================================
-- Migration: 000002_add_tx_branch_accounts.up.sql (MySQL 8.0 & TiDB Cloud)
-- Purpose: Store target customer account and branch vault account on pending
--          transactions so Maker-Checker approvals can resolve & lock accounts.
-- ==============================================================================

ALTER TABLE transactions
    ADD COLUMN target_account_id CHAR(36) NULL AFTER description;

ALTER TABLE transactions
    ADD COLUMN vault_account_id CHAR(36) NULL AFTER target_account_id;
