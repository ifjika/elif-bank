# Security Guide & Threat Modeling: AethelCore Engine

Dokumen ini mendokumentasikan kebijakan keamanan siber, matriks otorisasi berbasis peran (RBAC), mitigasi model ancaman (OWASP Top 10 API Security), aturan sanitasi input, dan manajemen rahasia (*secrets management*).

---

## 1. Matriks Otorisasi Berbasis Peran (RBAC Matrix)

Sistem membedakan hak akses berdasarkan klaim token JWT pengguna:

| Modul / Endpoint | Nasabah (Customer) | Teller (Maker) | Supervisor (Checker) | System Admin |
| :--- | :---: | :---: | :---: | :---: |
| **Cek Saldo Akun Sendiri** | ✅ | ❌ | ❌ | ❌ |
| **Transfer P2P Antar-Rekening** | ✅ (Limit Rp 25 Jt) | ❌ | ❌ | ❌ |
| **Cek Informasi Rekening Nasabah** | ❌ | ✅ | ✅ | ✅ |
| **Input Setor/Tarik Kas Cabang** | ❌ | ✅ (Maker) | ❌ | ❌ |
| **Approve Transaksi Kas > 50 Juta** | ❌ | ❌ | ✅ (Checker) | ❌ |
| **Memicu Eksekusi Manual EOD** | ❌ | ❌ | ❌ | ✅ |
| **Akses GraphQL Audit & Log** | ❌ | ❌ | ✅ | ✅ |

### 1.1 Aturan Maker-Checker Kriptografis (Anti-Self-Approval)
1. **Validasi Identitas Ganda**: Backend secara tegas memverifikasi:
   $$\text{maker\_id} \neq \text{checker\_id}$$
2. **Kunci Verifikasi Terpisah**: Token JWT untuk Maker dan Checker ditandatangani menggunakan *keypair* atau *claims* khusus yang diverifikasi secara independen oleh API Gateway.

---

## 2. Mitigasi Model Ancaman (Threat Modeling - OWASP API Security)

### 2.1 Broken Object Level Authorization (BOLA / IDOR)
* **Vektor Serangan**: Nasabah jahat mengganti parameter `account_id` pada URL request untuk membaca saldo atau mendebit rekening nasabah lain.
* **Mitigasi**:
  * API Gateway memverifikasi kecocokan antara `account_id` di database dengan `customer_id` yang tertera secara kriptografis di dalam payload JWT pemanggil.
  * Nasabah tidak pernah diizinkan mengirim `source_account_id` bebas tanpa validasi kepemilikan.

### 2.2 Replay Attacks & Network Retries
* **Vektor Serangan**: Jaringan internet seluler nasabah mengalami *drop*, menyebabkan aplikasi mobile mengirimkan ulang payload transfer beberapa kali secara otomatis, memicu pemotongan saldo berkali-kali.
* **Mitigasi**:
  * Penerapan header wajib **`X-Idempotency-Key`** (berisi UUID unik yang dibuat oleh aplikasi mobile).
  * Gateway melakukan atomic acquire pada Redis:
    ```go
    // Status awal disimpan dalam Redis dengan TTL 24 jam
    SETNX idempotency:{key} "PROCESSING" EX 86400
    ```
  * Jika kunci sudah ada dan statusnya `SUCCESS`, Gateway langsung mengembalikan salinan respons sebelumnya tanpa menyentuh database transaksi.

### 2.3 Race Condition & Saldo Negatif (Concurrency Attacks)
* **Vektor Serangan**: Nasabah melakukan 10 transaksi penarikan simultan senilai Rp 1.000.000 dari saldo awal yang hanya sebesar Rp 1.000.000 menggunakan bot terdistribusi.
* **Mitigasi**:
  * Penguncian baris database mutlak menggunakan `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`.
  * Penambahan constraint database: `CONSTRAINT chk_positive_balance CHECK (balance >= 0)`.
  * Jika saldo kurang, transaksi otomatis gagal dan di-rollback seketika.

### 2.4 SQL Injection & Mass Assignment
* **Vektor Serangan**: Memasukkan karakter kutip atau instruksi SQL berbahaya pada kolom deskripsi transaksi atau pencarian nama.
* **Mitigasi**:
  * Larangan keras merangkai string SQL (*string concatenation*). Seluruh query SQL wajib menggunakan *parameterized queries* bawaan driver `pgx` dan `sqlx`.
  * Sanitasi dan validasi format input DTO menggunakan pustaka `go-playground/validator`.

---

## 3. Manajemen Rahasia & Kunci (Secrets Management)

1. **Prinsip Nol Rahasia pada Kode (*Zero Hardcoded Secrets*)**:
   * Password database, private key JWT, dan token Redis dilarang keras disimpan di dalam source code repository.
   * Seluruh konfigurasi diinjeksi melalui *Environment Variables* atau file `.env` yang terdaftar pada `.gitignore`.
2. **Hashing Password Kredensial**:
   * Seluruh kata sandi pengguna di tabel `app_users` di-hash menggunakan algoritma **Argon2id** atau **Bcrypt** dengan *cost factor* $\ge 12$.
3. **Penyimpanan Jejak Audit Tanpa Data PII Terbuka**:
   * Log transaksi yang dicatat ke sistem telemetri (Stdout, Grafana Loki) tidak boleh memuat nomor PIN, CVV kartu, atau token otentikasi utuh.
