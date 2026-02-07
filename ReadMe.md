# ISP Site Checker

Приложение для мониторинга доступности веб-доменов из ISPManager.

## Описание

Сервис периодически проверяет доступность доменов и их поддоменов, отправляет уведомления на email при обнаружении что сайт открыт или HTTP ошибка. Список доменов получается из ISPManager через утилиту mgrctl. Проверка выполняется напрямую по IP-адресу, минуя DNS.

## Запуск

```bash
go run cmd/app/main.go -config config/config.toml [-debug]
```

## Конфигурация

```toml
mgrctl_path = "path/to/mgrctl"

scrape_interval = "10s"
send_interval = "10s"

[smtp]
username = "user1"
password = "password"
host = "smtp.example.ru"
port = "587"

[email]
from = "Служба проверки доменов <user1@example.ru>"
to = ["receiver@www.example.ru"]
subject = "Test"
```

## TODO

### Баги
- Обработать дублирование поддоменов (когда поддомен создан отдельно в панели)
- Обработать поддомены при поддоменности через внутреннюю папку