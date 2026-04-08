package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
)

func (s *Store) SetCaptchaSettings(ctx context.Context, settings domain.CaptchaSettings) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE captcha_settings
		SET enabled=$3, mode=$4, timeout_seconds=$5, rules_required=$6, auto_unmute_seconds=$7, kick_on_timeout=$8, button_text=$9, failure_action=$10, challenge_digits=$11, updated_at=NOW()
		WHERE bot_id=$1 AND chat_id=$2
	`, settings.BotID, settings.ChatID, settings.Enabled, settings.Mode, settings.TimeoutSeconds, settings.RulesRequired, settings.AutoUnmuteSeconds, settings.KickOnTimeout, settings.ButtonText, settings.FailureAction, settings.ChallengeDigits)
	return err
}

func (s *Store) CreateCaptchaChallenge(ctx context.Context, challenge domain.CaptchaChallenge) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO captcha_challenges (id, bot_id, chat_id, user_id, prompt, answer, message_id, expires_at, status, mode, rules_required, rules_accepted, timeout_action, failure_action)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', $9, $10, $11, $12, $13)
		ON CONFLICT (bot_id, chat_id, user_id) WHERE status = 'pending'
		DO UPDATE SET
			id = EXCLUDED.id,
			prompt = EXCLUDED.prompt,
			answer = EXCLUDED.answer,
			message_id = EXCLUDED.message_id,
			expires_at = EXCLUDED.expires_at,
			mode = EXCLUDED.mode,
			rules_required = EXCLUDED.rules_required,
			rules_accepted = EXCLUDED.rules_accepted,
			timeout_action = EXCLUDED.timeout_action,
			failure_action = EXCLUDED.failure_action,
			created_at = NOW()
	`, challenge.ID, challenge.BotID, challenge.ChatID, challenge.UserID, challenge.Prompt, challenge.Answer, challenge.MessageID, challenge.ExpiresAt, challenge.Mode, challenge.RulesRequired, challenge.RulesAccepted, challenge.TimeoutAction, challenge.FailureAction)
	return err
}

func (s *Store) GetCaptchaChallengeByID(ctx context.Context, challengeID string) (domain.CaptchaChallenge, error) {
	var challenge domain.CaptchaChallenge
	err := s.pool.QueryRow(ctx, `
		SELECT id, bot_id, chat_id, user_id, prompt, answer, message_id, expires_at, status, mode, rules_required, rules_accepted, timeout_action, failure_action
		FROM captcha_challenges
		WHERE id=$1
		LIMIT 1
	`, challengeID).Scan(
		&challenge.ID, &challenge.BotID, &challenge.ChatID, &challenge.UserID, &challenge.Prompt, &challenge.Answer, &challenge.MessageID, &challenge.ExpiresAt, &challenge.Status, &challenge.Mode, &challenge.RulesRequired, &challenge.RulesAccepted, &challenge.TimeoutAction, &challenge.FailureAction,
	)
	return challenge, err
}

func (s *Store) GetPendingCaptchaChallenge(ctx context.Context, botID string, chatID int64, userID int64) (domain.CaptchaChallenge, error) {
	var challenge domain.CaptchaChallenge
	err := s.pool.QueryRow(ctx, `
		SELECT id, bot_id, chat_id, user_id, prompt, answer, message_id, expires_at, status, mode, rules_required, rules_accepted, timeout_action, failure_action
		FROM captcha_challenges
		WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3 AND status='pending'
		ORDER BY created_at DESC
		LIMIT 1
	`, botID, chatID, userID).Scan(
		&challenge.ID, &challenge.BotID, &challenge.ChatID, &challenge.UserID, &challenge.Prompt, &challenge.Answer, &challenge.MessageID, &challenge.ExpiresAt, &challenge.Status, &challenge.Mode, &challenge.RulesRequired, &challenge.RulesAccepted, &challenge.TimeoutAction, &challenge.FailureAction,
	)
	return challenge, err
}

func (s *Store) MarkCaptchaRulesAccepted(ctx context.Context, challengeID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE captcha_challenges SET rules_accepted=TRUE WHERE id=$1`, challengeID)
	return err
}

func (s *Store) MarkCaptchaSolved(ctx context.Context, challengeID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE captcha_challenges SET status='solved' WHERE id=$1`, challengeID)
	return err
}

func (s *Store) ListExpiredCaptchaChallenges(ctx context.Context, now time.Time, limit int) ([]domain.CaptchaChallenge, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, bot_id, chat_id, user_id, prompt, answer, message_id, expires_at, status, mode, rules_required, rules_accepted, timeout_action, failure_action
		FROM captcha_challenges
		WHERE status='pending' AND expires_at <= $1
		ORDER BY expires_at ASC
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var challenges []domain.CaptchaChallenge
	for rows.Next() {
		var challenge domain.CaptchaChallenge
		if err := rows.Scan(&challenge.ID, &challenge.BotID, &challenge.ChatID, &challenge.UserID, &challenge.Prompt, &challenge.Answer, &challenge.MessageID, &challenge.ExpiresAt, &challenge.Status, &challenge.Mode, &challenge.RulesRequired, &challenge.RulesAccepted, &challenge.TimeoutAction, &challenge.FailureAction); err != nil {
			return nil, err
		}
		challenges = append(challenges, challenge)
	}
	return challenges, rows.Err()
}

func (s *Store) MarkCaptchaExpired(ctx context.Context, challengeID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE captcha_challenges SET status='expired' WHERE id=$1`, challengeID)
	return err
}

func (s *Store) SetAFK(ctx context.Context, state domain.AFKState) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO afk_states (bot_id, user_id, reason, set_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (bot_id, user_id) DO UPDATE SET reason=EXCLUDED.reason, set_at=EXCLUDED.set_at
	`, state.BotID, state.UserID, state.Reason, state.SetAt)
	return err
}

func (s *Store) ClearAFK(ctx context.Context, botID string, userID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM afk_states WHERE bot_id=$1 AND user_id=$2`, botID, userID)
	return err
}

func (s *Store) GetAFK(ctx context.Context, botID string, userID int64) (domain.AFKState, error) {
	var state domain.AFKState
	err := s.pool.QueryRow(ctx, `SELECT bot_id, user_id, reason, set_at FROM afk_states WHERE bot_id=$1 AND user_id=$2`, botID, userID).Scan(&state.BotID, &state.UserID, &state.Reason, &state.SetAt)
	if err == pgx.ErrNoRows {
		return domain.AFKState{}, nil
	}
	return state, err
}
