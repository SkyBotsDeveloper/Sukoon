package content

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/service/admin"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) HandleCommand(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "save":
		return true, s.save(ctx, rt)
	case "notes", "saved":
		return true, s.notes(ctx, rt)
	case "get":
		return true, s.get(ctx, rt)
	case "clear":
		return true, s.clear(ctx, rt)
	case "filter":
		return true, s.filter(ctx, rt)
	case "filters":
		return true, s.filters(ctx, rt)
	case "stop":
		return true, s.stop(ctx, rt)
	case "welcome", "setwelcome":
		return true, s.welcome(ctx, rt)
	case "goodbye", "setgoodbye":
		return true, s.goodbye(ctx, rt)
	case "setrules":
		return true, s.setRules(ctx, rt)
	case "resetrules":
		return true, s.resetRules(ctx, rt)
	case "rules":
		return true, s.rules(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) HandleMessage(ctx context.Context, rt *runtime.Context) (bool, error) {
	text := strings.TrimSpace(rt.Text())
	if text == "" {
		return false, nil
	}

	if strings.HasPrefix(text, "#") {
		name := strings.TrimPrefix(strings.Fields(text)[0], "#")
		if name != "" {
			note, err := rt.Store.GetNote(ctx, rt.Bot.ID, rt.ChatID(), strings.ToLower(name))
			if err == nil {
				replyMarkup, err := buttonsFromJSON(note.ButtonsJSON)
				if err != nil {
					return false, err
				}
				_, err = rt.Client.SendMessage(ctx, rt.ChatID(), note.Text, telegram.SendMessageOptions{ParseMode: note.ParseMode, ReplyMarkup: replyMarkup})
				return true, err
			}
			if err != pgx.ErrNoRows {
				return false, err
			}
		}
	}

	filters, err := rt.Store.ListFilters(ctx, rt.Bot.ID, rt.ChatID())
	if err != nil {
		return false, err
	}
	lowerText := strings.ToLower(text)
	for _, filter := range filters {
		trigger := strings.ToLower(filter.Trigger)
		if strings.Contains(lowerText, trigger) {
			replyMarkup, err := buttonsFromJSON(filter.ButtonsJSON)
			if err != nil {
				return false, err
			}
			_, err = rt.Client.SendMessage(ctx, rt.ChatID(), filter.ResponseText, telegram.SendMessageOptions{ParseMode: filter.ParseMode, ReplyMarkup: replyMarkup})
			return true, err
		}
	}

	return false, nil
}

func (s *Service) HandleJoin(ctx context.Context, rt *runtime.Context, user telegram.User) error {
	if !rt.RuntimeBundle.Settings.WelcomeEnabled || strings.TrimSpace(rt.RuntimeBundle.Settings.WelcomeText) == "" {
		return nil
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), serviceutil.RenderTemplate(rt.RuntimeBundle.Settings.WelcomeText, user, rt.Message.Chat), telegram.SendMessageOptions{})
	return err
}

func (s *Service) HandleLeave(ctx context.Context, rt *runtime.Context, user telegram.User) error {
	if !rt.RuntimeBundle.Settings.GoodbyeEnabled || strings.TrimSpace(rt.RuntimeBundle.Settings.GoodbyeText) == "" {
		return nil
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), serviceutil.RenderTemplate(rt.RuntimeBundle.Settings.GoodbyeText, user, rt.Message.Chat), telegram.SendMessageOptions{})
	return err
}

func (s *Service) save(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	name, body, err := splitNameAndBody(rt.Command.RawArgs)
	if err != nil {
		return fmt.Errorf("usage: /save <name> <text>")
	}
	text, buttonsJSON, err := parseStoredContent(body)
	if err != nil {
		return err
	}
	if err := rt.Store.UpsertNote(ctx, domain.Note{
		BotID:       rt.Bot.ID,
		ChatID:      rt.ChatID(),
		Name:        strings.ToLower(name),
		Text:        text,
		ParseMode:   "",
		ButtonsJSON: buttonsJSON,
		CreatedBy:   rt.ActorID(),
	}); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Saved note %s.", name), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) notes(ctx context.Context, rt *runtime.Context) error {
	notes, err := rt.Store.ListNotes(ctx, rt.Bot.ID, rt.ChatID())
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No saved notes.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	names := make([]string, 0, len(notes))
	for _, note := range notes {
		names = append(names, "#"+note.Name)
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Saved notes: "+strings.Join(names, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) get(ctx context.Context, rt *runtime.Context) error {
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /get <name>")
	}
	note, err := rt.Store.GetNote(ctx, rt.Bot.ID, rt.ChatID(), strings.ToLower(rt.Command.Args[0]))
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("note not found")
		}
		return err
	}
	replyMarkup, err := buttonsFromJSON(note.ButtonsJSON)
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), note.Text, rt.ReplyOptions(telegram.SendMessageOptions{ParseMode: note.ParseMode, ReplyMarkup: replyMarkup}))
	return err
}

func (s *Service) clear(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /clear <name>")
	}
	if err := rt.Store.DeleteNote(ctx, rt.Bot.ID, rt.ChatID(), strings.ToLower(rt.Command.Args[0])); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Note removed.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) filter(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	trigger, body, err := splitNameAndBody(rt.Command.RawArgs)
	if err != nil {
		return fmt.Errorf("usage: /filter <trigger> <response>")
	}
	response, buttonsJSON, err := parseStoredContent(body)
	if err != nil {
		return err
	}
	if err := rt.Store.UpsertFilter(ctx, domain.FilterRule{
		BotID:        rt.Bot.ID,
		ChatID:       rt.ChatID(),
		Trigger:      strings.ToLower(trigger),
		MatchMode:    "contains",
		ResponseText: response,
		ParseMode:    "",
		ButtonsJSON:  buttonsJSON,
		CreatedBy:    rt.ActorID(),
	}); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Filter %s saved.", trigger), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) stop(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	items := serviceutil.SplitBulkItems(rt.Command.RawArgs)
	if len(items) == 0 {
		return fmt.Errorf("usage: /stop <trigger>")
	}
	for _, item := range items {
		if err := rt.Store.DeleteFilter(ctx, rt.Bot.ID, rt.ChatID(), strings.ToLower(item)); err != nil {
			return err
		}
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Removed %d filter(s).", len(items)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) filters(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	filters, err := rt.Store.ListFilters(ctx, rt.Bot.ID, rt.ChatID())
	if err != nil {
		return err
	}
	if len(filters) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No saved filters.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	parts := make([]string, 0, len(filters))
	for _, filter := range filters {
		parts = append(parts, filter.Trigger)
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Saved filters: "+strings.Join(parts, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) welcome(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.Settings.WelcomeEnabled {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Welcome is "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := admin.ParseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	text := rt.RuntimeBundle.Settings.WelcomeText
	if len(rt.Command.Args) > 1 {
		text = strings.TrimSpace(strings.Join(rt.Command.Args[1:], " "))
	}
	if err := rt.Store.SetWelcome(ctx, rt.Bot.ID, rt.ChatID(), enabled, text); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Welcome updated.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) goodbye(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.Settings.GoodbyeEnabled {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Goodbye is "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := admin.ParseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	text := rt.RuntimeBundle.Settings.GoodbyeText
	if len(rt.Command.Args) > 1 {
		text = strings.TrimSpace(strings.Join(rt.Command.Args[1:], " "))
	}
	if err := rt.Store.SetGoodbye(ctx, rt.Bot.ID, rt.ChatID(), enabled, text); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Goodbye updated.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setRules(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if strings.TrimSpace(rt.Command.RawArgs) == "" {
		return fmt.Errorf("usage: /setrules <text>")
	}
	if err := rt.Store.SetRules(ctx, rt.Bot.ID, rt.ChatID(), rt.Command.RawArgs); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Rules updated.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) resetRules(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if err := rt.Store.SetRules(ctx, rt.Bot.ID, rt.ChatID(), ""); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Rules cleared.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) rules(ctx context.Context, rt *runtime.Context) error {
	if strings.TrimSpace(rt.RuntimeBundle.Settings.RulesText) == "" {
		return fmt.Errorf("rules are not set")
	}
	if rt.Message != nil && rt.Message.Chat.Type != "private" {
		text := "Rules are available below. Open them in PM for a cleaner view, or use the inline button to show them here."
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{
			ReplyMarkup: serviceutil.Markup(
				[]telegram.InlineKeyboardButton{
					{Text: "Open PM", URL: serviceutil.BotDeepLink(rt.Bot.Username, fmt.Sprintf("rules_%d", rt.ChatID()))},
					{Text: "Show Here", CallbackData: "ux:rules:show"},
				},
				[]telegram.InlineKeyboardButton{
					{Text: "Help", URL: serviceutil.BotDeepLink(rt.Bot.Username, "help_ruleswelcome")},
					{Text: "Website", URL: serviceutil.WebsiteURL},
				},
				[]telegram.InlineKeyboardButton{
					{Text: "Close", CallbackData: "ux:close"},
				},
			),
		}))
		return err
	}

	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), rt.RuntimeBundle.Settings.RulesText, rt.ReplyOptions(telegram.SendMessageOptions{
		ReplyMarkup: serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Help", CallbackData: "ux:help:ruleswelcome"},
				{Text: "Website", URL: serviceutil.WebsiteURL},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: "ux:start:home"},
				{Text: "Close", CallbackData: "ux:close"},
			},
		),
	}))
	return err
}
