package checker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/kias-hack/isp-site-checker/internal/notify"
)

func resultHandler(ctx context.Context, wg *sync.WaitGroup, resultPipe <-chan *Task, notifier notify.Notifier) {
	defer wg.Done()

	for {
		select {
		case task := <-resultPipe:
			logger := slog.With("component", "resultHandler", "site", task.Site, "owner", task.Owner)

			if task.Result.Err == nil && task.Result.StatusCode == http.StatusUnauthorized {
				notifier.Success(task.Site, fmt.Sprintf("Сайт %s закрыт - %d\r\nВладелец - %s", task.Site, task.Result.StatusCode, task.Owner))
				logger.Debug("result received, site closed")
				continue
			}

			logger.Debug("result received, processing")

			msg := strings.Builder{}

			msg.WriteString("Проверка домена выявила проблему\n")
			msg.WriteString(fmt.Sprintf("Сайт: %s\n", task.Site))
			msg.WriteString(fmt.Sprintf("Владелец: %s\n", task.Owner))
			msg.WriteString(fmt.Sprintf("Время: %s\n", task.Result.Timestamp))

			if task.Result.Err != nil {
				logger.Debug("error in result", "err", task.Result.Err)
				msg.WriteString(fmt.Sprintf("Произошла ошибка: %s", task.Result.Err.Error()))
			} else {
				logger.Debug("invalid status in result", "status_code", task.Result.StatusCode)
				msg.WriteString(fmt.Sprintf("Код ответа: %d", task.Result.StatusCode))
			}

			notifier.Fail(task.Site, msg.String())
		case <-ctx.Done():
			return
		}
	}
}
