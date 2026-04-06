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

func (s *Service) language(ctx context.Context, rt *runtime.Context) error {
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "language.current", rt.RuntimeBundle.Settings.Language), telegram.SendMessageOptions{})
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
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(language, "language.updated", language), telegram.SendMessageOptions{})
	return err
}

func (s *Service) privacy(ctx context.Context, rt *runtime.Context) error {
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "privacy.info"), telegram.SendMessageOptions{})
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
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "privacy.export", string(body)), telegram.SendMessageOptions{})
	return err
}

func (s *Service) forgetMe(ctx context.Context, rt *runtime.Context) error {
	if len(rt.Command.Args) == 0 || !strings.EqualFold(rt.Command.Args[0], "confirm") {
		return fmt.Errorf("usage: /forgetme confirm")
	}
	if err := rt.Store.DeleteUserData(ctx, rt.Bot.ID, rt.ActorID()); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), i18n.T(rt.RuntimeBundle.Settings.Language, "privacy.deleted"), telegram.SendMessageOptions{})
	return err
}
