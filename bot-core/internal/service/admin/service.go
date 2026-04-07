package admin

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
	case "approval":
		return true, s.approvalStatus(ctx, rt)
	case "approve":
		return true, s.approve(ctx, rt, true)
	case "unapprove":
		return true, s.approve(ctx, rt, false)
	case "approved":
		return true, s.listApproved(ctx, rt)
	case "unapproveall":
		return true, s.unapproveAll(ctx, rt)
	case "disable":
		return true, s.disable(ctx, rt, true)
	case "enable":
		return true, s.disable(ctx, rt, false)
	case "disabled":
		return true, s.listDisabled(ctx, rt)
	case "logchannel", "setlog", "unsetlog", "log", "nolog":
		return true, s.logChannel(ctx, rt)
	case "logcategories":
		return true, s.logCategories(ctx, rt)
	case "reports":
		return true, s.reports(ctx, rt)
	case "report":
		return true, s.report(ctx, rt)
	case "admins", "adminlist":
		return true, s.admins(ctx, rt)
	case "cleancommands", "cleancommand", "keepcommand":
		return true, s.cleanCommands(ctx, rt)
	case "cleancommandtypes":
		return true, s.cleanCommandTypes(ctx, rt)
	case "cleanservice":
		return true, s.cleanService(ctx, rt)
	case "nocleanservice":
		return true, s.noCleanService(ctx, rt)
	case "cleanservicetypes":
		return true, s.cleanServiceTypes(ctx, rt)
	case "purge":
		return true, s.purge(ctx, rt)
	case "del":
		return true, s.del(ctx, rt)
	case "pin":
		return true, s.pin(ctx, rt)
	case "unpin":
		return true, s.unpin(ctx, rt)
	case "unpinall":
		return true, s.unpinAll(ctx, rt)
	case "mod":
		return true, s.mod(ctx, rt, true)
	case "unmod":
		return true, s.mod(ctx, rt, false)
	case "muter":
		return true, s.muter(ctx, rt, true)
	case "unmuter":
		return true, s.muter(ctx, rt, false)
	case "mods":
		return true, s.listMods(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) ensureChatAdmin(rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	return nil
}

func (s *Service) ensureDeletePerm(rt *runtime.Context) error {
	if !rt.ActorPermissions.CanDeleteMessages {
		return fmt.Errorf("delete messages permission required")
	}
	return nil
}

func (s *Service) ensurePromotePerm(rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	if !rt.ActorPermissions.CanPromoteMembers {
		return fmt.Errorf("add admins permission required")
	}
	return nil
}

func (s *Service) ensurePinPerm(rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	if !rt.ActorPermissions.CanPinMessages {
		return fmt.Errorf("pin messages permission required")
	}
	return nil
}

func (s *Service) approve(ctx context.Context, rt *runtime.Context, approved bool) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	target, _, err := moderationTarget(ctx, rt)
	if err != nil {
		return err
	}
	if err := rt.Store.SetApproval(ctx, rt.Bot.ID, rt.ChatID(), target.UserID, rt.ActorID(), approved); err != nil {
		return err
	}
	text := "Approved " + target.Name + "."
	if !approved {
		text = "Removed approval for " + target.Name + "."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) approvalStatus(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	target, _, err := moderationTarget(ctx, rt)
	if err != nil {
		return fmt.Errorf("usage: /approval <reply|user>")
	}
	approved, err := rt.Store.IsApproved(ctx, rt.Bot.ID, rt.ChatID(), target.UserID)
	if err != nil {
		return err
	}
	text := target.Name + " is not approved."
	if approved {
		text = target.Name + " is approved."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) listApproved(ctx context.Context, rt *runtime.Context) error {
	approvedUsers, err := rt.Store.ListApprovedUsers(ctx, rt.Bot.ID, rt.ChatID())
	if err != nil {
		return err
	}
	if len(approvedUsers) == 0 {
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "No approved users.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	parts := make([]string, 0, len(approvedUsers))
	for _, userID := range approvedUsers {
		parts = append(parts, strconv.FormatInt(userID, 10))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Approved users: "+strings.Join(parts, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) unapproveAll(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	approvedUsers, err := rt.Store.ListApprovedUsers(ctx, rt.Bot.ID, rt.ChatID())
	if err != nil {
		return err
	}
	if len(approvedUsers) == 0 {
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "No approved users to remove.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	for _, userID := range approvedUsers {
		if err := rt.Store.SetApproval(ctx, rt.Bot.ID, rt.ChatID(), userID, rt.ActorID(), false); err != nil {
			return err
		}
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Removed approvals for %d user(s).", len(approvedUsers)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) disable(ctx context.Context, rt *runtime.Context, disabled bool) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /disable <command>")
	}
	command := strings.TrimPrefix(strings.ToLower(rt.Command.Args[0]), "/")
	if err := rt.Store.SetDisabledCommand(ctx, rt.Bot.ID, rt.ChatID(), command, disabled, rt.ActorID()); err != nil {
		return err
	}
	action := "Disabled"
	if !disabled {
		action = "Enabled"
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s /%s.", action, command), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) listDisabled(ctx context.Context, rt *runtime.Context) error {
	if len(rt.RuntimeBundle.DisabledCommands) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No disabled commands.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	commands := make([]string, 0, len(rt.RuntimeBundle.DisabledCommands))
	for command := range rt.RuntimeBundle.DisabledCommands {
		commands = append(commands, "/"+command)
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Disabled: "+strings.Join(commands, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) logChannel(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	switch rt.Command.Name {
	case "unsetlog", "nolog":
		if err := rt.Store.SetLogChannel(ctx, rt.Bot.ID, rt.ChatID(), nil); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Log channel disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	case "setlog":
		if len(rt.Command.Args) == 0 {
			return fmt.Errorf("usage: /setlog <chat_id>")
		}
	}
	if len(rt.Command.Args) == 0 {
		if rt.RuntimeBundle.Settings.LogChannelID == nil {
			_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No log channel configured.", rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Current log channel: %d", *rt.RuntimeBundle.Settings.LogChannelID), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}

	arg := strings.ToLower(rt.Command.Args[0])
	if arg == "off" {
		if err := rt.Store.SetLogChannel(ctx, rt.Bot.ID, rt.ChatID(), nil); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Log channel disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}

	channelID, err := strconv.ParseInt(rt.Command.Args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("usage: /logchannel <chat_id|off>")
	}
	if err := rt.Store.SetLogChannel(ctx, rt.Bot.ID, rt.ChatID(), &channelID); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Log channel set to %d.", channelID), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) logCategories(ctx context.Context, rt *runtime.Context) error {
	text := "Current log categories: moderation, antispam, antiabuse, antibio, reports."
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) reports(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.Settings.ReportsEnabled {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Reports are currently "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := ParseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	if err := rt.Store.SetReportsEnabled(ctx, rt.Bot.ID, rt.ChatID(), enabled); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Reports %s.", toggleWord(enabled)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) report(ctx context.Context, rt *runtime.Context) error {
	if !rt.RuntimeBundle.Settings.ReportsEnabled {
		return fmt.Errorf("reports are disabled")
	}
	if rt.RuntimeBundle.Settings.LogChannelID == nil {
		return fmt.Errorf("log channel is not configured")
	}
	target, reason, err := moderationTarget(ctx, rt)
	if err != nil {
		return err
	}
	reportText := fmt.Sprintf("Report from %d against %s in %d. %s", rt.ActorID(), target.Name, rt.ChatID(), strings.TrimSpace(reason))
	_, err = rt.Client.SendMessage(ctx, *rt.RuntimeBundle.Settings.LogChannelID, reportText, telegram.SendMessageOptions{})
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Report sent.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) admins(ctx context.Context, rt *runtime.Context) error {
	admins, err := rt.Client.GetChatAdministrators(ctx, rt.ChatID())
	if err != nil {
		return err
	}
	if len(admins) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No visible chat admins.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	parts := make([]string, 0, len(admins))
	for _, admin := range admins {
		if admin.IsAnonymous {
			parts = append(parts, "Anonymous admin")
			continue
		}
		label := serviceutil.DisplayName(admin.User)
		if admin.Status == "creator" {
			label += " [owner]"
		}
		parts = append(parts, label)
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Chat admins: "+strings.Join(parts, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) cleanCommands(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	if rt.Command.Name == "keepcommand" {
		if err := rt.Store.SetCleanCommands(ctx, rt.Bot.ID, rt.ChatID(), false); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Clean commands disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.Settings.CleanCommands {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Clean commands is "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := ParseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	if err := rt.Store.SetCleanCommands(ctx, rt.Bot.ID, rt.ChatID(), enabled); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Clean commands %s.", toggleWord(enabled)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) cleanCommandTypes(ctx context.Context, rt *runtime.Context) error {
	text := "Clean command types: command messages only in the current build. Service-message cleanup is configured separately with /cleanservice."
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) cleanService(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Clean service: join=%t leave=%t pin=%t title=%t photo=%t other=%t videochat=%t", rt.RuntimeBundle.Settings.CleanServiceJoin, rt.RuntimeBundle.Settings.CleanServiceLeave, rt.RuntimeBundle.Settings.CleanServicePin, rt.RuntimeBundle.Settings.CleanServiceTitle, rt.RuntimeBundle.Settings.CleanServicePhoto, rt.RuntimeBundle.Settings.CleanServiceOther, rt.RuntimeBundle.Settings.CleanServiceVideoChat), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	if len(rt.Command.Args) == 1 {
		if enabled, err := ParseToggle(rt.Command.Args[0]); err == nil {
			if err := rt.Store.SetCleanService(ctx, rt.Bot.ID, rt.ChatID(), "all", enabled); err != nil {
				return err
			}
			_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Clean service all set to %s.", toggleWord(enabled)), rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}
	}
	targets := rt.Command.Args
	enabled := true
	if len(rt.Command.Args) > 1 {
		if toggled, err := ParseToggle(rt.Command.Args[len(rt.Command.Args)-1]); err == nil {
			enabled = toggled
			targets = rt.Command.Args[:len(rt.Command.Args)-1]
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("usage: /cleanservice <on|off|join|leave|pin|title|photo|other|videochat|all> [on|off]")
	}
	for _, target := range targets {
		if err := rt.Store.SetCleanService(ctx, rt.Bot.ID, rt.ChatID(), strings.ToLower(target), enabled); err != nil {
			return err
		}
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Clean service %s set to %s.", strings.Join(targets, ", "), toggleWord(enabled)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) noCleanService(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureChatAdmin(rt); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /nocleanservice <join|leave|pin|title|photo|other|videochat|all>")
	}
	for _, target := range rt.Command.Args {
		if err := rt.Store.SetCleanService(ctx, rt.Bot.ID, rt.ChatID(), strings.ToLower(target), false); err != nil {
			return err
		}
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Requested cleanservice types were disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) cleanServiceTypes(ctx context.Context, rt *runtime.Context) error {
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Cleanservice types: all, join, leave, pin, title, photo, other, videochat", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) purge(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureDeletePerm(rt); err != nil {
		return err
	}
	if s.jobs == nil {
		return fmt.Errorf("jobs service is not available")
	}
	if len(rt.Command.Args) > 0 && strings.EqualFold(rt.Command.Args[0], "status") {
		return s.jobStatus(ctx, rt, jobs.KindPurge)
	}

	var fromMessageID int64
	var toMessageID int64
	switch {
	case rt.Message != nil && rt.Message.ReplyToMessage != nil:
		fromMessageID = rt.Message.ReplyToMessage.MessageID
		toMessageID = rt.Message.MessageID
	case len(rt.Command.Args) > 0:
		count, err := strconv.Atoi(rt.Command.Args[0])
		if err != nil || count < 1 {
			return fmt.Errorf("usage: /purge <count> or reply to a message")
		}
		fromMessageID = rt.Message.MessageID - int64(count) + 1
		if fromMessageID < 1 {
			fromMessageID = 1
		}
		toMessageID = rt.Message.MessageID
	default:
		return fmt.Errorf("usage: /purge <count> or reply to a message")
	}

	total := int(toMessageID-fromMessageID) + 1
	job, err := s.jobs.Enqueue(ctx, rt.Bot.ID, jobs.KindPurge, rt.ActorID(), rt.ChatID(), jobs.PurgePayload{
		ChatID:        rt.ChatID(),
		FromMessageID: fromMessageID,
		ToMessageID:   toMessageID,
	}, total)
	if err != nil {
		return err
	}
	s.jobs.NotifyQueued(ctx, rt.Bot, rt.ChatID(), job, fmt.Sprintf("Purge of %d message(s)", total))
	return nil
}

func (s *Service) del(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureDeletePerm(rt); err != nil {
		return err
	}
	if rt.Message == nil || rt.Message.ReplyToMessage == nil {
		return fmt.Errorf("reply to a message to delete it")
	}
	if err := rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.ReplyToMessage.MessageID); err != nil {
		return err
	}
	_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
	return nil
}

func (s *Service) pin(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensurePinPerm(rt); err != nil {
		return err
	}
	if rt.Message == nil || rt.Message.ReplyToMessage == nil {
		return fmt.Errorf("reply to a message to pin it")
	}
	disableNotification := len(rt.Command.Args) > 0 && strings.EqualFold(rt.Command.Args[0], "quiet")
	if err := rt.Client.PinChatMessage(ctx, rt.ChatID(), rt.Message.ReplyToMessage.MessageID, disableNotification); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Message pinned.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) unpin(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensurePinPerm(rt); err != nil {
		return err
	}
	var messageID *int64
	if rt.Message != nil && rt.Message.ReplyToMessage != nil {
		messageID = &rt.Message.ReplyToMessage.MessageID
	}
	if err := rt.Client.UnpinChatMessage(ctx, rt.ChatID(), messageID); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Message unpinned.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) unpinAll(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensurePinPerm(rt); err != nil {
		return err
	}
	if err := rt.Client.UnpinAllChatMessages(ctx, rt.ChatID()); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "All pinned messages were removed.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) mod(ctx context.Context, rt *runtime.Context, enabled bool) error {
	return s.setSilentRole(ctx, rt, enabled, "mod", "mod power")
}

func (s *Service) muter(ctx context.Context, rt *runtime.Context, enabled bool) error {
	return s.setSilentRole(ctx, rt, enabled, "muter", "mute power")
}

func (s *Service) setSilentRole(ctx context.Context, rt *runtime.Context, enabled bool, role string, label string) error {
	if err := s.ensurePromotePerm(rt); err != nil {
		return err
	}
	target, _, err := moderationTarget(ctx, rt)
	if err != nil {
		return err
	}
	if err := rt.Store.SetChatRole(ctx, rt.Bot.ID, rt.ChatID(), target.UserID, role, rt.ActorID(), enabled); err != nil {
		return err
	}
	text := "Granted " + label + " to " + target.Name + "."
	if !enabled {
		text = "Removed " + label + " from " + target.Name + "."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) listMods(ctx context.Context, rt *runtime.Context) error {
	mods, err := rt.Store.ListChatRoleUsers(ctx, rt.Bot.ID, rt.ChatID(), "mod")
	if err != nil {
		return err
	}
	muters, err := rt.Store.ListChatRoleUsers(ctx, rt.Bot.ID, rt.ChatID(), "muter")
	if err != nil {
		return err
	}
	if len(mods) == 0 && len(muters) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No silent mods are configured.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	parts := make([]string, 0, len(mods)+len(muters))
	for _, user := range mods {
		parts = append(parts, serviceutil.DisplayNameFromProfile(user)+" [mod]")
	}
	for _, user := range muters {
		parts = append(parts, serviceutil.DisplayNameFromProfile(user)+" [muter]")
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Silent staff: "+strings.Join(parts, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) jobStatus(ctx context.Context, rt *runtime.Context, kind string) error {
	if len(rt.Command.Args) > 1 {
		job, err := rt.Store.GetJob(ctx, rt.Command.Args[1])
		if err != nil {
			return err
		}
		if job.Kind != kind {
			return fmt.Errorf("job %s is not a %s job", job.ID, kind)
		}
		return s.sendJobStatus(ctx, rt, job)
	}
	jobsList, err := rt.Store.ListRecentJobs(ctx, rt.Bot.ID, 5)
	if err != nil {
		return err
	}
	lines := make([]string, 0, len(jobsList))
	for _, job := range jobsList {
		if job.Kind != kind {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s %d/%d", job.ID, job.Status, job.Progress, job.Total))
	}
	if len(lines) == 0 {
		lines = append(lines, "No recent jobs.")
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
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
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func ParseToggle(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "enable", "enabled", "yes":
		return true, nil
	case "off", "disable", "disabled", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected on or off")
	}
}

func toggleWord(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func moderationTarget(ctx context.Context, rt *runtime.Context) (serviceutil.Target, string, error) {
	if rt.Message == nil {
		return serviceutil.Target{}, "", fmt.Errorf("message context required")
	}
	target, err := serviceutil.ResolveTarget(ctx, rt, rt.Command.Args)
	if err != nil {
		return serviceutil.Target{}, "", err
	}
	if rt.Message.ReplyToMessage != nil {
		return target, strings.TrimSpace(strings.Join(rt.Command.Args, " ")), nil
	}
	if len(rt.Command.Args) <= 1 {
		return target, "", nil
	}
	return target, strings.TrimSpace(strings.Join(rt.Command.Args[1:], " ")), nil
}
