package federation

import (
	"context"
	"fmt"
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
	case "delfed":
		return true, s.delFed(ctx, rt)
	case "joinfed":
		return true, s.joinFed(ctx, rt)
	case "leavefed":
		return true, s.leaveFed(ctx, rt)
	case "fedinfo":
		return true, s.fedInfo(ctx, rt)
	case "fedadmins":
		return true, s.fedAdmins(ctx, rt)
	case "myfeds":
		return true, s.myFeds(ctx, rt)
	case "fedpromote":
		return true, s.fedPromote(ctx, rt, true)
	case "feddemote":
		return true, s.fedPromote(ctx, rt, false)
	case "fban":
		return true, s.fban(ctx, rt, true)
	case "unfban":
		return true, s.fban(ctx, rt, false)
	case "fedtransfer":
		return true, s.transfer(ctx, rt)
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
	if _, banned, err := rt.Store.GetFederationBan(ctx, federation.ID, rt.ActorID()); err != nil {
		return false, err
	} else if banned {
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
	if _, banned, err := rt.Store.GetFederationBan(ctx, federation.ID, user.ID); err != nil {
		return err
	} else if banned {
		return rt.Client.BanChatMember(ctx, rt.ChatID(), user.ID, nil, true)
	}
	return nil
}

func (s *Service) newFed(ctx context.Context, rt *runtime.Context) error {
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /newfed <short_name> [display name]")
	}
	shortName := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	displayName := shortName
	if len(rt.Command.Args) > 1 {
		displayName = strings.TrimSpace(strings.Join(rt.Command.Args[1:], " "))
	}
	federation, err := rt.Store.CreateFederation(ctx, domain.Federation{
		ID:          util.RandomID(18),
		BotID:       rt.Bot.ID,
		ShortName:   shortName,
		DisplayName: displayName,
		OwnerUserID: rt.ActorID(),
	})
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Federation created: %s (%s)", federation.DisplayName, federation.ShortName), telegram.SendMessageOptions{})
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
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Federation deleted.", telegram.SendMessageOptions{})
	return err
}

func (s *Service) joinFed(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("chat admin rights required")
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
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Joined federation %s.", federation.DisplayName), telegram.SendMessageOptions{})
	return err
}

func (s *Service) leaveFed(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("chat admin rights required")
	}
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	if err := s.ensureFedAdmin(ctx, rt, federation); err != nil {
		return err
	}
	if err := rt.Store.LeaveFederation(ctx, federation.ID, rt.Bot.ID, rt.ChatID()); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Left federation %s.", federation.DisplayName), telegram.SendMessageOptions{})
	return err
}

func (s *Service) fedInfo(ctx context.Context, rt *runtime.Context) error {
	federation, err := s.resolveFederation(ctx, rt, "")
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
		"Federation %s\nid=%s\nshort=%s\nowner=%d\nchats=%d\nadmins=%d",
		federation.DisplayName,
		federation.ID,
		federation.ShortName,
		federation.OwnerUserID,
		len(chats),
		len(admins),
	)
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{})
	return err
}

func (s *Service) fedAdmins(ctx context.Context, rt *runtime.Context) error {
	federation, err := s.resolveFederation(ctx, rt, "")
	if err != nil {
		return err
	}
	admins, err := rt.Store.ListFederationAdmins(ctx, federation.ID)
	if err != nil {
		return err
	}
	if len(admins) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No federation admins.", telegram.SendMessageOptions{})
		return err
	}
	lines := make([]string, 0, len(admins))
	for _, admin := range admins {
		name := fmt.Sprintf("%d", admin.UserID)
		if user, err := rt.Store.GetUserByID(ctx, admin.UserID); err == nil && user.ID != 0 {
			name = serviceutil.DisplayNameFromProfile(user)
		}
		lines = append(lines, fmt.Sprintf("%s (%s)", name, admin.Role))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), telegram.SendMessageOptions{})
	return err
}

func (s *Service) myFeds(ctx context.Context, rt *runtime.Context) error {
	federations, err := rt.Store.ListFederationsForUser(ctx, rt.Bot.ID, rt.ActorID())
	if err != nil {
		return err
	}
	if len(federations) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "You are not managing any federations.", telegram.SendMessageOptions{})
		return err
	}
	lines := make([]string, 0, len(federations))
	for _, federation := range federations {
		lines = append(lines, fmt.Sprintf("%s (%s)", federation.DisplayName, federation.ShortName))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), telegram.SendMessageOptions{})
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
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{})
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
	if err := rt.Store.TransferFederation(ctx, federation.ID, target.UserID); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Federation ownership transferred to "+target.Name+".", telegram.SendMessageOptions{})
	return err
}

func (s *Service) resolveFederation(ctx context.Context, rt *runtime.Context, explicit string) (domain.Federation, error) {
	ref := strings.TrimSpace(explicit)
	if ref == "" {
		federation, err := rt.Store.GetFederationByChat(ctx, rt.Bot.ID, rt.ChatID())
		if err != nil {
			return domain.Federation{}, fmt.Errorf("this chat is not linked to a federation")
		}
		return federation, nil
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
