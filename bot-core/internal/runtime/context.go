package runtime

import (
	"context"
	"log/slog"
	"strings"

	"sukoon/bot-core/internal/commands"
	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/state"
	"sukoon/bot-core/internal/telegram"
)

type Context struct {
	Base             context.Context
	Logger           *slog.Logger
	Store            persistence.Store
	State            state.Store
	Bot              domain.BotInstance
	Client           telegram.Client
	Update           telegram.Update
	Message          *telegram.Message
	CallbackQuery    *telegram.CallbackQuery
	Command          commands.Parsed
	CommandOK        bool
	RuntimeBundle    domain.RuntimeBundle
	KnownChatAdmins  map[int64]struct{}
	ActorPermissions ActorPermissions
	TargetChatID     int64
	TargetChat       *telegram.Chat
}

type ActorPermissions struct {
	IsOwner            bool
	IsSudo             bool
	IsChatCreator      bool
	IsChatAdmin        bool
	IsSilentMod        bool
	CanDeleteMessages  bool
	CanMuteMembers     bool
	CanRestrictMembers bool
	CanChangeInfo      bool
	CanPinMessages     bool
	CanPromoteMembers  bool
}

func (c *Context) ChatID() int64 {
	if c.Message != nil {
		return c.Message.Chat.ID
	}
	if c.CallbackQuery != nil && c.CallbackQuery.Message != nil {
		return c.CallbackQuery.Message.Chat.ID
	}
	return 0
}

func (c *Context) ActorID() int64 {
	if c.Message != nil && c.Message.From != nil {
		return c.Message.From.ID
	}
	if c.CallbackQuery != nil {
		return c.CallbackQuery.From.ID
	}
	return 0
}

func (c *Context) Text() string {
	if c.Message == nil {
		return ""
	}
	if strings.TrimSpace(c.Message.Text) != "" {
		return c.Message.Text
	}
	return c.Message.Caption
}

func (c *Context) ReplyOptions(options telegram.SendMessageOptions) telegram.SendMessageOptions {
	if options.ReplyToMessageID == 0 && c.Message != nil {
		options.ReplyToMessageID = c.Message.MessageID
	}
	return options
}

func (c *Context) ReplyMediaOptions(options telegram.SendMediaOptions) telegram.SendMediaOptions {
	if options.ReplyToMessageID == 0 && c.Message != nil {
		options.ReplyToMessageID = c.Message.MessageID
	}
	return options
}
