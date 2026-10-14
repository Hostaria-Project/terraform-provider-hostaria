BIN := terraform-provider-hostaria
MOCK_ADDR ?= 127.0.0.1:18080
TF ?= terraform

.PHONY: build test acc mock apply-local destroy-local apply-smoke destroy-smoke tidy

build:
	go build -o $(BIN) .

tidy:
	go mod tidy

test:
	go test ./...

acc: build
	TF_ACC=1 go test ./internal/provider -count=1 -timeout 10m -v

mock:
	go run ./cmd/mock-api -addr $(MOCK_ADDR)

apply-local: build
	cd examples/local && TF_CLI_CONFIG_FILE=$$PWD/dev.tfrc $(TF) apply -auto-approve

destroy-local:
	cd examples/local && TF_CLI_CONFIG_FILE=$$PWD/dev.tfrc $(TF) destroy -auto-approve

apply-smoke: build
	cd examples/smoke && TF_CLI_CONFIG_FILE=$$PWD/dev.tfrc $(TF) apply -auto-approve

destroy-smoke:
	cd examples/smoke && TF_CLI_CONFIG_FILE=$$PWD/dev.tfrc $(TF) destroy -auto-approve
