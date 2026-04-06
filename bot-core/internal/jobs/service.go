package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/util"
)

const (
	KindBroadcast       = "broadcast"
	KindPurge           = "purge"
	KindGlobalBan       = "gban"
	KindGlobalUnban     = "ungban"
	KindFederationBan   = "fban"
	KindFederationUnban = "unfban"
)

type Service struct {
	store   persistence.Store
	factory telegram.Factory
	logger  *slog.Logger
}

type BroadcastPayload struct {
	Mode string `json:"mode"`
	Text string `json:"text"`
}

type PurgePayload struct {
	ChatID        int64 `json:"chat_id"`
	FromMessageID int64 `json:"from_message_id"`
	ToMessageID   int64 `json:"to_message_id"`
}

type GlobalBanPayload struct {
	UserID int64  `json:"user_id"`
	Reason string `json:"reason"`
}

type FederationBanPayload struct {
	FederationID string `json:"federation_id"`
	UserID       int64  `json:"user_id"`
	Reason       string `json:"reason"`
}

func New(store persistence.Store, factory telegram.Factory, logger *slog.Logger) *Service {
	return &Service{
		store:   store,
		factory: factory,
		logger:  logger,
	}
}

func (s *Service) Enqueue(ctx context.Context, botID string, kind string, requestedBy int64, reportChatID int64, payload any, total int) (domain.Job, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.Job{}, err
	}
	job := domain.Job{
		ID:           util.RandomID(18),
		BotID:        botID,
		Kind:         kind,
		Status:       "pending",
		RequestedBy:  requestedBy,
		ReportChatID: reportChatID,
		PayloadJSON:  body,
		Total:        total,
		MaxAttempts:  5,
		AvailableAt:  time.Now(),
	}
	if err := s.store.CreateJob(ctx, job); err != nil {
		return domain.Job{}, err
	}
	return job, nil
}

func (s *Service) Process(ctx context.Context, job domain.Job) (string, error) {
	switch job.Kind {
	case KindBroadcast:
		return s.processBroadcast(ctx, job)
	case KindPurge:
		return s.processPurge(ctx, job)
	case KindGlobalBan:
		return s.processGlobalBan(ctx, job, true)
	case KindGlobalUnban:
		return s.processGlobalBan(ctx, job, false)
	case KindFederationBan:
		return s.processFederationBan(ctx, job, true)
	case KindFederationUnban:
		return s.processFederationBan(ctx, job, false)
	default:
		return "", fmt.Errorf("unsupported job kind %s", job.Kind)
	}
}

func (s *Service) NotifyQueued(ctx context.Context, bot domain.BotInstance, chatID int64, job domain.Job, description string) {
	if chatID == 0 {
		return
	}
	client := s.factory.ForBot(bot)
	_, _ = client.SendMessage(ctx, chatID, fmt.Sprintf("%s queued. Job ID: `%s`", description, job.ID), telegram.SendMessageOptions{
		ParseMode: "Markdown",
	})
}

func (s *Service) NotifyResult(ctx context.Context, job domain.Job, status string, summary string) {
	if job.ReportChatID == 0 {
		return
	}
	bot, err := s.store.GetBotByID(ctx, job.BotID)
	if err != nil {
		s.logger.Error("job result bot lookup failed", "job_id", job.ID, "error", err)
		return
	}
	client := s.factory.ForBot(bot)
	text := fmt.Sprintf("Job `%s` %s.\n%s", job.ID, status, summary)
	_, _ = client.SendMessage(ctx, job.ReportChatID, text, telegram.SendMessageOptions{ParseMode: "Markdown"})
}

func (s *Service) processBroadcast(ctx context.Context, job domain.Job) (string, error) {
	var payload BroadcastPayload
	if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
		return "", fmt.Errorf("decode broadcast payload: %w", err)
	}
	if strings.TrimSpace(payload.Text) == "" {
		return "", fmt.Errorf("broadcast text is required")
	}
	bot, err := s.store.GetBotByID(ctx, job.BotID)
	if err != nil {
		return "", err
	}
	client := s.factory.ForBot(bot)
	chats, err := s.store.ListChats(ctx, job.BotID)
	if err != nil {
		return "", err
	}
	if len(chats) == 0 {
		return "No chats are currently registered for this bot.", nil
	}

	total := len(chats)
	if err := s.store.UpdateJobProgress(ctx, job.ID, "processing", 0, total, ""); err != nil {
		return "", err
	}

	var sent, failed int
	var failures []string
	for idx, chat := range chats {
		if _, err := client.SendMessage(ctx, chat.ID, payload.Text, telegram.SendMessageOptions{}); err != nil {
			failed++
			failures = append(failures, fmt.Sprintf("%d:%v", chat.ID, err))
		} else {
			sent++
		}
		if shouldFlushProgress(idx+1, total) {
			errText := truncateFailures(failures)
			if err := s.store.UpdateJobProgress(ctx, job.ID, "processing", idx+1, total, errText); err != nil {
				return "", err
			}
		}
	}
	return fmt.Sprintf("Broadcast finished. sent=%d failed=%d total=%d", sent, failed, total), nil
}

func (s *Service) processPurge(ctx context.Context, job domain.Job) (string, error) {
	var payload PurgePayload
	if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
		return "", fmt.Errorf("decode purge payload: %w", err)
	}
	if payload.ChatID == 0 || payload.FromMessageID == 0 || payload.ToMessageID == 0 {
		return "", fmt.Errorf("purge payload is incomplete")
	}
	if payload.FromMessageID > payload.ToMessageID {
		payload.FromMessageID, payload.ToMessageID = payload.ToMessageID, payload.FromMessageID
	}

	bot, err := s.store.GetBotByID(ctx, job.BotID)
	if err != nil {
		return "", err
	}
	client := s.factory.ForBot(bot)
	total := int(payload.ToMessageID-payload.FromMessageID) + 1
	if total < 1 {
		return "Nothing to purge.", nil
	}
	if err := s.store.UpdateJobProgress(ctx, job.ID, "processing", 0, total, ""); err != nil {
		return "", err
	}

	var deleted, failed int
	var failures []string
	progress := 0
	for messageID := payload.FromMessageID; messageID <= payload.ToMessageID; messageID++ {
		progress++
		if err := client.DeleteMessage(ctx, payload.ChatID, messageID); err != nil {
			failed++
			failures = append(failures, fmt.Sprintf("%d:%v", messageID, err))
		} else {
			deleted++
		}
		if shouldFlushProgress(progress, total) {
			errText := truncateFailures(failures)
			if err := s.store.UpdateJobProgress(ctx, job.ID, "processing", progress, total, errText); err != nil {
				return "", err
			}
		}
	}
	return fmt.Sprintf("Purge finished. deleted=%d failed=%d total=%d", deleted, failed, total), nil
}

func (s *Service) processGlobalBan(ctx context.Context, job domain.Job, ban bool) (string, error) {
	var payload GlobalBanPayload
	if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
		return "", fmt.Errorf("decode global ban payload: %w", err)
	}
	bot, err := s.store.GetBotByID(ctx, job.BotID)
	if err != nil {
		return "", err
	}
	client := s.factory.ForBot(bot)
	chats, err := s.store.ListChats(ctx, job.BotID)
	if err != nil {
		return "", err
	}
	total := len(chats)
	if total == 0 {
		return "No chats are currently registered for this bot.", nil
	}
	if err := s.store.UpdateJobProgress(ctx, job.ID, "processing", 0, total, ""); err != nil {
		return "", err
	}

	var succeeded, failed int
	var failures []string
	for idx, chat := range chats {
		var opErr error
		if ban {
			opErr = client.BanChatMember(ctx, chat.ID, payload.UserID, nil, true)
		} else {
			opErr = client.UnbanChatMember(ctx, chat.ID, payload.UserID, false)
		}
		if opErr != nil {
			failed++
			failures = append(failures, fmt.Sprintf("%d:%v", chat.ID, opErr))
		} else {
			succeeded++
		}
		if shouldFlushProgress(idx+1, total) {
			errText := truncateFailures(failures)
			if err := s.store.UpdateJobProgress(ctx, job.ID, "processing", idx+1, total, errText); err != nil {
				return "", err
			}
		}
	}
	action := "Global ban"
	if !ban {
		action = "Global unban"
	}
	return fmt.Sprintf("%s finished. succeeded=%d failed=%d total=%d", action, succeeded, failed, total), nil
}

func (s *Service) processFederationBan(ctx context.Context, job domain.Job, ban bool) (string, error) {
	var payload FederationBanPayload
	if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
		return "", fmt.Errorf("decode federation payload: %w", err)
	}
	bot, err := s.store.GetBotByID(ctx, job.BotID)
	if err != nil {
		return "", err
	}
	client := s.factory.ForBot(bot)
	chats, err := s.store.ListFederationChats(ctx, payload.FederationID)
	if err != nil {
		return "", err
	}
	sort.Slice(chats, func(i, j int) bool { return chats[i].ID < chats[j].ID })
	total := len(chats)
	if total == 0 {
		return "Federation has no registered chats.", nil
	}
	if err := s.store.UpdateJobProgress(ctx, job.ID, "processing", 0, total, ""); err != nil {
		return "", err
	}

	var succeeded, failed int
	var failures []string
	for idx, chat := range chats {
		var opErr error
		if ban {
			opErr = client.BanChatMember(ctx, chat.ID, payload.UserID, nil, true)
		} else {
			opErr = client.UnbanChatMember(ctx, chat.ID, payload.UserID, false)
		}
		if opErr != nil {
			failed++
			failures = append(failures, fmt.Sprintf("%d:%v", chat.ID, opErr))
		} else {
			succeeded++
		}
		if shouldFlushProgress(idx+1, total) {
			errText := truncateFailures(failures)
			if err := s.store.UpdateJobProgress(ctx, job.ID, "processing", idx+1, total, errText); err != nil {
				return "", err
			}
		}
	}
	action := "Federation ban"
	if !ban {
		action = "Federation unban"
	}
	return fmt.Sprintf("%s finished. succeeded=%d failed=%d total=%d", action, succeeded, failed, total), nil
}

func shouldFlushProgress(progress int, total int) bool {
	if progress >= total {
		return true
	}
	return progress%10 == 0
}

func truncateFailures(failures []string) string {
	if len(failures) == 0 {
		return ""
	}
	if len(failures) > 5 {
		failures = failures[:5]
	}
	return strings.Join(failures, "; ")
}
