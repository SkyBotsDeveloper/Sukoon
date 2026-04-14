package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
)

func (s *Store) SetChatConnection(ctx context.Context, botID string, userID int64, chatID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO chat_connections (bot_id, user_id, chat_id, connected_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (bot_id, user_id) DO UPDATE SET
			chat_id = EXCLUDED.chat_id,
			connected_at = NOW()
	`, botID, userID, chatID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO chat_connection_history (bot_id, user_id, chat_id, connected_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (bot_id, user_id, chat_id) DO UPDATE SET
			connected_at = NOW()
	`, botID, userID, chatID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ClearChatConnection(ctx context.Context, botID string, userID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM chat_connections WHERE bot_id=$1 AND user_id=$2`, botID, userID)
	return err
}

func (s *Store) GetChatConnection(ctx context.Context, botID string, userID int64) (domain.ChatConnection, error) {
	return s.scanChatConnection(ctx, `
		SELECT c.bot_id, c.user_id, c.chat_id, ch.chat_type, ch.title, ch.username, c.connected_at
		FROM chat_connections c
		JOIN chats ch ON ch.bot_id = c.bot_id AND ch.telegram_chat_id = c.chat_id
		WHERE c.bot_id=$1 AND c.user_id=$2
	`, botID, userID)
}

func (s *Store) GetLastChatConnection(ctx context.Context, botID string, userID int64) (domain.ChatConnection, error) {
	return s.scanChatConnection(ctx, `
		SELECT h.bot_id, h.user_id, h.chat_id, ch.chat_type, ch.title, ch.username, h.connected_at
		FROM chat_connection_history h
		JOIN chats ch ON ch.bot_id = h.bot_id AND ch.telegram_chat_id = h.chat_id
		WHERE h.bot_id=$1 AND h.user_id=$2
		ORDER BY h.connected_at DESC
		LIMIT 1
	`, botID, userID)
}

func (s *Store) ListChatConnections(ctx context.Context, botID string, userID int64, limit int) ([]domain.ChatConnection, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT h.bot_id, h.user_id, h.chat_id, ch.chat_type, ch.title, ch.username, h.connected_at
		FROM chat_connection_history h
		JOIN chats ch ON ch.bot_id = h.bot_id AND ch.telegram_chat_id = h.chat_id
		WHERE h.bot_id=$1 AND h.user_id=$2
		ORDER BY h.connected_at DESC
		LIMIT $3
	`, botID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	connections := []domain.ChatConnection{}
	for rows.Next() {
		var connection domain.ChatConnection
		if err := rows.Scan(&connection.BotID, &connection.UserID, &connection.ChatID, &connection.ChatType, &connection.ChatTitle, &connection.ChatUsername, &connection.ConnectedAt); err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}
	return connections, rows.Err()
}

func (s *Store) scanChatConnection(ctx context.Context, query string, args ...any) (domain.ChatConnection, error) {
	var connection domain.ChatConnection
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&connection.BotID,
		&connection.UserID,
		&connection.ChatID,
		&connection.ChatType,
		&connection.ChatTitle,
		&connection.ChatUsername,
		&connection.ConnectedAt,
	)
	if err == pgx.ErrNoRows {
		return domain.ChatConnection{}, err
	}
	return connection, err
}
