package integration

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/models"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/repository"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Worker struct {
	client  *resty.Client
	baseURL string
}

func NewWorker(client *resty.Client, baseURL string) *Worker {
	return &Worker{
		client:  client,
		baseURL: baseURL,
	}
}

func (w *Worker) getAccrualResponse(ctx context.Context, number string) (*models.AccrualResponse, error) {
	var order models.AccrualResponse
	resp, err := w.client.R().
		SetContext(ctx).
		SetResult(&order).
		Get(w.baseURL + "/api/orders/" + number)

	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		return &order, nil
	}
	if resp.StatusCode() == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		delay := 30 * time.Second
		if retry, err := strconv.Atoi(resp.Header().Get("Retry-After")); err == nil && retry > 0 {
			delay = time.Duration(retry) * time.Second
		}
		return nil, &RateLimitError{RetryAfter: delay}
	}
	return nil, errors.New(resp.Status())
}

func (w *Worker) RunCollector(ctx context.Context, store repository.Storage, log *zap.SugaredLogger, jobs chan []string) {
	defer close(jobs)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			orders, err := store.GetProcessingNumbers(ctx)
			if err != nil {
				log.Errorw("some error happened when getting processing orders", "error", err)
				continue
			}
			select {
			case jobs <- orders:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (w *Worker) RunWorker(ctx context.Context, store repository.Storage, log *zap.SugaredLogger, jobs <-chan []string) {
	for job := range jobs {
		if len(job) == 0 {
			continue
		}
		for _, number := range job {
			var order *models.AccrualResponse
			order, err := w.getAccrualResponse(ctx, number)
			if rateLimit, ok := errors.AsType[*RateLimitError](err); ok {
				select {
				case <-time.After(rateLimit.RetryAfter):
				case <-ctx.Done():
					return
				}
				break
			}
			if err != nil {
				log.Errorw("ошибка отправки", "number", number, "error", err)
				continue
			}
			if order == nil {
				continue
			}

			err = store.UpdateOrderStatus(ctx, number, models.MapOrderStatus(order.Status), order.Accrual)
			if err != nil {
				log.Errorw("обновление статуса завершилось с ошибкой", "number", number, "error", err)
				continue
			}
		}
	}
}
