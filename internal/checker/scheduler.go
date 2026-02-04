package checker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kias-hack/isp-site-checker/internal/isp"
)

func sсheduler(ctx context.Context, wg *sync.WaitGroup, ticker <-chan struct{}, taskPipe chan<- *Task, getDomains isp.GetWebDomainsFunc) {
	defer wg.Done()

	for {
		select {
		case <-ticker:
			slog.Debug("начинаю проверку доменов, получаю список доменов", "component", "scheduler")

			domains, err := getDomains()
			if err != nil {
				slog.Error("при получении списка доменов из ISPManager произошла ошибка", "err", err, "component", "scheduler")
				continue
			}

			for _, domainInfo := range domains {
				logger := slog.With("component", "scheduler", "name", domainInfo.Name, "owner", domainInfo.Owner)

				for _, site := range domainInfo.Sites {
					logger.Debug("отправлена задача на обработку", "site", site)

					taskPipe <- &Task{
						DomainId:   domainInfo.Id,
						DomainName: domainInfo.Name,
						Owner:      domainInfo.Owner,
						Site:       site,
						Connection: struct {
							Addr string
							Port string
						}{
							Port: domainInfo.Port,
							Addr: domainInfo.IPAddr,
						},
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
