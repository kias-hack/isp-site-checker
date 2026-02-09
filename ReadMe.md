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
email = "user@example.ru"
password = "password"
host = "smtp.example.ru"
port = "587"

[email]
from = "Служба проверки доменов <user@example.ru>"
to = ["receiver@example.ru"]
subject = "Тема письма"
```

- **smtp.email** — полный адрес для авторизации на SMTP (для Яндекса и др. обязателен формат user@domain).
- **smtp.host** — необязателен: если не указан, подставляется MX-хост домена из `email` (например smtp.yandex.ru для @yandex.ru).
- **send_timeout** — таймаут одной попытки отправки письма. **send_interval** должен быть больше **send_timeout** минимум на 2 секунды.

## TODO

### Баги
- Обработать дублирование поддоменов (когда поддомен создан отдельно в панели)
- Обработать поддомены при поддоменности через внутреннюю папку