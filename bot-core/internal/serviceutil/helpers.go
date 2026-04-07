package serviceutil

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

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
	return RenderChatTemplate(template, user, chat, "")
}

func RenderChatTemplate(template string, user telegram.User, chat telegram.Chat, rules string) string {
	fullName := strings.TrimSpace(strings.TrimSpace(user.FirstName + " " + user.LastName))
	if fullName == "" {
		fullName = DisplayName(user)
	}
	username := user.Username
	if username != "" {
		username = "@" + strings.TrimPrefix(username, "@")
	}
	mention := username
	if mention == "" {
		mention = fullName
	}
	replacer := strings.NewReplacer(
		"{first}", user.FirstName,
		"{last}", user.LastName,
		"{fullname}", fullName,
		"{mention}", mention,
		"{username}", username,
		"{id}", strconv.FormatInt(user.ID, 10),
		"{chat}", chat.Title,
		"{chatname}", chat.Title,
		"{rules}", rules,
		"{rules:same}", rules,
	)
	return replacer.Replace(template)
}

func RenderStoredMessage(template string, user telegram.User, chat telegram.Chat, rules string) string {
	return RenderChatTemplate(pickRandomContent(template), user, chat, rules)
}

func pickRandomContent(raw string) string {
	parts := strings.Split(raw, "%%%")
	if len(parts) == 1 {
		return strings.TrimSpace(raw)
	}
	options := make([]string, 0, len(parts))
	for _, part := range parts {
		option := strings.TrimSpace(part)
		if option != "" {
			options = append(options, option)
		}
	}
	if len(options) == 0 {
		return strings.TrimSpace(raw)
	}
	if len(options) == 1 {
		return options[0]
	}
	picker := rand.New(rand.NewSource(time.Now().UnixNano()))
	return options[picker.Intn(len(options))]
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
