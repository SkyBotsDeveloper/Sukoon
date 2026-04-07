package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/util"
	"sukoon/bot-core/migrations"
)

type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

var _ persistence.Store = (*Store)(nil)

func New(ctx context.Context, databaseURL string, logger *slog.Logger) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Store{pool: pool, logger: logger}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		version := entry.Name()

		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}

		content, err := migrations.Files.ReadFile(version)
		if err != nil {
			return err
		}

		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			tx.Rollback(ctx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}
		s.logger.Info("migration applied", "version", version)
	}

	return nil
}

func (s *Store) UpsertPrimaryBot(ctx context.Context, bot domain.BotInstance, ownerUserIDs []int64) (domain.BotInstance, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.BotInstance{}, err
	}
	defer tx.Rollback(ctx)

	if bot.ID == "" {
		bot.ID = util.RandomID(16)
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO bot_instances (id, slug, display_name, telegram_token, webhook_key, webhook_secret, username, is_primary, created_by_user_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, 0, 'active')
		ON CONFLICT (slug) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			telegram_token = EXCLUDED.telegram_token,
			webhook_key = EXCLUDED.webhook_key,
			webhook_secret = EXCLUDED.webhook_secret,
			username = EXCLUDED.username,
			is_primary = TRUE,
			created_by_user_id = EXCLUDED.created_by_user_id,
			status = 'active',
			updated_at = NOW()
		RETURNING id, slug, display_name, telegram_token, webhook_key, webhook_secret, username, is_primary, created_by_user_id, status, created_at, updated_at
	`, bot.ID, bot.Slug, bot.DisplayName, bot.TelegramToken, bot.WebhookKey, bot.WebhookSecret, bot.Username).Scan(
		&bot.ID, &bot.Slug, &bot.DisplayName, &bot.TelegramToken, &bot.WebhookKey, &bot.WebhookSecret, &bot.Username, &bot.IsPrimary, &bot.CreatedByUserID, &bot.Status, &bot.CreatedAt, &bot.UpdatedAt,
	); err != nil {
		return domain.BotInstance{}, err
	}

	for _, ownerID := range ownerUserIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bot_roles (bot_id, user_id, role)
			VALUES ($1, $2, 'owner')
			ON CONFLICT DO NOTHING
		`, bot.ID, ownerID); err != nil {
			return domain.BotInstance{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.BotInstance{}, err
	}
	return bot, nil
}

func (s *Store) ResolveBotByWebhookKey(ctx context.Context, webhookKey string) (domain.BotInstance, error) {
	return s.getBot(ctx, `SELECT id, slug, display_name, telegram_token, webhook_key, webhook_secret, username, is_primary, created_by_user_id, status, created_at, updated_at FROM bot_instances WHERE webhook_key=$1 AND status='active'`, webhookKey)
}

func (s *Store) GetBotByID(ctx context.Context, botID string) (domain.BotInstance, error) {
	return s.getBot(ctx, `SELECT id, slug, display_name, telegram_token, webhook_key, webhook_secret, username, is_primary, created_by_user_id, status, created_at, updated_at FROM bot_instances WHERE id=$1`, botID)
}

func (s *Store) getBot(ctx context.Context, query string, arg any) (domain.BotInstance, error) {
	var bot domain.BotInstance
	err := s.pool.QueryRow(ctx, query, arg).Scan(
		&bot.ID,
		&bot.Slug,
		&bot.DisplayName,
		&bot.TelegramToken,
		&bot.WebhookKey,
		&bot.WebhookSecret,
		&bot.Username,
		&bot.IsPrimary,
		&bot.CreatedByUserID,
		&bot.Status,
		&bot.CreatedAt,
		&bot.UpdatedAt,
	)
	if err != nil {
		return domain.BotInstance{}, err
	}
	return bot, nil
}

func (s *Store) CreateCloneBot(ctx context.Context, bot domain.BotInstance, ownerUserID int64) (domain.BotInstance, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.BotInstance{}, err
	}
	defer tx.Rollback(ctx)

	if bot.ID == "" {
		bot.ID = util.RandomID(16)
	}
	bot.IsPrimary = false
	bot.CreatedByUserID = ownerUserID
	bot.Status = "active"

	var existingClones int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM bot_instances
		WHERE created_by_user_id = $1 AND is_primary = FALSE AND status = 'active'
	`, ownerUserID).Scan(&existingClones); err != nil {
		return domain.BotInstance{}, err
	}
	if existingClones > 0 {
		return domain.BotInstance{}, persistence.ErrCloneLimitReached
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO bot_instances (id, slug, display_name, telegram_token, webhook_key, webhook_secret, username, is_primary, created_by_user_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE, $8, 'active')
		RETURNING id, slug, display_name, telegram_token, webhook_key, webhook_secret, username, is_primary, created_by_user_id, status, created_at, updated_at
	`, bot.ID, bot.Slug, bot.DisplayName, bot.TelegramToken, bot.WebhookKey, bot.WebhookSecret, bot.Username, ownerUserID).Scan(
		&bot.ID, &bot.Slug, &bot.DisplayName, &bot.TelegramToken, &bot.WebhookKey, &bot.WebhookSecret, &bot.Username, &bot.IsPrimary, &bot.CreatedByUserID, &bot.Status, &bot.CreatedAt, &bot.UpdatedAt,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "bot_instances_one_active_clone_per_owner" {
			return domain.BotInstance{}, persistence.ErrCloneLimitReached
		}
		return domain.BotInstance{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO bot_roles (bot_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT DO NOTHING
	`, bot.ID, ownerUserID); err != nil {
		return domain.BotInstance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.BotInstance{}, err
	}
	return bot, nil
}

func (s *Store) DeleteBotInstance(ctx context.Context, botID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM bot_instances WHERE id=$1`, botID)
	return err
}

func (s *Store) ListOwnedBots(ctx context.Context, ownerUserID int64) ([]domain.BotInstance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT b.id, b.slug, b.display_name, b.telegram_token, b.webhook_key, b.webhook_secret, b.username, b.is_primary, b.created_by_user_id, b.status, b.created_at, b.updated_at
		FROM bot_instances b
		JOIN bot_roles r ON r.bot_id = b.id
		WHERE r.user_id = $1 AND r.role = 'owner'
		ORDER BY b.created_at ASC
	`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bots []domain.BotInstance
	for rows.Next() {
		var bot domain.BotInstance
		if err := rows.Scan(&bot.ID, &bot.Slug, &bot.DisplayName, &bot.TelegramToken, &bot.WebhookKey, &bot.WebhookSecret, &bot.Username, &bot.IsPrimary, &bot.CreatedByUserID, &bot.Status, &bot.CreatedAt, &bot.UpdatedAt); err != nil {
			return nil, err
		}
		bots = append(bots, bot)
	}
	return bots, rows.Err()
}

func (s *Store) EnqueueUpdate(ctx context.Context, botID string, updateID int64, payload []byte) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO telegram_updates (bot_id, update_id, payload_json)
		VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (bot_id, update_id) DO NOTHING
	`, botID, updateID, payload)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) ClaimPendingUpdates(ctx context.Context, workerID string, limit int) ([]domain.QueuedUpdate, error) {
	rows, err := s.pool.Query(ctx, `
		WITH picked AS (
			SELECT id
			FROM telegram_updates
			WHERE status = 'pending'
			  AND available_at <= NOW()
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE telegram_updates AS t
		SET status = 'processing',
		    attempts = attempts + 1,
		    locked_at = NOW(),
		    locked_by = $2
		WHERE t.id IN (SELECT id FROM picked)
		RETURNING t.id, t.bot_id, t.update_id, t.payload_json::text, t.attempts, t.created_at
	`, limit, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var updates []domain.QueuedUpdate
	for rows.Next() {
		var update domain.QueuedUpdate
		var payload string
		if err := rows.Scan(&update.ID, &update.BotID, &update.UpdateID, &payload, &update.Attempts, &update.CreatedAt); err != nil {
			return nil, err
		}
		update.Payload = []byte(payload)
		updates = append(updates, update)
	}
	return updates, rows.Err()
}

func (s *Store) MarkUpdateCompleted(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE telegram_updates
		SET status='completed', processed_at=NOW(), locked_at=NULL, locked_by=NULL
		WHERE id=$1
	`, id)
	return err
}

func (s *Store) MarkUpdateRetry(ctx context.Context, id int64, attempts int, lastError string, availableAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE telegram_updates
		SET status='pending', last_error=$2, available_at=$3, locked_at=NULL, locked_by=NULL
		WHERE id=$1
	`, id, lastError, availableAt)
	return err
}

func (s *Store) MarkUpdateDead(ctx context.Context, id int64, lastError string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE telegram_updates
		SET status='dead', last_error=$2, processed_at=NOW(), locked_at=NULL, locked_by=NULL
		WHERE id=$1
	`, id, lastError)
	return err
}

func (s *Store) EnsureChat(ctx context.Context, botID string, chat telegram.Chat) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO chats (bot_id, telegram_chat_id, chat_type, title, username)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (bot_id, telegram_chat_id) DO UPDATE SET
			chat_type = EXCLUDED.chat_type,
			title = EXCLUDED.title,
			username = EXCLUDED.username,
			updated_at = NOW()
	`, botID, chat.ID, chat.Type, chat.Title, chat.Username); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO chat_settings (bot_id, chat_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, botID, chat.ID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO moderation_settings (bot_id, chat_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, botID, chat.ID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO antiflood_settings (bot_id, chat_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, botID, chat.ID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO captcha_settings (bot_id, chat_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, botID, chat.ID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO antiabuse_settings (bot_id, chat_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, botID, chat.ID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO antibio_settings (bot_id, chat_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, botID, chat.ID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Store) EnsureUser(ctx context.Context, user telegram.User) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, username, first_name, last_name, is_bot, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			is_bot = EXCLUDED.is_bot,
			updated_at = NOW()
	`, user.ID, user.Username, user.FirstName, user.LastName, user.IsBot)
	return err
}

func (s *Store) GetUserByID(ctx context.Context, userID int64) (domain.UserProfile, error) {
	var user domain.UserProfile
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, first_name, last_name, is_bot
		FROM users
		WHERE id=$1
	`, userID).Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.IsBot)
	return user, err
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (domain.UserProfile, error) {
	var user domain.UserProfile
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, first_name, last_name, is_bot
		FROM users
		WHERE LOWER(username)=LOWER($1)
	`, username).Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.IsBot)
	return user, err
}

func (s *Store) GetBotRoles(ctx context.Context, botID string, userID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT role FROM bot_roles WHERE bot_id=$1 AND user_id=$2`, botID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (s *Store) SetBotRole(ctx context.Context, botID string, userID int64, role string, enabled bool) error {
	if enabled {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO bot_roles (bot_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, botID, userID, role)
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM bot_roles WHERE bot_id=$1 AND user_id=$2 AND role=$3`, botID, userID, role)
	return err
}

func (s *Store) ListBotRoleUsers(ctx context.Context, botID string, role string) ([]domain.UserProfile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.username, u.first_name, u.last_name, u.is_bot
		FROM bot_roles r
		JOIN users u ON u.id = r.user_id
		WHERE r.bot_id=$1 AND r.role=$2
		ORDER BY u.id ASC
	`, botID, role)
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

func (s *Store) GetChatRoles(ctx context.Context, botID string, chatID int64, userID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT role FROM chat_roles WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3`, botID, chatID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (s *Store) SetChatRole(ctx context.Context, botID string, chatID int64, userID int64, role string, grantedBy int64, enabled bool) error {
	if enabled {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO chat_roles (bot_id, chat_id, user_id, role, granted_by)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT DO NOTHING
		`, botID, chatID, userID, role, grantedBy)
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM chat_roles WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3 AND role=$4`, botID, chatID, userID, role)
	return err
}

func (s *Store) ListChatRoleUsers(ctx context.Context, botID string, chatID int64, role string) ([]domain.UserProfile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.username, u.first_name, u.last_name, u.is_bot
		FROM chat_roles r
		JOIN users u ON u.id = r.user_id
		WHERE r.bot_id=$1 AND r.chat_id=$2 AND r.role=$3
		ORDER BY u.id ASC
	`, botID, chatID, role)
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

func (s *Store) IsApproved(ctx context.Context, botID string, chatID int64, userID int64) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM approvals WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3)`, botID, chatID, userID).Scan(&exists)
	return exists, err
}

func (s *Store) SetApproval(ctx context.Context, botID string, chatID int64, userID int64, approvedBy int64, approved bool) error {
	if approved {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO approvals (bot_id, chat_id, user_id, approved_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (bot_id, chat_id, user_id) DO UPDATE SET
				approved_by = EXCLUDED.approved_by,
				approved_at = NOW()
		`, botID, chatID, userID, approvedBy)
		return err
	}

	_, err := s.pool.Exec(ctx, `DELETE FROM approvals WHERE bot_id=$1 AND chat_id=$2 AND user_id=$3`, botID, chatID, userID)
	return err
}

func (s *Store) ListApprovedUsers(ctx context.Context, botID string, chatID int64) ([]int64, error) {
	rows, err := s.pool.Query(ctx, `SELECT user_id FROM approvals WHERE bot_id=$1 AND chat_id=$2 ORDER BY approved_at ASC`, botID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		result = append(result, userID)
	}
	return result, rows.Err()
}
