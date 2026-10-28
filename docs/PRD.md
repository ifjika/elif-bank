# Product Requirements Document (PRD)
## Proyek: AethelCore - Mini Core Banking System (CBS)

Dokumen ini menjelaskan kebutuhan produk, logika bisnis perbankan, batasan sistem, dan target metrik untuk AethelCore. Dokumen ini dirancang dengan bahasa yang mudah dipahami bagi pemula yang baru mempelajari arsitektur perbankan dan backend Golang.

---

## 1. Pengantar: Apa itu Core Banking System (CBS)?

Di dunia perbankan riil, **Core Banking System (CBS)** adalah "jantung" dari seluruh operasional bank. Sistem ini bertanggung jawab atas:
1. Membuka dan mengelola rekening nasabah.
2. Memproses mutasi saldo (setor, tarik, transfer).
3. Menjaga buku besar akuntansi (*General Ledger*) agar selalu seimbang (*balanced*).
4. Menjalankan proses penutupan buku harian (*End of Day* / EOD).
5. Mencegah kecurangan (*fraud*) melalui kontrol otorisasi ganda (*Four-Eyes Principle / Maker-Checker*).

Tujuan proyek ini adalah membangun replika miniatur CBS berstandar industri dengan teknologi modern yang dapat dibuktikan keandalannya saat wawancara kerja teknis di perbankan atau fintech.

---

## 2. Pengguna Sistem (User Personas)

| Persona | Peran | Tanggung Jawab Utama |
| :--- | :--- | :--- |
| **Nasabah (Customer)** | Pengguna akhir kanal digital (*Mobile Banking*) | Melakukan transfer dana, cek saldo, melihat riwayat mutasi, dan menerima bunga simpanan. |
| **Teller (Maker)** | Staf operasional cabang bank | Melakukan transaksi fisik setor tunai dan tarik tunai nasabah di konter cabang. |
| **Supervisor (Checker)** | Pejabat operasional cabang bank | Memverifikasi dan menyetujui (*approve*) transaksi berisiko tinggi (misal: penarikan kas bernilai besar). |
| **Sistem Otomatis (Worker/Daemon)** | Pekerja latar belakang (*background worker*) | Menjalankan kalkulasi bunga harian (EOD), simulasi lalu lintas beban tinggi, dan audit integritas data. |

---

## 3. Ruang Lingkup & Spesifikasi Fitur Utama

### 3.1 Manajemen Rekening Simpanan (Funding & Accounts)
* **Jenis Akun**:
  * *Tabungan Individu*: Rekening simpanan likuid nasabah perorangan dengan suku bunga harian.
  * *Giro Komersial*: Rekening operasional perusahaan/pedagang dengan mutasi volume tinggi.
  * *Deposito Berjangka*: Rekening simpanan berjangka dengan penalti penarikan sebelum jatuh tempo.
* **State Machine Rekening**:
  * `ACTIVE`: Rekening aktif dan dapat menerima mutasi debit/kredit.
  * `FROZEN`: Rekening dibekukan sementara (misal: permintaan penegak hukum atau anomali transaksi). Menolak semua transaksi keluar (debit).
  * `DORMANT`: Rekening pasif tanpa transaksi nasabah selama > 180 hari. Memerlukan verifikasi ulang untuk diaktifkan.
  * `CLOSED`: Rekening ditutup permanen dengan saldo nol. Menolak seluruh transaksi secara mutlak.
* **Pemisahan Saldo Efektif & Saldo Tertahan**:
  * $\text{Total Saldo} = \text{Available Balance} + \text{Hold Balance}$
  * Saldo tertahan (*hold balance*) digunakan untuk jaminan pinjaman atau pre-otorisasi tanpa mendebit uang secara langsung.

---

### 3.2 Mesin Buku Besar Berpasangan (Double-Entry General Ledger)
* **Prinsip Dasar**: Tidak boleh ada manipulasi saldo langsung yang bersifat sepihak (`UPDATE accounts SET balance = balance - 1000`).
* **Aturan Jurnal**: Setiap kali ada uang bergerak, sistem wajib membuat minimal 2 baris entri jurnal (*ledger entries*):
  * Akun Sumber / Aset berkurang $\to$ dicatat di sisi yang sesuai.
  * Akun Tujuan / Liabilitas bertambah $\to$ dicatat di sisi lawannya.
* **Invarian Akuntansi**:
  $$\sum \text{Debit} - \sum \text{Kredit} = 0$$
  Jika dalam satu transaksi total debit $\neq$ total kredit, database wajib melakukan `ROLLBACK` seketika.

---

### 3.3 Tata Kelola Cabang: Maker-Checker (Four-Eyes Principle)
* **Ambang Batas (Threshold)**: Transaksi kas cabang bernilai $\le \text{Rp50.000.000}$ dapat diproses langsung oleh Teller (*Maker*).
* **Otorisasi Ganda**: Transaksi kas bernilai $> \text{Rp50.000.000}$ otomatis masuk ke status `PENDING_APPROVAL`.
* **Pemisahan Tugas**:
  * Teller pembuat transaksi (`maker_id`) dilarang keras menyetujui transaksinya sendiri.
  * Supervisor cabang (`checker_id`) harus menggunakan kredensial/token tersendiri untuk menyetujui (*APPROVE*) atau menolak (*REJECT*).

---

### 3.4 Penutupan Hari Otomatis (End-of-Day / EOD Engine)
* Berjalan secara terjadwal untuk mengemulasikan pergantian hari kalender bank:
  1. **Akrual Bunga**: Menghitung bunga harian dari saldo akhir setiap rekening aktif:
     $$\text{Bunga Harian} = \frac{\text{Saldo Akhir} \times \text{Suku Bunga}}{365}$$
  2. **Pajak Penghasilan (PPh Final 20%)**: Memotong pajak otomatis atas pendapatan bunga sesuai regulasi keuangan Indonesia.
  3. **Biaya Administrasi**: Memotong biaya pemeliharaan rekening bulanan.
  4. **Pemeriksaan Kolektibilitas Kredit**: Mengelompokkan pinjaman nasabah ke dalam kategori lancar hingga macet (KOL 1–5) berdasarkan *Days Past Due* (DPD).
  5. **Rekonsiliasi Neraca Saldo**: Menghitung total aktiva, pasiva, dan ekuitas untuk memvalidasi persamaan dasar akuntansi.

---

### 3.5 Generator Transaksi Otonom (Cron Simulator)
* Menghasilkan beban lalu lintas transaksi sintetis secara berkelanjutan untuk:
  * Menguji performa sistem di bawah beban konkurensi tinggi.
  * Membuktikan ketahanan sistem terhadap *race condition* dan *deadlock*.
  * Menampilkan metrik *real-time* pada dashboard pemantauan (Grafana).

---

## 4. Persyaratan Non-Fungsional (NFR)

1. **Jaminan Transaksi ACID**: Atomicity, Consistency, Isolation, dan Durability dijamin penuh oleh database engine.
2. **Determinisme Konkurensi**: Penanganan transfer antar-rekening yang berjalan simultan tidak boleh menghasilkan *race condition* atau saldo negatif yang tidak sah.
3. **Auditability (Tamper-Evident)**: Setiap baris mutasi buku besar dihubungkan dengan rantai kriptografis SHA-256 sehingga manipulasi data manual di database dapat langsung terdeteksi.
4. **Idempotensi API**: Kanal ingress wajib menolak atau mengembalikan respons terduplikasi secara aman jika klien mengirimkan request ulang yang sama akibat kendala jaringan.
5. **Performa & Latensi**:
   * Throughput transaksi mutasi target: $\ge 2.500 \text{ TPS}$.
   * Latensi 99th percentile (p99): $\le 20 \text{ ms}$.
