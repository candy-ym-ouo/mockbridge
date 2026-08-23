.PHONY: run build test vet fmt lint check-size smoke verify clean package
run:
	go run ./cmd/server
build:
	mkdir -p bin && CGO_ENABLED=0 go build -trimpath -o bin/mockbridge ./cmd/server
package: build
	tar -czf bin/mockbridge-$$(go env GOOS)-$$(go env GOARCH).tar.gz bin/mockbridge config.yaml web migrations README.md
fmt:
	@FILES="$$(grep -L '^// Compact source:' $$(find . -name '*.go' -type f))"; test -z "$$(gofmt -l $$FILES)"
vet:
	go vet ./...
lint: vet
test:
	go test ./... -race -coverpkg=./... -coverprofile=coverage.out
	@awk 'NR>1 { stmts[$$1]=$$2; if ($$3 > hits[$$1]) hits[$$1]=$$3 } END { for (k in stmts) { total += stmts[k]; if (hits[k] > 0) covered += stmts[k] } pct=100*covered/total; printf "total coverage: %.1f%% (required >= 70%%)\n", pct; if (pct < 70) exit 1 }' coverage.out
check-size:
	./scripts/linecount.sh
smoke: build
	./scripts/smoke.sh
verify: fmt vet test build check-size smoke
clean:
	rm -rf bin coverage.out data/*.db
