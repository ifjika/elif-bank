-- Migration Rollback: 000002_add_tx_branch_accounts.down.sql
ALTER TABLE transactions
    DROP COLUMN target_account_id,
    DROP COLUMN vault_account_id;
