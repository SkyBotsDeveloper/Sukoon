package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"sukoon/bot-core/internal/domain"
)

type Client interface {
	SendMessage(ctx context.Context, chatID int64, text string, options SendMessageOptions) (Message, error)
	SendPhoto(ctx context.Context, chatID int64, photo string, options SendPhotoOptions) (Message, error)
	EditMessageText(ctx context.Context, chatID int64, messageID int64, text string, options EditMessageTextOptions) error
	DeleteMessage(ctx context.Context, chatID int64, messageID int64) error
	PinChatMessage(ctx context.Context, chatID int64, messageID int64, disableNotification bool) error
	UnpinChatMessage(ctx context.Context, chatID int64, messageID *int64) error
	UnpinAllChatMessages(ctx context.Context, chatID int64) error
	PromoteChatMember(ctx context.Context, chatID int64, userID int64, permissions PromotePermissions) error
	BanChatMember(ctx context.Context, chatID int64, userID int64, untilDate *time.Time, revokeMessages bool) error
	UnbanChatMember(ctx context.Context, chatID int64, userID int64, onlyIfBanned bool) error
	RestrictChatMember(ctx context.Context, chatID int64, userID int64, permissions RestrictPermissions, untilDate *time.Time) error
	GetChatAdministrators(ctx context.Context, chatID int64) ([]ChatAdministrator, error)
	GetChat(ctx context.Context, chatID int64) (Chat, error)
	GetMe(ctx context.Context) (User, error)
	SetWebhook(ctx context.Context, options SetWebhookOptions) error
	DeleteWebhook(ctx context.Context) error
	LeaveChat(ctx context.Context, chatID int64) error
	AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string, showAlert bool) error
}

type Factory interface {
	ForBot(bot domain.BotInstance) Client
}

type HTTPFactory struct {
	baseURL        string
	requestTimeout time.Duration
	maxRetries     int
	initialBackoff time.Duration
	logger         *slog.Logger
}

func NewHTTPFactory(baseURL string, requestTimeout time.Duration, maxRetries int, initialBackoff time.Duration, logger *slog.Logger) *HTTPFactory {
	return &HTTPFactory{
		baseURL:        baseURL,
		requestTimeout: requestTimeout,
		maxRetries:     maxRetries,
		initialBackoff: initialBackoff,
		logger:         logger,
	}
}

func (f *HTTPFactory) ForBot(bot domain.BotInstance) Client {
	return &HTTPClient{
		baseURL:        f.baseURL,
		token:          bot.TelegramToken,
		client:         &http.Client{Timeout: f.requestTimeout},
		logger:         f.logger.With("bot_id", bot.ID, "bot_slug", bot.Slug),
		maxRetries:     f.maxRetries,
		initialBackoff: f.initialBackoff,
	}
}

type HTTPClient struct {
	baseURL        string
	token          string
	client         *http.Client
	logger         *slog.Logger
	maxRetries     int
	initialBackoff time.Duration
}

type apiResponse[T any] struct {
	OK          bool                `json:"ok"`
	Result      T                   `json:"result"`
	Description string              `json:"description"`
	ErrorCode   int                 `json:"error_code"`
	Parameters  *responseParameters `json:"parameters,omitempty"`
}

type responseParameters struct {
	RetryAfter int `json:"retry_after,omitempty"`
}

func (c *HTTPClient) SendMessage(ctx context.Context, chatID int64, text string, options SendMessageOptions) (Message, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	if options.ReplyToMessageID != 0 {
		payload["reply_to_message_id"] = options.ReplyToMessageID
	}
	if options.ParseMode != "" {
		payload["parse_mode"] = options.ParseMode
	}
	if options.DisableWebPagePreview {
		payload["disable_web_page_preview"] = true
	}
	if options.DisableNotification {
		payload["disable_notification"] = true
	}
	if options.ProtectContent {
		payload["protect_content"] = true
	}
	if options.ReplyMarkup != nil {
		payload["reply_markup"] = options.ReplyMarkup
	}
	return request[Message](ctx, c, "sendMessage", payload)
}

func (c *HTTPClient) SendPhoto(ctx context.Context, chatID int64, photo string, options SendPhotoOptions) (Message, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"photo":   photo,
	}
	if options.ReplyToMessageID != 0 {
		payload["reply_to_message_id"] = options.ReplyToMessageID
	}
	if options.Caption != "" {
		payload["caption"] = options.Caption
	}
	if options.ParseMode != "" {
		payload["parse_mode"] = options.ParseMode
	}
	if options.ReplyMarkup != nil {
		payload["reply_markup"] = options.ReplyMarkup
	}
	return request[Message](ctx, c, "sendPhoto", payload)
}

func (c *HTTPClient) EditMessageText(ctx context.Context, chatID int64, messageID int64, text string, options EditMessageTextOptions) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}
	if options.ParseMode != "" {
		payload["parse_mode"] = options.ParseMode
	}
	if options.DisableWebPagePreview {
		payload["disable_web_page_preview"] = true
	}
	if options.ReplyMarkup != nil {
		payload["reply_markup"] = options.ReplyMarkup
	}
	_, err := request[Message](ctx, c, "editMessageText", payload)
	return err
}

func (c *HTTPClient) DeleteMessage(ctx context.Context, chatID int64, messageID int64) error {
	_, err := request[bool](ctx, c, "deleteMessage", map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	})
	return err
}

func (c *HTTPClient) PinChatMessage(ctx context.Context, chatID int64, messageID int64, disableNotification bool) error {
	_, err := request[bool](ctx, c, "pinChatMessage", map[string]any{
		"chat_id":              chatID,
		"message_id":           messageID,
		"disable_notification": disableNotification,
	})
	return err
}

func (c *HTTPClient) UnpinChatMessage(ctx context.Context, chatID int64, messageID *int64) error {
	payload := map[string]any{"chat_id": chatID}
	if messageID != nil {
		payload["message_id"] = *messageID
	}
	_, err := request[bool](ctx, c, "unpinChatMessage", payload)
	return err
}

func (c *HTTPClient) UnpinAllChatMessages(ctx context.Context, chatID int64) error {
	_, err := request[bool](ctx, c, "unpinAllChatMessages", map[string]any{
		"chat_id": chatID,
	})
	return err
}

func (c *HTTPClient) PromoteChatMember(ctx context.Context, chatID int64, userID int64, permissions PromotePermissions) error {
	_, err := request[bool](ctx, c, "promoteChatMember", map[string]any{
		"chat_id":              chatID,
		"user_id":              userID,
		"can_delete_messages":  permissions.CanDeleteMessages,
		"can_restrict_members": permissions.CanRestrictMembers,
		"can_change_info":      permissions.CanChangeInfo,
		"can_pin_messages":     permissions.CanPinMessages,
		"can_promote_members":  permissions.CanPromoteMembers,
	})
	return err
}

func (c *HTTPClient) BanChatMember(ctx context.Context, chatID int64, userID int64, untilDate *time.Time, revokeMessages bool) error {
	payload := map[string]any{
		"chat_id":         chatID,
		"user_id":         userID,
		"revoke_messages": revokeMessages,
	}
	if untilDate != nil {
		payload["until_date"] = untilDate.Unix()
	}
	_, err := request[bool](ctx, c, "banChatMember", payload)
	return err
}

func (c *HTTPClient) UnbanChatMember(ctx context.Context, chatID int64, userID int64, onlyIfBanned bool) error {
	_, err := request[bool](ctx, c, "unbanChatMember", map[string]any{
		"chat_id":        chatID,
		"user_id":        userID,
		"only_if_banned": onlyIfBanned,
	})
	return err
}

func (c *HTTPClient) RestrictChatMember(ctx context.Context, chatID int64, userID int64, permissions RestrictPermissions, untilDate *time.Time) error {
	payload := map[string]any{
		"chat_id":     chatID,
		"user_id":     userID,
		"permissions": permissions,
	}
	if untilDate != nil {
		payload["until_date"] = untilDate.Unix()
	}
	_, err := request[bool](ctx, c, "restrictChatMember", payload)
	return err
}

func (c *HTTPClient) GetChatAdministrators(ctx context.Context, chatID int64) ([]ChatAdministrator, error) {
	return request[[]ChatAdministrator](ctx, c, "getChatAdministrators", map[string]any{
		"chat_id": chatID,
	})
}

func (c *HTTPClient) GetChat(ctx context.Context, chatID int64) (Chat, error) {
	return request[Chat](ctx, c, "getChat", map[string]any{
		"chat_id": chatID,
	})
}

func (c *HTTPClient) GetMe(ctx context.Context) (User, error) {
	return request[User](ctx, c, "getMe", map[string]any{})
}

func (c *HTTPClient) SetWebhook(ctx context.Context, options SetWebhookOptions) error {
	_, err := request[bool](ctx, c, "setWebhook", map[string]any{
		"url":          options.URL,
		"secret_token": options.SecretToken,
	})
	return err
}

func (c *HTTPClient) DeleteWebhook(ctx context.Context) error {
	_, err := request[bool](ctx, c, "deleteWebhook", map[string]any{})
	return err
}

func (c *HTTPClient) LeaveChat(ctx context.Context, chatID int64) error {
	_, err := request[bool](ctx, c, "leaveChat", map[string]any{
		"chat_id": chatID,
	})
	return err
}

func (c *HTTPClient) AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string, showAlert bool) error {
	_, err := request[bool](ctx, c, "answerCallbackQuery", map[string]any{
		"callback_query_id": callbackQueryID,
		"text":              text,
		"show_alert":        showAlert,
	})
	return err
}

func request[T any](ctx context.Context, c *HTTPClient, method string, payload map[string]any) (T, error) {
	var zero T

	body, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}

	var lastErr error
	backoff := c.initialBackoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method), bytes.NewReader(body))
		if err != nil {
			return zero, err
		}
		req.Header.Set("Content-Type", "application/json")

		start := time.Now()
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return zero, err
		}

		respBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return zero, readErr
		}

		c.logger.Debug("telegram request",
			"method", method,
			"attempt", attempt+1,
			"status_code", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		var apiResp apiResponse[T]
		if err := json.Unmarshal(respBytes, &apiResp); err != nil {
			lastErr = fmt.Errorf("telegram decode failed: %w", err)
			if attempt < c.maxRetries {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return zero, lastErr
		}

		if apiResp.OK {
			return apiResp.Result, nil
		}

		if apiResp.Parameters != nil && apiResp.Parameters.RetryAfter > 0 && attempt < c.maxRetries {
			time.Sleep(time.Duration(apiResp.Parameters.RetryAfter) * time.Second)
			continue
		}

		lastErr = fmt.Errorf("telegram api error %d: %s", apiResp.ErrorCode, apiResp.Description)
		if apiResp.ErrorCode >= 500 && attempt < c.maxRetries {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		return zero, lastErr
	}

	return zero, lastErr
}
