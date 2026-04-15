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
	case "stopall":
		return true, s.stopAll(ctx, rt)
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
	message := rt.Message
	text := strings.TrimSpace(rt.Text())
	if text == "" {
		return false, nil
	}

	if strings.HasPrefix(text, "#") {
		name := strings.TrimPrefix(strings.Fields(text)[0], "#")
		if name != "" {
			note, err := rt.Store.GetNote(ctx, rt.Bot.ID, dataChatID(rt), strings.ToLower(name))
			if err == nil {
				user := telegram.User{}
				if message.From != nil {
					user = *message.From
				}
				rendered, err := renderStoredPayload(note.Text, note.ButtonsJSON, user, dataChat(rt), rt.RuntimeBundle.Settings.RulesText, rt.Bot.Username, dataChatID(rt))
				if err != nil {
					return false, err
				}
				_, err = rt.Client.SendMessage(ctx, rt.ChatID(), rendered.Text, telegram.SendMessageOptions{
					ParseMode:             coalesceParseMode(note.ParseMode, rendered.ParseMode),
					ReplyMarkup:           rendered.ReplyMarkup,
					DisableWebPagePreview: rendered.DisableWebPagePreview,
					EnableWebPagePreview:  rendered.EnableWebPagePreview,
					ShowPreviewAboveText:  rendered.ShowPreviewAboveText,
					DisableNotification:   rendered.DisableNotification,
					ProtectContent:        rendered.ProtectContent,
				})
				return true, err
			}
			if err != pgx.ErrNoRows {
				return false, err
			}
		}
	}

	filters, err := rt.Store.ListFilters(ctx, rt.Bot.ID, dataChatID(rt))
	if err != nil {
		return false, err
	}
	lowerText := strings.ToLower(text)
	for _, filter := range filters {
		if filterMatches(filter, lowerText) {
			user := telegram.User{}
			if message.From != nil {
				user = *message.From
			}
			noFormat := hasFilterKeyword(lowerText, strings.ToLower(filter.Trigger), "noformat")
			force := hasFilterKeyword(lowerText, strings.ToLower(filter.Trigger), "force")
			rendered, ok, err := renderFilterResponse(filter, user, dataChat(rt), rt.RuntimeBundle.Settings.RulesText, rt, noFormat, force)
			if err != nil || !ok {
				return false, err
			}
			if rendered.MediaFileID != "" && !noFormat {
				_, err = rt.Client.SendMedia(ctx, rt.ChatID(), rendered.MediaType, rendered.MediaFileID, rt.ReplyMediaOptions(telegram.SendMediaOptions{
					Caption:               rendered.Text,
					ParseMode:             coalesceParseMode(filter.ParseMode, rendered.ParseMode),
					ReplyMarkup:           rendered.ReplyMarkup,
					DisableNotification:   rendered.DisableNotification,
					ProtectContent:        rendered.ProtectContent,
					HasSpoiler:            rendered.HasMediaSpoiler,
					ShowCaptionAboveMedia: rendered.ShowPreviewAboveText,
				}))
				return true, err
			}
			_, err = rt.Client.SendMessage(ctx, rt.ChatID(), rendered.Text, rt.ReplyOptions(telegram.SendMessageOptions{
				ParseMode:             coalesceParseMode(filter.ParseMode, rendered.ParseMode),
				ReplyMarkup:           rendered.ReplyMarkup,
				DisableNotification:   rendered.DisableNotification,
				ProtectContent:        rendered.ProtectContent,
				DisableWebPagePreview: rendered.DisableWebPagePreview,
				EnableWebPagePreview:  rendered.EnableWebPagePreview,
				ShowPreviewAboveText:  rendered.ShowPreviewAboveText,
			}))
			return true, err
		}
	}

	return false, nil
}

func (s *Service) HandleJoin(ctx context.Context, rt *runtime.Context, user telegram.User) error {
	if !rt.RuntimeBundle.Settings.WelcomeEnabled || strings.TrimSpace(rt.RuntimeBundle.Settings.WelcomeText) == "" {
		return nil
	}
	rendered, err := renderStoredPayload(rt.RuntimeBundle.Settings.WelcomeText, "", user, rt.Message.Chat, rt.RuntimeBundle.Settings.RulesText, rt.Bot.Username, rt.ChatID())
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), rendered.Text, telegram.SendMessageOptions{
		ParseMode:             rendered.ParseMode,
		ReplyMarkup:           rendered.ReplyMarkup,
		DisableWebPagePreview: rendered.DisableWebPagePreview,
		EnableWebPagePreview:  rendered.EnableWebPagePreview,
		ShowPreviewAboveText:  rendered.ShowPreviewAboveText,
		DisableNotification:   rendered.DisableNotification,
		ProtectContent:        rendered.ProtectContent,
	})
	return err
}

func (s *Service) HandleLeave(ctx context.Context, rt *runtime.Context, user telegram.User) error {
	if !rt.RuntimeBundle.Settings.GoodbyeEnabled || strings.TrimSpace(rt.RuntimeBundle.Settings.GoodbyeText) == "" {
		return nil
	}
	rendered, err := renderStoredPayload(rt.RuntimeBundle.Settings.GoodbyeText, "", user, rt.Message.Chat, rt.RuntimeBundle.Settings.RulesText, rt.Bot.Username, rt.ChatID())
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), rendered.Text, telegram.SendMessageOptions{
		ParseMode:             rendered.ParseMode,
		ReplyMarkup:           rendered.ReplyMarkup,
		DisableWebPagePreview: rendered.DisableWebPagePreview,
		EnableWebPagePreview:  rendered.EnableWebPagePreview,
		ShowPreviewAboveText:  rendered.ShowPreviewAboveText,
		DisableNotification:   rendered.DisableNotification,
		ProtectContent:        rendered.ProtectContent,
	})
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
		ChatID:      dataChatID(rt),
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
	notes, err := rt.Store.ListNotes(ctx, rt.Bot.ID, dataChatID(rt))
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
	note, err := rt.Store.GetNote(ctx, rt.Bot.ID, dataChatID(rt), strings.ToLower(rt.Command.Args[0]))
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("note not found")
		}
		return err
	}
	user := telegram.User{}
	if rt.Message != nil && rt.Message.From != nil {
		user = *rt.Message.From
	}
	rendered, err := renderStoredPayload(note.Text, note.ButtonsJSON, user, dataChat(rt), rt.RuntimeBundle.Settings.RulesText, rt.Bot.Username, dataChatID(rt))
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), rendered.Text, rt.ReplyOptions(telegram.SendMessageOptions{
		ParseMode:             coalesceParseMode(note.ParseMode, rendered.ParseMode),
		ReplyMarkup:           rendered.ReplyMarkup,
		DisableWebPagePreview: rendered.DisableWebPagePreview,
		EnableWebPagePreview:  rendered.EnableWebPagePreview,
		ShowPreviewAboveText:  rendered.ShowPreviewAboveText,
		DisableNotification:   rendered.DisableNotification,
		ProtectContent:        rendered.ProtectContent,
	}))
	return err
}

func (s *Service) clear(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /clear <name>")
	}
	if err := rt.Store.DeleteNote(ctx, rt.Bot.ID, dataChatID(rt), strings.ToLower(rt.Command.Args[0])); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Note removed.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) filter(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	definitions, err := parseFilterDefinitions(rt.Command.RawArgs)
	if err != nil {
		if rt.Message == nil || rt.Message.ReplyToMessage == nil {
			return fmt.Errorf("usage: /filter <trigger> <response>")
		}
		definitions, err = parseFilterTriggersOnly(rt.Command.RawArgs)
		if err != nil {
			return fmt.Errorf("usage: /filter <trigger> <response>")
		}
	}
	replyText := ""
	replyButtons := ""
	replyMediaType := ""
	replyMediaFileID := ""
	if rt.Message != nil && rt.Message.ReplyToMessage != nil && strings.TrimSpace(rt.Command.RawArgs) != "" {
		if fallbackDefs, fallbackErr := parseFilterTriggersOnly(rt.Command.RawArgs); fallbackErr == nil {
			if text, buttonsJSON, mediaType, mediaFileID, replyErr := parseFilterReplyPayload(rt.Message.ReplyToMessage); replyErr == nil && len(fallbackDefs) > 0 {
				if onlyTriggers(definitions) {
					definitions = fallbackDefs
					replyText = text
					replyButtons = buttonsJSON
					replyMediaType = mediaType
					replyMediaFileID = mediaFileID
				}
			}
		}
	}
	if onlyTriggers(definitions) && strings.TrimSpace(replyText) == "" && replyMediaFileID == "" {
		return fmt.Errorf("usage: /filter <trigger> <response>")
	}
	for _, definition := range definitions {
		response := replyText
		buttonsJSON := replyButtons
		mediaType := replyMediaType
		mediaFileID := replyMediaFileID
		if definition.Body != "" {
			var parseErr error
			response, buttonsJSON, parseErr = parseStoredContent(definition.Body)
			if parseErr != nil {
				return parseErr
			}
			mediaType = ""
			mediaFileID = ""
		}
		if err := rt.Store.UpsertFilter(ctx, domain.FilterRule{
			BotID:               rt.Bot.ID,
			ChatID:              dataChatID(rt),
			Trigger:             strings.ToLower(definition.Trigger),
			MatchMode:           definition.MatchMode,
			ResponseText:        response,
			ResponseMediaType:   mediaType,
			ResponseMediaFileID: mediaFileID,
			ParseMode:           "",
			ButtonsJSON:         buttonsJSON,
			CreatedBy:           rt.ActorID(),
		}); err != nil {
			return err
		}
	}
	label := definitions[0].Trigger
	if len(definitions) > 1 {
		label = fmt.Sprintf("%d filters", len(definitions))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Filter %s saved.", label), rt.ReplyOptions(telegram.SendMessageOptions{}))
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
		if err := rt.Store.DeleteFilter(ctx, rt.Bot.ID, dataChatID(rt), normalizeFilterTrigger(item)); err != nil {
			return err
		}
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Removed %d filter(s).", len(items)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) stopAll(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsOwner && !rt.ActorPermissions.IsSudo && !rt.ActorPermissions.IsChatCreator {
		return fmt.Errorf("chat creator rights required")
	}
	filters, err := rt.Store.ListFilters(ctx, rt.Bot.ID, dataChatID(rt))
	if err != nil {
		return err
	}
	if len(filters) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No saved filters to remove.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	for _, filter := range filters {
		if err := rt.Store.DeleteFilter(ctx, rt.Bot.ID, dataChatID(rt), strings.ToLower(filter.Trigger)); err != nil {
			return err
		}
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Removed %d filter(s).", len(filters)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) filters(ctx context.Context, rt *runtime.Context) error {
	filters, err := rt.Store.ListFilters(ctx, rt.Bot.ID, dataChatID(rt))
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
	if err := rt.Store.SetWelcome(ctx, rt.Bot.ID, dataChatID(rt), enabled, text); err != nil {
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
	if err := rt.Store.SetGoodbye(ctx, rt.Bot.ID, dataChatID(rt), enabled, text); err != nil {
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
	if err := rt.Store.SetRules(ctx, rt.Bot.ID, dataChatID(rt), rt.Command.RawArgs); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Rules updated.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) resetRules(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if err := rt.Store.SetRules(ctx, rt.Bot.ID, dataChatID(rt), ""); err != nil {
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
					{Text: "Open PM", URL: serviceutil.BotDeepLink(rt.Bot.Username, fmt.Sprintf("rules_%d", dataChatID(rt)))},
					{Text: "Show Here", CallbackData: "ux:rules:show"},
				},
				[]telegram.InlineKeyboardButton{
					{Text: "Help", URL: serviceutil.BotDeepLink(rt.Bot.Username, "help_main")},
					{Text: "Website", URL: serviceutil.WebsiteURL},
				},
				[]telegram.InlineKeyboardButton{
					{Text: "Close", CallbackData: "ux:close"},
				},
			),
		}))
		return err
	}

	user := telegram.User{}
	if rt.Message != nil && rt.Message.From != nil {
		user = *rt.Message.From
	}
	rendered, err := renderStoredPayload(rt.RuntimeBundle.Settings.RulesText, "", user, dataChat(rt), rt.RuntimeBundle.Settings.RulesText, rt.Bot.Username, dataChatID(rt))
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), rendered.Text, rt.ReplyOptions(telegram.SendMessageOptions{
		ParseMode: rendered.ParseMode,
		ReplyMarkup: serviceutil.Markup(
			[]telegram.InlineKeyboardButton{
				{Text: "Help", CallbackData: "ux:help:root"},
				{Text: "Website", URL: serviceutil.WebsiteURL},
			},
			[]telegram.InlineKeyboardButton{
				{Text: "Back", CallbackData: "ux:start:home"},
				{Text: "Close", CallbackData: "ux:close"},
			},
		),
		DisableWebPagePreview: rendered.DisableWebPagePreview,
		EnableWebPagePreview:  rendered.EnableWebPagePreview,
		ShowPreviewAboveText:  rendered.ShowPreviewAboveText,
		DisableNotification:   rendered.DisableNotification,
		ProtectContent:        rendered.ProtectContent,
	}))
	return err
}

func dataChatID(rt *runtime.Context) int64 {
	if rt.TargetChatID != 0 {
		return rt.TargetChatID
	}
	return rt.ChatID()
}

func dataChat(rt *runtime.Context) telegram.Chat {
	if rt.TargetChat != nil {
		return *rt.TargetChat
	}
	if rt.Message != nil {
		return rt.Message.Chat
	}
	return telegram.Chat{ID: dataChatID(rt)}
}

func filterMatches(filter domain.FilterRule, lowerText string) bool {
	trigger := strings.ToLower(strings.TrimSpace(filter.Trigger))
	if trigger == "" {
		return false
	}
	text := strings.TrimSpace(lowerText)
	switch strings.ToLower(strings.TrimSpace(filter.MatchMode)) {
	case "exact":
		return text == trigger || text == trigger+" noformat" || text == trigger+" force"
	case "prefix":
		return strings.HasPrefix(text, trigger)
	default:
		return strings.Contains(lowerText, trigger)
	}
}

func hasFilterKeyword(lowerText string, trigger string, keyword string) bool {
	text := strings.TrimSpace(lowerText)
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	trigger = strings.ToLower(strings.TrimSpace(trigger))
	return text == trigger+" "+keyword || strings.HasSuffix(text, " "+keyword)
}

func normalizeFilterTrigger(item string) string {
	item = strings.ToLower(strings.Trim(strings.TrimSpace(item), `"`))
	switch {
	case strings.HasPrefix(item, "exact:"):
		return strings.TrimSpace(item[len("exact:"):])
	case strings.HasPrefix(item, "prefix:"):
		return strings.TrimSpace(item[len("prefix:"):])
	default:
		return item
	}
}

func onlyTriggers(definitions []filterDefinition) bool {
	if len(definitions) == 0 {
		return false
	}
	for _, definition := range definitions {
		if strings.TrimSpace(definition.Body) != "" {
			return false
		}
	}
	return true
}

func coalesceParseMode(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}
