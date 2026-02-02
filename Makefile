toolgen:
	go build -o mgrctl cmd/mgrctl-test/main.go

run:
	go run cmd/app/main.go -config config/config.toml -debug

test:
	go test -v ./...

mockgen:
	mockgen -source=internal/notify/notifier.go -destination=internal/notify/mock_notifier.go -package=notify

generate:
	go generate ./...

# Установка mockgen (требует доступ к сети)
install-mockgen:
	GOPROXY=https://proxy.golang.org,direct go install go.uber.org/mock/mockgen@latest