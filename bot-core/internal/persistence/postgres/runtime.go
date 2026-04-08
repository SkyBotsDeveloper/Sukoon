package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
)

func (s *Store) LoadRuntimeBundle(ctx context.Context, botID string, chatID int64) (domain.RuntimeBundle, error) {
	bundle := domain.RuntimeBundle{
		DisabledCommands: map[string]struct{}{},
		Locks:            map[string]domain.LockRule{},
	}

	err := s.pool.QueryRow(ctx, `
		SELECT
			cs.bot_id, cs.chat_id, cs.language, cs.reports_enabled, cs.log_channel_id,
			cs.clean_commands, cs.disabled_delete, cs.disable_admins, cs.admin_errors, cs.anon_admins, cs.clean_service_join, cs.clean_service_leave, cs.clean_service_pin, cs.clean_service_title, cs.clean_service_photo, cs.clean_service_other, cs.clean_service_videochat,
			cs.welcome_enabled, cs.welcome_text, cs.goodbye_enabled, cs.goodbye_text, cs.rules_text,
			ms.warn_limit, ms.warn_mode,
			afs.enabled, afs.flood_limit, afs.timed_limit, afs.window_seconds, afs.action, afs.action_duration_seconds, afs.clear_all,
			cps.enabled, cps.mode, cps.timeout_seconds, cps.failure_action, cps.challenge_digits,
			aas.enabled, aas.action,
			abs.enabled, abs.action
		FROM chat_settings cs
		JOIN moderation_settings ms ON ms.bot_id = cs.bot_id AND ms.chat_id = cs.chat_id
		JOIN antiflood_settings afs ON afs.bot_id = cs.bot_id AND afs.chat_id = cs.chat_id
		JOIN captcha_settings cps ON cps.bot_id = cs.bot_id AND cps.chat_id = cs.chat_id
		JOIN antiabuse_settings aas ON aas.bot_id = cs.bot_id AND aas.chat_id = cs.chat_id
		JOIN antibio_settings abs ON abs.bot_id = cs.bot_id AND abs.chat_id = cs.chat_id
		WHERE cs.bot_id=$1 AND cs.chat_id=$2
	`, botID, chatID).Scan(
		&bundle.Settings.BotID,
		&bundle.Settings.ChatID,
		&bundle.Settings.Language,
		&bundle.Settings.ReportsEnabled,
		&bundle.Settings.LogChannelID,
		&bundle.Settings.CleanCommands,
		&bundle.Settings.DisabledDelete,
		&bundle.Settings.DisableAdmins,
		&bundle.Settings.AdminErrors,
		&bundle.Settings.AnonAdmins,
		&bundle.Settings.CleanServiceJoin,
		&bundle.Settings.CleanServiceLeave,
		&bundle.Settings.CleanServicePin,
		&bundle.Settings.CleanServiceTitle,
		&bundle.Settings.CleanServicePhoto,
		&bundle.Settings.CleanServiceOther,
		&bundle.Settings.CleanServiceVideoChat,
		&bundle.Settings.WelcomeEnabled,
		&bundle.Settings.WelcomeText,
		&bundle.Settings.GoodbyeEnabled,
		&bundle.Settings.GoodbyeText,
		&bundle.Settings.RulesText,
		&bundle.Moderation.WarnLimit,
		&bundle.Moderation.WarnMode,
		&bundle.Antiflood.Enabled,
		&bundle.Antiflood.Limit,
		&bundle.Antiflood.TimedLimit,
		&bundle.Antiflood.WindowSeconds,
		&bundle.Antiflood.Action,
		&bundle.Antiflood.ActionDurationSeconds,
		&bundle.Antiflood.ClearAll,
		&bundle.Captcha.Enabled,
		&bundle.Captcha.Mode,
		&bundle.Captcha.TimeoutSeconds,
		&bundle.Captcha.FailureAction,
		&bundle.Captcha.ChallengeDigits,
		&bundle.AntiAbuse.Enabled,
		&bundle.AntiAbuse.Action,
		&bundle.AntiBio.Enabled,
		&bundle.AntiBio.Action,
	)
	if err != nil {
		return domain.RuntimeBundle{}, err
	}

	rows, err := s.pool.Query(ctx, `SELECT command FROM disabled_commands WHERE bot_id=$1 AND chat_id=$2`, botID, chatID)
	if err != nil {
		return domain.RuntimeBundle{}, err
	}
	for rows.Next() {
		var command string
		if err := rows.Scan(&command); err != nil {
			rows.Close()
			return domain.RuntimeBundle{}, err
		}
		bundle.DisabledCommands[command] = struct{}{}
	}
	rows.Close()

	lockRows, err := s.pool.Query(ctx, `SELECT lock_type, action FROM locks WHERE bot_id=$1 AND chat_id=$2`, botID, chatID)
	if err != nil {
		return domain.RuntimeBundle{}, err
	}
	for lockRows.Next() {
		var lock domain.LockRule
		lock.BotID = botID
		lock.ChatID = chatID
		if err := lockRows.Scan(&lock.LockType, &lock.Action); err != nil {
			lockRows.Close()
			return domain.RuntimeBundle{}, err
		}
		bundle.Locks[lock.LockType] = lock
	}
	lockRows.Close()

	blockRows, err := s.pool.Query(ctx, `
		SELECT id, pattern, match_mode, action, created_by, created_at
		FROM blocklist_rules
		WHERE bot_id=$1 AND chat_id=$2
		ORDER BY id ASC
	`, botID, chatID)
	if err != nil {
		return domain.RuntimeBundle{}, err
	}
	defer blockRows.Close()
	for blockRows.Next() {
		var rule domain.BlocklistRule
		rule.BotID = botID
		rule.ChatID = chatID
		if err := blockRows.Scan(&rule.ID, &rule.Pattern, &rule.MatchMode, &rule.Action, &rule.CreatedBy, &rule.CreatedAt); err != nil {
			return domain.RuntimeBundle{}, err
		}
		bundle.Blocklist = append(bundle.Blocklist, rule)
	}

	return bundle, blockRows.Err()
}

func (s *Store) SetLanguage(ctx context.Context, botID string, chatID int64, language string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET language=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, language)
	return err
}

func (s *Store) SetDisabledCommand(ctx context.Context, botID string, chatID int64, command string, disabled bool, changedBy int64) error {
	if disabled {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO disabled_commands (bot_id, chat_id, command, changed_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (bot_id, chat_id, command) DO UPDATE SET changed_by=EXCLUDED.changed_by, changed_at=NOW()
		`, botID, chatID, command, changedBy)
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM disabled_commands WHERE bot_id=$1 AND chat_id=$2 AND command=$3`, botID, chatID, command)
	return err
}

func (s *Store) SetDisabledDelete(ctx context.Context, botID string, chatID int64, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET disabled_delete=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, enabled)
	return err
}

func (s *Store) SetDisableAdmins(ctx context.Context, botID string, chatID int64, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET disable_admins=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, enabled)
	return err
}

func (s *Store) SetAdminErrors(ctx context.Context, botID string, chatID int64, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET admin_errors=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, enabled)
	return err
}

func (s *Store) SetAnonAdmins(ctx context.Context, botID string, chatID int64, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET anon_admins=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, enabled)
	return err
}

func (s *Store) SetWarnConfig(ctx context.Context, botID string, chatID int64, limit int, mode string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE moderation_settings
		SET warn_limit=$3, warn_mode=$4, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, limit, mode)
	return err
}

func (s *Store) IncrementWarnings(ctx context.Context, botID string, chatID int64, userID int64, reason string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		INSERT INTO warnings (bot_id, chat_id, user_id, warning_count, last_reason)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (bot_id, chat_id, user_id) DO UPDATE SET
			warning_count = warnings.warning_count + 1,
			last_reason = EXCLUDED.last_reason,
			updated_at = NOW()
		RETURNING warning_count
	`, botID, chatID, userID, reason).Scan(&count)
	return count, err
}

func (s *Store) ResetWarnings(ctx context.Context, botID string, chatID int64, userID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM warnings WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3`, botID, chatID, userID)
	return err
}

func (s *Store) GetWarnings(ctx context.Context, botID string, chatID int64, userID int64) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT warning_count FROM warnings WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3`, botID, chatID, userID).Scan(&count)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	return count, err
}

func (s *Store) SetLogChannel(ctx context.Context, botID string, chatID int64, logChannelID *int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET log_channel_id=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, logChannelID)
	return err
}

func (s *Store) SetReportsEnabled(ctx context.Context, botID string, chatID int64, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET reports_enabled=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, enabled)
	return err
}

func (s *Store) SetCleanCommands(ctx context.Context, botID string, chatID int64, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET clean_commands=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, enabled)
	return err
}

func (s *Store) SetCleanService(ctx context.Context, botID string, chatID int64, target string, enabled bool) error {
	query := `
		UPDATE chat_settings
		SET updated_at=NOW()
	`
	switch target {
	case "join":
		query += `, clean_service_join=$3`
	case "leave":
		query += `, clean_service_leave=$3`
	case "pin":
		query += `, clean_service_pin=$3`
	case "title":
		query += `, clean_service_title=$3`
	case "photo":
		query += `, clean_service_photo=$3`
	case "other":
		query += `, clean_service_other=$3`
	case "videochat":
		query += `, clean_service_videochat=$3`
	default:
		query += `, clean_service_join=$3, clean_service_leave=$3, clean_service_pin=$3, clean_service_title=$3, clean_service_photo=$3, clean_service_other=$3, clean_service_videochat=$3`
	}
	query += ` WHERE bot_id=$1 AND chat_id=$2`
	_, err := s.pool.Exec(ctx, query, botID, chatID, enabled)
	return err
}
