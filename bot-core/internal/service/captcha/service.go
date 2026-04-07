package captcha

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/util"
)

type Service struct {
	store   persistence.Store
	factory telegram.Factory
	logger  *slog.Logger
}

func New(store persistence.Store, factory telegram.Factory, logger *slog.Logger) *Service {
	return &Service{
		store:   store,
		factory: factory,
		logger:  logger,
	}
}

func (s *Service) HandleCommand(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "captcha":
		return s.handleCaptcha(ctx, rt)
	case "captchamode":
		return true, s.captchaMode(ctx, rt)
	case "captchakick":
		return true, s.captchaKick(ctx, rt)
	case "captchakicktime":
		return true, s.captchaKickTime(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) handleCaptcha(ctx context.Context, rt *runtime.Context) (bool, error) {
	if !rt.ActorPermissions.IsChatAdmin {
		return true, fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.Captcha.Enabled {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha is "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return true, err
	}

	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return true, err
	}
	settings := normalizedSettings(rt)
	settings.Enabled = enabled
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return true, err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha updated.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return true, err
}

func (s *Service) captchaMode(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha mode is "+settings.Mode+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	mode := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	if mode != "button" {
		return fmt.Errorf("only button captcha mode is supported")
	}
	settings.Mode = mode
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha mode set to button.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) captchaKick(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha failure action is "+settings.FailureAction+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	action := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	switch action {
	case "kick", "mute", "ban":
	default:
		return fmt.Errorf("captcha failure action must be kick, mute, or ban")
	}
	settings.FailureAction = action
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha failure action set to "+action+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) captchaKickTime(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Captcha timeout is %d seconds.", settings.TimeoutSeconds), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	seconds, err := strconv.Atoi(rt.Command.Args[0])
	if err != nil || seconds < 10 {
		return fmt.Errorf("captcha timeout must be at least 10 seconds")
	}
	settings.TimeoutSeconds = seconds
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Captcha timeout set to %d seconds.", seconds), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) HandleJoin(ctx context.Context, rt *runtime.Context, user telegram.User) error {
	if !rt.RuntimeBundle.Captcha.Enabled {
		return nil
	}

	if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), user.ID, telegram.RestrictPermissions{CanSendMessages: false}, nil); err != nil {
		return err
	}

	left := rand.IntN(9) + 1
	right := rand.IntN(9) + 1
	answer := left + right
	challengeID := util.RandomID(12)

	options := []int{answer, answer + 1, answer + 2, answer - 1}
	if options[3] <= 0 {
		options[3] = answer + 3
	}
	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	buttons := make([][]telegram.InlineKeyboardButton, 0, 2)
	row := make([]telegram.InlineKeyboardButton, 0, 2)
	for idx, option := range options {
		row = append(row, telegram.InlineKeyboardButton{
			Text:         strconv.Itoa(option),
			CallbackData: fmt.Sprintf("captcha:%s:%d", challengeID, option),
		})
		if len(row) == 2 || idx == len(options)-1 {
			buttons = append(buttons, row)
			row = make([]telegram.InlineKeyboardButton, 0, 2)
		}
	}

	prompt := fmt.Sprintf("Welcome %s. Tap the answer for %d + %d within %d seconds.", user.FirstName, left, right, rt.RuntimeBundle.Captcha.TimeoutSeconds)
	msg, err := rt.Client.SendMessage(ctx, rt.ChatID(), prompt, telegram.SendMessageOptions{
		ReplyMarkup: &telegram.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
	if err != nil {
		return err
	}

	return rt.Store.CreateCaptchaChallenge(ctx, domain.CaptchaChallenge{
		ID:            challengeID,
		BotID:         rt.Bot.ID,
		ChatID:        rt.ChatID(),
		UserID:        user.ID,
		Prompt:        prompt,
		Answer:        strconv.Itoa(answer),
		MessageID:     msg.MessageID,
		ExpiresAt:     time.Now().Add(time.Duration(rt.RuntimeBundle.Captcha.TimeoutSeconds) * time.Second),
		Status:        "pending",
		FailureAction: rt.RuntimeBundle.Captcha.FailureAction,
	})
}

func (s *Service) HandleCallback(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.CallbackQuery == nil || !strings.HasPrefix(rt.CallbackQuery.Data, "captcha:") {
		return false, nil
	}
	parts := strings.Split(rt.CallbackQuery.Data, ":")
	if len(parts) != 3 {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Malformed captcha.", true)
	}

	challenge, err := rt.Store.GetPendingCaptchaChallenge(ctx, rt.Bot.ID, rt.ChatID(), rt.CallbackQuery.From.ID)
	if err != nil {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Captcha not found.", true)
	}
	if challenge.ID != parts[1] {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Wrong captcha.", true)
	}
	if time.Now().After(challenge.ExpiresAt) {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Captcha expired.", true)
	}
	if parts[2] != challenge.Answer {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Wrong answer.", false)
	}

	if err := rt.Store.MarkCaptchaSolved(ctx, challenge.ID); err != nil {
		return true, err
	}
	if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), rt.CallbackQuery.From.ID, telegram.RestrictPermissions{CanSendMessages: true}, nil); err != nil {
		return true, err
	}
	if rt.CallbackQuery.Message != nil {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.CallbackQuery.Message.MessageID)
	}
	return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Verification complete.", false)
}

func (s *Service) SweepExpired(ctx context.Context) error {
	challenges, err := s.store.ListExpiredCaptchaChallenges(ctx, time.Now(), 100)
	if err != nil {
		return err
	}
	for _, challenge := range challenges {
		bot, err := s.store.GetBotByID(ctx, challenge.BotID)
		if err != nil {
			s.logger.Error("load bot for captcha sweep failed", "challenge_id", challenge.ID, "error", err)
			continue
		}
		client := s.factory.ForBot(bot)
		switch challenge.FailureAction {
		case "ban":
			if err := client.BanChatMember(ctx, challenge.ChatID, challenge.UserID, nil, true); err != nil {
				s.logger.Error("captcha ban failed", "challenge_id", challenge.ID, "error", err)
				continue
			}
		case "mute":
			if err := client.RestrictChatMember(ctx, challenge.ChatID, challenge.UserID, telegram.RestrictPermissions{CanSendMessages: false}, nil); err != nil {
				s.logger.Error("captcha mute failed", "challenge_id", challenge.ID, "error", err)
				continue
			}
		default:
			until := time.Now().Add(30 * time.Second)
			if err := client.BanChatMember(ctx, challenge.ChatID, challenge.UserID, &until, true); err != nil {
				s.logger.Error("captcha kick failed", "challenge_id", challenge.ID, "error", err)
				continue
			}
			if err := client.UnbanChatMember(ctx, challenge.ChatID, challenge.UserID, true); err != nil {
				s.logger.Error("captcha unban after kick failed", "challenge_id", challenge.ID, "error", err)
				continue
			}
		}
		if challenge.MessageID != 0 {
			_ = client.DeleteMessage(ctx, challenge.ChatID, challenge.MessageID)
		}
		_ = s.store.MarkCaptchaExpired(ctx, challenge.ID)
	}
	return nil
}

func parseToggle(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "enable", "enabled":
		return true, nil
	case "off", "disable", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("expected on or off")
	}
}

func normalizedSettings(rt *runtime.Context) domain.CaptchaSettings {
	settings := rt.RuntimeBundle.Captcha
	settings.BotID = rt.Bot.ID
	settings.ChatID = rt.ChatID()
	if settings.Mode == "" {
		settings.Mode = "button"
	}
	if settings.TimeoutSeconds == 0 {
		settings.TimeoutSeconds = 120
	}
	if settings.FailureAction == "" {
		settings.FailureAction = "kick"
	}
	if settings.ChallengeDigits == 0 {
		settings.ChallengeDigits = 2
	}
	return settings
}
