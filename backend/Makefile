# BalanzaPro3.0 backend. Integration tests need a disposable PostgreSQL:
#   export BP_TEST_DATABASE_URL=postgres://user@127.0.0.1:5432/bp_test?sslmode=disable
# Never point it at a real or production database: tests create and drop schemas.

.PHONY: fmt-check vet test test-unit test-integration build

fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

test-unit:
	go test -race -count=1 ./internal/platform/qty/... ./internal/inventory/domain/... ./internal/inventory/application/...

test-integration:
	@test -n "$$BP_TEST_DATABASE_URL" || (echo "BP_TEST_DATABASE_URL is required" && exit 1)
	BP_REQUIRE_DB=1 go test -race -count=1 ./...

test: fmt-check vet test-integration

build:
	go build -o bin/bpserver ./cmd/bpserver
