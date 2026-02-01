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
scrape_interval = "60s"
mgrctl_path = "/usr/local/mgr5/sbin/mgrctl"
recipient = "admin@example.com"

[smtp]
username = "user@example.com"
password = "password"
host = "smtp.example.com"
port = 465
from = "noreply@example.com"
```

## TODO

### Критичные
- Внедрить context в HTTP запросы к серверам с таймаутами
- Внедрить context в отправку email с таймаутами

### Улучшения
- Заменить отправку почты на интерфейс для упрощения тестирования
- Добавить операционные теги в логирование (op: "notification", op: "check")
- Реализовать ограничение частоты отправки уведомлений
- Отправлять уведомления о восстановлении доступности

### Баги
- Обработать дублирование поддоменов (когда поддомен создан отдельно в панели)