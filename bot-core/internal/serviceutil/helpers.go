package serviceutil

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
)

type Target struct {
	UserID int64
	Name   string
}

func ResolveTarget(ctx context.Context, rt *runtime.Context, args []string) (Target, error) {
	message := rt.Message
	if message != nil && message.ReplyToMessage != nil && message.ReplyToMessage.From != nil {
		return Target{
			UserID: message.ReplyToMessage.From.ID,
			Name:   DisplayName(*message.ReplyToMessage.From),
		}, nil
	}

	if len(args) == 0 {
		return Target{}, fmt.Errorf("reply to a user or pass a user id/username")
	}

	target := strings.TrimSpace(args[0])
	if strings.HasPrefix(target, "@") {
		user, err := rt.Store.GetUserByUsername(ctx, strings.TrimPrefix(target, "@"))
		if err != nil {
			return Target{}, fmt.Errorf("could not resolve username %s", target)
		}
		return Target{
			UserID: user.ID,
			Name:   DisplayNameFromProfile(user),
		}, nil
	}

	userID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return Target{}, fmt.Errorf("could not parse target user id")
	}
	user, err := rt.Store.GetUserByID(ctx, userID)
	if err == nil {
		return Target{
			UserID: user.ID,
			Name:   DisplayNameFromProfile(user),
		}, nil
	}
	return Target{
		UserID: userID,
		Name:   target,
	}, nil
}

func DisplayName(user telegram.User) string {
	if user.Username != "" {
		return "@" + user.Username
	}
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName + " " + user.LastName))
	if name == "" {
		return strconv.FormatInt(user.ID, 10)
	}
	return name
}

func RenderTemplate(template string, user telegram.User, chat telegram.Chat) string {
	replacer := strings.NewReplacer(
		"{first}", user.FirstName,
		"{last}", user.LastName,
		"{username}", user.Username,
		"{id}", strconv.FormatInt(user.ID, 10),
		"{chat}", chat.Title,
	)
	return replacer.Replace(template)
}

func DisplayNameFromProfile(user domain.UserProfile) string {
	if user.Username != "" {
		return "@" + user.Username
	}
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName + " " + user.LastName))
	if name == "" {
		return strconv.FormatInt(user.ID, 10)
	}
	return name
}
