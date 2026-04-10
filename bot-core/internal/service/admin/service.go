package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/jobs"
	"sukoon/bot-core/internal/permissions"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

type Service struct {
	jobs        *jobs.Service
	permissions *permissions.Service
}

func New(jobService *jobs.Service, permissionsService *permissions.Service) *Service {
	return &Service{jobs: jobService, permissions: permissionsService}
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
	case "disableable":
		return true, s.disableable(ctx, rt)
	case "disabledel":
		return true, s.disabledDelete(ctx, rt)
	case "disableadmin":
		return true, s.disableAdmins(ctx, rt)
	case "disabled":
		return true, s.listDisabled(ctx, rt)
	case "logchannel", "setlog", "unsetlog":
		return true, s.logChannel(ctx, rt)
	case "log", "nolog":
		return true, s.logCategoryToggle(ctx, rt)
	case "logcategories":
		return true, s.logCategories(ctx, rt)
	case "reports":
		return true, s.reports(ctx, rt)
	case "report":
		return true, s.report(ctx, rt)
	case "promote":
		return true, s.promote(ctx, rt, true)
	case "demote":
		return true, s.promote(ctx, rt, false)
	case "admins", "adminlist":
		return true, s.admins(ctx, rt)
	case "admincache":
		return true, s.adminCache(ctx, rt)
	case "anonadmin":
		return true, s.anonAdmin(ctx, rt)
	case "adminerror":
		return true, s.adminError(ctx, rt)
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

func (s *Service) ensureChatAdmin(ctx context.Context, rt *runtime.Context) (bool, error) {
	if !rt.ActorPermissions.IsChatAdmin {
		if isAnonymousAdminMessage(rt) {
			return false, s.sendPermissionNotice(ctx, rt, "Anonymous admins need /anonadmin on before they can use admin commands.", true)
		}
		return false, s.sendPermissionNotice(ctx, rt, "You need to be admin to do this.", true)
	}
	return true, nil
}

func (s *Service) ensureDeletePerm(ctx context.Context, rt *runtime.Context) (bool, error) {
	ok, err := s.ensureChatAdmin(ctx, rt)
	if !ok || err != nil {
		return ok, err
	}
	if !rt.ActorPermissions.CanDeleteMessages {
		return false, s.sendPermissionNotice(ctx, rt, "Delete messages permission required.", false)
	}
	return true, nil
}

func (s *Service) ensurePromotePerm(ctx context.Context, rt *runtime.Context) (bool, error) {
	ok, err := s.ensureChatAdmin(ctx, rt)
	if !ok || err != nil {
		return ok, err
	}
	if !rt.ActorPermissions.CanPromoteMembers {
		return false, s.sendPermissionNotice(ctx, rt, "Add admins permission required.", false)
	}
	return true, nil
}

func (s *Service) ensureChatOwner(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.ActorPermissions.IsOwner || rt.ActorPermissions.IsSudo {
		return true, nil
	}
	if !rt.ActorPermissions.IsChatAdmin {
		return false, s.sendPermissionNotice(ctx, rt, "Only the chat owner can do this.", true)
	}
	admins, err := s.chatAdministrators(ctx, rt, false)
	if err != nil {
		return false, err
	}
	for _, admin := range admins {
		if admin.User.ID == rt.ActorID() && admin.Status == "creator" {
			return true, nil
		}
	}
	return false, s.sendPermissionNotice(ctx, rt, "Only the chat owner can do this.", true)
}

func (s *Service) ensurePinPerm(ctx context.Context, rt *runtime.Context) (bool, error) {
	ok, err := s.ensureChatAdmin(ctx, rt)
	if !ok || err != nil {
		return ok, err
	}
	if !rt.ActorPermissions.CanPinMessages {
		return false, s.sendPermissionNotice(ctx, rt, "Pin messages permission required.", false)
	}
	return true, nil
}

func (s *Service) sendPermissionNotice(ctx context.Context, rt *runtime.Context, text string, respectAdminErrors bool) error {
	if respectAdminErrors && !rt.RuntimeBundle.Settings.AdminErrors {
		return nil
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func isAnonymousAdminMessage(rt *runtime.Context) bool {
	return rt.Message != nil && rt.Message.SenderChat != nil && rt.Message.SenderChat.ID == rt.ChatID()
}

func (s *Service) approve(ctx context.Context, rt *runtime.Context, approved bool) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	target, reason, err := moderationTarget(ctx, rt)
	if err != nil {
		return err
	}
	if err := rt.Store.SetApproval(ctx, rt.Bot.ID, rt.ChatID(), target.UserID, rt.ActorID(), approved, reason); err != nil {
		return err
	}
	text := "Approved " + target.Name + "."
	if approved && strings.TrimSpace(reason) != "" {
		text += " Reason: " + reason
	}
	if !approved {
		text = "Removed approval for " + target.Name + "."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategoryAdmin, fmt.Sprintf("approval: actor=%d target=%d approved=%t reason=%s", rt.ActorID(), target.UserID, approved, reason))
	}
	return err
}

func (s *Service) approvalStatus(ctx context.Context, rt *runtime.Context) error {
	target, _, err := moderationTarget(ctx, rt)
	if err != nil {
		return fmt.Errorf("usage: /approval <reply|user>")
	}
	approval, err := rt.Store.GetApproval(ctx, rt.Bot.ID, rt.ChatID(), target.UserID)
	approved := err == nil
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	text := target.Name + " is not approved."
	if approved {
		text = target.Name + " is approved."
		if strings.TrimSpace(approval.Reason) != "" {
			text += " Reason: " + approval.Reason
		}
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) listApproved(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
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
		profile, err := rt.Store.GetUserByID(ctx, userID)
		if err == nil {
			if profile.Username != "" {
				parts = append(parts, "@"+profile.Username)
				continue
			}
			name := strings.TrimSpace(strings.TrimSpace(profile.FirstName + " " + profile.LastName))
			if name != "" {
				parts = append(parts, fmt.Sprintf("%s (%d)", name, userID))
				continue
			}
		}
		parts = append(parts, strconv.FormatInt(userID, 10))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Approved users: "+strings.Join(parts, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) unapproveAll(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatOwner(ctx, rt); err != nil || !ok {
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
		if err := rt.Store.SetApproval(ctx, rt.Bot.ID, rt.ChatID(), userID, rt.ActorID(), false, ""); err != nil {
			return err
		}
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Removed approvals for %d user(s).", len(approvedUsers)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategoryAdmin, fmt.Sprintf("approval: actor=%d action=unapproveall count=%d", rt.ActorID(), len(approvedUsers)))
	}
	return err
}

func (s *Service) disable(ctx context.Context, rt *runtime.Context, disabled bool) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /disable <command>")
	}
	command := strings.TrimPrefix(strings.ToLower(rt.Command.Args[0]), "/")
	if isProtectedDisabledCommand(command) {
		return fmt.Errorf("/%s cannot be disabled", command)
	}
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

func (s *Service) disableable(ctx context.Context, rt *runtime.Context) error {
	text := strings.Join([]string{
		"Disableable Commands",
		"",
		"Sukoon disables the exact command name you pass. Admins bypass disabled commands by default unless /disableadmin is enabled.",
		"",
		"/approve, /unapprove, /approved, /unapproveall, /approval",
		"/ban, /dban, /sban, /tban, /unban",
		"/mute, /dmute, /smute, /tmute, /unmute",
		"/kick, /dkick, /skick, /kickme",
		"/warn, /warns, /resetwarns, /setwarnlimit, /setwarnmode",
		"/lock, /unlock, /locks, /locktypes",
		"/addblocklist, /rmbl, /rmblocklist, /unblocklistall, /blocklist",
		"/setflood, /flood, /setfloodtimer, /floodmode, /clearflood",
		"/captcha, /captchamode, /captcharules, /captchamutetime, /captchakick, /captchakicktime, /setcaptchatext, /resetcaptchatext",
		"/cleancommands, /cleancommand, /keepcommand, /cleancommandtypes",
		"/cleanservice, /nocleanservice, /cleanservicetypes",
		"/logchannel, /setlog, /unsetlog, /log, /nolog, /logcategories",
		"/report, /reports",
		"/save, /notes, /saved, /get, /clear",
		"/filter, /filters, /stop, /stopall",
		"/welcome, /setwelcome, /goodbye, /setgoodbye",
		"/setrules, /resetrules, /rules",
		"/antiabuse, /antibio, /free, /unfree, /freelist",
		"/newfed, /renamefed, /delfed, /joinfed, /leavefed, /fedinfo, /fedadmins, /myfeds, /fedpromote, /feddemote, /feddemoteme, /fban, /unfban, /fedtransfer, /chatfed",
	}, "\n")
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) disabledDelete(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.Settings.DisabledDelete {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Disabled command deletion is "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := ParseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	if err := rt.Store.SetDisabledDelete(ctx, rt.Bot.ID, rt.ChatID(), enabled); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Disabled command deletion %s.", toggleWord(enabled)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) disableAdmins(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.Settings.DisableAdmins {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Disable-admin mode is "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := ParseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	if err := rt.Store.SetDisableAdmins(ctx, rt.Bot.ID, rt.ChatID(), enabled); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Disable-admin mode %s.", toggleWord(enabled)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) listDisabled(ctx context.Context, rt *runtime.Context) error {
	if len(rt.RuntimeBundle.DisabledCommands) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("No disabled commands.\n/delete on disabled commands: %s\n/disableadmin: %s", onOff(rt.RuntimeBundle.Settings.DisabledDelete), onOff(rt.RuntimeBundle.Settings.DisableAdmins)), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	commands := make([]string, 0, len(rt.RuntimeBundle.DisabledCommands))
	for command := range rt.RuntimeBundle.DisabledCommands {
		commands = append(commands, "/"+command)
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Disabled: "+strings.Join(commands, ", ")+fmt.Sprintf("\n/delete on disabled commands: %s\n/disableadmin: %s", onOff(rt.RuntimeBundle.Settings.DisabledDelete), onOff(rt.RuntimeBundle.Settings.DisableAdmins)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func isProtectedDisabledCommand(command string) bool {
	switch strings.TrimSpace(strings.ToLower(command)) {
	case "", "disable", "enable", "disabled", "disableable", "disabledel", "disableadmin", "start", "help":
		return true
	default:
		return false
	}
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func (s *Service) logChannel(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	switch rt.Command.Name {
	case "unsetlog":
		if err := rt.Store.SetLogChannel(ctx, rt.Bot.ID, rt.ChatID(), nil); err != nil {
			return err
		}
		rt.RuntimeBundle.Settings.LogChannelID = nil
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Log channel disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	case "setlog":
		channelID, label, err := s.resolveLogChannelTarget(ctx, rt)
		if err != nil {
			return err
		}
		if err := rt.Store.SetLogChannel(ctx, rt.Bot.ID, rt.ChatID(), &channelID); err != nil {
			return err
		}
		rt.RuntimeBundle.Settings.LogChannelID = &channelID
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Log channel set to %s.", label), rt.ReplyOptions(telegram.SendMessageOptions{}))
		if err == nil {
			_ = serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategorySettings, fmt.Sprintf("settings: actor=%d logchannel=%d", rt.ActorID(), channelID))
		}
		return err
	}
	if len(rt.Command.Args) == 0 {
		if rt.RuntimeBundle.Settings.LogChannelID == nil {
			_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No log channel configured.", rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Current log channel: "+s.logChannelLabel(ctx, rt, *rt.RuntimeBundle.Settings.LogChannelID), rt.ReplyOptions(telegram.SendMessageOptions{}))
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
	rt.RuntimeBundle.Settings.LogChannelID = &channelID
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Log channel set to %s.", s.logChannelLabel(ctx, rt, channelID)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategorySettings, fmt.Sprintf("settings: actor=%d logchannel=%d", rt.ActorID(), channelID))
	}
	return err
}

func (s *Service) logCategories(ctx context.Context, rt *runtime.Context) error {
	text := strings.Join([]string{
		"Log categories:",
		"",
		fmt.Sprintf("- settings: Bot configuration updates. [%s]", enabledLabel(rt.RuntimeBundle.Settings.LogCategorySettings)),
		fmt.Sprintf("- admin: Manual admin actions such as bans, mutes, kicks, warns, and approvals. [%s]", enabledLabel(rt.RuntimeBundle.Settings.LogCategoryAdmin)),
		fmt.Sprintf("- user: User-driven actions such as /kickme and other member-side moderation flows. [%s]", enabledLabel(rt.RuntimeBundle.Settings.LogCategoryUser)),
		fmt.Sprintf("- automated: Automatic moderation triggers such as locks, blocklists, antiflood, antiraid, antiabuse, and antibio. [%s]", enabledLabel(rt.RuntimeBundle.Settings.LogCategoryAutomated)),
		fmt.Sprintf("- reports: User reports sent through /report. [%s]", enabledLabel(rt.RuntimeBundle.Settings.LogCategoryReports)),
		fmt.Sprintf("- other: Extra bot events that do not fit the other categories. [%s]", enabledLabel(rt.RuntimeBundle.Settings.LogCategoryOther)),
	}, "\n")
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) logCategoryToggle(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	if len(rt.Command.Args) == 0 {
		if rt.Command.Name == "log" {
			return fmt.Errorf("usage: /log <category>")
		}
		return fmt.Errorf("usage: /nolog <category>")
	}
	categories, err := parseLogCategories(rt.Command.Args)
	if err != nil {
		return err
	}
	enabled := rt.Command.Name == "log"
	if err := rt.Store.SetLogCategories(ctx, rt.Bot.ID, rt.ChatID(), categories, enabled); err != nil {
		return err
	}
	applyLogCategories(&rt.RuntimeBundle.Settings, categories, enabled)
	verb := "Enabled"
	if !enabled {
		verb = "Disabled"
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s log categories: %s.", verb, strings.Join(categories, ", ")), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategorySettings, fmt.Sprintf("settings: actor=%d logcategories enabled=%t values=%s", rt.ActorID(), enabled, strings.Join(categories, ",")))
	}
	return err
}

func (s *Service) reports(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
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
	if err := serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategoryReports, reportText); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Report sent.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) admins(ctx context.Context, rt *runtime.Context) error {
	admins, err := s.chatAdministrators(ctx, rt, false)
	if err != nil {
		return err
	}
	parts := make([]string, 0, len(admins))
	for _, admin := range admins {
		if admin.IsAnonymous {
			continue
		}
		label := serviceutil.DisplayName(admin.User)
		if admin.Status == "creator" {
			label += " [owner]"
		}
		parts = append(parts, label)
	}
	if len(parts) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No visible chat admins.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Chat admins: "+strings.Join(parts, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) promote(ctx context.Context, rt *runtime.Context, enabled bool) error {
	if ok, err := s.ensurePromotePerm(ctx, rt); err != nil || !ok {
		return err
	}
	target, _, err := moderationTarget(ctx, rt)
	if err != nil {
		if enabled {
			return fmt.Errorf("usage: /promote <reply|user>")
		}
		return fmt.Errorf("usage: /demote <reply|user>")
	}

	perms := telegram.PromotePermissions{}
	if enabled {
		perms = telegram.PromotePermissions{
			CanDeleteMessages:  rt.ActorPermissions.CanDeleteMessages,
			CanRestrictMembers: rt.ActorPermissions.CanRestrictMembers,
			CanChangeInfo:      rt.ActorPermissions.CanChangeInfo,
			CanPinMessages:     rt.ActorPermissions.CanPinMessages,
			CanPromoteMembers:  false,
		}
	}
	if err := rt.Client.PromoteChatMember(ctx, rt.ChatID(), target.UserID, perms); err != nil {
		return err
	}
	if _, err := s.chatAdministrators(ctx, rt, true); err != nil {
		rt.Logger.Warn("admin cache refresh failed", "error", err)
	}
	text := "Promoted " + target.Name + "."
	if !enabled {
		text = "Demoted " + target.Name + "."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) adminCache(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	admins, err := s.chatAdministrators(ctx, rt, true)
	if err != nil {
		return err
	}
	visible := 0
	for _, admin := range admins {
		if admin.IsAnonymous {
			continue
		}
		visible++
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Admin cache refreshed. Visible admins: %d.", visible), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) chatAdministrators(ctx context.Context, rt *runtime.Context, refresh bool) ([]telegram.ChatAdministrator, error) {
	if s.permissions == nil {
		return rt.Client.GetChatAdministrators(ctx, rt.ChatID())
	}
	if refresh {
		return s.permissions.RefreshChatAdministrators(ctx, rt.Bot.ID, rt.ChatID(), rt.Client)
	}
	return s.permissions.ChatAdministrators(ctx, rt.Bot.ID, rt.ChatID(), rt.Client)
}

func (s *Service) anonAdmin(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.Settings.AnonAdmins {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Anonymous admin mode is "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := ParseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	if err := rt.Store.SetAnonAdmins(ctx, rt.Bot.ID, rt.ChatID(), enabled); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Anonymous admin mode %s.", toggleWord(enabled)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) adminError(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	if len(rt.Command.Args) == 0 {
		status := "on"
		if !rt.RuntimeBundle.Settings.AdminErrors {
			status = "off"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Admin errors are "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := ParseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	if err := rt.Store.SetAdminErrors(ctx, rt.Bot.ID, rt.ChatID(), enabled); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Admin errors %s.", toggleWord(enabled)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) cleanCommands(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
		return err
	}
	if len(rt.Command.Args) == 0 {
		if rt.Command.Name == "keepcommand" {
			if err := rt.Store.SetCleanCommandTypes(ctx, rt.Bot.ID, rt.ChatID(), []string{"all"}, false); err != nil {
				return err
			}
			applyCleanCommandTypes(&rt.RuntimeBundle.Settings, []string{"all"}, false)
			_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Clean command categories disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}
		enabled := rt.RuntimeBundle.Settings.EnabledCleanCommandCategories()
		if len(enabled) == 0 {
			_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Clean Commands is off. Use /cleancommand <all|admin|user|other> to enable categories.", rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Currently cleaned command types: "+strings.Join(enabled, ", ")+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled := rt.Command.Name != "keepcommand"
	if toggled, err := ParseToggle(rt.Command.Args[0]); err == nil {
		enabled = toggled
		if rt.Command.Name == "keepcommand" {
			enabled = false
		}
		if err := rt.Store.SetCleanCommandTypes(ctx, rt.Bot.ID, rt.ChatID(), []string{"all"}, enabled); err != nil {
			return err
		}
		applyCleanCommandTypes(&rt.RuntimeBundle.Settings, []string{"all"}, enabled)
		verb := "Enabled"
		if !enabled {
			verb = "Disabled"
		}
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s clean command category: all.", verb), rt.ReplyOptions(telegram.SendMessageOptions{}))
		if err == nil {
			_ = serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategorySettings, fmt.Sprintf("settings: actor=%d cleancommands enabled=%t categories=all", rt.ActorID(), enabled))
		}
		return err
	}
	categories, err := parseCleanCommandTypes(rt.Command.Args)
	if err != nil {
		return err
	}
	if err := rt.Store.SetCleanCommandTypes(ctx, rt.Bot.ID, rt.ChatID(), categories, enabled); err != nil {
		return err
	}
	applyCleanCommandTypes(&rt.RuntimeBundle.Settings, categories, enabled)
	verb := "Enabled"
	if !enabled {
		verb = "Disabled"
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s clean command categories: %s.", verb, strings.Join(categories, ", ")), rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err == nil {
		_ = serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategorySettings, fmt.Sprintf("settings: actor=%d cleancommands enabled=%t categories=%s", rt.ActorID(), enabled, strings.Join(categories, ",")))
	}
	return err
}

func (s *Service) cleanCommandTypes(ctx context.Context, rt *runtime.Context) error {
	text := strings.Join([]string{
		"Clean command types:",
		"",
		"- all: Delete all command messages sent to the group.",
		"- admin: Delete admin-only commands such as /ban, /mute, /setwelcome, and settings changes.",
		"- user: Delete user-facing commands such as /get, /rules, /report, and /help.",
		"- other: Delete commands Sukoon does not recognise, including commands for other bots.",
		"",
		"Use /cleancommand <type> to enable one or more categories, and /keepcommand <type> to stop deleting them.",
	}, "\n")
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) cleanService(ctx context.Context, rt *runtime.Context) error {
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
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
	if ok, err := s.ensureChatAdmin(ctx, rt); err != nil || !ok {
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
	if ok, err := s.ensureDeletePerm(ctx, rt); err != nil || !ok {
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
	if ok, err := s.ensureDeletePerm(ctx, rt); err != nil || !ok {
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
	if ok, err := s.ensurePinPerm(ctx, rt); err != nil || !ok {
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
	if ok, err := s.ensurePinPerm(ctx, rt); err != nil || !ok {
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
	if ok, err := s.ensurePinPerm(ctx, rt); err != nil || !ok {
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
	if ok, err := s.ensurePromotePerm(ctx, rt); err != nil || !ok {
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

func enabledLabel(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func parseCleanCommandTypes(args []string) ([]string, error) {
	return parseCategories(args, map[string]string{
		"all":    "all",
		"admin":  "admin",
		"admins": "admin",
		"user":   "user",
		"users":  "user",
		"other":  "other",
	}, "clean command type")
}

func parseLogCategories(args []string) ([]string, error) {
	return parseCategories(args, map[string]string{
		"all":       "all",
		"settings":  "settings",
		"setting":   "settings",
		"admin":     "admin",
		"admins":    "admin",
		"user":      "user",
		"users":     "user",
		"automated": "automated",
		"auto":      "automated",
		"reports":   "reports",
		"report":    "reports",
		"other":     "other",
	}, "log category")
}

func parseCategories(args []string, allowed map[string]string, label string) ([]string, error) {
	seen := map[string]struct{}{}
	categories := make([]string, 0, len(args))
	for _, arg := range args {
		category, ok := allowed[strings.ToLower(strings.TrimSpace(arg))]
		if !ok {
			return nil, fmt.Errorf("unknown %s: %s", label, arg)
		}
		if _, exists := seen[category]; exists {
			continue
		}
		seen[category] = struct{}{}
		categories = append(categories, category)
	}
	if len(categories) == 0 {
		return nil, fmt.Errorf("at least one %s is required", label)
	}
	return categories, nil
}

func applyCleanCommandTypes(settings *domain.ChatSettings, categories []string, enabled bool) {
	for _, category := range categories {
		switch category {
		case "all":
			settings.CleanCommandAll = enabled
			if !enabled {
				settings.CleanCommandAdmin = false
				settings.CleanCommandUser = false
				settings.CleanCommandOther = false
			}
		case "admin":
			settings.CleanCommandAdmin = enabled
		case "user":
			settings.CleanCommandUser = enabled
		case "other":
			settings.CleanCommandOther = enabled
		}
	}
	settings.CleanCommands = settings.CleanCommandAll || settings.CleanCommandAdmin || settings.CleanCommandUser || settings.CleanCommandOther
}

func applyLogCategories(settings *domain.ChatSettings, categories []string, enabled bool) {
	for _, category := range categories {
		switch category {
		case "all":
			settings.LogCategorySettings = enabled
			settings.LogCategoryAdmin = enabled
			settings.LogCategoryUser = enabled
			settings.LogCategoryAutomated = enabled
			settings.LogCategoryReports = enabled
			settings.LogCategoryOther = enabled
		case "settings":
			settings.LogCategorySettings = enabled
		case "admin":
			settings.LogCategoryAdmin = enabled
		case "user":
			settings.LogCategoryUser = enabled
		case "automated":
			settings.LogCategoryAutomated = enabled
		case "reports":
			settings.LogCategoryReports = enabled
		case "other":
			settings.LogCategoryOther = enabled
		}
	}
}

func (s *Service) resolveLogChannelTarget(ctx context.Context, rt *runtime.Context) (int64, string, error) {
	if len(rt.Command.Args) > 0 {
		channelID, err := strconv.ParseInt(rt.Command.Args[0], 10, 64)
		if err != nil {
			return 0, "", fmt.Errorf("usage: /setlog <chat_id> or forward /setlog from the log channel")
		}
		return channelID, s.logChannelLabel(ctx, rt, channelID), nil
	}
	if rt.Message != nil && rt.Message.ForwardFromChat != nil {
		channel := rt.Message.ForwardFromChat
		if channel.Type != "channel" {
			return 0, "", fmt.Errorf("forward /setlog from the channel you want to use for logs")
		}
		return channel.ID, logChatLabel(*channel), nil
	}
	return 0, "", fmt.Errorf("usage: /setlog <chat_id> or forward /setlog from the log channel")
}

func (s *Service) logChannelLabel(ctx context.Context, rt *runtime.Context, channelID int64) string {
	if rt.Client != nil {
		if chat, err := rt.Client.GetChat(ctx, channelID); err == nil {
			return logChatLabel(chat)
		}
	}
	return fmt.Sprintf("%d", channelID)
}

func logChatLabel(chat telegram.Chat) string {
	if strings.TrimSpace(chat.Title) != "" {
		return fmt.Sprintf("%s (%d)", chat.Title, chat.ID)
	}
	if strings.TrimSpace(chat.Username) != "" {
		return fmt.Sprintf("@%s (%d)", chat.Username, chat.ID)
	}
	return fmt.Sprintf("%d", chat.ID)
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
