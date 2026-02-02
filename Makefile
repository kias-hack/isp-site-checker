toolgen:
	go build -o mgrctl cmd/mgrctl-test/main.go

run:
	go run cmd/app/main.go -config config/config.toml -debug

test:
	go test -v ./...