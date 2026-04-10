package utility

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"sukoon/bot-core/internal/commands"
	"sukoon/bot-core/internal/i18n"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

type Service struct{}

const donateImageURL = "https://files.catbox.moe/25hv2j.jpg"
const donateCaption = "Hey, thanks for wanting to donate! Sukoon is entirely run by volunteers, so this means a lot.\nWe accept only UPI as donation method."

func New() *Service {
	return &Service{}
}

func (s *Service) Handle(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "start":
		return true, s.start(ctx, rt)
	case "help":
		return true, s.help(ctx, rt)
	case "donate":
		return true, s.donate(ctx, rt)
	case "setlang", "language":
		return true, s.language(ctx, rt)
	case "privacy":
		return true, s.privacy(ctx, rt)
	case "mydata":
		return true, s.myData(ctx, rt)
	case "forgetme":
		return true, s.forgetMe(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) HandleCallback(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.CallbackQuery == nil || !strings.HasPrefix(rt.CallbackQuery.Data, "ux:") {
		return false, nil
	}

	fastAck := s.ShouldFastPathCallback(rt.CallbackQuery.Data)
	if fastAck {
		if err := rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "", false); err != nil {
			return true, err
		}
	}

	var err error
	switch rt.CallbackQuery.Data {
	case callbackStartHome:
		err = s.sendCallbackPageWithOptions(ctx, rt, startLandingText(), startLandingMarkup(rt.Bot.Username), "HTML", true)
	case callbackStartClone:
		err = s.sendCallbackPageWithOptions(ctx, rt, cloneLandingText(), cloneLandingMarkup(rt.Bot.Username), "HTML", false)
	case callbackHelpMain:
		err = s.sendHelpCallbackPage(ctx, rt, helpRoot)
	case callbackPrivacy:
		err = s.sendCallbackPageWithOptions(ctx, rt, privacyText(), privacyMarkup(rt.Bot.Username), "HTML", false)
	case callbackRulesShow:
		if strings.TrimSpace(rt.RuntimeBundle.Settings.RulesText) == "" {
			err = fmt.Errorf("rules are not set")
			break
		}
		chatTitle := ""
		chat := telegram.Chat{}
		if rt.CallbackQuery.Message != nil {
			chatTitle = rt.CallbackQuery.Message.Chat.Title
			chat = rt.CallbackQuery.Message.Chat
		}
		err = s.sendCallbackPage(ctx, rt, rulesText(chatTitle, serviceutil.RenderStoredMessage(rt.RuntimeBundle.Settings.RulesText, rt.CallbackQuery.From, chat, rt.RuntimeBundle.Settings.RulesText)), rulesShownHereMarkup(rt.Bot.Username, rt.ChatID()))
	case callbackClose:
		err = s.closeCallbackMessage(ctx, rt)
	default:
		section, ok := helpSectionFromCallback(rt.CallbackQuery.Data)
		if !ok {
			return false, nil
		}
		err = s.sendHelpCallbackPage(ctx, rt, section)
	}

	if !fastAck {
		if ackErr := rt.Client.AnswerCallbackQuery(ctx, rt.CallbackQuery.ID, "", false); ackErr != nil && err == nil {
			err = ackErr
		}
	}
	return true, err
}

func (s *Service) start(ctx context.Context, rt *runtime.Context) error {
	if !isPrivateChat(rt) {
		return s.sendPMGuidance(ctx, rt,
			"Sukoon is active in this group. Open PM for the full help menu, personal tools, and cleaner navigation.",
			"help_main",
		)
	}

	payload := strings.ToLower(strings.TrimSpace(rt.Command.RawArgs))
	switch {
	case payload == "", payload == "home":
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), startLandingText(), rt.ReplyOptions(telegram.SendMessageOptions{
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
			ReplyMarkup:           startLandingMarkup(rt.Bot.Username),
		}))
		return err
	case payload == "help", payload == "help_main":
		return s.sendHelpMessage(ctx, rt, helpRoot)
	case strings.HasPrefix(payload, "help_"):
		section := normalizeHelpSection(strings.TrimPrefix(payload, "help_"))
		if section == "" {
			return fmt.Errorf("unknown help section")
		}
		return s.sendHelpMessage(ctx, rt, section)
	case strings.HasPrefix(payload, "rules_"):
		return s.startRules(ctx, rt, strings.TrimPrefix(payload, "rules_"))
	case payload == "privacy":
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), privacyText(), rt.ReplyOptions(telegram.SendMessageOptions{
			ParseMode:   "HTML",
			ReplyMarkup: privacyMarkup(rt.Bot.Username),
		}))
		return err
	default:
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), startLandingText(), rt.ReplyOptions(telegram.SendMessageOptions{
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
			ReplyMarkup:           startLandingMarkup(rt.Bot.Username),
		}))
		return err
	}
}

func (s *Service) help(ctx context.Context, rt *runtime.Context) error {
	if !isPrivateChat(rt) {
		payload := "help_main"
		if len(rt.Command.Args) > 0 {
			if section := normalizeHelpSection(rt.Command.Args[0]); section != "" {
				payload = "help_" + section
			}
		}
		return s.sendPMGuidance(ctx, rt,
			"The full help menu is easier to browse in PM. Open Sukoon privately for category buttons and cleaner help pages.",
			payload,
		)
	}

	if len(rt.Command.Args) > 0 {
		section := normalizeHelpSection(rt.Command.Args[0])
		if section != "" {
			return s.sendHelpMessage(ctx, rt, section)
		}
	}

	return s.sendHelpMessage(ctx, rt, helpRoot)
}

func (s *Service) donate(ctx context.Context, rt *runtime.Context) error {
	options := telegram.SendPhotoOptions{}
	if rt.Message != nil {
		options.ReplyToMessageID = rt.Message.MessageID
	}
	options.Caption = donateCaption
	_, err := rt.Client.SendPhoto(ctx, rt.ChatID(), donateImageURL, options)
	return err
}

func (s *Service) language(ctx context.Context, rt *runtime.Context) error {
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "language.current", rt.RuntimeBundle.Settings.Language), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	language := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	if !i18n.IsSupported(language) {
		return fmt.Errorf("unsupported language. supported: %s", strings.Join(i18n.SupportedLanguages(), ", "))
	}
	if err := rt.Store.SetLanguage(ctx, rt.Bot.ID, rt.ChatID(), language); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(language, "language.updated", language), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) privacy(ctx context.Context, rt *runtime.Context) error {
	if !isPrivateChat(rt) {
		return s.sendPMGuidance(ctx, rt,
			"Privacy and personal data controls are better handled in PM.",
			"privacy",
		)
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), privacyText(), rt.ReplyOptions(telegram.SendMessageOptions{
		ParseMode:   "HTML",
		ReplyMarkup: privacyMarkup(rt.Bot.Username),
	}))
	return err
}

func (s *Service) myData(ctx context.Context, rt *runtime.Context) error {
	if !isPrivateChat(rt) {
		return s.sendPMGuidance(ctx, rt,
			"Use /mydata in PM so your exported data stays private.",
			"privacy",
		)
	}
	export, err := rt.Store.ExportUserData(ctx, rt.Bot.ID, rt.ActorID())
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "privacy.export", string(body)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) forgetMe(ctx context.Context, rt *runtime.Context) error {
	if !isPrivateChat(rt) {
		return s.sendPMGuidance(ctx, rt,
			"Use /forgetme in PM before deleting personal data for this bot.",
			"privacy",
		)
	}
	if len(rt.Command.Args) == 0 || !strings.EqualFold(rt.Command.Args[0], "confirm") {
		return fmt.Errorf("usage: /forgetme confirm")
	}
	if err := rt.Store.DeleteUserData(ctx, rt.Bot.ID, rt.ActorID()); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "privacy.deleted"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) sendPMGuidance(ctx context.Context, rt *runtime.Context, text string, payload string) error {
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{
		ReplyMarkup: pmGuidanceMarkup(rt.Bot.Username, payload),
	}))
	return err
}

func (s *Service) sendHelpMessage(ctx context.Context, rt *runtime.Context, section string) error {
	parseMode, disablePreview := helpSectionOptions(section)
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), helpPageText(section), rt.ReplyOptions(telegram.SendMessageOptions{
		ParseMode:             parseMode,
		DisableWebPagePreview: disablePreview,
		ReplyMarkup:           helpMarkup(section, rt.Bot.Username),
	}))
	return err
}

func (s *Service) sendHelpCallbackPage(ctx context.Context, rt *runtime.Context, section string) error {
	parseMode, disablePreview := helpSectionOptions(section)
	return s.sendCallbackPageWithOptions(ctx, rt, helpPageText(section), helpMarkup(section, rt.Bot.Username), parseMode, disablePreview)
}

func (s *Service) sendCallbackPage(ctx context.Context, rt *runtime.Context, text string, markup *telegram.InlineKeyboardMarkup) error {
	return s.sendCallbackPageWithOptions(ctx, rt, text, markup, "", false)
}

func (s *Service) sendCallbackPageWithOptions(ctx context.Context, rt *runtime.Context, text string, markup *telegram.InlineKeyboardMarkup, parseMode string, disablePreview bool) error {
	if rt.CallbackQuery == nil || rt.CallbackQuery.Message == nil {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{ParseMode: parseMode, DisableWebPagePreview: disablePreview, ReplyMarkup: markup})
		return err
	}
	err := rt.Client.EditMessageText(ctx, rt.ChatID(), rt.CallbackQuery.Message.MessageID, text, telegram.EditMessageTextOptions{
		ParseMode:             parseMode,
		DisableWebPagePreview: disablePreview,
		ReplyMarkup:           markup,
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

func (s *Service) startRules(ctx context.Context, rt *runtime.Context, rawChatID string) error {
	chatID, err := strconv.ParseInt(strings.TrimSpace(rawChatID), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid rules target")
	}
	bundle, err := rt.Store.LoadRuntimeBundle(ctx, rt.Bot.ID, chatID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(bundle.Settings.RulesText) == "" {
		return fmt.Errorf("rules are not set")
	}
	chatTitle := "this group"
	if chat, err := rt.Client.GetChat(ctx, chatID); err == nil && strings.TrimSpace(chat.Title) != "" {
		chatTitle = chat.Title
	}
	requester := telegram.User{}
	if rt.Message != nil && rt.Message.From != nil {
		requester = *rt.Message.From
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), rulesText(chatTitle, serviceutil.RenderStoredMessage(bundle.Settings.RulesText, requester, telegram.Chat{ID: chatID, Title: chatTitle}, bundle.Settings.RulesText)), rt.ReplyOptions(telegram.SendMessageOptions{
		ReplyMarkup: rulesPMMarkup(rt.Bot.Username),
	}))
	return err
}

func isPrivateChat(rt *runtime.Context) bool {
	if rt.Message != nil {
		return rt.Message.Chat.Type == "private"
	}
	if rt.CallbackQuery != nil && rt.CallbackQuery.Message != nil {
		return rt.CallbackQuery.Message.Chat.Type == "private"
	}
	return false
}

func (s *Service) ShouldFastPathCommand(command commands.Parsed) bool {
	switch command.Name {
	case "start", "help", "donate", "privacy":
		return true
	default:
		return false
	}
}

func (s *Service) ShouldFastPathCallback(data string) bool {
	switch data {
	case callbackStartHome, callbackStartClone, callbackHelpMain, callbackPrivacy, callbackClose:
		return true
	default:
		return strings.HasPrefix(data, callbackHelpPrefix)
	}
}

func helpSectionFromCallback(data string) (string, bool) {
	switch data {
	case "ux:help:main":
		return helpRoot, true
	case callbackHelpMain:
		return helpRoot, true
	}
	if !strings.HasPrefix(data, callbackHelpPrefix) {
		return "", false
	}
	section := normalizeHelpSection(strings.TrimPrefix(data, callbackHelpPrefix))
	if section == "" {
		return "", false
	}
	return section, true
}

func helpMarkup(section string, username string) *telegram.InlineKeyboardMarkup {
	if section == helpRoot {
		return helpLandingMarkup(username)
	}
	return helpSectionMarkup(section, username)
}
