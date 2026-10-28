# Testing Guide & Quality Assurance: AethelCore Engine

This document defines testing standards, the Arrange-Act-Assert (AAA) pattern for unit and integration tests, interface mocking rules, concurrency load simulation, and automated test runners in the CI/CD pipeline.

---

## 1. Banking Testing Philosophy

In banking software, software defects are not mere cosmetic issues; they represent direct financial loss and regulatory non-compliance. Testing must guarantee:

1. **Zero Math Errors**: Zero penny-rounding discrepancy under any calculation scenario.
2. **Concurrency Determinism**: Zero instances of race conditions where concurrent goroutines overdraw an account.
3. **Absolute Accounting Invariants**: $\sum \text{Debit} - \sum \text{Credit} = 0$ in every single transaction mutation.

---

## 2. Unit Test Structure: Arrange-Act-Assert (AAA) Pattern

Every Go test function must follow three isolated phases:

```go
func TestTransferUsecase_InsufficientBalance(t *testing.T) {
    // 1. ARRANGE (Setup initial state, dummy data, and repository mocks)
    mockRepo := new(MockLedgerRepo)
    usecase := NewTransferUsecase(mockRepo)

    sourceAcc := &domain.Account{
        ID:      "acc_001",
        Balance: 500000, // Balance is only IDR 500,000
        Status:  "ACTIVE",
    }
    mockRepo.On("GetByID", "acc_001").Return(sourceAcc, nil)

    // 2. ACT (Execute use case with an amount exceeding available balance)
    err := usecase.ExecuteTransfer(context.Background(), TransferCmd{
        SourceID:      "acc_001",
        DestinationID: "acc_002",
        Amount:        1000000, // Attempting to transfer IDR 1,000,000
    })

    // 3. ASSERT (Verify outcome: must fail with ErrInsufficientFunds, balance untouched)
    assert.Error(t, err)
    assert.Equal(t, domain.ErrInsufficientFunds, err)
    assert.Equal(t, int64(500000), sourceAcc.Balance)
}
```

---

## 3. Concurrency Testing & Race Condition Detection

To verify concurrency resilience, integration tests are executed with Go's race detector:

```bash
go test -race -v ./...
```

### 3.1 Concurrency Stress Test Simulation (500 Goroutines)

Tests simulate 500 parallel goroutines concurrently attempting to debit IDR 10,000 from an account holding only IDR 100,000.

- **Expectation**: Exactly the first 10 goroutines succeed; the remaining 490 goroutines receive `Insufficient Funds` errors. The final account balance must remain exactly **IDR 0** (no negative balances allowed).

---

## 4. CI/CD Runner Configuration (GitHub Actions)

The `.github/workflows/ci-pipeline.yml` workflow automatically runs the complete test suite on every git push:

```yaml
name: AethelCore CI Pipeline

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  test-and-lint:
    name: Run Tests & Linters
    runs-on: ubuntu-latest

    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_DATABASE: aethel_bank_test
          MYSQL_ROOT_PASSWORD: test_root_password
        ports:
          - 3306:3306
        options: >-
          --health-cmd="mysqladmin ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Setup Go Runtime
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"
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

This automated pipeline provides automated quality gating, ensuring code correctness before pull requests can be merged.
