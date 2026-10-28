-- ==============================================================================
-- Migration: 000001_init_auth_schema.up.sql (MariaDB 11)
-- Dokumen Terkait: docs/TSD.md (Seksi 1.2) & docs/SECURITY_GUIDE.md (Seksi 1)
-- Fungsi: Inisialisasi skema Kantor Cabang & Manajemen Pengguna (RBAC)
-- ==============================================================================

-- 1. Tabel Kantor Cabang Bank
CREATE TABLE IF NOT EXISTS bank_branches (
    id VARCHAR(36) PRIMARY KEY,
    branch_code VARCHAR(10) UNIQUE NOT NULL,
    branch_name VARCHAR(100) NOT NULL,
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Tabel Pengguna Aplikasi (Nasabah, Teller, Supervisor, Admin)
CREATE TABLE IF NOT EXISTS app_users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL, -- Hash Bcrypt/Argon2id (panjang standar 60+ chars)
    full_name VARCHAR(100) NOT NULL,
    role ENUM('CUSTOMER', 'TELLER', 'SUPERVISOR', 'ADMIN') NOT NULL,
    branch_id VARCHAR(36) NULL,
    status ENUM('ACTIVE', 'SUSPENDED') DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_users_branch FOREIGN KEY (branch_id) REFERENCES bank_branches(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_users_role ON app_users(role);
CREATE INDEX idx_users_username ON app_users(username);
