package utility

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sukoon/bot-core/internal/i18n"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Handle(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "start":
		return true, s.start(ctx, rt)
	case "help":
		return true, s.help(ctx, rt)
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

func (s *Service) start(ctx context.Context, rt *runtime.Context) error {
	text := "Sukoon is online and ready.\nUse /help to see the command groups and common moderation flows."
	if rt.Message != nil && rt.Message.Chat.Type == "private" {
		text = strings.Join([]string{
			"Welcome to Sukoon.",
			"Sukoon is a Telegram moderation and group-management bot built for fast admin workflows.",
			"Add the bot to a group, grant the permissions you want it to enforce, then use /help for the main command set.",
		}, "\n")
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) help(ctx context.Context, rt *runtime.Context) error {
	lines := []string{
		"Sukoon help",
		"",
		"Moderation:",
		"/ban, /tban, /unban, /mute, /tmute, /unmute, /kick, /warn, /warns, /resetwarns",
		"",
		"Admin:",
		"/approve, /unapprove, /approved, /disable, /enable, /disabled, /logchannel, /reports, /report",
		"/cleancommands, /cleanservice, /nocleanservice, /cleanservicetypes, /pin, /unpin, /unpinall, /mods",
		"",
		"Protection:",
		"/lock, /unlock, /locks, /addblocklist, /rmbl, /blocklist, /setflood, /setfloodmode, /captcha",
		"",
		"Content and info:",
		"/setrules, /rules, /save, /get, /clear, /filter, /stop, /welcome, /goodbye",
		"",
		"Utility:",
		"/start, /help, /privacy, /mydata, /forgetme confirm, /setlang",
		"",
		"Tip: for most moderation actions, reply to the target user's message before running the command.",
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
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
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "privacy.info"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) myData(ctx context.Context, rt *runtime.Context) error {
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
	if len(rt.Command.Args) == 0 || !strings.EqualFold(rt.Command.Args[0], "confirm") {
		return fmt.Errorf("usage: /forgetme confirm")
	}
	if err := rt.Store.DeleteUserData(ctx, rt.Bot.ID, rt.ActorID()); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "privacy.deleted"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}
