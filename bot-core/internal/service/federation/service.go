package federation

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/util"
)

type Service struct {
	jobs *jobs.Service
}

func New(jobService *jobs.Service) *Service {
	return &Service{jobs: jobService}
}

func (s *Service) Handle(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "newfed":
		return true, s.newFed(ctx, rt)
	case "renamefed":
		return true, s.renameFed(ctx, rt)
	case "delfed":
		return true, s.delFed(ctx, rt)
	case "joinfed":
		return true, s.joinFed(ctx, rt)
	case "leavefed":
		return true, s.leaveFed(ctx, rt)
	case "fedinfo":
		return true, s.fedInfo(ctx, rt)
	case "chatfed":
		return true, s.chatFed(ctx, rt)
	case "fedadmins":
		return true, s.fedAdmins(ctx, rt)
	case "myfeds":
		return true, s.myFeds(ctx, rt)
	case "fedpromote":
		return true, s.fedPromote(ctx, rt, true)
	case "feddemote":
		return true, s.fedPromote(ctx, rt, false)
	case "feddemoteme":
		return true, s.fedDemoteMe(ctx, rt)
	case "fban":
		return true, s.fban(ctx, rt, true)
	case "unfban":
		return true, s.fban(ctx, rt, false)
	case "fedtransfer":
		return true, s.transfer(ctx, rt)
	case "fednotif":
		return true, s.fedNotif(ctx, rt)
	case "fedreason":
		return true, s.fedReason(ctx, rt)
	case "subfed":
		return true, s.subFed(ctx, rt, true)
	case "unsubfed":
		return true, s.subFed(ctx, rt, false)
	case "fedsubs":
		return true, s.fedSubs(ctx, rt)
	case "fedexport":
		return true, s.fedExport(ctx, rt)
	case "fedimport":
		return true, s.fedImport(ctx, rt)
	case "setfedlog":
		return true, s.setFedLog(ctx, rt, true)
	case "unsetfedlog":
		return true, s.setFedLog(ctx, rt, false)
	case "setfedlang":
		return true, s.setFedLang(ctx, rt)
	case "fedstat":
		return true, s.fedStat(ctx, rt)
	case "quietfed":
		return true, s.quietFed(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) HandleMessage(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.ActorPermissions.IsOwner || rt.ActorPermissions.IsSudo || rt.ActorPermissions.IsChatAdmin {
		return false, nil
	}
	federation, err := rt.Store.GetFederationByChat(ctx, rt.Bot.ID, rt.ChatID())
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, _, banned, err := s.effectiveFederationBan(ctx, rt, federation, rt.ActorID())
	if err != nil {
		return false, err
	}
	if banned {
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), nil, true); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) HandleJoin(ctx context.Context, rt *runtime.Context, user telegram.User) error {
	federation, err := rt.Store.GetFederationByChat(ctx, rt.Bot.ID, rt.ChatID())
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	sourceFederation, ban, banned, err := s.effectiveFederationBan(ctx, rt, federation, user.ID)
	if err != nil {
		return err
	}
	if banned {
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), user.ID, nil, true); err != nil {
			return err
		}
		quiet, err := rt.Store.GetFederationChatQuiet(ctx, federation.ID, rt.Bot.ID, rt.ChatID())
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if !quiet {
			reason := strings.TrimSpace(ban.Reason)
			if reason == "" {
				reason = "No reason provided."
			}
			_, _ = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Banned %s because they are fbanned in %s.\nReason: %s", serviceutil.DisplayName(user), sourceFederation.DisplayName, reason), telegram.SendMessageOptions{})
		}
	}
	return nil
}

func (s *Service) newFed(ctx context.Context, rt *runtime.Context) error {
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /newfed <fedname>")
	}
	if existing, err := rt.Store.GetFederationOwnedByUser(ctx, rt.Bot.ID, rt.ActorID()); err == nil && existing.ID != "" {
		_, sendErr := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("You already own federation %s (%s). Delete it with /delfed before creating another.", existing.DisplayName, existing.ShortName), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return sendErr
	} else if err != nil && err != pgx.ErrNoRows {
		return err
	}
	shortName := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	displayName := shortName
	if len(rt.Command.Args) > 1 {
		displayName = strings.TrimSpace(strings.Join(rt.Command.Args[1:], " "))
	}
	if len(displayName) > 64 {
		displayName = displayName[:64]
	}
	federation, err := rt.Store.CreateFederation(ctx, domain.Federation{
		ID:            util.RandomID(18),
		BotID:         rt.Bot.ID,
		ShortName:     shortName,
		DisplayName:   displayName,
		OwnerUserID:   rt.ActorID(),
		NotifyActions: true,
		LogLanguage:   "en",
	})
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Federation created: %s (%s)\nID: %s", federation.DisplayName, federation.ShortName, federation.ID), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) renameFed(ctx context.Context, rt *runtime.Context) error {
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /renamefed <short_name> [display name]")
	}
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	shortName := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	displayName := shortName
	if len(rt.Command.Args) > 1 {
		displayName = strings.TrimSpace(strings.Join(rt.Command.Args[1:], " "))
	}
	if err := rt.Store.RenameFederation(ctx, federation.ID, shortName, displayName); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Federation renamed to %s (%s).", displayName, shortName), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("Federation renamed by %d to %s (%s).", rt.ActorID(), displayName, shortName))
	}
	return err
}

func (s *Service) delFed(ctx context.Context, rt *runtime.Context) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if err := rt.Store.DeleteFederation(ctx, federation.ID); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Federation deleted.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) joinFed(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsOwner && !rt.ActorPermissions.IsSudo && !rt.ActorPermissions.IsChatCreator {
		return fmt.Errorf("chat creator rights required")
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /joinfed <federation>")
	}
	federation, err := s.resolveFederation(ctx, rt, rt.Command.Args[0])
	if err != nil {
		return err
	}
	if err := s.ensureFedAdmin(ctx, rt, federation); err != nil {
		return err
	}
	if err := rt.Store.JoinFederation(ctx, federation.ID, rt.Bot.ID, rt.ChatID()); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Joined federation %s.", federation.DisplayName), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("Chat %d joined federation %s.", rt.ChatID(), federation.ShortName))
	}
	return err
}

func (s *Service) leaveFed(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsOwner && !rt.ActorPermissions.IsSudo && !rt.ActorPermissions.IsChatCreator {
		return fmt.Errorf("chat creator rights required")
	}
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := rt.Store.LeaveFederation(ctx, federation.ID, rt.Bot.ID, rt.ChatID()); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Left federation %s.", federation.DisplayName), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("Chat %d left federation %s.", rt.ChatID(), federation.ShortName))
	}
	return err
}

func (s *Service) fedInfo(ctx context.Context, rt *runtime.Context) error {
	explicit := ""
	if len(rt.Command.Args) > 0 {
		explicit = rt.Command.Args[0]
	}
	federation, err := s.resolveFederation(ctx, rt, explicit)
	if err != nil {
		return err
	}
	chats, err := rt.Store.ListFederationChats(ctx, federation.ID)
	if err != nil {
		return err
	}
	admins, err := rt.Store.ListFederationAdmins(ctx, federation.ID)
	if err != nil {
		return err
	}
	text := fmt.Sprintf(
		"Federation %s\nID: %s\nShort name: %s\nOwner: %d\nChats: %d\nAdmins: %d\nFed notifications: %s\nFedban reason required: %s\nLog language: %s",
		federation.DisplayName,
		federation.ID,
		federation.ShortName,
		federation.OwnerUserID,
		len(chats),
		len(admins),
		onOff(federation.NotifyActions),
		onOff(federation.RequireReason),
		federation.LogLanguage,
	)
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) chatFed(ctx context.Context, rt *runtime.Context) error {
	federation, err := rt.Store.GetFederationByChat(ctx, rt.Bot.ID, rt.ChatID())
	if err == pgx.ErrNoRows {
		_, sendErr := rt.Client.SendMessage(ctx, rt.ChatID(), "This chat is not linked to a federation.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return sendErr
	}
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("This chat is linked to federation %s (%s).\nID: %s", federation.DisplayName, federation.ShortName, federation.ID), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) fedAdmins(ctx context.Context, rt *runtime.Context) error {
	explicit := ""
	if len(rt.Command.Args) > 0 {
		explicit = rt.Command.Args[0]
	}
	federation, err := s.resolveFederation(ctx, rt, explicit)
	if err != nil {
		return err
	}
	admins, err := rt.Store.ListFederationAdmins(ctx, federation.ID)
	if err != nil {
		return err
	}
	if len(admins) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No federation admins.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	lines := []string{"Federation admins for " + federation.DisplayName + ":"}
	for _, admin := range admins {
		name := fmt.Sprintf("%d", admin.UserID)
		if user, err := rt.Store.GetUserByID(ctx, admin.UserID); err == nil && user.ID != 0 {
			name = serviceutil.DisplayNameFromProfile(user)
		}
		lines = append(lines, fmt.Sprintf("- %s (%s)", name, admin.Role))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) myFeds(ctx context.Context, rt *runtime.Context) error {
	federations, err := rt.Store.ListFederationsForUser(ctx, rt.Bot.ID, rt.ActorID())
	if err != nil {
		return err
	}
	if len(federations) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "You are not managing any federations.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	lines := []string{"Federations you manage:"}
	for _, federation := range federations {
		lines = append(lines, fmt.Sprintf("- %s (%s)", federation.DisplayName, federation.ShortName))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) fedPromote(ctx context.Context, rt *runtime.Context, enabled bool) error {
	federation, target, err := s.resolveFedTarget(ctx, rt)
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if err := rt.Store.SetFederationAdmin(ctx, federation.ID, target.UserID, "admin", enabled); err != nil {
		return err
	}
	text := "Federation admin added: " + target.Name
	if !enabled {
		text = "Federation admin removed: " + target.Name
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("Federation admin update by %d: %s enabled=%t", rt.ActorID(), target.Name, enabled))
	}
	return err
}

func (s *Service) fedDemoteMe(ctx context.Context, rt *runtime.Context) error {
	explicit := ""
	if len(rt.Command.Args) > 0 {
		explicit = rt.Command.Args[0]
	}
	federation, err := s.resolveFederation(ctx, rt, explicit)
	if err != nil {
		return err
	}
	if federation.OwnerUserID == rt.ActorID() {
		return fmt.Errorf("federation owner cannot demote themselves")
	}
	if err := rt.Store.SetFederationAdmin(ctx, federation.ID, rt.ActorID(), "admin", false); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "You were removed from the federation admin list.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("Federation admin %d demoted themselves.", rt.ActorID()))
	}
	return err
}

func (s *Service) fban(ctx context.Context, rt *runtime.Context, enabled bool) error {
	if !rt.ActorPermissions.CanRestrictMembers {
		return fmt.Errorf("restrict permission required")
	}
	if s.jobs == nil {
		return fmt.Errorf("jobs service is not available")
	}
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedAdmin(ctx, rt, federation); err != nil {
		return err
	}
	target, reason, err := targetAndReason(ctx, rt)
	if err != nil {
		return err
	}
	if federation.RequireReason && strings.TrimSpace(reason) == "" && enabled {
		return fmt.Errorf("this federation requires a reason for fbans")
	}
	if err := rt.Store.SetFederationBan(ctx, domain.FederationBan{
		FederationID: federation.ID,
		UserID:       target.UserID,
		Reason:       reason,
		BannedBy:     rt.ActorID(),
	}, enabled); err != nil {
		return err
	}
	chats, err := rt.Store.ListFederationChats(ctx, federation.ID)
	if err != nil {
		return err
	}
	kind := jobs.KindFederationBan
	description := "Federation ban"
	if !enabled {
		kind = jobs.KindFederationUnban
		description = "Federation unban"
	}
	job, err := s.jobs.Enqueue(ctx, rt.Bot.ID, kind, rt.ActorID(), rt.ChatID(), jobs.FederationBanPayload{
		FederationID: federation.ID,
		UserID:       target.UserID,
		Reason:       reason,
	}, len(chats))
	if err != nil {
		return err
	}
	_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("%s queued by %d for %s. Reason: %s", description, rt.ActorID(), target.Name, emptyDash(reason)))
	s.jobs.NotifyQueued(ctx, rt.Bot, rt.ChatID(), job, description+" for "+target.Name)
	return nil
}

func (s *Service) transfer(ctx context.Context, rt *runtime.Context) error {
	federation, target, err := s.resolveFedTarget(ctx, rt)
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if existing, err := rt.Store.GetFederationOwnedByUser(ctx, rt.Bot.ID, target.UserID); err == nil && existing.ID != "" && existing.ID != federation.ID {
		return fmt.Errorf("target user already owns a federation")
	} else if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if err := rt.Store.TransferFederation(ctx, federation.ID, target.UserID); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Federation ownership transferred to "+target.Name+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		updated, loadErr := rt.Store.GetFederationByID(ctx, federation.ID)
		if loadErr == nil {
			federation = updated
		}
		_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("Federation ownership transferred to %s by %d.", target.Name, rt.ActorID()))
	}
	return err
}

func (s *Service) fedNotif(ctx context.Context, rt *runtime.Context) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Federation PM notifications are "+onOff(federation.NotifyActions)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	federation.NotifyActions = enabled
	if err := rt.Store.UpdateFederationSettings(ctx, federation.ID, federation.NotifyActions, federation.RequireReason, federation.LogChatID, federation.LogLanguage); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Federation PM notifications "+onOff(enabled)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) fedReason(ctx context.Context, rt *runtime.Context) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Federation ban reasons are "+onOff(federation.RequireReason)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	federation.RequireReason = enabled
	if err := rt.Store.UpdateFederationSettings(ctx, federation.ID, federation.NotifyActions, federation.RequireReason, federation.LogChatID, federation.LogLanguage); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Federation ban reason requirement "+onOff(enabled)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) subFed(ctx context.Context, rt *runtime.Context, enabled bool) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /subfed <FedID>")
	}
	target, err := s.resolveFederation(ctx, rt, rt.Command.Args[0])
	if err != nil {
		return err
	}
	if target.ID == federation.ID {
		return fmt.Errorf("cannot subscribe a federation to itself")
	}
	if err := rt.Store.SetFederationSubscription(ctx, federation.ID, target.ID, enabled); err != nil {
		return err
	}
	action := "Subscribed to"
	if !enabled {
		action = "Unsubscribed from"
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s federation %s (%s).", action, target.DisplayName, target.ShortName), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("%s federation %s by %d.", strings.ToLower(action), target.ShortName, rt.ActorID()))
	}
	return err
}

func (s *Service) fedSubs(ctx context.Context, rt *runtime.Context) error {
	explicit := ""
	if len(rt.Command.Args) > 0 {
		explicit = rt.Command.Args[0]
	}
	federation, err := s.resolveFederation(ctx, rt, explicit)
	if err != nil {
		return err
	}
	subs, err := rt.Store.ListFederationSubscriptions(ctx, federation.ID)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "This federation is not subscribed to any other federation.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	lines := []string{"Federations subscribed by " + federation.DisplayName + ":"}
	for _, sub := range subs {
		lines = append(lines, fmt.Sprintf("- %s (%s)", sub.DisplayName, sub.ShortName))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) fedExport(ctx context.Context, rt *runtime.Context) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	format := "csv"
	if len(rt.Command.Args) > 0 {
		format = strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	}
	bans, err := rt.Store.ListFederationBans(ctx, federation.ID)
	if err != nil {
		return err
	}
	text, err := renderFedExport(format, bans)
	if err != nil {
		return err
	}
	if text == "" {
		text = "No users are banned in this federation."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) fedImport(ctx context.Context, rt *runtime.Context) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if len(rt.Command.Args) < 2 {
		return fmt.Errorf("usage: /fedimport <overwrite|keep> <csv|minicsv|json|human> [data]")
	}
	mode := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	format := strings.ToLower(strings.TrimSpace(rt.Command.Args[1]))
	if mode != "overwrite" && mode != "keep" {
		return fmt.Errorf("usage: /fedimport <overwrite|keep> <csv|minicsv|json|human> [data]")
	}
	source := strings.TrimSpace(strings.Join(rt.Command.Args[2:], " "))
	if source == "" && rt.Message != nil && rt.Message.ReplyToMessage != nil {
		source = strings.TrimSpace(rt.Message.ReplyToMessage.Text)
	}
	if source == "" {
		return fmt.Errorf("reply to exported data or pass data after the format")
	}
	bans, err := parseFedImport(format, source, federation.ID, rt.ActorID())
	if err != nil {
		return err
	}
	if len(bans) > 100 {
		return fmt.Errorf("fedimport accepts at most 100 users per command")
	}
	if mode == "overwrite" {
		existing, err := rt.Store.ListFederationBans(ctx, federation.ID)
		if err != nil {
			return err
		}
		for _, ban := range existing {
			if err := rt.Store.SetFederationBan(ctx, ban, false); err != nil {
				return err
			}
		}
	}
	for _, ban := range bans {
		if err := rt.Store.SetFederationBan(ctx, ban, true); err != nil {
			return err
		}
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Imported %d federation bans.", len(bans)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = s.notifyFederationEvent(ctx, rt, federation, fmt.Sprintf("Imported %d federation bans by %d.", len(bans), rt.ActorID()))
	}
	return err
}

func (s *Service) setFedLog(ctx context.Context, rt *runtime.Context, enabled bool) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if enabled {
		chatID := rt.ChatID()
		federation.LogChatID = &chatID
	} else {
		federation.LogChatID = nil
	}
	if err := rt.Store.UpdateFederationSettings(ctx, federation.ID, federation.NotifyActions, federation.RequireReason, federation.LogChatID, federation.LogLanguage); err != nil {
		return err
	}
	text := "Federation log unset."
	if enabled {
		text = "Federation log set to this chat."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setFedLang(ctx context.Context, rt *runtime.Context) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedOwner(ctx, rt, federation); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Federation log language: "+federation.LogLanguage+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	language := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	if len(language) > 16 {
		return fmt.Errorf("language code is too long")
	}
	federation.LogLanguage = language
	if err := rt.Store.UpdateFederationSettings(ctx, federation.ID, federation.NotifyActions, federation.RequireReason, federation.LogChatID, federation.LogLanguage); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Federation log language set to "+language+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) fedStat(ctx context.Context, rt *runtime.Context) error {
	var (
		targetID = rt.ActorID()
		name     = "you"
		fedRef   string
	)
	switch len(rt.Command.Args) {
	case 0:
	case 1:
		if looksLikeUserRef(rt.Command.Args[0]) {
			target, err := serviceutil.ResolveTarget(ctx, rt, rt.Command.Args)
			if err != nil {
				return err
			}
			targetID = target.UserID
			name = target.Name
		} else {
			fedRef = rt.Command.Args[0]
		}
	default:
		target, err := serviceutil.ResolveTarget(ctx, rt, rt.Command.Args[:1])
		if err != nil {
			return err
		}
		targetID = target.UserID
		name = target.Name
		fedRef = rt.Command.Args[1]
	}
	if fedRef != "" {
		federation, err := s.resolveFederation(ctx, rt, fedRef)
		if err != nil {
			return err
		}
		ban, banned, err := rt.Store.GetFederationBan(ctx, federation.ID, targetID)
		if err != nil {
			return err
		}
		if !banned {
			_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s is not fbanned in %s.", name, federation.DisplayName), rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s is fbanned in %s.\nReason: %s", name, federation.DisplayName, emptyDash(ban.Reason)), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	bans, err := rt.Store.ListFederationBansForUser(ctx, targetID)
	if err != nil {
		return err
	}
	if len(bans) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s has no federation bans.", name), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	lines := []string{fmt.Sprintf("Federation bans for %s:", name)}
	for _, ban := range bans {
		federation, err := rt.Store.GetFederationByID(ctx, ban.FederationID)
		label := ban.FederationID
		if err == nil {
			label = federation.DisplayName + " (" + federation.ShortName + ")"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", label, emptyDash(ban.Reason)))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) quietFed(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsOwner && !rt.ActorPermissions.IsSudo && !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("chat admin rights required")
	}
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		enabled, err := rt.Store.GetFederationChatQuiet(ctx, federation.ID, rt.Bot.ID, rt.ChatID())
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		_, sendErr := rt.Client.SendMessage(ctx, rt.ChatID(), "Quiet federation notifications are "+onOff(enabled)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return sendErr
	}
	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	if err := rt.Store.SetFederationChatQuiet(ctx, federation.ID, rt.Bot.ID, rt.ChatID(), enabled); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Quiet federation notifications "+onOff(enabled)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) resolveFederation(ctx context.Context, rt *runtime.Context, explicit string) (domain.Federation, error) {
	ref := strings.TrimSpace(explicit)
	if ref == "" {
		federation, err := rt.Store.GetFederationByChat(ctx, rt.Bot.ID, rt.ChatID())
		if err == nil {
			return federation, nil
		}
		if err != pgx.ErrNoRows {
			return domain.Federation{}, err
		}
		if owned, ownedErr := rt.Store.GetFederationOwnedByUser(ctx, rt.Bot.ID, rt.ActorID()); ownedErr == nil {
			return owned, nil
		}
		return domain.Federation{}, fmt.Errorf("this chat is not linked to a federation")
	}
	if federation, err := rt.Store.GetFederationByID(ctx, ref); err == nil {
		return federation, nil
	}
	federation, err := rt.Store.GetFederationByShortName(ctx, rt.Bot.ID, ref)
	if err != nil {
		return domain.Federation{}, fmt.Errorf("federation not found")
	}
	return federation, nil
}

func (s *Service) effectiveFederationBan(ctx context.Context, rt *runtime.Context, federation domain.Federation, userID int64) (domain.Federation, domain.FederationBan, bool, error) {
	if ban, banned, err := rt.Store.GetFederationBan(ctx, federation.ID, userID); err != nil {
		return domain.Federation{}, domain.FederationBan{}, false, err
	} else if banned {
		return federation, ban, true, nil
	}
	subs, err := rt.Store.ListFederationSubscriptions(ctx, federation.ID)
	if err != nil {
		return domain.Federation{}, domain.FederationBan{}, false, err
	}
	for _, sub := range subs {
		ban, banned, err := rt.Store.GetFederationBan(ctx, sub.ID, userID)
		if err != nil {
			return domain.Federation{}, domain.FederationBan{}, false, err
		}
		if banned {
			return sub, ban, true, nil
		}
	}
	return domain.Federation{}, domain.FederationBan{}, false, nil
}

func (s *Service) notifyFederationEvent(ctx context.Context, rt *runtime.Context, federation domain.Federation, text string) error {
	if federation.LogChatID != nil {
		_, _ = rt.Client.SendMessage(ctx, *federation.LogChatID, text, telegram.SendMessageOptions{})
	}
	if federation.NotifyActions && federation.OwnerUserID != 0 && federation.OwnerUserID != rt.ActorID() {
		_, _ = rt.Client.SendMessage(ctx, federation.OwnerUserID, text, telegram.SendMessageOptions{})
	}
	return nil
}

func (s *Service) ensureFedOwner(ctx context.Context, rt *runtime.Context, federation domain.Federation) error {
	if rt.ActorPermissions.IsOwner || rt.ActorPermissions.IsSudo {
		return nil
	}
	if federation.OwnerUserID != rt.ActorID() {
		return fmt.Errorf("federation owner required")
	}
	return nil
}

func (s *Service) ensureFedAdmin(ctx context.Context, rt *runtime.Context, federation domain.Federation) error {
	if rt.ActorPermissions.IsOwner || rt.ActorPermissions.IsSudo || federation.OwnerUserID == rt.ActorID() {
		return nil
	}
	admins, err := rt.Store.ListFederationAdmins(ctx, federation.ID)
	if err != nil {
		return err
	}
	for _, admin := range admins {
		if admin.UserID == rt.ActorID() {
			return nil
		}
	}
	return fmt.Errorf("federation admin required")
}

func (s *Service) resolveFedTarget(ctx context.Context, rt *runtime.Context) (domain.Federation, serviceutil.Target, error) {
	var (
		federation domain.Federation
		target     serviceutil.Target
		err        error
	)
	if current, currentErr := rt.Store.GetFederationByChat(ctx, rt.Bot.ID, rt.ChatID()); currentErr == nil {
		federation = current
		target, err = serviceutil.ResolveTarget(ctx, rt, rt.Command.Args)
		if err != nil {
			return domain.Federation{}, serviceutil.Target{}, err
		}
		return federation, target, nil
	}
	if len(rt.Command.Args) < 2 {
		return domain.Federation{}, serviceutil.Target{}, fmt.Errorf("usage: /command <federation> <user>")
	}
	federation, err = s.resolveFederation(ctx, rt, rt.Command.Args[0])
	if err != nil {
		return domain.Federation{}, serviceutil.Target{}, err
	}
	target, err = serviceutil.ResolveTarget(ctx, rt, rt.Command.Args[1:])
	if err != nil {
		return domain.Federation{}, serviceutil.Target{}, err
	}
	return federation, target, nil
}

func targetAndReason(ctx context.Context, rt *runtime.Context) (serviceutil.Target, string, error) {
	target, err := serviceutil.ResolveTarget(ctx, rt, rt.Command.Args)
	if err != nil {
		return serviceutil.Target{}, "", err
	}
	if rt.Message != nil && rt.Message.ReplyToMessage != nil {
		return target, strings.TrimSpace(strings.Join(rt.Command.Args, " ")), nil
	}
	if len(rt.Command.Args) <= 1 {
		return target, "", nil
	}
	return target, strings.TrimSpace(strings.Join(rt.Command.Args[1:], " ")), nil
}

func parseToggle(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "yes", "enable", "enabled", "true":
		return true, nil
	case "off", "no", "disable", "disabled", "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected yes/no/on/off")
	}
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func looksLikeUserRef(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "@") {
		return true
	}
	_, err := strconv.ParseInt(value, 10, 64)
	return err == nil
}

func renderFedExport(format string, bans []domain.FederationBan) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "csv":
		var b strings.Builder
		w := csv.NewWriter(&b)
		_ = w.Write([]string{"user_id", "reason", "banned_by", "banned_at"})
		for _, ban := range bans {
			_ = w.Write([]string{
				strconv.FormatInt(ban.UserID, 10),
				ban.Reason,
				strconv.FormatInt(ban.BannedBy, 10),
				ban.BannedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
		w.Flush()
		return b.String(), w.Error()
	case "minicsv":
		var b strings.Builder
		w := csv.NewWriter(&b)
		for _, ban := range bans {
			_ = w.Write([]string{strconv.FormatInt(ban.UserID, 10), ban.Reason})
		}
		w.Flush()
		return b.String(), w.Error()
	case "json":
		payload, err := json.MarshalIndent(bans, "", "  ")
		if err != nil {
			return "", err
		}
		return string(payload), nil
	case "human":
		lines := make([]string, 0, len(bans))
		for _, ban := range bans {
			lines = append(lines, fmt.Sprintf("- %d: %s", ban.UserID, emptyDash(ban.Reason)))
		}
		return strings.Join(lines, "\n"), nil
	default:
		return "", fmt.Errorf("supported export formats: csv, minicsv, json, human")
	}
}

func parseFedImport(format string, source string, federationID string, actorID int64) ([]domain.FederationBan, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv", "minicsv":
		reader := csv.NewReader(strings.NewReader(source))
		reader.FieldsPerRecord = -1
		records, err := reader.ReadAll()
		if err != nil {
			return nil, err
		}
		var bans []domain.FederationBan
		for idx, record := range records {
			if len(record) == 0 {
				continue
			}
			if idx == 0 && strings.EqualFold(strings.TrimSpace(record[0]), "user_id") {
				continue
			}
			userID, err := strconv.ParseInt(strings.TrimSpace(record[0]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid user id %q", record[0])
			}
			reason := ""
			if len(record) > 1 {
				reason = strings.TrimSpace(record[1])
			}
			bans = append(bans, domain.FederationBan{
				FederationID: federationID,
				UserID:       userID,
				Reason:       reason,
				BannedBy:     actorID,
			})
		}
		return bans, nil
	case "json":
		var bans []domain.FederationBan
		if err := json.Unmarshal([]byte(source), &bans); err != nil {
			return nil, err
		}
		for idx := range bans {
			bans[idx].FederationID = federationID
			if bans[idx].BannedBy == 0 {
				bans[idx].BannedBy = actorID
			}
		}
		return bans, nil
	case "human":
		lines := strings.Split(source, "\n")
		var bans []domain.FederationBan
		for _, line := range lines {
			line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
			if line == "" {
				continue
			}
			userPart, reason, _ := strings.Cut(line, ":")
			userID, err := strconv.ParseInt(strings.TrimSpace(userPart), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid user id %q", userPart)
			}
			bans = append(bans, domain.FederationBan{
				FederationID: federationID,
				UserID:       userID,
				Reason:       strings.TrimSpace(reason),
				BannedBy:     actorID,
			})
		}
		return bans, nil
	default:
		return nil, fmt.Errorf("supported import formats: csv, minicsv, json, human")
	}
}
