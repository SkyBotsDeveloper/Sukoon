package afk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) HandleCommand(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.Command.Name != "afk" {
		return false, nil
	}

	reason := strings.TrimSpace(rt.Command.RawArgs)
	if err := rt.Store.SetAFK(ctx, domainAFKState(rt, reason)); err != nil {
		return true, err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "AFK set.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return true, err
}

func (s *Service) HandleMessage(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.Message == nil || rt.Message.From == nil {
		return false, nil
	}

	if rt.CommandOK && rt.Command.Name == "afk" {
		return false, nil
	}

	state, err := rt.Store.GetAFK(ctx, rt.Bot.ID, rt.ActorID())
	if err != nil {
		return false, err
	}
	if state.UserID != 0 {
		if err := rt.Store.ClearAFK(ctx, rt.Bot.ID, rt.ActorID()); err != nil {
			return false, err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Welcome back, AFK removed.", telegram.SendMessageOptions{})
		if err != nil {
			return false, err
		}
	}

	if rt.Message.ReplyToMessage == nil || rt.Message.ReplyToMessage.From == nil {
		return false, nil
	}
	targetState, err := rt.Store.GetAFK(ctx, rt.Bot.ID, rt.Message.ReplyToMessage.From.ID)
	if err != nil {
		return false, err
	}
	if targetState.UserID == 0 {
		return false, nil
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s is AFK: %s", serviceutil.DisplayName(*rt.Message.ReplyToMessage.From), targetState.Reason), telegram.SendMessageOptions{})
	return true, err
}

func domainAFKState(rt *runtime.Context, reason string) domain.AFKState {
	return domain.AFKState{
		BotID:  rt.Bot.ID,
		UserID: rt.ActorID(),
		Reason: reason,
		SetAt:  time.Now(),
	}
}
