# Design System & UX Standards: AethelCore Console

Dokumen ini mendefinisikan standar antarmuka visual (UI) dan pengalaman pengguna (UX) untuk web konsol perbankan AethelCore (Portal Teller, Supervisor Cabang, dan Simulasi Nasabah). Dokumen ini memberikan acuan desain yang kredibel, presisi, dan mencerminkan aplikasi finansial enterprise.

---

## 1. Prinsip Desain Perbankan

1. **Kejelasan Angka Finansial**: Angka moneter dan persentase tidak boleh mengalami pergeseran visual atau terpotong. Menggunakan *tabular figures* (angka berlebar sama).
2. **Kredibilitas & Keamanan**: Palet warna didominasi nuansa biru laut (*Deep Navy/Slate*) yang memberikan kesan stabil, aman, dan dapat dipercaya.
3. **Pemberian Konfirmasi Berlapis**: Transaksi yang mengubah saldo atau status rekening wajib memiliki langkah tinjauan (*review step*) dan konfirmasi eksplisit sebelum dieksekusi.
4. **Status Tanpa Ambiguitas**: Status transaksi dan rekening harus menggunakan warna kontras dan label tegas (misal: Hijau untuk sukses/aktif, Merah untuk gagal/blokir, Kuning untuk persetujuan supervisor).

---

## 2. Token Desain (Design Tokens)

### 2.1 Palet Warna (Color Palette)

| Token | Nilai Hex | Fungsi / Penggunaan |
| :--- | :--- | :--- |
| `--color-brand-primary` | `#0F172A` (Slate 900) | Warna latar utama bar navigasi, header, dan teks judul utama. |
| `--color-brand-accent` | `#2563EB` (Blue 600) | Tombol aksi primer, tautan interaktif, indikator fokus aktif. |
| `--color-surface-bg` | `#F8FAFC` (Slate 50) | Latar belakang seluruh halaman aplikasi. |
| `--color-surface-card` | `#FFFFFF` (White) | Latar belakang kartu (*card*), tabel data, dan jendela dialog (*modal*). |
| `--color-border-subtle`| `#E2E8F0` (Slate 200) | Garis pemisah tabel, batas kartu, dan garis input form. |
| `--color-status-success`| `#16A34A` (Green 600) | Badge status `ACTIVE`, mutasi kredit (dana masuk), transaksi sukses. |
| `--color-status-warning`| `#D97706` (Amber 600) | Badge status `PENDING_APPROVAL`, `DORMANT`, peringatan limit transaksi. |
| `--color-status-danger` | `#DC2626` (Red 600) | Badge status `FROZEN`, `CLOSED`, mutasi debit (dana keluar), transaksi gagal. |

### 2.2 Tipografi (Typography)

* **Font Utama**: `Inter`, `-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, sans-serif.
* **Monetary Font Utility**: Wajib menerapkan kelas CSS `font-mono` atau `font-variant-numeric: tabular-nums` pada seluruh komponen yang menampilkan angka saldo, nomor rekening, dan nominal transfer.
* **Hirarki Ukuran Teks**:
  * `Heading 1` (Judul Halaman): 24px / Semibold (600)
  * `Heading 2` (Judul Kartu/Seksi): 18px / Semibold (600)
  * `Heading 3` (Sub-judul): 14px / Medium (500)
  * `Body Regular` (Teks Penjelas): 14px / Regular (400)
  * `Caption / Meta`: 12px / Regular (400) - Slate 500
  * `Display Balance` (Nilai Saldo Besar): 32px / Bold (700) - Tabular Nums

### 2.3 Skala Spacing & Radius Sudut

* **Spacing Grid**: Kelipatan 4px (4px, 8px, 12px, 16px, 24px, 32px, 48px).
* **Border Radius**:
  * Tombol & Input: `6px` (Rounded Medium)
  * Kartu & Kontainer: `10px` (Rounded Large)
  * Badge & Tag: `9999px` (Full Pill)

---

## 3. Komponen Standar UI

### 3.1 Kartu Saldo Rekening (Account Balance Card)
* Menampilkan Nomor Rekening (dengan tombol copy), Nama Pemilik, Tipe Rekening (Tabungan/Giro), Saldo Efektif, dan Saldo Tertahan.
* Format Saldo: Selalu menggunakan prefix mata uang `IDR` dan pemisah ribuan titik (contoh: `Rp 150.000.000,00`).

### 3.2 Badge Status (State Badges)
Komponen penanda status rekening dan transaksi:
* `<Badge variant="success">ACTIVE</Badge>`
* `<Badge variant="warning">PENDING APPROVAL</Badge>`
* `<Badge variant="danger">FROZEN</Badge>`
* `<Badge variant="neutral">CLOSED</Badge>`

### 3.3 Modal Otorisasi Maker-Checker
* Jendela pop-up khusus untuk Supervisor (Checker) saat menyetujui transaksi kas bernilai $> \text{Rp50.000.000}$.
* Elemen Wajib:
  1. Ringkasan transaksi: Nama Nasabah, Nomor Rekening, Jumlah Kas, Nama Teller (Maker), Timestamp pembuatan.
  2. Peringatan limit otorisasi perbankan.
  3. Input catatan/alasan (*approval notes*).
  4. Dua tombol tegas: **Tolak Transaksi** (Merah) dan **Setujui Transaksi** (Hijau).

### 3.4 Tabel Jurnal Buku Besar (General Ledger Table)
* Menampilkan entri mutasi double-entry secara berpasangan.
* Kolom: Timestamp, Transaction Reference ID, Account Number, Tipe (`DR` / `CR`), Nominal, Saldo Setelah Mutasi, dan Verifikasi Hash SHA-256 (icon gembok hijau jika rantai valid).

---

## 4. Aturan Pengalaman Pengguna (UX Rules)

1. **Proteksi Double-Click (Debounce & State Disable)**:
   Tombol submit transaksi finansial harus langsung dinonaktifkan (*disabled*) dan menampilkan indikator loading seketika saat diklik untuk mencegah duplikasi pengiriman data dari sisi browser.
2. **Penyembunyian Data Sensitif (Masking)**:
   Kredensial atau informasi identitas sensitif (seperti NIK atau PIN) harus dimasking secara default dengan opsi toggle "Show/Hide".
3. **Format Input Moneter Otomatis**:
   Input nominal transfer atau setor tunai harus memformat angka ribuan secara *real-time* saat diketik oleh user (contoh: user mengetik `50000000`, form menampilkan `Rp 50.000.000`).
4. **Notifikasi Toast yang Jelas**:
   Setiap transaksi sukses atau gagal harus memunculkan notifikasi toast dengan menyertakan `Reference ID` transaksi untuk memudahkan pelacakan audit.
