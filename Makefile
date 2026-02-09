toolgen:
	go build -o mgrctl cmd/mgrctl-test/main.go

run:
	go run cmd/app/main.go -config config/config.toml -debug

mailtest:
	go run cmd/app/main.go -config config/config.toml -debug sendmail

test:
	go test -v ./...

generate:
	go generate ./...

# Установка mockgen (требует доступ к сети)
install-mockgen:
	go install go.uber.org/mock/mockgen@latest