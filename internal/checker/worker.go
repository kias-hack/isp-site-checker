package checker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

func worker(ctx context.Context, wg *sync.WaitGroup, taskPipe <-chan *Task, resultPipe chan<- *Task, n int) {
	defer wg.Done()

	slog.Debug("worker started", "component", fmt.Sprintf("worker[%d]", n))

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-taskPipe:
			logger := slog.With("component", fmt.Sprintf("worker[%d]", n), "site", task.Site, "owner", task.Owner)

			logger.Debug("получена задача для обработки", "task", task)

			client := createClient(task.Connection.Addr, task.Connection.Port)
			defer client.CloseIdleConnections()

			url := fmt.Sprintf("http://%s/", task.Site)

			task.Result.Timestamp = time.Now()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				logger.Debug("произошла ошибка при создании запроса", "err", err)
				task.Result.Err = err
			} else {
				resp, err := client.Do(req)

				if err != nil {
					logger.Debug("произошла ошибка при подключении к серверу", "err", err)
					task.Result.Err = err
				} else {
					logger.Debug("получен статус от сервера", "status", resp.StatusCode)
					task.Result.StatusCode = resp.StatusCode
				}
			}

			resultPipe <- task
		}
	}
}
