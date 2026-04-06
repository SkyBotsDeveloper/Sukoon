package owner

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

type Service struct {
	jobs *jobs.Service
}

func New(jobService *jobs.Service) *Service {
	return &Service{jobs: jobService}
}

func (s *Service) Handle(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "broadcast":
		return true, s.broadcast(ctx, rt)
	case "stats":
		return true, s.stats(ctx, rt)
	case "gban":
		return true, s.gban(ctx, rt, true)
	case "ungban":
		return true, s.gban(ctx, rt, false)
	case "bluser":
		return true, s.bluser(ctx, rt, true)
	case "unbluser":
		return true, s.bluser(ctx, rt, false)
	case "blchat":
		return true, s.blchat(ctx, rt, true)
	case "unblchat":
		return true, s.blchat(ctx, rt, false)
	case "addsudo":
		return true, s.sudo(ctx, rt, true)
	case "rmsudo":
		return true, s.sudo(ctx, rt, false)
	default:
		return false, nil
	}
}

func (s *Service) HandleMessage(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.ActorPermissions.IsOwner || rt.ActorPermissions.IsSudo {
		return false, nil
	}
	if rt.ChatID() == 0 {
		return false, nil
	}
	if _, blacklisted, err := rt.Store.GetGlobalBlacklistChat(ctx, rt.Bot.ID, rt.ChatID()); err != nil {
		return false, err
	} else if blacklisted {
		_ = rt.Client.LeaveChat(ctx, rt.ChatID())
		return true, nil
	}
	if rt.ActorID() == 0 {
		return false, nil
	}
	if _, blacklisted, err := rt.Store.GetGlobalBlacklistUser(ctx, rt.Bot.ID, rt.ActorID()); err != nil {
		return false, err
	} else if blacklisted {
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), nil, true); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) HandleJoin(ctx context.Context, rt *runtime.Context, user telegram.User) error {
	if _, blacklisted, err := rt.Store.GetGlobalBlacklistChat(ctx, rt.Bot.ID, rt.ChatID()); err != nil {
		return err
	} else if blacklisted {
		return rt.Client.LeaveChat(ctx, rt.ChatID())
	}
	if _, blacklisted, err := rt.Store.GetGlobalBlacklistUser(ctx, rt.Bot.ID, user.ID); err != nil {
		return err
	} else if blacklisted {
		return rt.Client.BanChatMember(ctx, rt.ChatID(), user.ID, nil, true)
	}
	return nil
}

func (s *Service) ensureOwnerOrSudo(rt *runtime.Context) error {
	if !rt.ActorPermissions.IsOwner && !rt.ActorPermissions.IsSudo {
		return fmt.Errorf("owner or sudo permission required")
	}
	return nil
}

func (s *Service) ensureOwner(rt *runtime.Context) error {
	if !rt.ActorPermissions.IsOwner {
		return fmt.Errorf("owner permission required")
	}
	return nil
}

func (s *Service) broadcast(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureOwnerOrSudo(rt); err != nil {
		return err
	}
	if s.jobs == nil {
		return fmt.Errorf("jobs service is not available")
	}
	if len(rt.Command.Args) == 0 {
		return s.broadcastStatus(ctx, rt)
	}
	if strings.EqualFold(rt.Command.Args[0], "status") {
		return s.broadcastStatus(ctx, rt)
	}
	mode := strings.ToLower(rt.Command.Args[0])
	if mode != "chats" && mode != "all" {
		return fmt.Errorf("usage: /broadcast <chats|all> <message> or /broadcast status [job_id]")
	}
	text := strings.TrimSpace(strings.TrimPrefix(rt.Command.RawArgs, rt.Command.Args[0]))
	if text == "" {
		return fmt.Errorf("broadcast text is required")
	}
	chats, err := rt.Store.ListChats(ctx, rt.Bot.ID)
	if err != nil {
		return err
	}
	job, err := s.jobs.Enqueue(ctx, rt.Bot.ID, jobs.KindBroadcast, rt.ActorID(), rt.ChatID(), jobs.BroadcastPayload{
		Mode: mode,
		Text: text,
	}, len(chats))
	if err != nil {
		return err
	}
	s.jobs.NotifyQueued(ctx, rt.Bot, rt.ChatID(), job, "Broadcast")
	return nil
}

func (s *Service) broadcastStatus(ctx context.Context, rt *runtime.Context) error {
	if len(rt.Command.Args) > 1 {
		job, err := rt.Store.GetJob(ctx, rt.Command.Args[1])
		if err != nil {
			return err
		}
		return s.sendJobStatus(ctx, rt, job)
	}
	jobsList, err := rt.Store.ListRecentJobs(ctx, rt.Bot.ID, 8)
	if err != nil {
		return err
	}
	lines := make([]string, 0, len(jobsList))
	for _, job := range jobsList {
		if job.Kind != jobs.KindBroadcast && job.Kind != jobs.KindGlobalBan && job.Kind != jobs.KindGlobalUnban {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s %d/%d", job.ID, job.Status, job.Progress, job.Total))
	}
	if len(lines) == 0 {
		lines = append(lines, "No recent owner jobs.")
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), telegram.SendMessageOptions{})
	return err
}

func (s *Service) stats(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureOwnerOrSudo(rt); err != nil {
		return err
	}
	stats, err := rt.Store.GetStats(ctx, rt.Bot.ID)
	if err != nil {
		return err
	}
	text := fmt.Sprintf(
		"Stats\nchats=%d\nusers=%d\nclones=%d\nfederations=%d\njobs=%d\ndead_jobs=%d",
		stats.ChatCount,
		stats.UserCount,
		stats.CloneCount,
		stats.FederationCount,
		stats.JobCount,
		stats.DeadJobCount,
	)
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{})
	return err
}

func (s *Service) gban(ctx context.Context, rt *runtime.Context, enabled bool) error {
	if err := s.ensureOwnerOrSudo(rt); err != nil {
		return err
	}
	if s.jobs == nil {
		return fmt.Errorf("jobs service is not available")
	}
	target, reason, err := targetAndReason(ctx, rt)
	if err != nil {
		return err
	}
	entry := domain.GlobalBlacklistUser{
		BotID:     rt.Bot.ID,
		UserID:    target.UserID,
		Reason:    reason,
		CreatedBy: rt.ActorID(),
	}
	if err := rt.Store.SetGlobalBlacklistUser(ctx, entry, enabled); err != nil {
		return err
	}
	chats, err := rt.Store.ListChats(ctx, rt.Bot.ID)
	if err != nil {
		return err
	}
	kind := jobs.KindGlobalBan
	description := "Global ban"
	if !enabled {
		kind = jobs.KindGlobalUnban
		description = "Global unban"
	}
	job, err := s.jobs.Enqueue(ctx, rt.Bot.ID, kind, rt.ActorID(), rt.ChatID(), jobs.GlobalBanPayload{
		UserID: target.UserID,
		Reason: reason,
	}, len(chats))
	if err != nil {
		return err
	}
	s.jobs.NotifyQueued(ctx, rt.Bot, rt.ChatID(), job, description+" for "+target.Name)
	return nil
}

func (s *Service) bluser(ctx context.Context, rt *runtime.Context, enabled bool) error {
	if err := s.ensureOwnerOrSudo(rt); err != nil {
		return err
	}
	target, reason, err := targetAndReason(ctx, rt)
	if err != nil {
		return err
	}
	if err := rt.Store.SetGlobalBlacklistUser(ctx, domain.GlobalBlacklistUser{
		BotID:     rt.Bot.ID,
		UserID:    target.UserID,
		Reason:    reason,
		CreatedBy: rt.ActorID(),
	}, enabled); err != nil {
		return err
	}
	text := "User globally blacklisted: " + target.Name
	if !enabled {
		text = "Global user blacklist removed: " + target.Name
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{})
	return err
}

func (s *Service) blchat(ctx context.Context, rt *runtime.Context, enabled bool) error {
	if err := s.ensureOwnerOrSudo(rt); err != nil {
		return err
	}
	chatID := rt.ChatID()
	reason := strings.TrimSpace(rt.Command.RawArgs)
	if len(rt.Command.Args) > 0 {
		if parsed, err := strconv.ParseInt(rt.Command.Args[0], 10, 64); err == nil {
			chatID = parsed
			if len(rt.Command.Args) > 1 {
				reason = strings.TrimSpace(strings.Join(rt.Command.Args[1:], " "))
			} else {
				reason = ""
			}
		}
	}
	if chatID == 0 {
		return fmt.Errorf("usage: /blchat [chat_id] [reason]")
	}
	if err := rt.Store.SetGlobalBlacklistChat(ctx, domain.GlobalBlacklistChat{
		BotID:     rt.Bot.ID,
		ChatID:    chatID,
		Reason:    reason,
		CreatedBy: rt.ActorID(),
	}, enabled); err != nil {
		return err
	}
	text := fmt.Sprintf("Chat %d blacklisted.", chatID)
	if !enabled {
		text = fmt.Sprintf("Chat %d removed from blacklist.", chatID)
	}
	if _, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{}); err != nil {
		return err
	}
	if enabled && chatID == rt.ChatID() {
		_ = rt.Client.LeaveChat(ctx, chatID)
	}
	return nil
}

func (s *Service) sudo(ctx context.Context, rt *runtime.Context, enabled bool) error {
	if err := s.ensureOwner(rt); err != nil {
		return err
	}
	target, _, err := targetAndReason(ctx, rt)
	if err != nil {
		return err
	}
	if err := rt.Store.SetBotRole(ctx, rt.Bot.ID, target.UserID, "sudo", enabled); err != nil {
		return err
	}
	text := "Added sudo: " + target.Name
	if !enabled {
		text = "Removed sudo: " + target.Name
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{})
	return err
}

func (s *Service) sendJobStatus(ctx context.Context, rt *runtime.Context, job domain.Job) error {
	text := fmt.Sprintf("Job %s\nkind=%s\nstatus=%s\nprogress=%d/%d", job.ID, job.Kind, job.Status, job.Progress, job.Total)
	if strings.TrimSpace(job.ResultSummary) != "" {
		text += "\nresult: " + job.ResultSummary
	}
	if strings.TrimSpace(job.Error) != "" {
		text += "\nerror: " + job.Error
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, telegram.SendMessageOptions{})
	return err
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
