package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/telegram"
)

func (s *Store) SetAntiAbuseSettings(ctx context.Context, settings domain.AntiAbuseSettings) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE antiabuse_settings
		SET enabled=$3, action=$4, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, settings.BotID, settings.ChatID, settings.Enabled, settings.Action)
	return err
}

func (s *Store) SetAntiBioSettings(ctx context.Context, settings domain.AntiBioSettings) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE antibio_settings
		SET enabled=$3, action=$4, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, settings.BotID, settings.ChatID, settings.Enabled, settings.Action)
	return err
}

func (s *Store) IsAntiBioExempt(ctx context.Context, botID string, chatID int64, userID int64) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM antibio_exemptions WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3
		)
	`, botID, chatID, userID).Scan(&exists)
	return exists, err
}

func (s *Store) SetAntiBioExemption(ctx context.Context, botID string, chatID int64, userID int64, addedBy int64, exempt bool) error {
	if exempt {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO antibio_exemptions (bot_id, chat_id, user_id, added_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (bot_id, chat_id, user_id) DO UPDATE SET
				added_by = EXCLUDED.added_by,
				added_at = NOW()
		`, botID, chatID, userID, addedBy)
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM antibio_exemptions WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3`, botID, chatID, userID)
	return err
}

func (s *Store) ListAntiBioExemptions(ctx context.Context, botID string, chatID int64) ([]domain.UserProfile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.username, u.first_name, u.last_name, u.is_bot
		FROM antibio_exemptions e
		JOIN users u ON u.id = e.user_id
		WHERE e.bot_id=$1 AND e.chat_id=$2
		ORDER BY e.added_at ASC
	`, botID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.UserProfile
	for rows.Next() {
		var user domain.UserProfile
		if err := rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.IsBot); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) CreateJob(ctx context.Context, job domain.Job) error {
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 5
	}
	if job.AvailableAt.IsZero() {
		job.AvailableAt = time.Now()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO jobs (id, bot_id, kind, status, requested_by, report_chat_id, progress, total, attempts, max_attempts, payload_json, error_text, result_summary, available_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12, $13, $14)
	`, job.ID, job.BotID, job.Kind, job.Status, job.RequestedBy, job.ReportChatID, job.Progress, job.Total, job.Attempts, job.MaxAttempts, job.PayloadJSON, job.Error, job.ResultSummary, job.AvailableAt)
	return err
}

func (s *Store) ClaimPendingJobs(ctx context.Context, workerID string, limit int) ([]domain.Job, error) {
	rows, err := s.pool.Query(ctx, `
		WITH picked AS (
			SELECT id
			FROM jobs
			WHERE status IN ('pending', 'retry')
			  AND available_at <= NOW()
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE jobs AS j
		SET status='processing',
		    attempts = attempts + 1,
		    locked_at = NOW(),
		    locked_by = $2,
		    updated_at = NOW()
		WHERE j.id IN (SELECT id FROM picked)
		RETURNING id, bot_id, kind, status, requested_by, report_chat_id, progress, total, attempts, max_attempts, payload_json::text, error_text, result_summary, available_at, locked_at, locked_by, created_at, updated_at, completed_at
	`, limit, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var job domain.Job
		var payload string
		if err := rows.Scan(&job.ID, &job.BotID, &job.Kind, &job.Status, &job.RequestedBy, &job.ReportChatID, &job.Progress, &job.Total, &job.Attempts, &job.MaxAttempts, &payload, &job.Error, &job.ResultSummary, &job.AvailableAt, &job.LockedAt, &job.LockedBy, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt); err != nil {
			return nil, err
		}
		job.PayloadJSON = []byte(payload)
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) GetJob(ctx context.Context, jobID string) (domain.Job, error) {
	var job domain.Job
	var payload string
	err := s.pool.QueryRow(ctx, `
		SELECT id, bot_id, kind, status, requested_by, report_chat_id, progress, total, attempts, max_attempts, payload_json::text, error_text, result_summary, available_at, locked_at, locked_by, created_at, updated_at, completed_at
		FROM jobs
		WHERE id=$1
	`, jobID).Scan(&job.ID, &job.BotID, &job.Kind, &job.Status, &job.RequestedBy, &job.ReportChatID, &job.Progress, &job.Total, &job.Attempts, &job.MaxAttempts, &payload, &job.Error, &job.ResultSummary, &job.AvailableAt, &job.LockedAt, &job.LockedBy, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt)
	if err != nil {
		return domain.Job{}, err
	}
	job.PayloadJSON = []byte(payload)
	return job, nil
}

func (s *Store) ListRecentJobs(ctx context.Context, botID string, limit int) ([]domain.Job, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, bot_id, kind, status, requested_by, report_chat_id, progress, total, attempts, max_attempts, payload_json::text, error_text, result_summary, available_at, locked_at, locked_by, created_at, updated_at, completed_at
		FROM jobs
		WHERE bot_id=$1
		ORDER BY created_at DESC
		LIMIT $2
	`, botID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var job domain.Job
		var payload string
		if err := rows.Scan(&job.ID, &job.BotID, &job.Kind, &job.Status, &job.RequestedBy, &job.ReportChatID, &job.Progress, &job.Total, &job.Attempts, &job.MaxAttempts, &payload, &job.Error, &job.ResultSummary, &job.AvailableAt, &job.LockedAt, &job.LockedBy, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt); err != nil {
			return nil, err
		}
		job.PayloadJSON = []byte(payload)
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) UpdateJobProgress(ctx context.Context, jobID string, status string, progress int, total int, errText string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs
		SET status=$2, progress=$3, total=$4, error_text=$5, updated_at=NOW()
		WHERE id=$1
	`, jobID, status, progress, total, errText)
	return err
}

func (s *Store) MarkJobCompleted(ctx context.Context, jobID string, resultSummary string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs
		SET status='completed', result_summary=$2, completed_at=NOW(), locked_at=NULL, locked_by=NULL, updated_at=NOW()
		WHERE id=$1
	`, jobID, resultSummary)
	return err
}

func (s *Store) MarkJobRetry(ctx context.Context, jobID string, attempts int, errText string, availableAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs
		SET status='retry', attempts=$2, error_text=$3, available_at=$4, locked_at=NULL, locked_by=NULL, updated_at=NOW()
		WHERE id=$1
	`, jobID, attempts, errText, availableAt)
	return err
}

func (s *Store) MarkJobDead(ctx context.Context, jobID string, errText string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs
		SET status='dead', error_text=$2, completed_at=NOW(), locked_at=NULL, locked_by=NULL, updated_at=NOW()
		WHERE id=$1
	`, jobID, errText)
	return err
}

func (s *Store) ListChats(ctx context.Context, botID string) ([]telegram.Chat, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT telegram_chat_id, chat_type, title, username
		FROM chats
		WHERE bot_id=$1
		ORDER BY telegram_chat_id ASC
	`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []telegram.Chat
	for rows.Next() {
		var chat telegram.Chat
		if err := rows.Scan(&chat.ID, &chat.Type, &chat.Title, &chat.Username); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *Store) GetStats(ctx context.Context, botID string) (domain.BotStats, error) {
	stats := domain.BotStats{BotID: botID}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM chats WHERE bot_id=$1`, botID).Scan(&stats.ChatCount); err != nil {
		return domain.BotStats{}, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.UserCount); err != nil {
		return domain.BotStats{}, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM bot_instances WHERE created_by_user_id != 0`).Scan(&stats.CloneCount); err != nil {
		return domain.BotStats{}, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM federations WHERE bot_id=$1`, botID).Scan(&stats.FederationCount); err != nil {
		return domain.BotStats{}, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE bot_id=$1`, botID).Scan(&stats.JobCount); err != nil {
		return domain.BotStats{}, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE bot_id=$1 AND status='dead'`, botID).Scan(&stats.DeadJobCount); err != nil {
		return domain.BotStats{}, err
	}
	return stats, nil
}

func (s *Store) SetGlobalBlacklistUser(ctx context.Context, entry domain.GlobalBlacklistUser, enabled bool) error {
	if enabled {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO global_blacklist_users (bot_id, user_id, reason, created_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (bot_id, user_id) DO UPDATE SET
				reason = EXCLUDED.reason,
				created_by = EXCLUDED.created_by,
				created_at = NOW()
		`, entry.BotID, entry.UserID, entry.Reason, entry.CreatedBy)
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM global_blacklist_users WHERE bot_id=$1 AND user_id=$2`, entry.BotID, entry.UserID)
	return err
}

func (s *Store) GetGlobalBlacklistUser(ctx context.Context, botID string, userID int64) (domain.GlobalBlacklistUser, bool, error) {
	var entry domain.GlobalBlacklistUser
	err := s.pool.QueryRow(ctx, `
		SELECT bot_id, user_id, reason, created_by, created_at
		FROM global_blacklist_users
		WHERE bot_id=$1 AND user_id=$2
	`, botID, userID).Scan(&entry.BotID, &entry.UserID, &entry.Reason, &entry.CreatedBy, &entry.CreatedAt)
	if err == pgx.ErrNoRows {
		return domain.GlobalBlacklistUser{}, false, nil
	}
	return entry, err == nil, err
}

func (s *Store) ListGlobalBlacklistUsers(ctx context.Context, botID string) ([]domain.GlobalBlacklistUser, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT bot_id, user_id, reason, created_by, created_at
		FROM global_blacklist_users
		WHERE bot_id=$1
		ORDER BY created_at ASC
	`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.GlobalBlacklistUser
	for rows.Next() {
		var entry domain.GlobalBlacklistUser
		if err := rows.Scan(&entry.BotID, &entry.UserID, &entry.Reason, &entry.CreatedBy, &entry.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (s *Store) SetGlobalBlacklistChat(ctx context.Context, entry domain.GlobalBlacklistChat, enabled bool) error {
	if enabled {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO global_blacklist_chats (bot_id, chat_id, reason, created_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (bot_id, chat_id) DO UPDATE SET
				reason = EXCLUDED.reason,
				created_by = EXCLUDED.created_by,
				created_at = NOW()
		`, entry.BotID, entry.ChatID, entry.Reason, entry.CreatedBy)
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM global_blacklist_chats WHERE bot_id=$1 AND chat_id=$2`, entry.BotID, entry.ChatID)
	return err
}

func (s *Store) GetGlobalBlacklistChat(ctx context.Context, botID string, chatID int64) (domain.GlobalBlacklistChat, bool, error) {
	var entry domain.GlobalBlacklistChat
	err := s.pool.QueryRow(ctx, `
		SELECT bot_id, chat_id, reason, created_by, created_at
		FROM global_blacklist_chats
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID).Scan(&entry.BotID, &entry.ChatID, &entry.Reason, &entry.CreatedBy, &entry.CreatedAt)
	if err == pgx.ErrNoRows {
		return domain.GlobalBlacklistChat{}, false, nil
	}
	return entry, err == nil, err
}

func (s *Store) ListGlobalBlacklistChats(ctx context.Context, botID string) ([]domain.GlobalBlacklistChat, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT bot_id, chat_id, reason, created_by, created_at
		FROM global_blacklist_chats
		WHERE bot_id=$1
		ORDER BY created_at ASC
	`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.GlobalBlacklistChat
	for rows.Next() {
		var entry domain.GlobalBlacklistChat
		if err := rows.Scan(&entry.BotID, &entry.ChatID, &entry.Reason, &entry.CreatedBy, &entry.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (s *Store) CreateFederation(ctx context.Context, federation domain.Federation) (domain.Federation, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO federations (id, bot_id, short_name, display_name, owner_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, bot_id, short_name, display_name, owner_user_id, created_at
	`, federation.ID, federation.BotID, federation.ShortName, federation.DisplayName, federation.OwnerUserID).Scan(
		&federation.ID, &federation.BotID, &federation.ShortName, &federation.DisplayName, &federation.OwnerUserID, &federation.CreatedAt,
	)
	if err != nil {
		return domain.Federation{}, err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO federation_admins (federation_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT DO NOTHING
	`, federation.ID, federation.OwnerUserID)
	return federation, err
}

func (s *Store) DeleteFederation(ctx context.Context, federationID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM federations WHERE id=$1`, federationID)
	return err
}

func (s *Store) GetFederationByID(ctx context.Context, federationID string) (domain.Federation, error) {
	var federation domain.Federation
	err := s.pool.QueryRow(ctx, `
		SELECT id, bot_id, short_name, display_name, owner_user_id, created_at
		FROM federations
		WHERE id=$1
	`, federationID).Scan(&federation.ID, &federation.BotID, &federation.ShortName, &federation.DisplayName, &federation.OwnerUserID, &federation.CreatedAt)
	return federation, err
}

func (s *Store) GetFederationByShortName(ctx context.Context, botID string, shortName string) (domain.Federation, error) {
	var federation domain.Federation
	err := s.pool.QueryRow(ctx, `
		SELECT id, bot_id, short_name, display_name, owner_user_id, created_at
		FROM federations
		WHERE bot_id=$1 AND LOWER(short_name)=LOWER($2)
	`, botID, shortName).Scan(&federation.ID, &federation.BotID, &federation.ShortName, &federation.DisplayName, &federation.OwnerUserID, &federation.CreatedAt)
	return federation, err
}

func (s *Store) GetFederationByChat(ctx context.Context, botID string, chatID int64) (domain.Federation, error) {
	var federation domain.Federation
	err := s.pool.QueryRow(ctx, `
		SELECT f.id, f.bot_id, f.short_name, f.display_name, f.owner_user_id, f.created_at
		FROM federation_chats fc
		JOIN federations f ON f.id = fc.federation_id
		WHERE fc.bot_id=$1 AND fc.chat_id=$2
	`, botID, chatID).Scan(&federation.ID, &federation.BotID, &federation.ShortName, &federation.DisplayName, &federation.OwnerUserID, &federation.CreatedAt)
	return federation, err
}

func (s *Store) ListFederationsForUser(ctx context.Context, botID string, userID int64) ([]domain.Federation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT f.id, f.bot_id, f.short_name, f.display_name, f.owner_user_id, f.created_at
		FROM federations f
		LEFT JOIN federation_admins fa ON fa.federation_id = f.id
		WHERE f.bot_id=$1 AND (f.owner_user_id=$2 OR fa.user_id=$2)
		ORDER BY f.created_at ASC
	`, botID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Federation
	for rows.Next() {
		var federation domain.Federation
		if err := rows.Scan(&federation.ID, &federation.BotID, &federation.ShortName, &federation.DisplayName, &federation.OwnerUserID, &federation.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, federation)
	}
	return result, rows.Err()
}

func (s *Store) JoinFederation(ctx context.Context, federationID string, botID string, chatID int64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO federation_chats (federation_id, bot_id, chat_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, federationID, botID, chatID)
	return err
}

func (s *Store) LeaveFederation(ctx context.Context, federationID string, botID string, chatID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM federation_chats WHERE federation_id=$1 AND bot_id=$2 AND chat_id=$3`, federationID, botID, chatID)
	return err
}

func (s *Store) ListFederationChats(ctx context.Context, federationID string) ([]telegram.Chat, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.telegram_chat_id, c.chat_type, c.title, c.username
		FROM federation_chats fc
		JOIN chats c ON c.bot_id = fc.bot_id AND c.telegram_chat_id = fc.chat_id
		WHERE fc.federation_id=$1
		ORDER BY c.telegram_chat_id ASC
	`, federationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []telegram.Chat
	for rows.Next() {
		var chat telegram.Chat
		if err := rows.Scan(&chat.ID, &chat.Type, &chat.Title, &chat.Username); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *Store) SetFederationAdmin(ctx context.Context, federationID string, userID int64, role string, enabled bool) error {
	if enabled {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO federation_admins (federation_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (federation_id, user_id) DO UPDATE SET role=EXCLUDED.role, added_at=NOW()
		`, federationID, userID, role)
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM federation_admins WHERE federation_id=$1 AND user_id=$2`, federationID, userID)
	return err
}

func (s *Store) ListFederationAdmins(ctx context.Context, federationID string) ([]domain.FederationAdmin, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT federation_id, user_id, role, added_at
		FROM federation_admins
		WHERE federation_id=$1
		ORDER BY added_at ASC
	`, federationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []domain.FederationAdmin
	for rows.Next() {
		var admin domain.FederationAdmin
		if err := rows.Scan(&admin.FederationID, &admin.UserID, &admin.Role, &admin.AddedAt); err != nil {
			return nil, err
		}
		admins = append(admins, admin)
	}
	return admins, rows.Err()
}

func (s *Store) TransferFederation(ctx context.Context, federationID string, newOwnerUserID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE federations SET owner_user_id=$2 WHERE id=$1`, federationID, newOwnerUserID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO federation_admins (federation_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT (federation_id, user_id) DO UPDATE SET role='owner', added_at=NOW()
	`, federationID, newOwnerUserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SetFederationBan(ctx context.Context, ban domain.FederationBan, enabled bool) error {
	if enabled {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO federation_bans (federation_id, user_id, reason, banned_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (federation_id, user_id) DO UPDATE SET
				reason = EXCLUDED.reason,
				banned_by = EXCLUDED.banned_by,
				banned_at = NOW()
		`, ban.FederationID, ban.UserID, ban.Reason, ban.BannedBy)
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM federation_bans WHERE federation_id=$1 AND user_id=$2`, ban.FederationID, ban.UserID)
	return err
}

func (s *Store) GetFederationBan(ctx context.Context, federationID string, userID int64) (domain.FederationBan, bool, error) {
	var ban domain.FederationBan
	err := s.pool.QueryRow(ctx, `
		SELECT federation_id, user_id, reason, banned_by, banned_at
		FROM federation_bans
		WHERE federation_id=$1 AND user_id=$2
	`, federationID, userID).Scan(&ban.FederationID, &ban.UserID, &ban.Reason, &ban.BannedBy, &ban.BannedAt)
	if err == pgx.ErrNoRows {
		return domain.FederationBan{}, false, nil
	}
	return ban, err == nil, err
}

func (s *Store) ExportUserData(ctx context.Context, botID string, userID int64) (map[string]any, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	export := map[string]any{
		"user": user,
	}

	var warnings []map[string]any
	rows, err := s.pool.Query(ctx, `SELECT chat_id, warning_count, last_reason, updated_at FROM warnings WHERE bot_id=$1 AND user_id=$2`, botID, userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var chatID int64
		var count int
		var reason string
		var updatedAt time.Time
		if err := rows.Scan(&chatID, &count, &reason, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		warnings = append(warnings, map[string]any{"chat_id": chatID, "count": count, "reason": reason, "updated_at": updatedAt})
	}
	rows.Close()
	export["warnings"] = warnings

	var approvals []int64
	rows, err = s.pool.Query(ctx, `SELECT chat_id FROM approvals WHERE bot_id=$1 AND user_id=$2`, botID, userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			rows.Close()
			return nil, err
		}
		approvals = append(approvals, chatID)
	}
	rows.Close()
	export["approved_in_chats"] = approvals

	afk, err := s.GetAFK(ctx, botID, userID)
	if err == nil && afk.UserID != 0 {
		export["afk"] = afk
	}

	exemptRows, err := s.pool.Query(ctx, `SELECT chat_id FROM antibio_exemptions WHERE bot_id=$1 AND user_id=$2`, botID, userID)
	if err != nil {
		return nil, err
	}
	var exemptChatIDs []int64
	for exemptRows.Next() {
		var chatID int64
		if err := exemptRows.Scan(&chatID); err != nil {
			exemptRows.Close()
			return nil, err
		}
		exemptChatIDs = append(exemptChatIDs, chatID)
	}
	exemptRows.Close()
	export["antibio_exemptions"] = exemptChatIDs

	return export, nil
}

func (s *Store) DeleteUserData(ctx context.Context, botID string, userID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM afk_states WHERE bot_id=$1 AND user_id=$2`, []any{botID, userID}},
		{`DELETE FROM warnings WHERE bot_id=$1 AND user_id=$2`, []any{botID, userID}},
		{`DELETE FROM approvals WHERE bot_id=$1 AND user_id=$2`, []any{botID, userID}},
		{`DELETE FROM antibio_exemptions WHERE bot_id=$1 AND user_id=$2`, []any{botID, userID}},
		{`DELETE FROM chat_roles WHERE bot_id=$1 AND user_id=$2 AND role IN ('mod', 'muter')`, []any{botID, userID}},
		{`DELETE FROM bot_roles WHERE bot_id=$1 AND user_id=$2 AND role='sudo'`, []any{botID, userID}},
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query.sql, query.args...); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func encodePayload(payload any) []byte {
	body, _ := json.Marshal(payload)
	return body
}
