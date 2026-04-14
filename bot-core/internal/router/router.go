package router

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

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
	cacheMu     sync.Mutex
	seenChats   map[string]time.Time
	seenUsers   map[int64]time.Time
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
		seenChats:   map[string]time.Time{},
		seenUsers:   map[int64]time.Time{},
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

	var parsed commands.Parsed
	var commandOK bool
	if message != nil {
		parsed, commandOK = commands.Parse(textFromMessage(message), bot.Username)
	}

	baseLogger := r.logger.With("bot_id", bot.ID, "chat_id", chat.ID, "update_id", update.UpdateID)
	if callback != nil && r.utility != nil && r.utility.ShouldFastPathCallback(callback.Data) {
		rt := &runtime.Context{
			Base:          ctx,
			Logger:        baseLogger,
			Store:         r.store,
			State:         r.state,
			Bot:           bot,
			Client:        client,
			Update:        update,
			CallbackQuery: callback,
		}
		handled, err := r.utility.HandleCallback(ctx, rt)
		if handled || err != nil {
			return err
		}
	}
	if message != nil && chat.Type == "private" && commandOK && r.utility != nil && r.utility.ShouldFastPathCommand(parsed) && !shouldBypassUtilityFastPath(parsed) {
		rt := &runtime.Context{
			Base:      ctx,
			Logger:    baseLogger,
			Store:     r.store,
			State:     r.state,
			Bot:       bot,
			Client:    client,
			Update:    update,
			Message:   message,
			Command:   parsed,
			CommandOK: commandOK,
		}
		handled, err := r.utility.Handle(ctx, rt)
		if handled || err != nil {
			return err
		}
	}

	if err := r.ensureChatIfNeeded(ctx, bot.ID, chat); err != nil {
		return err
	}
	if message != nil {
		if message.From != nil {
			if err := r.ensureUserIfNeeded(ctx, *message.From); err != nil {
				return err
			}
		}
		if message.ReplyToMessage != nil && message.ReplyToMessage.From != nil {
			if err := r.ensureUserIfNeeded(ctx, *message.ReplyToMessage.From); err != nil {
				return err
			}
		}
		for _, member := range message.NewChatMembers {
			if err := r.ensureUserIfNeeded(ctx, member); err != nil {
				return err
			}
		}
		if message.LeftChatMember != nil {
			if err := r.ensureUserIfNeeded(ctx, *message.LeftChatMember); err != nil {
				return err
			}
		}
	}
	if callback != nil {
		if err := r.ensureUserIfNeeded(ctx, callback.From); err != nil {
			return err
		}
	}
	bundle, err := r.store.LoadRuntimeBundle(ctx, bot.ID, chat.ID)
	if err != nil {
		return err
	}

	rt := &runtime.Context{
		Base:            ctx,
		Logger:          baseLogger,
		Store:           r.store,
		State:           r.state,
		Bot:             bot,
		Client:          client,
		Update:          update,
		Message:         message,
		CallbackQuery:   callback,
		Command:         parsed,
		CommandOK:       commandOK,
		RuntimeBundle:   bundle,
		KnownChatAdmins: map[int64]struct{}{},
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
	if message != nil && chat.Type == "private" && commandOK && isConnectionAwareCommand(parsed.Name) {
		if err := r.applyConnectedChatTarget(ctx, bot, client, rt); err != nil {
			return err
		}
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

	if len(message.NewChatMembers) > 0 && (rt.RuntimeBundle.AntiRaid.AutoThreshold > 0 || (rt.RuntimeBundle.AntiRaid.EnabledUntil != nil && rt.RuntimeBundle.AntiRaid.EnabledUntil.After(time.Now()))) {
		admins, err := client.GetChatAdministrators(ctx, chat.ID)
		if err == nil {
			for _, admin := range admins {
				rt.KnownChatAdmins[admin.User.ID] = struct{}{}
			}
		} else {
			rt.Logger.Warn("antiraid admin lookup failed", "error", err)
		}
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
		if r.antispam != nil {
			handled, err := r.antispam.HandleJoin(ctx, rt, member)
			if err != nil {
				return err
			}
			if handled {
				continue
			}
		}
		captchaHandled, err := r.captcha.HandleJoin(ctx, rt, member)
		if err != nil {
			return err
		}
		if captchaHandled {
			continue
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
			if rt.TargetChatID != 0 && rt.Message != nil && rt.Message.Chat.Type == "private" {
				goto dispatchCommand
			}
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
				if err == nil && shouldCleanHandledCommand(rt.Command.Name, rt.RuntimeBundle.Settings) {
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

	if shouldCleanUnhandledCommandMessage(rt.Text(), rt.RuntimeBundle.Settings) {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
	}

	return nil
}

func (r *Router) ensureChatIfNeeded(ctx context.Context, botID string, chat telegram.Chat) error {
	key := fmt.Sprintf("%s:%d", botID, chat.ID)
	if !r.shouldRefreshChat(key) {
		return nil
	}
	if err := r.store.EnsureChat(ctx, botID, chat); err != nil {
		return err
	}
	r.markChatRefreshed(key)
	return nil
}

func (r *Router) applyConnectedChatTarget(ctx context.Context, bot domain.BotInstance, client telegram.Client, rt *runtime.Context) error {
	connection, err := r.store.GetChatConnection(ctx, bot.ID, rt.ActorID())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}
	target := telegram.Chat{
		ID:       connection.ChatID,
		Type:     connection.ChatType,
		Title:    connection.ChatTitle,
		Username: connection.ChatUsername,
	}
	bundle, err := r.store.LoadRuntimeBundle(ctx, bot.ID, target.ID)
	if err != nil {
		return err
	}
	perms, err := r.permissions.Load(ctx, bot.ID, rt.ActorID(), target.ID, target.Type, client)
	if err != nil {
		return err
	}
	rt.TargetChatID = target.ID
	rt.TargetChat = &target
	rt.RuntimeBundle = bundle
	rt.ActorPermissions = perms
	rt.Logger = rt.Logger.With("target_chat_id", target.ID)
	return nil
}

func (r *Router) ensureUserIfNeeded(ctx context.Context, user telegram.User) error {
	if !r.shouldRefreshUser(user.ID) {
		return nil
	}
	if err := r.store.EnsureUser(ctx, user); err != nil {
		return err
	}
	r.markUserRefreshed(user.ID)
	return nil
}

func (r *Router) shouldRefreshChat(key string) bool {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	expiresAt, ok := r.seenChats[key]
	return !ok || time.Now().After(expiresAt)
}

func (r *Router) markChatRefreshed(key string) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.seenChats[key] = time.Now().Add(5 * time.Minute)
}

func (r *Router) shouldRefreshUser(userID int64) bool {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	expiresAt, ok := r.seenUsers[userID]
	return !ok || time.Now().After(expiresAt)
}

func (r *Router) markUserRefreshed(userID int64) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.seenUsers[userID] = time.Now().Add(10 * time.Minute)
}

func textFromMessage(message *telegram.Message) string {
	if message.Text != "" {
		return message.Text
	}
	return message.Caption
}

func shouldBypassUtilityFastPath(parsed commands.Parsed) bool {
	if parsed.Name != "start" {
		return false
	}
	raw := strings.TrimSpace(parsed.RawArgs)
	return len(raw) >= len("captcha_") && strings.EqualFold(raw[:len("captcha_")], "captcha_")
}

func shouldCleanServiceMessage(message *telegram.Message, settings domain.ChatSettings) bool {
	if message == nil {
		return false
	}
	switch cleanServiceCategory(message) {
	case "join":
		return settings.CleanServiceJoin
	case "leave":
		return settings.CleanServiceLeave
	case "pin":
		return settings.CleanServicePin
	case "title":
		return settings.CleanServiceTitle
	case "photo":
		return settings.CleanServicePhoto
	case "videochat":
		return settings.CleanServiceVideoChat
	case "other":
		return settings.CleanServiceOther
	default:
		return false
	}
}

func cleanServiceCategory(message *telegram.Message) string {
	switch {
	case len(message.NewChatMembers) > 0:
		return "join"
	case message.LeftChatMember != nil:
		return "leave"
	case message.PinnedMessage != nil:
		return "pin"
	case message.NewChatTitle != "" || message.ForumTopicCreated != nil || message.ForumTopicEdited != nil:
		return "title"
	case len(message.NewChatPhoto) > 0 || message.DeleteChatPhoto || message.ChatBackgroundSet != nil:
		return "photo"
	case message.VideoChatStarted != nil || message.VideoChatEnded != nil || message.VideoChatParticipantsInvited != nil || message.VideoChatScheduled != nil:
		return "videochat"
	case message.GroupChatCreated || message.SupergroupChatCreated || message.ChannelChatCreated ||
		message.MessageAutoDeleteTimerChanged != nil || message.SuccessfulPayment != nil ||
		message.RefundedPayment != nil || message.ProximityAlertTriggered != nil ||
		message.WebAppData != nil || message.ChecklistTasksDone != nil ||
		message.ChecklistTasksAdded != nil || message.BoostAdded != nil:
		return "other"
	default:
		return ""
	}
}
