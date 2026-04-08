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
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/util"
)

const (
	defaultCaptchaButtonText = "Click here to prove you're human"
	failedCaptchaBanWindow   = 10 * time.Minute
)

var captchaTextWords = []string{
	"shield",
	"signal",
	"bridge",
	"forest",
	"planet",
	"rocket",
	"sunrise",
	"anchor",
}

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
	case "start":
		raw := strings.TrimSpace(rt.Command.RawArgs)
		if len(raw) < len("captcha_") || !strings.EqualFold(raw[:len("captcha_")], "captcha_") {
			return false, nil
		}
		return true, s.startChallenge(ctx, rt, raw[len("captcha_"):])
	case "captcha":
		return s.handleCaptcha(ctx, rt)
	case "captchamode":
		return true, s.captchaMode(ctx, rt)
	case "captcharules":
		return true, s.captchaRules(ctx, rt)
	case "captchamutetime":
		return true, s.captchaMuteTime(ctx, rt)
	case "captchakick":
		return true, s.captchaKick(ctx, rt)
	case "captchakicktime":
		return true, s.captchaKickTime(ctx, rt)
	case "setcaptchatext":
		return true, s.setCaptchaText(ctx, rt)
	case "resetcaptchatext":
		return true, s.resetCaptchaText(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) handleCaptcha(ctx context.Context, rt *runtime.Context) (bool, error) {
	if !rt.ActorPermissions.IsChatAdmin {
		return true, fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), captchaStatusText(settings), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return true, err
	}

	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return true, err
	}
	if enabled && !rt.RuntimeBundle.Settings.WelcomeEnabled {
		return true, fmt.Errorf("enable welcome messages before turning captcha on")
	}
	settings.Enabled = enabled
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return true, err
	}
	text := "Captcha disabled."
	if enabled {
		text = "Captcha enabled."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
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
	switch mode {
	case "button", "math", "text", "text2":
	default:
		return fmt.Errorf("captcha mode must be button, math, text, or text2")
	}
	settings.Mode = mode
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha mode set to "+mode+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) captchaRules(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha rules are "+onOff(settings.RulesRequired)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	if enabled && strings.TrimSpace(rt.RuntimeBundle.Settings.RulesText) == "" {
		return fmt.Errorf("set rules first before enabling captcha rules")
	}
	settings.RulesRequired = enabled
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha rules "+toggleWord(enabled)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) captchaMuteTime(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	if len(rt.Command.Args) == 0 {
		value := "off"
		if settings.AutoUnmuteSeconds > 0 {
			value = humanizeCaptchaDuration(time.Duration(settings.AutoUnmuteSeconds) * time.Second)
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha mute time is "+value+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	arg := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	if arg == "off" || arg == "no" {
		settings.AutoUnmuteSeconds = 0
		if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha mute time disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	duration, err := parseCaptchaDuration(rt.Command.Args[0])
	if err != nil || duration < time.Minute {
		return fmt.Errorf("usage: /captchamutetime <Xw/d/h/m> or off")
	}
	settings.AutoUnmuteSeconds = int(duration.Seconds())
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha mute time set to "+humanizeCaptchaDuration(duration)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) captchaKick(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha kicks are "+onOff(settings.KickOnTimeout)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	settings.KickOnTimeout = enabled
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha kicks "+toggleWord(enabled)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) captchaKickTime(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha kick time is "+humanizeCaptchaDuration(time.Duration(settings.TimeoutSeconds)*time.Second)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	duration, err := parseCaptchaDuration(rt.Command.Args[0])
	if err != nil || duration < 5*time.Minute || duration > 24*time.Hour {
		return fmt.Errorf("captcha kick time must be between 5m and 1d")
	}
	settings.TimeoutSeconds = int(duration.Seconds())
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha kick time set to "+humanizeCaptchaDuration(duration)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setCaptchaText(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	text := strings.TrimSpace(rt.Command.RawArgs)
	if err := validateCaptchaButtonText(text); err != nil {
		return err
	}
	settings := normalizedSettings(rt)
	settings.ButtonText = text
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha button text updated.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) resetCaptchaText(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizedSettings(rt)
	settings.ButtonText = defaultCaptchaButtonText
	if err := rt.Store.SetCaptchaSettings(ctx, settings); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Captcha button text reset.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) startChallenge(ctx context.Context, rt *runtime.Context, challengeID string) error {
	if !isPrivateChat(rt) {
		return fmt.Errorf("open the bot in PM to solve this captcha")
	}
	challenge, err := s.store.GetCaptchaChallengeByID(ctx, strings.TrimSpace(challengeID))
	if err != nil || challenge.Status != "pending" {
		return fmt.Errorf("captcha not found")
	}
	if challenge.UserID != rt.ActorID() {
		return fmt.Errorf("this captcha is not for you")
	}
	if time.Now().After(challenge.ExpiresAt) && challenge.TimeoutAction != "none" {
		return fmt.Errorf("captcha expired")
	}
	if challenge.Mode == "button" {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Open the group and press the captcha button there.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}

	if challenge.RulesRequired && !challenge.RulesAccepted {
		text, markup, err := s.rulesPrompt(ctx, rt, challenge)
		if err != nil {
			return err
		}
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{ReplyMarkup: markup}))
		return err
	}

	text, markup := s.challengePrompt(rt.Bot.Username, challenge)
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{ReplyMarkup: markup}))
	return err
}

func (s *Service) HandleJoin(ctx context.Context, rt *runtime.Context, user telegram.User) (bool, error) {
	settings := normalizedSettings(rt)
	if !settings.Enabled || !rt.RuntimeBundle.Settings.WelcomeEnabled {
		return false, nil
	}

	restrictUntil := captchaRestrictionUntil(settings)
	if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), user.ID, telegram.RestrictPermissions{CanSendMessages: false}, restrictUntil); err != nil {
		return true, err
	}

	challengeID := util.RandomID(12)
	solvePrompt, answer := generateCaptchaPrompt(settings.Mode)
	timeoutAction, expiresAt := captchaTimeout(settings)
	messageText, markup := captchaJoinMessage(rt, user, settings, challengeID)
	msg, err := rt.Client.SendMessage(ctx, rt.ChatID(), messageText, telegram.SendMessageOptions{
		ReplyMarkup: markup,
	})
	if err != nil {
		return true, err
	}

	if err := rt.Store.CreateCaptchaChallenge(ctx, domain.CaptchaChallenge{
		ID:            challengeID,
		BotID:         rt.Bot.ID,
		ChatID:        rt.ChatID(),
		UserID:        user.ID,
		Prompt:        solvePrompt,
		Answer:        answer,
		MessageID:     msg.MessageID,
		ExpiresAt:     expiresAt,
		Status:        "pending",
		Mode:          settings.Mode,
		RulesRequired: settings.RulesRequired,
		RulesAccepted: false,
		TimeoutAction: timeoutAction,
		FailureAction: settings.FailureAction,
	}); err != nil {
		return true, err
	}
	return true, nil
}

func (s *Service) HandleCallback(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.CallbackQuery == nil || !strings.HasPrefix(rt.CallbackQuery.Data, "captcha:") {
		return false, nil
	}
	parts := strings.SplitN(rt.CallbackQuery.Data, ":", 4)
	if len(parts) < 3 {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Malformed captcha.", true)
	}

	action := parts[1]
	challengeID := parts[2]
	answer := ""
	if len(parts) == 4 {
		answer = parts[3]
	}

	challenge, err := s.store.GetCaptchaChallengeByID(ctx, challengeID)
	if err != nil || challenge.Status != "pending" {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Captcha not found.", true)
	}
	if challenge.UserID != rt.CallbackQuery.From.ID {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "This captcha isn't for you.", true)
	}
	if time.Now().After(challenge.ExpiresAt) && challenge.TimeoutAction != "none" {
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Captcha expired.", true)
	}

	switch action {
	case "button":
		return true, s.handleButtonCallback(ctx, rt, challenge)
	case "rules":
		return true, s.handleRulesCallback(ctx, rt, challenge)
	case "answer":
		return true, s.handleAnswerCallback(ctx, rt, challenge, answer)
	default:
		return true, rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Malformed captcha.", true)
	}
}

func (s *Service) handleButtonCallback(ctx context.Context, rt *runtime.Context, challenge domain.CaptchaChallenge) error {
	if challenge.Mode != "button" {
		return rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Open PM to continue.", true)
	}
	if challenge.RulesRequired && !challenge.RulesAccepted {
		text, markup, err := s.rulesPrompt(ctx, rt, challenge)
		if err != nil {
			return err
		}
		if err := s.editCallbackMessage(ctx, rt, text, markup); err != nil {
			return err
		}
		return rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Accept the rules to continue.", false)
	}
	return s.completeChallenge(ctx, rt, challenge)
}

func (s *Service) handleRulesCallback(ctx context.Context, rt *runtime.Context, challenge domain.CaptchaChallenge) error {
	if challenge.RulesRequired && !challenge.RulesAccepted {
		if err := s.store.MarkCaptchaRulesAccepted(ctx, challenge.ID); err != nil {
			return err
		}
		challenge.RulesAccepted = true
	}

	if challenge.Mode == "button" {
		return s.completeChallenge(ctx, rt, challenge)
	}

	text, markup := s.challengePrompt(rt.Bot.Username, challenge)
	if err := s.editCallbackMessage(ctx, rt, text, markup); err != nil {
		return err
	}
	return rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Rules accepted.", false)
}

func (s *Service) handleAnswerCallback(ctx context.Context, rt *runtime.Context, challenge domain.CaptchaChallenge, answer string) error {
	if challenge.RulesRequired && !challenge.RulesAccepted {
		return rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Accept the rules first.", true)
	}
	if strings.TrimSpace(answer) == "" {
		return rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Malformed captcha.", true)
	}
	if answer != challenge.Answer {
		return s.failChallenge(ctx, rt, challenge)
	}
	return s.completeChallenge(ctx, rt, challenge)
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
		switch challenge.TimeoutAction {
		case "kick":
			until := time.Now().Add(30 * time.Second)
			if err := client.BanChatMember(ctx, challenge.ChatID, challenge.UserID, &until, true); err != nil {
				s.logger.Error("captcha kick failed", "challenge_id", challenge.ID, "error", err)
				continue
			}
			if err := client.UnbanChatMember(ctx, challenge.ChatID, challenge.UserID, true); err != nil {
				s.logger.Error("captcha unban after kick failed", "challenge_id", challenge.ID, "error", err)
				continue
			}
		case "unmute":
			if err := client.RestrictChatMember(ctx, challenge.ChatID, challenge.UserID, telegram.RestrictPermissions{CanSendMessages: true}, nil); err != nil {
				s.logger.Error("captcha auto-unmute failed", "challenge_id", challenge.ID, "error", err)
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

func (s *Service) rulesPrompt(ctx context.Context, rt *runtime.Context, challenge domain.CaptchaChallenge) (string, *telegram.InlineKeyboardMarkup, error) {
	bundle, err := rt.Store.LoadRuntimeBundle(ctx, rt.Bot.ID, challenge.ChatID)
	if err != nil {
		return "", nil, err
	}
	chat := telegram.Chat{ID: challenge.ChatID, Title: "this chat"}
	if rt.CallbackQuery != nil && rt.CallbackQuery.Message != nil && rt.CallbackQuery.Message.Chat.ID == challenge.ChatID {
		chat = rt.CallbackQuery.Message.Chat
	} else if fetched, err := rt.Client.GetChat(ctx, challenge.ChatID); err == nil {
		chat = fetched
	}

	user := telegram.User{ID: challenge.UserID, FirstName: "there"}
	if rt.CallbackQuery != nil && rt.CallbackQuery.From.ID == challenge.UserID {
		user = rt.CallbackQuery.From
	} else if rt.Message != nil && rt.Message.From != nil && rt.Message.From.ID == challenge.UserID {
		user = *rt.Message.From
	}

	lines := []string{"Accept the rules before you can speak."}
	renderedRules := strings.TrimSpace(serviceutil.RenderStoredMessage(bundle.Settings.RulesText, user, chat, bundle.Settings.RulesText))
	if renderedRules != "" {
		lines = append(lines, "", renderedRules)
	}
	return strings.Join(lines, "\n"), serviceutil.Markup(
		[]telegram.InlineKeyboardButton{
			{Text: "Accept Rules", CallbackData: fmt.Sprintf("captcha:rules:%s", challenge.ID)},
		},
	), nil
}

func (s *Service) challengePrompt(username string, challenge domain.CaptchaChallenge) (string, *telegram.InlineKeyboardMarkup) {
	text := strings.Join([]string{
		"Complete the captcha to finish verification.",
		"",
		challenge.Prompt,
	}, "\n")
	return text, challengeMarkup(username, challenge)
}

func (s *Service) completeChallenge(ctx context.Context, rt *runtime.Context, challenge domain.CaptchaChallenge) error {
	if err := s.store.MarkCaptchaSolved(ctx, challenge.ID); err != nil {
		return err
	}
	if err := rt.Client.RestrictChatMember(ctx, challenge.ChatID, challenge.UserID, telegram.RestrictPermissions{CanSendMessages: true}, nil); err != nil {
		return err
	}
	if challenge.MessageID != 0 {
		_ = rt.Client.DeleteMessage(ctx, challenge.ChatID, challenge.MessageID)
	}
	if rt.CallbackQuery != nil && rt.CallbackQuery.Message != nil && rt.CallbackQuery.Message.Chat.ID != challenge.ChatID {
		if err := rt.Client.EditMessageText(ctx, rt.CallbackQuery.Message.Chat.ID, rt.CallbackQuery.Message.MessageID, "Verification complete. You can return to the group now.", telegram.EditMessageTextOptions{}); err != nil && !strings.Contains(strings.ToLower(err.Error()), "message is not modified") {
			return err
		}
	}
	if rt.CallbackQuery != nil {
		return rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Verification complete.", false)
	}
	return nil
}

func (s *Service) failChallenge(ctx context.Context, rt *runtime.Context, challenge domain.CaptchaChallenge) error {
	until := time.Now().Add(failedCaptchaBanWindow)
	if err := rt.Client.BanChatMember(ctx, challenge.ChatID, challenge.UserID, &until, true); err != nil {
		return err
	}
	if challenge.MessageID != 0 {
		_ = rt.Client.DeleteMessage(ctx, challenge.ChatID, challenge.MessageID)
	}
	if err := s.store.MarkCaptchaExpired(ctx, challenge.ID); err != nil {
		return err
	}
	if rt.CallbackQuery != nil && rt.CallbackQuery.Message != nil && rt.CallbackQuery.Message.Chat.ID != challenge.ChatID {
		if err := rt.Client.EditMessageText(ctx, rt.CallbackQuery.Message.Chat.ID, rt.CallbackQuery.Message.MessageID, "Verification failed. Rejoin and try again.", telegram.EditMessageTextOptions{}); err != nil && !strings.Contains(strings.ToLower(err.Error()), "message is not modified") {
			return err
		}
	}
	return rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "Wrong captcha.", true)
}

func (s *Service) editCallbackMessage(ctx context.Context, rt *runtime.Context, text string, markup *telegram.InlineKeyboardMarkup) error {
	if rt.CallbackQuery == nil || rt.CallbackQuery.Message == nil {
		return nil
	}
	err := rt.Client.EditMessageText(ctx, rt.CallbackQuery.Message.Chat.ID, rt.CallbackQuery.Message.MessageID, text, telegram.EditMessageTextOptions{
		ReplyMarkup: markup,
	})
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "message is not modified") {
		return nil
	}
	return err
}

func captchaStatusText(settings domain.CaptchaSettings) string {
	lines := []string{
		"Captcha settings:",
		fmt.Sprintf("- Enabled: %s", onOff(settings.Enabled)),
		fmt.Sprintf("- Mode: %s", settings.Mode),
		fmt.Sprintf("- Rules required: %s", onOff(settings.RulesRequired)),
		fmt.Sprintf("- Auto-unmute: %s", formatOptionalCaptchaTime(settings.AutoUnmuteSeconds)),
		fmt.Sprintf("- Kick unsolved users: %s", onOff(settings.KickOnTimeout)),
		fmt.Sprintf("- Kick after: %s", humanizeCaptchaDuration(time.Duration(settings.TimeoutSeconds)*time.Second)),
		fmt.Sprintf("- Button text: %s", settings.ButtonText),
	}
	return strings.Join(lines, "\n")
}

func captchaJoinMessage(rt *runtime.Context, user telegram.User, settings domain.CaptchaSettings, challengeID string) (string, *telegram.InlineKeyboardMarkup) {
	base := fmt.Sprintf("Welcome %s.", serviceutil.DisplayName(user))
	if strings.TrimSpace(rt.RuntimeBundle.Settings.WelcomeText) != "" && rt.Message != nil {
		base = serviceutil.RenderStoredMessage(rt.RuntimeBundle.Settings.WelcomeText, user, rt.Message.Chat, rt.RuntimeBundle.Settings.RulesText)
	}

	instructions := []string{}
	if settings.Mode == "button" {
		instructions = append(instructions, "Press the button below to prove you're human.")
	} else {
		instructions = append(instructions, "Open PM and complete the captcha before you can speak.")
	}
	if settings.RulesRequired {
		instructions = append(instructions, "You'll need to accept the chat rules first.")
	}
	switch {
	case settings.KickOnTimeout:
		instructions = append(instructions, fmt.Sprintf("If you ignore it for %s, you'll be kicked.", humanizeCaptchaDuration(time.Duration(settings.TimeoutSeconds)*time.Second)))
	case settings.AutoUnmuteSeconds > 0:
		instructions = append(instructions, fmt.Sprintf("If you don't solve it, you'll be automatically unmuted after %s.", humanizeCaptchaDuration(time.Duration(settings.AutoUnmuteSeconds)*time.Second)))
	}

	button := telegram.InlineKeyboardButton{Text: settings.ButtonText}
	if settings.Mode == "button" {
		button.CallbackData = fmt.Sprintf("captcha:button:%s", challengeID)
	} else {
		button.URL = serviceutil.BotDeepLink(rt.Bot.Username, "captcha_"+challengeID)
	}

	return strings.Join([]string{base, "", strings.Join(instructions, " ")}, "\n"), serviceutil.Markup(
		[]telegram.InlineKeyboardButton{button},
	)
}

func challengeMarkup(username string, challenge domain.CaptchaChallenge) *telegram.InlineKeyboardMarkup {
	switch challenge.Mode {
	case "math":
		return numericChallengeMarkup(challenge.ID, challenge.Answer)
	case "text":
		return optionChallengeMarkup(challenge.ID, challenge.Answer, wordOptions(challenge.Answer))
	case "text2":
		return optionChallengeMarkup(challenge.ID, challenge.Answer, codeOptions(challenge.Answer))
	default:
		return serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Open the group", URL: serviceutil.BotURL(username)},
			},
		)
	}
}

func numericChallengeMarkup(challengeID string, answer string) *telegram.InlineKeyboardMarkup {
	value, _ := strconv.Atoi(answer)
	options := []string{
		strconv.Itoa(value),
		strconv.Itoa(value + 1),
		strconv.Itoa(maxInt(1, value-1)),
		strconv.Itoa(value + 2),
	}
	return optionChallengeMarkup(challengeID, answer, uniqueOptions(options, answer))
}

func optionChallengeMarkup(challengeID string, answer string, options []string) *telegram.InlineKeyboardMarkup {
	rows := make([][]telegram.InlineKeyboardButton, 0, 2)
	row := make([]telegram.InlineKeyboardButton, 0, 2)
	for idx, option := range options {
		row = append(row, telegram.InlineKeyboardButton{
			Text:         option,
			CallbackData: fmt.Sprintf("captcha:answer:%s:%s", challengeID, option),
		})
		if len(row) == 2 || idx == len(options)-1 {
			rows = append(rows, row)
			row = make([]telegram.InlineKeyboardButton, 0, 2)
		}
	}
	return serviceutil.Markup(rows...)
}

func generateCaptchaPrompt(mode string) (string, string) {
	switch mode {
	case "math":
		left := rand.IntN(9) + 1
		right := rand.IntN(9) + 1
		return fmt.Sprintf("Solve: %d + %d", left, right), strconv.Itoa(left + right)
	case "text":
		answer := captchaTextWords[rand.IntN(len(captchaTextWords))]
		return "Select the matching word: " + answer, answer
	case "text2":
		answer := randomCaptchaCode(4)
		return "Pick the exact code: " + answer, answer
	default:
		return "Press the button to verify yourself.", "button"
	}
}

func wordOptions(answer string) []string {
	options := []string{answer}
	for _, word := range captchaTextWords {
		if word == answer {
			continue
		}
		options = append(options, word)
		if len(options) == 4 {
			break
		}
	}
	return uniqueOptions(options, answer)
}

func codeOptions(answer string) []string {
	options := []string{answer}
	for len(options) < 4 {
		options = append(options, mutateCaptchaCode(answer))
	}
	return uniqueOptions(options, answer)
}

func uniqueOptions(options []string, answer string) []string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option == "" {
			continue
		}
		if _, ok := seen[option]; ok {
			continue
		}
		seen[option] = struct{}{}
		ordered = append(ordered, option)
	}
	if _, ok := seen[answer]; !ok {
		ordered = append(ordered, answer)
	}
	rand.Shuffle(len(ordered), func(i, j int) {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	})
	if len(ordered) > 4 {
		ordered = ordered[:4]
	}
	return ordered
}

func randomCaptchaCode(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return b.String()
}

func mutateCaptchaCode(answer string) string {
	if answer == "" {
		return randomCaptchaCode(4)
	}
	raw := []byte(answer)
	idx := rand.IntN(len(raw))
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	raw[idx] = alphabet[rand.IntN(len(alphabet))]
	mutated := string(raw)
	if mutated == answer {
		raw[idx] = alphabet[(rand.IntN(len(alphabet)-1)+1)%len(alphabet)]
		mutated = string(raw)
	}
	return mutated
}

func captchaRestrictionUntil(settings domain.CaptchaSettings) *time.Time {
	if settings.KickOnTimeout || settings.AutoUnmuteSeconds <= 0 {
		return nil
	}
	until := time.Now().Add(time.Duration(settings.AutoUnmuteSeconds) * time.Second)
	return &until
}

func captchaTimeout(settings domain.CaptchaSettings) (string, time.Time) {
	switch {
	case settings.KickOnTimeout:
		return "kick", time.Now().Add(time.Duration(settings.TimeoutSeconds) * time.Second)
	case settings.AutoUnmuteSeconds > 0:
		return "unmute", time.Now().Add(time.Duration(settings.AutoUnmuteSeconds) * time.Second)
	default:
		return "none", time.Now().Add(100 * 365 * 24 * time.Hour)
	}
}

func validateCaptchaButtonText(value string) error {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return fmt.Errorf("usage: /setcaptchatext <text>")
	case strings.Contains(value, "\n"):
		return fmt.Errorf("captcha button text must be plain single-line text")
	case strings.Contains(value, "{") || strings.Contains(value, "}"):
		return fmt.Errorf("captcha button text cannot use template variables")
	case len([]rune(value)) > 64:
		return fmt.Errorf("captcha button text must be 64 characters or fewer")
	default:
		return nil
	}
}

func parseToggle(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "enable", "enabled", "yes":
		return true, nil
	case "off", "disable", "disabled", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected on or off")
	}
}

func parseCaptchaDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, fmt.Errorf("duration is required")
	}
	if duration, err := time.ParseDuration(value); err == nil {
		return duration, nil
	}
	if len(value) < 2 {
		return 0, fmt.Errorf("invalid duration")
	}
	unit := value[len(value)-1]
	amount, err := strconv.Atoi(value[:len(value)-1])
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("invalid duration")
	}
	switch unit {
	case 'm':
		return time.Duration(amount) * time.Minute, nil
	case 'h':
		return time.Duration(amount) * time.Hour, nil
	case 'd':
		return time.Duration(amount) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(amount) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid duration")
	}
}

func humanizeCaptchaDuration(duration time.Duration) string {
	if duration <= 0 {
		return "off"
	}
	switch {
	case duration%(7*24*time.Hour) == 0:
		return fmt.Sprintf("%dw", int(duration/(7*24*time.Hour)))
	case duration%(24*time.Hour) == 0:
		return fmt.Sprintf("%dd", int(duration/(24*time.Hour)))
	case duration%time.Hour == 0:
		return fmt.Sprintf("%dh", int(duration/time.Hour))
	case duration%time.Minute == 0:
		return fmt.Sprintf("%dm", int(duration/time.Minute))
	default:
		return fmt.Sprintf("%ds", int(duration/time.Second))
	}
}

func formatOptionalCaptchaTime(seconds int) string {
	if seconds <= 0 {
		return "off"
	}
	return humanizeCaptchaDuration(time.Duration(seconds) * time.Second)
}

func normalizedSettings(rt *runtime.Context) domain.CaptchaSettings {
	settings := rt.RuntimeBundle.Captcha
	settings.BotID = rt.Bot.ID
	settings.ChatID = rt.ChatID()
	if settings.Mode == "" {
		settings.Mode = "button"
	}
	switch settings.Mode {
	case "button", "math", "text", "text2":
	default:
		settings.Mode = "button"
	}
	if settings.TimeoutSeconds == 0 {
		settings.TimeoutSeconds = 5 * 60
	}
	if settings.ButtonText == "" {
		settings.ButtonText = defaultCaptchaButtonText
	}
	if settings.FailureAction == "" {
		settings.FailureAction = "kick"
	}
	if settings.ChallengeDigits == 0 {
		settings.ChallengeDigits = 2
	}
	return settings
}

func toggleWord(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func isPrivateChat(rt *runtime.Context) bool {
	return rt.Message != nil && rt.Message.Chat.Type == "private"
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
