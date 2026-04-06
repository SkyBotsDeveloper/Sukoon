package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/observability"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/processor"
	"sukoon/bot-core/internal/service/captcha"
)

type Service struct {
	store         persistence.Store
	processor     *processor.Processor
	jobs          *jobs.Service
	captcha       *captcha.Service
	concurrency   int
	pollInterval  time.Duration
	logger        *slog.Logger
	maxRetryCount int
	metrics       observability.Metrics
}

func New(store persistence.Store, processor *processor.Processor, jobsService *jobs.Service, captcha *captcha.Service, concurrency int, pollInterval time.Duration, logger *slog.Logger) *Service {
	return NewWithMetrics(store, processor, jobsService, captcha, concurrency, pollInterval, logger, observability.NewNoopMetrics())
}

func NewWithMetrics(store persistence.Store, processor *processor.Processor, jobsService *jobs.Service, captcha *captcha.Service, concurrency int, pollInterval time.Duration, logger *slog.Logger, metrics observability.Metrics) *Service {
	if concurrency < 1 {
		concurrency = 1
	}
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	return &Service{
		store:         store,
		processor:     processor,
		jobs:          jobsService,
		captcha:       captcha,
		concurrency:   concurrency,
		pollInterval:  pollInterval,
		logger:        logger,
		maxRetryCount: 5,
		metrics:       metrics,
	}
}

func (s *Service) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, s.concurrency+1)

	for i := 0; i < s.concurrency; i++ {
		workerID := fmt.Sprintf("worker-%d", i+1)
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if err := s.runWorker(ctx, id); err != nil && ctx.Err() == nil {
				errCh <- err
			}
		}(workerID)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.captcha.SweepExpired(ctx); err != nil {
					s.logger.Error("captcha sweep failed", "error", err)
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		wg.Wait()
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Service) runWorker(ctx context.Context, workerID string) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		didWork, err := s.processUpdates(ctx, workerID)
		if err != nil {
			return err
		}
		jobWork, err := s.processJobs(ctx, workerID)
		if err != nil {
			return err
		}
		if !didWork && !jobWork {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(s.pollInterval):
				continue
			}
		}
	}
}

func (s *Service) processUpdates(ctx context.Context, workerID string) (bool, error) {
	updates, err := s.store.ClaimPendingUpdates(ctx, workerID, 1)
	if err != nil {
		return false, err
	}
	if len(updates) == 0 {
		return false, nil
	}

	for _, update := range updates {
		if err := s.processor.Process(ctx, update); err != nil {
			s.logger.Error("update processing failed", "update_id", update.UpdateID, "queue_id", update.ID, "error", err)
			if update.Attempts >= s.maxRetryCount {
				s.logger.Error("update moved to dead-letter", "update_id", update.UpdateID, "queue_id", update.ID, "attempts", update.Attempts, "error", err)
				s.metrics.IncCounter("update_dead_total", "bot_id", update.BotID)
				if deadErr := s.store.MarkUpdateDead(ctx, update.ID, err.Error()); deadErr != nil {
					return true, deadErr
				}
				continue
			}
			backoff := time.Duration(update.Attempts*update.Attempts) * time.Second
			if backoff < time.Second {
				backoff = time.Second
			}
			s.logger.Warn("update scheduled for retry", "update_id", update.UpdateID, "queue_id", update.ID, "attempts", update.Attempts, "backoff_ms", backoff.Milliseconds(), "error", err)
			s.metrics.IncCounter("update_retries_total", "bot_id", update.BotID)
			if retryErr := s.store.MarkUpdateRetry(ctx, update.ID, update.Attempts, err.Error(), time.Now().Add(backoff)); retryErr != nil {
				return true, retryErr
			}
			continue
		}
		if err := s.store.MarkUpdateCompleted(ctx, update.ID); err != nil {
			return true, err
		}
	}
	return true, nil
}

func (s *Service) processJobs(ctx context.Context, workerID string) (bool, error) {
	if s.jobs == nil {
		return false, nil
	}
	jobsToRun, err := s.store.ClaimPendingJobs(ctx, workerID, 1)
	if err != nil {
		return false, err
	}
	if len(jobsToRun) == 0 {
		return false, nil
	}

	for _, job := range jobsToRun {
		s.logger.Info("job started", "job_id", job.ID, "kind", job.Kind, "attempt", job.Attempts)
		summary, err := s.jobs.Process(ctx, job)
		if err != nil {
			s.logger.Error("job processing failed", "job_id", job.ID, "kind", job.Kind, "attempt", job.Attempts, "error", err)
			maxAttempts := job.MaxAttempts
			if maxAttempts < 1 {
				maxAttempts = s.maxRetryCount
			}
			if job.Attempts >= maxAttempts {
				s.logger.Error("job moved to dead-letter", "job_id", job.ID, "kind", job.Kind, "attempts", job.Attempts, "error", err)
				s.metrics.IncCounter("job_dead_total", "bot_id", job.BotID, "kind", job.Kind)
				if deadErr := s.store.MarkJobDead(ctx, job.ID, err.Error()); deadErr != nil {
					return true, deadErr
				}
				s.jobs.NotifyResult(ctx, job, "failed permanently", err.Error())
				continue
			}
			backoff := time.Duration(job.Attempts*job.Attempts) * time.Second
			if backoff < time.Second {
				backoff = time.Second
			}
			s.logger.Warn("job scheduled for retry", "job_id", job.ID, "kind", job.Kind, "attempts", job.Attempts, "backoff_ms", backoff.Milliseconds(), "error", err)
			s.metrics.IncCounter("job_retries_total", "bot_id", job.BotID, "kind", job.Kind)
			if retryErr := s.store.MarkJobRetry(ctx, job.ID, job.Attempts, err.Error(), time.Now().Add(backoff)); retryErr != nil {
				return true, retryErr
			}
			continue
		}
		if err := s.store.MarkJobCompleted(ctx, job.ID, summary); err != nil {
			return true, err
		}
		s.metrics.IncCounter("jobs_completed_total", "bot_id", job.BotID, "kind", job.Kind)
		s.jobs.NotifyResult(ctx, job, "completed", summary)
	}
	return true, nil
}
