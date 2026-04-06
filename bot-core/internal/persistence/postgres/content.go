package postgres

import (
	"context"

	"sukoon/bot-core/internal/domain"
)

func (s *Store) UpsertNote(ctx context.Context, note domain.Note) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notes (bot_id, chat_id, name, text, parse_mode, buttons_json, created_by)
		VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), '[]')::jsonb, $7)
		ON CONFLICT (bot_id, chat_id, name) DO UPDATE SET
			text = EXCLUDED.text,
			parse_mode = EXCLUDED.parse_mode,
			buttons_json = EXCLUDED.buttons_json,
			created_by = EXCLUDED.created_by,
			updated_at = NOW()
	`, note.BotID, note.ChatID, note.Name, note.Text, note.ParseMode, note.ButtonsJSON, note.CreatedBy)
	return err
}

func (s *Store) GetNote(ctx context.Context, botID string, chatID int64, name string) (domain.Note, error) {
	var note domain.Note
	err := s.pool.QueryRow(ctx, `
		SELECT bot_id, chat_id, name, text, parse_mode, buttons_json::text, created_by, created_at, updated_at
		FROM notes
		WHERE bot_id=$1 AND chat_id=$2 AND name=$3
	`, botID, chatID, name).Scan(
		&note.BotID, &note.ChatID, &note.Name, &note.Text, &note.ParseMode, &note.ButtonsJSON, &note.CreatedBy, &note.CreatedAt, &note.UpdatedAt,
	)
	return note, err
}

func (s *Store) DeleteNote(ctx context.Context, botID string, chatID int64, name string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM notes WHERE bot_id=$1 AND chat_id=$2 AND name=$3`, botID, chatID, name)
	return err
}

func (s *Store) UpsertFilter(ctx context.Context, filter domain.FilterRule) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO filters (bot_id, chat_id, trigger, match_mode, response_text, parse_mode, buttons_json, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE(NULLIF($7, ''), '[]')::jsonb, $8)
		ON CONFLICT (bot_id, chat_id, trigger) DO UPDATE SET
			match_mode = EXCLUDED.match_mode,
			response_text = EXCLUDED.response_text,
			parse_mode = EXCLUDED.parse_mode,
			buttons_json = EXCLUDED.buttons_json,
			created_by = EXCLUDED.created_by,
			updated_at = NOW()
	`, filter.BotID, filter.ChatID, filter.Trigger, filter.MatchMode, filter.ResponseText, filter.ParseMode, filter.ButtonsJSON, filter.CreatedBy)
	return err
}

func (s *Store) DeleteFilter(ctx context.Context, botID string, chatID int64, trigger string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM filters WHERE bot_id=$1 AND chat_id=$2 AND trigger=$3`, botID, chatID, trigger)
	return err
}

func (s *Store) ListFilters(ctx context.Context, botID string, chatID int64) ([]domain.FilterRule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, bot_id, chat_id, trigger, match_mode, response_text, parse_mode, buttons_json::text, created_by, created_at, updated_at
		FROM filters
		WHERE bot_id=$1 AND chat_id=$2
		ORDER BY id ASC
	`, botID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []domain.FilterRule
	for rows.Next() {
		var filter domain.FilterRule
		if err := rows.Scan(&filter.ID, &filter.BotID, &filter.ChatID, &filter.Trigger, &filter.MatchMode, &filter.ResponseText, &filter.ParseMode, &filter.ButtonsJSON, &filter.CreatedBy, &filter.CreatedAt, &filter.UpdatedAt); err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, rows.Err()
}

func (s *Store) SetWelcome(ctx context.Context, botID string, chatID int64, enabled bool, text string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET welcome_enabled=$3, welcome_text=$4, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, enabled, text)
	return err
}

func (s *Store) SetGoodbye(ctx context.Context, botID string, chatID int64, enabled bool, text string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET goodbye_enabled=$3, goodbye_text=$4, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, enabled, text)
	return err
}

func (s *Store) SetRules(ctx context.Context, botID string, chatID int64, text string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE chat_settings
		SET rules_text=$3, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, botID, chatID, text)
	return err
}

func (s *Store) UpsertLock(ctx context.Context, lock domain.LockRule) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO locks (bot_id, chat_id, lock_type, action)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (bot_id, chat_id, lock_type) DO UPDATE SET action = EXCLUDED.action
	`, lock.BotID, lock.ChatID, lock.LockType, lock.Action)
	return err
}

func (s *Store) DeleteLock(ctx context.Context, botID string, chatID int64, lockType string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM locks WHERE bot_id=$1 AND chat_id=$2 AND lock_type=$3`, botID, chatID, lockType)
	return err
}

func (s *Store) ListLocks(ctx context.Context, botID string, chatID int64) ([]domain.LockRule, error) {
	rows, err := s.pool.Query(ctx, `SELECT bot_id, chat_id, lock_type, action FROM locks WHERE bot_id=$1 AND chat_id=$2 ORDER BY lock_type ASC`, botID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locks []domain.LockRule
	for rows.Next() {
		var lock domain.LockRule
		if err := rows.Scan(&lock.BotID, &lock.ChatID, &lock.LockType, &lock.Action); err != nil {
			return nil, err
		}
		locks = append(locks, lock)
	}
	return locks, rows.Err()
}

func (s *Store) AddBlocklistRule(ctx context.Context, rule domain.BlocklistRule) (domain.BlocklistRule, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO blocklist_rules (bot_id, chat_id, pattern, match_mode, action, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (bot_id, chat_id, pattern, match_mode) DO UPDATE SET
			action = EXCLUDED.action,
			created_by = EXCLUDED.created_by
		RETURNING id, created_at
	`, rule.BotID, rule.ChatID, rule.Pattern, rule.MatchMode, rule.Action, rule.CreatedBy).Scan(&rule.ID, &rule.CreatedAt)
	return rule, err
}

func (s *Store) DeleteBlocklistRule(ctx context.Context, botID string, chatID int64, pattern string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM blocklist_rules WHERE bot_id=$1 AND chat_id=$2 AND pattern=$3`, botID, chatID, pattern)
	return err
}

func (s *Store) ListBlocklistRules(ctx context.Context, botID string, chatID int64) ([]domain.BlocklistRule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, bot_id, chat_id, pattern, match_mode, action, created_by, created_at
		FROM blocklist_rules
		WHERE bot_id=$1 AND chat_id=$2
		ORDER BY id ASC
	`, botID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []domain.BlocklistRule
	for rows.Next() {
		var rule domain.BlocklistRule
		if err := rows.Scan(&rule.ID, &rule.BotID, &rule.ChatID, &rule.Pattern, &rule.MatchMode, &rule.Action, &rule.CreatedBy, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Store) SetAntiflood(ctx context.Context, settings domain.AntifloodSettings) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE antiflood_settings
		SET enabled=$3, flood_limit=$4, window_seconds=$5, action=$6, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, settings.BotID, settings.ChatID, settings.Enabled, settings.Limit, settings.WindowSeconds, settings.Action)
	return err
}
