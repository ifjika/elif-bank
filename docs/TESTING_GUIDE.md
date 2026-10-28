# Testing Guide & Quality Assurance: AethelCore Engine

Dokumen ini mendefinisikan standar pengujian kode (*testing standards*), pola penulisan tes unit/integrasi dengan pola AAA (Arrange-Act-Assert), aturan mocking interface, simulasi beban konkurensi, dan otomatisasi test runner di pipeline CI/CD.

---

## 1. Filosofi Pengujian Perbankan

Di sistem perbankan, *bug* bukan sekadar masalah tampilan, melainkan potensi kerugian finansial riil atau pelanggaran hukum perbankan. Oleh karena itu, pengujian harus menjamin:
1. **Zero Math Errors**: Tidak boleh ada pembulatan saldo yang hilang satu rupiah pun.
2. **Determinisme Konkurensi**: Tidak boleh ada kondisi di mana dua goroutine mendebit rekening yang sama dan lolos melebihi batas saldo (*race condition*).
3. **Invarian Akuntansi Mutlak**: $\sum \text{Debit} - \sum \text{Kredit} = 0$ dalam setiap operasi transaksi.

---

## 2. Struktur Tes Unit: Pola Arrange-Act-Assert (AAA)

Setiap fungsi tes di Golang wajib ditulis dengan pola tiga tahap yang terisolasi:

```go
func TestTransferUsecase_InsufficientBalance(t *testing.T) {
    // 1. ARRANGE (Siapkan kondisi awal, data tiruan, dan mock)
    mockRepo := new(MockLedgerRepo)
    usecase := NewTransferUsecase(mockRepo)
    
    sourceAcc := &domain.Account{
        ID:      "acc_001",
        Balance: 500000, // Saldo hanya Rp 500.000
        Status:  "ACTIVE",
    }
    mockRepo.On("GetByID", "acc_001").Return(sourceAcc, nil)

    // 2. ACT (Jalankan fungsi yang diuji dengan nominal yang melebihi saldo)
    err := usecase.ExecuteTransfer(context.Background(), TransferCmd{
        SourceID:      "acc_001",
        DestinationID: "acc_002",
        Amount:        1000000, // Mencoba transfer Rp 1.000.000
    })

    // 3. ASSERT (Pastikan hasil sesuai ekspektasi: harus error dan saldo tidak berubah)
    assert.Error(t, err)
    assert.Equal(t, domain.ErrInsufficientFunds, err)
    assert.Equal(t, int64(500000), sourceAcc.Balance)
}
```

---

## 3. Pengujian Konkurensi & Deteksi Race Condition

Untuk membuktikan keandalan sistem terhadap *race condition*, tes integrasi dijalankan menggunakan flag bawaan Go:
```bash
go test -race -v ./...
```

### 3.1 Skrip Simulasi Stress Test Konkurensi (500 Goroutines)
Pengujian dijalankan dengan membuat 500 goroutine paralel yang secara bersamaan mencoba mendebit Rp 10.000 dari rekening yang hanya memiliki saldo Rp 100.000.
* **Ekspektasi**: Tepat 10 goroutine pertama berhasil, dan 490 goroutine sisanya harus menerima pesan penolakan `Insufficient Funds`. Saldo akhir wajib tepat **Rp 0** (tidak boleh minus).

---

## 4. Konfigurasi CI/CD Runner (GitHub Actions)

File `.github/workflows/ci-pipeline.yml` akan secara otomatis menjalankan seluruh rangkaian tes setiap kali ada kode baru yang di-*push* ke repository:

```yaml
name: AethelCore CI Pipeline

on:
  push:
    branches: [ main, master ]
  pull_request:
    branches: [ main, master ]

jobs:
  test-and-lint:
    name: Run Tests & Linters
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: aethel_core_test
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_password
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Setup Go Runtime
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true

      - name: Run GolangCI-Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.60

      - name: Run Unit & Race Tests
        run: |
          go test -race -covermode=atomic -coverprofile=coverage.out ./...

      - name: Verify Code Coverage
        run: |
          go tool cover -func=coverage.out
```

Dengan pipeline ini, kredensial dan profesionalitas kode Anda terjamin karena setiap baris kode diuji secara otomatis di cloud GitHub secara gratis.
