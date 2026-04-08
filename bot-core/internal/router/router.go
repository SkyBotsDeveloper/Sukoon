package router

import (
	"context"
	"fmt"
	"log/slog"

	"sukoon/bot-core/internal/commands"
	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/permissions"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/service/admin"
	"sukoon/bot-core/internal/service/afk"
	antiabuseservice "sukoon/bot-core/internal/service/antiabuse"
	antibioservice "sukoon/bot-core/internal/service/antibio"
	"sukoon/bot-core/internal/service/antispam"
	"sukoon/bot-core/internal/service/captcha"
	cloneService "sukoon/bot-core/internal/service/clones"
	"sukoon/bot-core/internal/service/content"
	fedservice "sukoon/bot-core/internal/service/federation"
	"sukoon/bot-core/internal/service/moderation"
	ownerservice "sukoon/bot-core/internal/service/owner"
	utilityservice "sukoon/bot-core/internal/service/utility"
	"sukoon/bot-core/internal/state"
	"sukoon/bot-core/internal/telegram"
)

type Router struct {
	store       persistence.Store
	state       state.Store
	permissions *permissions.Service
	moderation  *moderation.Service
	admin       *admin.Service
	antispam    *antispam.Service
	content     *content.Service
	captcha     *captcha.Service
	afk         *afk.Service
	owner       *ownerservice.Service
	federation  *fedservice.Service
	clones      *cloneService.Service
	antiabuse   *antiabuseservice.Service
	antibio     *antibioservice.Service
	utility     *utilityservice.Service
	logger      *slog.Logger
}

func New(
	store persistence.Store,
	state state.Store,
	permissions *permissions.Service,
	moderation *moderation.Service,
	adminService *admin.Service,
	antispamService *antispam.Service,
	contentService *content.Service,
	captchaService *captcha.Service,
	afkService *afk.Service,
	ownerService *ownerservice.Service,
	federationService *fedservice.Service,
	cloneMgmtService *cloneService.Service,
	antiAbuseService *antiabuseservice.Service,
	antiBioService *antibioservice.Service,
	utilityService *utilityservice.Service,
	logger *slog.Logger,
) *Router {
	return &Router{
		store:       store,
		state:       state,
		permissions: permissions,
		moderation:  moderation,
		admin:       adminService,
		antispam:    antispamService,
		content:     contentService,
		captcha:     captchaService,
		afk:         afkService,
		owner:       ownerService,
		federation:  federationService,
		clones:      cloneMgmtService,
		antiabuse:   antiAbuseService,
		antibio:     antiBioService,
		utility:     utilityService,
		logger:      logger,
	}
}

func (r *Router) HandleUpdate(ctx context.Context, bot domain.BotInstance, client telegram.Client, update telegram.Update) error {
	message := update.Message
	callback := update.CallbackQuery

	var chat telegram.Chat
	if message != nil {
		chat = message.Chat
	} else if callback != nil && callback.Message != nil {
		chat = callback.Message.Chat
	} else {
		return nil
	}

	if err := r.store.EnsureChat(ctx, bot.ID, chat); err != nil {
		return err
	}
	if message != nil {
		if message.From != nil {
			if err := r.store.EnsureUser(ctx, *message.From); err != nil {
				return err
			}
		}
		if message.ReplyToMessage != nil && message.ReplyToMessage.From != nil {
			if err := r.store.EnsureUser(ctx, *message.ReplyToMessage.From); err != nil {
				return err
			}
		}
		for _, member := range message.NewChatMembers {
			if err := r.store.EnsureUser(ctx, member); err != nil {
				return err
			}
		}
		if message.LeftChatMember != nil {
			if err := r.store.EnsureUser(ctx, *message.LeftChatMember); err != nil {
				return err
			}
		}
	}
	if callback != nil {
		if err := r.store.EnsureUser(ctx, callback.From); err != nil {
			return err
		}
	}
	bundle, err := r.store.LoadRuntimeBundle(ctx, bot.ID, chat.ID)
	if err != nil {
		return err
	}

	var parsed commands.Parsed
	var commandOK bool
	if message != nil {
		parsed, commandOK = commands.Parse(textFromMessage(message), bot.Username)
	}

	rt := &runtime.Context{
		Base:          ctx,
		Logger:        r.logger.With("bot_id", bot.ID, "chat_id", chat.ID, "update_id", update.UpdateID),
		Store:         r.store,
		State:         r.state,
		Bot:           bot,
		Client:        client,
		Update:        update,
		Message:       message,
		CallbackQuery: callback,
		Command:       parsed,
		CommandOK:     commandOK,
		RuntimeBundle: bundle,
	}

	actorID := rt.ActorID()
	if actorID != 0 && chat.Type != "channel" {
		perms, err := r.permissions.Load(ctx, bot.ID, actorID, chat.ID, chat.Type, client)
		if err != nil {
			return err
		}
		rt.ActorPermissions = perms
	}
	if message != nil && message.SenderChat != nil && message.SenderChat.ID == chat.ID && rt.RuntimeBundle.Settings.AnonAdmins {
		rt.ActorPermissions.IsChatAdmin = true
		rt.ActorPermissions.CanDeleteMessages = true
		rt.ActorPermissions.CanMuteMembers = true
		rt.ActorPermissions.CanRestrictMembers = true
		rt.ActorPermissions.CanChangeInfo = true
		rt.ActorPermissions.CanPinMessages = true
		rt.ActorPermissions.CanPromoteMembers = true
	}

	if callback != nil {
		handled, err := r.captcha.HandleCallback(ctx, rt)
		if handled || err != nil {
			return err
		}
		if r.clones != nil {
			handled, err = r.clones.HandleCallback(ctx, rt)
			if handled || err != nil {
				return err
			}
		}
		if r.utility != nil {
			handled, err = r.utility.HandleCallback(ctx, rt)
			if handled || err != nil {
				return err
			}
		}
		return nil
	}

	if message == nil {
		return nil
	}

	if r.owner != nil {
		handled, err := r.owner.HandleMessage(ctx, rt)
		if handled || err != nil {
			return err
		}
	}
	if r.federation != nil {
		handled, err := r.federation.HandleMessage(ctx, rt)
		if handled || err != nil {
			return err
		}
	}
	if r.antiabuse != nil {
		handled, err := r.antiabuse.HandleMessage(ctx, rt)
		if handled || err != nil {
			return err
		}
	}
	if shouldCleanServiceMessage(message, rt.RuntimeBundle.Settings) {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
	}

	for _, member := range message.NewChatMembers {
		if member.IsBot {
			continue
		}
		if r.owner != nil {
			if err := r.owner.HandleJoin(ctx, rt, member); err != nil {
				return err
			}
		}
		if r.federation != nil {
			if err := r.federation.HandleJoin(ctx, rt, member); err != nil {
				return err
			}
		}
		if err := r.captcha.HandleJoin(ctx, rt, member); err != nil {
			return err
		}
		if err := r.content.HandleJoin(ctx, rt, member); err != nil {
			return err
		}
	}

	if message.LeftChatMember != nil {
		if err := r.content.HandleLeave(ctx, rt, *message.LeftChatMember); err != nil {
			return err
		}
	}

	if rt.CommandOK {
		if _, disabled := rt.RuntimeBundle.DisabledCommands[rt.Command.Name]; disabled && !rt.ActorPermissions.IsOwner && !rt.ActorPermissions.IsSudo {
			if rt.ActorPermissions.IsChatAdmin && !rt.RuntimeBundle.Settings.DisableAdmins {
				goto dispatchCommand
			}
			if rt.RuntimeBundle.Settings.DisabledDelete && rt.Message != nil {
				_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
				return nil
			}
			_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("/%s is disabled in this chat.", rt.Command.Name), rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}

	dispatchCommand:
		for _, handler := range []func(context.Context, *runtime.Context) (bool, error){
			r.moderation.Handle,
			r.admin.Handle,
			r.antispam.HandleCommand,
			r.content.HandleCommand,
			r.captcha.HandleCommand,
			r.afk.HandleCommand,
			r.owner.Handle,
			r.federation.Handle,
			r.clones.Handle,
			r.antiabuse.HandleCommand,
			r.antibio.HandleCommand,
			r.utility.Handle,
		} {
			handled, err := handler(ctx, rt)
			if handled || err != nil {
				if err == nil && rt.RuntimeBundle.Settings.CleanCommands {
					_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
				}
				return err
			}
		}
	}

	for _, handler := range []func(context.Context, *runtime.Context) (bool, error){
		r.afk.HandleMessage,
		r.antibio.HandleMessage,
		r.antispam.HandleMessage,
		r.content.HandleMessage,
	} {
		handled, err := handler(ctx, rt)
		if handled || err != nil {
			return err
		}
	}

	return nil
}

func textFromMessage(message *telegram.Message) string {
	if message.Text != "" {
		return message.Text
	}
	return message.Caption
}

func shouldCleanServiceMessage(message *telegram.Message, settings domain.ChatSettings) bool {
	if message == nil {
		return false
	}
	switch {
	case len(message.NewChatMembers) > 0:
		return settings.CleanServiceJoin
	case message.LeftChatMember != nil:
		return settings.CleanServiceLeave
	case message.PinnedMessage != nil:
		return settings.CleanServicePin
	case message.NewChatTitle != "":
		return settings.CleanServiceTitle
	case len(message.NewChatPhoto) > 0 || message.DeleteChatPhoto:
		return settings.CleanServicePhoto
	case message.VideoChatStarted != nil || message.VideoChatEnded != nil || message.VideoChatParticipantsInvited != nil || message.VideoChatScheduled != nil:
		return settings.CleanServiceVideoChat
	case message.GroupChatCreated || message.SupergroupChatCreated || message.ChannelChatCreated || message.MessageAutoDeleteTimerChanged != nil:
		return settings.CleanServiceOther
	default:
		return false
	}
}
