package clones

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/util"
)

type Service struct {
	store            persistence.Store
	factory          telegram.Factory
	publicWebhookURL string
	logger           *slog.Logger
}

func New(store persistence.Store, factory telegram.Factory, publicWebhookURL string, logger *slog.Logger) *Service {
	return &Service{
		store:            store,
		factory:          factory,
		publicWebhookURL: strings.TrimRight(publicWebhookURL, "/"),
		logger:           logger,
	}
}

func (s *Service) Handle(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "clone":
		return true, s.clone(ctx, rt)
	case "clones":
		return true, s.list(ctx, rt)
	case "rmclone":
		return true, s.remove(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) ensureOperator(rt *runtime.Context) error {
	if !rt.ActorPermissions.IsOwner && !rt.ActorPermissions.IsSudo {
		return fmt.Errorf("owner or sudo permission required")
	}
	return nil
}

func (s *Service) clone(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureOperator(rt); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /clone <bot_token> or /clone sync <clone>")
	}
	if strings.EqualFold(rt.Command.Args[0], "sync") {
		return s.sync(ctx, rt)
	}
	if s.publicWebhookURL == "" {
		return fmt.Errorf("PUBLIC_WEBHOOK_BASE_URL is required for clone creation")
	}

	token := strings.TrimSpace(rt.Command.Args[0])
	tempBot := domain.BotInstance{
		ID:            util.RandomID(16),
		Slug:          "pending-clone",
		DisplayName:   "Pending Clone",
		TelegramToken: token,
	}
	client := s.factory.ForBot(tempBot)
	me, err := client.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("validate clone token: %w", err)
	}
	if strings.TrimSpace(me.Username) == "" {
		return fmt.Errorf("clone bot username is required")
	}

	clone := domain.BotInstance{
		ID:            util.RandomID(16),
		Slug:          strings.ToLower(me.Username),
		DisplayName:   strings.TrimSpace(strings.TrimSpace(me.FirstName + " " + me.LastName)),
		TelegramToken: token,
		WebhookKey:    util.RandomID(24),
		WebhookSecret: util.RandomID(24),
		Username:      me.Username,
		IsPrimary:     false,
	}
	if clone.DisplayName == "" {
		clone.DisplayName = me.Username
	}
	clone, err = s.store.CreateCloneBot(ctx, clone, rt.ActorID())
	if err != nil {
		return err
	}

	if err := client.SetWebhook(ctx, telegram.SetWebhookOptions{
		URL:         fmt.Sprintf("%s/webhook/%s", s.publicWebhookURL, clone.WebhookKey),
		SecretToken: clone.WebhookSecret,
	}); err != nil {
		_ = s.store.DeleteBotInstance(ctx, clone.ID)
		return fmt.Errorf("set clone webhook: %w", err)
	}

	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Clone created: @%s", clone.Username), telegram.SendMessageOptions{})
	return err
}

func (s *Service) sync(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureOperator(rt); err != nil {
		return err
	}
	if s.publicWebhookURL == "" {
		return fmt.Errorf("PUBLIC_WEBHOOK_BASE_URL is required for clone sync")
	}
	if len(rt.Command.Args) < 2 {
		return fmt.Errorf("usage: /clone sync <clone>")
	}
	clone, err := s.resolveOwnedClone(ctx, rt.ActorID(), rt.Command.Args[1])
	if err != nil {
		return err
	}
	client := s.factory.ForBot(clone)
	if err := client.SetWebhook(ctx, telegram.SetWebhookOptions{
		URL:         fmt.Sprintf("%s/webhook/%s", s.publicWebhookURL, clone.WebhookKey),
		SecretToken: clone.WebhookSecret,
	}); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Webhook synced for @%s", clone.Username), telegram.SendMessageOptions{})
	return err
}

func (s *Service) list(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureOperator(rt); err != nil {
		return err
	}
	bots, err := s.store.ListOwnedBots(ctx, rt.ActorID())
	if err != nil {
		return err
	}
	lines := make([]string, 0, len(bots))
	for _, bot := range bots {
		if bot.IsPrimary {
			continue
		}
		lines = append(lines, fmt.Sprintf("@%s slug=%s status=%s", bot.Username, bot.Slug, bot.Status))
	}
	if len(lines) == 0 {
		lines = append(lines, "No clones found.")
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), telegram.SendMessageOptions{})
	return err
}

func (s *Service) remove(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureOperator(rt); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /rmclone <clone>")
	}
	clone, err := s.resolveOwnedClone(ctx, rt.ActorID(), rt.Command.Args[0])
	if err != nil {
		return err
	}
	client := s.factory.ForBot(clone)
	if err := client.DeleteWebhook(ctx); err != nil {
		s.logger.Warn("clone webhook delete failed", "bot_id", clone.ID, "error", err)
	}
	if err := s.store.DeleteBotInstance(ctx, clone.ID); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Clone removed: @%s", clone.Username), telegram.SendMessageOptions{})
	return err
}

func (s *Service) resolveOwnedClone(ctx context.Context, ownerUserID int64, ref string) (domain.BotInstance, error) {
	bots, err := s.store.ListOwnedBots(ctx, ownerUserID)
	if err != nil {
		return domain.BotInstance{}, err
	}
	ref = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ref)), "@")
	for _, bot := range bots {
		if bot.IsPrimary {
			continue
		}
		if strings.ToLower(bot.Username) == ref || strings.ToLower(bot.Slug) == ref || strings.ToLower(bot.ID) == ref {
			return bot, nil
		}
	}
	return domain.BotInstance{}, fmt.Errorf("clone not found")
}
