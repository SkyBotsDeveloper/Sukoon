package clones

import (
	"context"
	"errors"
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

const (
	cloneCallbackPrefix        = "clone:"
	cloneCallbackSelectPrefix  = cloneCallbackPrefix + "select:"
	cloneCallbackRestartPrefix = cloneCallbackPrefix + "restart:"
	cloneCallbackDeletePrefix  = cloneCallbackPrefix + "delete:"
	cloneCallbackConfirmPrefix = cloneCallbackPrefix + "confirm:"
	cloneCallbackBack          = cloneCallbackPrefix + "back"
	cloneCallbackClose         = cloneCallbackPrefix + "close"
)

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
	case "mybot", "mybots":
		return true, s.myBot(ctx, rt)
	case "rmclone":
		return true, s.remove(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) HandleCallback(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.CallbackQuery == nil || !strings.HasPrefix(rt.CallbackQuery.Data, cloneCallbackPrefix) {
		return false, nil
	}

	var err error
	if err = s.ensureOperator(rt); err == nil {
		switch data := rt.CallbackQuery.Data; {
		case data == cloneCallbackBack:
			err = s.editMyBotList(ctx, rt, "")
		case data == cloneCallbackClose:
			err = s.closeCallbackMessage(ctx, rt)
		case strings.HasPrefix(data, cloneCallbackSelectPrefix):
			err = s.editCloneActions(ctx, rt, strings.TrimPrefix(data, cloneCallbackSelectPrefix), "")
		case strings.HasPrefix(data, cloneCallbackRestartPrefix):
			err = s.restartClone(ctx, rt, strings.TrimPrefix(data, cloneCallbackRestartPrefix))
		case strings.HasPrefix(data, cloneCallbackDeletePrefix):
			err = s.editCloneDeleteConfirm(ctx, rt, strings.TrimPrefix(data, cloneCallbackDeletePrefix))
		case strings.HasPrefix(data, cloneCallbackConfirmPrefix):
			err = s.deleteCloneFromCallback(ctx, rt, strings.TrimPrefix(data, cloneCallbackConfirmPrefix))
		default:
			return false, nil
		}
	}

	alertText := ""
	showAlert := false
	if err != nil {
		alertText = callbackErrorText(err)
		showAlert = true
	}
	if ackErr := rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, alertText, showAlert); ackErr != nil && err == nil {
		err = ackErr
	}
	return true, err
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
	if limited, err := s.prepareCloneSlot(ctx, rt); err != nil {
		return err
	} else if limited {
		return nil
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
		if errors.Is(err, persistence.ErrCloneLimitReached) {
			_, sendErr := rt.Client.SendMessage(ctx, rt.ChatID(), cloneLimitText(), telegram.SendMessageOptions{})
			if sendErr != nil {
				return sendErr
			}
			return nil
		}
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

func (s *Service) myBot(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureOperator(rt); err != nil {
		return err
	}
	return s.sendMyBotList(ctx, rt)
}

func (s *Service) prepareCloneSlot(ctx context.Context, rt *runtime.Context) (bool, error) {
	bots, err := s.store.ListOwnedBots(ctx, rt.ActorID())
	if err != nil {
		return false, err
	}
	for _, bot := range bots {
		if bot.IsPrimary || bot.Status != "active" {
			continue
		}
		if _, err := s.factory.ForBot(bot).GetMe(ctx); err != nil {
			s.logger.Warn("removing stale clone after token validation failed", "bot_id", bot.ID, "owner_user_id", rt.ActorID(), "error", err)
			if delErr := s.store.DeleteBotInstance(ctx, bot.ID); delErr != nil {
				return false, delErr
			}
			continue
		}
		_, sendErr := rt.Client.SendMessage(ctx, rt.ChatID(), cloneLimitText(), telegram.SendMessageOptions{})
		if sendErr != nil {
			return false, sendErr
		}
		return true, nil
	}
	return false, nil
}

func cloneLimitText() string {
	return "Only one Sukoon clone is allowed per account. Remove your existing clone with /rmclone before creating another."
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

func (s *Service) sendMyBotList(ctx context.Context, rt *runtime.Context) error {
	clones, err := s.listOwnedClones(ctx, rt.ActorID())
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), myBotListText(clones, ""), rt.ReplyOptions(telegram.SendMessageOptions{
		ReplyMarkup: myBotListMarkup(clones),
	}))
	return err
}

func (s *Service) editMyBotList(ctx context.Context, rt *runtime.Context, notice string) error {
	clones, err := s.listOwnedClones(ctx, rt.ActorID())
	if err != nil {
		return err
	}
	return s.editCallbackMessage(ctx, rt, myBotListText(clones, notice), myBotListMarkup(clones))
}

func (s *Service) editCloneActions(ctx context.Context, rt *runtime.Context, ref string, notice string) error {
	clone, err := s.resolveOwnedClone(ctx, rt.ActorID(), ref)
	if err != nil {
		return err
	}
	return s.editCallbackMessage(ctx, rt, cloneActionsText(clone, notice), cloneActionsMarkup(clone))
}

func (s *Service) restartClone(ctx context.Context, rt *runtime.Context, ref string) error {
	if s.publicWebhookURL == "" {
		return fmt.Errorf("clone restart is unavailable right now")
	}
	clone, err := s.resolveOwnedClone(ctx, rt.ActorID(), ref)
	if err != nil {
		return err
	}
	client := s.factory.ForBot(clone)
	if _, err := client.GetMe(ctx); err != nil {
		return fmt.Errorf("clone token validation failed; remove or replace this clone")
	}
	if err := client.SetWebhook(ctx, telegram.SetWebhookOptions{
		URL:         fmt.Sprintf("%s/webhook/%s", s.publicWebhookURL, clone.WebhookKey),
		SecretToken: clone.WebhookSecret,
	}); err != nil {
		return err
	}
	return s.editCloneActions(ctx, rt, ref, "Clone restarted successfully.")
}

func (s *Service) editCloneDeleteConfirm(ctx context.Context, rt *runtime.Context, ref string) error {
	clone, err := s.resolveOwnedClone(ctx, rt.ActorID(), ref)
	if err != nil {
		return err
	}
	return s.editCallbackMessage(ctx, rt, cloneDeleteConfirmText(clone), cloneDeleteConfirmMarkup(clone))
}

func (s *Service) deleteCloneFromCallback(ctx context.Context, rt *runtime.Context, ref string) error {
	clone, err := s.resolveOwnedClone(ctx, rt.ActorID(), ref)
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
	return s.editMyBotList(ctx, rt, fmt.Sprintf("Removed @%s.", clone.Username))
}

func (s *Service) listOwnedClones(ctx context.Context, ownerUserID int64) ([]domain.BotInstance, error) {
	bots, err := s.store.ListOwnedBots(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}
	clones := make([]domain.BotInstance, 0, len(bots))
	for _, bot := range bots {
		if bot.IsPrimary {
			continue
		}
		clones = append(clones, bot)
	}
	return clones, nil
}

func myBotListText(clones []domain.BotInstance, notice string) string {
	lines := []string{"Your Sukoon Bots", ""}
	if strings.TrimSpace(notice) != "" {
		lines = append(lines, notice, "")
	}
	if len(clones) == 0 {
		lines = append(lines, "No Sukoon clone is linked to your account right now.")
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "Select a bot below to manage it.")
	return strings.Join(lines, "\n")
}

func myBotListMarkup(clones []domain.BotInstance) *telegram.InlineKeyboardMarkup {
	rows := make([][]telegram.InlineKeyboardButton, 0, len(clones)+1)
	for _, clone := range clones {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: "@" + clone.Username, CallbackData: cloneCallbackSelectPrefix + clone.ID},
		})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: "Close", CallbackData: cloneCallbackClose},
	})
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func cloneActionsText(clone domain.BotInstance, notice string) string {
	lines := []string{
		"Manage Your Sukoon Clone",
		"",
		fmt.Sprintf("Bot: @%s", clone.Username),
		fmt.Sprintf("Status: %s", clone.Status),
	}
	if strings.TrimSpace(notice) != "" {
		lines = append(lines, "", notice)
	}
	lines = append(lines, "", "Choose what you want to do with this clone.")
	return strings.Join(lines, "\n")
}

func cloneActionsMarkup(clone domain.BotInstance) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "Restart", CallbackData: cloneCallbackRestartPrefix + clone.ID},
				{Text: "Delete", CallbackData: cloneCallbackDeletePrefix + clone.ID},
			},
			{
				{Text: "Back", CallbackData: cloneCallbackBack},
				{Text: "Close", CallbackData: cloneCallbackClose},
			},
		},
	}
}

func cloneDeleteConfirmText(clone domain.BotInstance) string {
	return strings.Join([]string{
		"Delete This Clone?",
		"",
		fmt.Sprintf("Bot: @%s", clone.Username),
		"",
		"This removes the clone from Sukoon and deletes its webhook.",
	}, "\n")
}

func cloneDeleteConfirmMarkup(clone domain.BotInstance) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "Confirm Delete", CallbackData: cloneCallbackConfirmPrefix + clone.ID},
			},
			{
				{Text: "Back", CallbackData: cloneCallbackSelectPrefix + clone.ID},
				{Text: "Close", CallbackData: cloneCallbackClose},
			},
		},
	}
}

func (s *Service) editCallbackMessage(ctx context.Context, rt *runtime.Context, text string, markup *telegram.InlineKeyboardMarkup) error {
	if rt.CallbackQuery == nil || rt.CallbackQuery.Message == nil {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{ReplyMarkup: markup})
		return err
	}
	err := rt.Client.EditMessageText(ctx, rt.ChatID(), rt.CallbackQuery.Message.MessageID, text, telegram.EditMessageTextOptions{
		ReplyMarkup: markup,
	})
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "message is not modified") {
		return nil
	}
	return err
}

func (s *Service) closeCallbackMessage(ctx context.Context, rt *runtime.Context) error {
	if rt.CallbackQuery == nil || rt.CallbackQuery.Message == nil {
		return nil
	}
	return rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.CallbackQuery.Message.MessageID)
}

func callbackErrorText(err error) string {
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "Action failed."
	}
	if len(text) > 180 {
		return text[:177] + "..."
	}
	return text
}
