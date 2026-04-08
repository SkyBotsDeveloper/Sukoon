package captcha_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"sukoon/bot-core/internal/domain"
	captchaservice "sukoon/bot-core/internal/service/captcha"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
	"sukoon/bot-core/internal/testsupport"
)

func TestCaptchaControlCommandsPersistSettings(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100401, Type: "supergroup", Title: "Captcha Controls"}

	commands := []string{
		"/welcome on Welcome {first}",
		"/setrules Be nice.",
		"/captchamode text2",
		"/captcharules on",
		"/captchamutetime 12h",
		"/captchakick on",
		"/captchakicktime 10m",
		"/setcaptchatext Tap to verify",
		"/captcha on",
		"/resetcaptchatext",
	}
	for idx, text := range commands {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(30 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("command %q failed: %v", text, err)
		}
	}

	bundle, err := h.Store.LoadRuntimeBundle(context.Background(), h.Bot.ID, chat.ID)
	if err != nil {
		t.Fatalf("load runtime bundle failed: %v", err)
	}
	if !bundle.Captcha.Enabled {
		t.Fatalf("expected captcha enabled, got %+v", bundle.Captcha)
	}
	if bundle.Captcha.Mode != "text2" || !bundle.Captcha.RulesRequired || !bundle.Captcha.KickOnTimeout {
		t.Fatalf("unexpected captcha flags: %+v", bundle.Captcha)
	}
	if bundle.Captcha.AutoUnmuteSeconds != int((12 * time.Hour).Seconds()) {
		t.Fatalf("expected 12h auto unmute, got %+v", bundle.Captcha)
	}
	if bundle.Captcha.TimeoutSeconds != int((10 * time.Minute).Seconds()) {
		t.Fatalf("expected 10m kick time, got %+v", bundle.Captcha)
	}
	if bundle.Captcha.ButtonText != "Click here to prove you're human" {
		t.Fatalf("expected reset button text, got %+v", bundle.Captcha)
	}
}

func TestCaptchaRequiresWelcomeBeforeEnable(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100402, Type: "supergroup", Title: "Captcha Welcome"}

	err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			From:      &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:      chat,
			Text:      "/captcha on",
		},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "welcome") {
		t.Fatalf("expected welcome requirement error, got %v", err)
	}
}

func TestCaptchaButtonRulesFlow(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	chat := telegram.Chat{ID: -100403, Type: "supergroup", Title: "Captcha Rules"}
	h.Client.ChatsByID[chat.ID] = chat

	for idx, text := range []string{
		"/welcome on Welcome {first}",
		"/setrules Respect the chat.",
		"/captcharules on",
		"/captcha on",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(20 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      chat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", text, err)
		}
	}

	joinUpdate := telegram.Update{
		UpdateID: 5,
		Message: &telegram.Message{
			MessageID:      30,
			From:           &telegram.User{ID: 1, FirstName: "Owner"},
			Chat:           chat,
			NewChatMembers: []telegram.User{{ID: 20, FirstName: "NewUser"}},
		},
	}
	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, joinUpdate); err != nil {
		t.Fatalf("join update failed: %v", err)
	}

	challenge, ok := h.Store.PendingCaptchaForUser(h.Bot.ID, chat.ID, 20)
	if !ok {
		t.Fatalf("expected pending captcha challenge")
	}
	if len(h.Client.Restrictions) == 0 || h.Client.Restrictions[0].UserID != 20 || h.Client.Restrictions[0].Permissions.CanSendMessages {
		t.Fatalf("expected initial mute restriction, got %+v", h.Client.Restrictions)
	}
	if len(h.Client.Messages) == 0 {
		t.Fatalf("expected captcha join prompt")
	}
	joinPrompt := h.Client.Messages[len(h.Client.Messages)-1]
	markup := requireMarkup(t, joinPrompt)
	assertButton(t, markup, 0, 0, "Click here to prove you're human", "captcha:button:"+challenge.ID, "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 6,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-captcha-button",
			From: telegram.User{ID: 20, FirstName: "NewUser"},
			Message: &telegram.Message{
				MessageID: challenge.MessageID,
				Chat:      chat,
			},
			Data: "captcha:button:" + challenge.ID,
		},
	}); err != nil {
		t.Fatalf("button callback failed: %v", err)
	}

	rulesPage := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(rulesPage.Text, "Accept the rules before you can speak.") || !strings.Contains(rulesPage.Text, "Respect the chat.") {
		t.Fatalf("expected rules prompt, got %q", rulesPage.Text)
	}
	rulesMarkup := requireEditedMarkup(t, rulesPage)
	assertButton(t, rulesMarkup, 0, 0, "Accept Rules", "captcha:rules:"+challenge.ID, "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 7,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-captcha-rules",
			From: telegram.User{ID: 20, FirstName: "NewUser"},
			Message: &telegram.Message{
				MessageID: challenge.MessageID,
				Chat:      chat,
			},
			Data: "captcha:rules:" + challenge.ID,
		},
	}); err != nil {
		t.Fatalf("rules callback failed: %v", err)
	}

	if last := h.Client.Restrictions[len(h.Client.Restrictions)-1]; !last.Permissions.CanSendMessages || last.UserID != 20 {
		t.Fatalf("expected final unrestrict call, got %+v", last)
	}
	if len(h.Client.DeletedMessages) == 0 || h.Client.DeletedMessages[len(h.Client.DeletedMessages)-1].MessageID != challenge.MessageID {
		t.Fatalf("expected captcha prompt deletion, got %+v", h.Client.DeletedMessages)
	}
}

func TestCaptchaPMModeFlowFromStartDeepLink(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	groupChat := telegram.Chat{ID: -100404, Type: "supergroup", Title: "Captcha PM"}
	privateChat := telegram.Chat{ID: 44, Type: "private"}
	h.Client.ChatsByID[groupChat.ID] = groupChat

	for idx, text := range []string{
		"/welcome on Welcome {first}",
		"/setrules Read the rules.",
		"/captcharules on",
		"/captchamode math",
		"/setcaptchatext Verify in PM",
		"/captcha on",
	} {
		if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
			UpdateID: int64(idx + 1),
			Message: &telegram.Message{
				MessageID: int64(40 + idx),
				From:      &telegram.User{ID: 1, FirstName: "Owner"},
				Chat:      groupChat,
				Text:      text,
			},
		}); err != nil {
			t.Fatalf("setup command %q failed: %v", text, err)
		}
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 7,
		Message: &telegram.Message{
			MessageID:      50,
			From:           &telegram.User{ID: 44, FirstName: "Solver"},
			Chat:           groupChat,
			NewChatMembers: []telegram.User{{ID: 44, FirstName: "Solver"}},
		},
	}); err != nil {
		t.Fatalf("join update failed: %v", err)
	}

	challenge, ok := h.Store.PendingCaptchaForUser(h.Bot.ID, groupChat.ID, 44)
	if !ok {
		t.Fatalf("expected pending captcha challenge")
	}
	joinPrompt := h.Client.Messages[len(h.Client.Messages)-1]
	joinMarkup := requireMarkup(t, joinPrompt)
	assertButton(t, joinMarkup, 0, 0, "Verify in PM", "", serviceutil.BotDeepLink(h.Bot.Username, "captcha_"+challenge.ID))

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 8,
		Message: &telegram.Message{
			MessageID: 60,
			From:      &telegram.User{ID: 44, FirstName: "Solver"},
			Chat:      privateChat,
			Text:      "/start captcha_" + challenge.ID,
		},
	}); err != nil {
		t.Fatalf("pm start captcha failed: %v", err)
	}

	pmRules := h.Client.Messages[len(h.Client.Messages)-1]
	if !strings.Contains(pmRules.Text, "Accept the rules before you can speak.") || !strings.Contains(pmRules.Text, "Read the rules.") {
		t.Fatalf("expected PM rules prompt, got %q", pmRules.Text)
	}
	pmRulesMarkup := requireMarkup(t, pmRules)
	assertButton(t, pmRulesMarkup, 0, 0, "Accept Rules", "captcha:rules:"+challenge.ID, "")

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 9,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-pm-rules",
			From: telegram.User{ID: 44, FirstName: "Solver"},
			Message: &telegram.Message{
				MessageID: pmRules.MessageID,
				Chat:      privateChat,
			},
			Data: "captcha:rules:" + challenge.ID,
		},
	}); err != nil {
		t.Fatalf("pm rules callback failed: %v", err)
	}

	pmChallenge := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(pmChallenge.Text, "Complete the captcha") || !strings.Contains(pmChallenge.Text, "Solve:") {
		t.Fatalf("expected PM challenge prompt, got %q", pmChallenge.Text)
	}
	pmChallengeMarkup := requireEditedMarkup(t, pmChallenge)
	foundAnswer := false
	for _, row := range pmChallengeMarkup.InlineKeyboard {
		for _, button := range row {
			if button.CallbackData == "captcha:answer:"+challenge.ID+":"+challenge.Answer {
				foundAnswer = true
			}
		}
	}
	if !foundAnswer {
		t.Fatalf("expected answer option for challenge %+v, got %+v", challenge, pmChallengeMarkup.InlineKeyboard)
	}

	if err := h.Router.HandleUpdate(context.Background(), h.Bot, h.Client, telegram.Update{
		UpdateID: 10,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-pm-answer",
			From: telegram.User{ID: 44, FirstName: "Solver"},
			Message: &telegram.Message{
				MessageID: pmRules.MessageID,
				Chat:      privateChat,
			},
			Data: "captcha:answer:" + challenge.ID + ":" + challenge.Answer,
		},
	}); err != nil {
		t.Fatalf("pm answer callback failed: %v", err)
	}

	if last := h.Client.Restrictions[len(h.Client.Restrictions)-1]; !last.Permissions.CanSendMessages || last.UserID != 44 || last.ChatID != groupChat.ID {
		t.Fatalf("expected PM solve to unrestrict group member, got %+v", last)
	}
	finalPM := h.Client.EditedMessages[len(h.Client.EditedMessages)-1]
	if !strings.Contains(finalPM.Text, "Verification complete.") {
		t.Fatalf("expected PM success page, got %q", finalPM.Text)
	}
}

func TestCaptchaSweepExpiredKickAndAutoUnmute(t *testing.T) {
	h := testsupport.NewHarness(slog.New(slog.NewTextHandler(io.Discard, nil)))
	service := captchaservice.New(h.Store, testsupport.StaticClientFactory{Client: h.Client}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := h.Store.CreateCaptchaChallenge(context.Background(), domainCaptchaChallenge("kick-1", h.Bot.ID, -100405, 90, "kick")); err != nil {
		t.Fatalf("create kick challenge failed: %v", err)
	}
	if err := h.Store.CreateCaptchaChallenge(context.Background(), domainCaptchaChallenge("mute-1", h.Bot.ID, -100406, 91, "unmute")); err != nil {
		t.Fatalf("create unmute challenge failed: %v", err)
	}

	if err := service.SweepExpired(context.Background()); err != nil {
		t.Fatalf("sweep failed: %v", err)
	}
	if len(h.Client.Bans) == 0 || h.Client.Bans[0].ChatID != -100405 || h.Client.Bans[0].UserID != 90 {
		t.Fatalf("expected captcha sweep kick ban call, got %+v", h.Client.Bans)
	}
	if len(h.Client.Unbans) == 0 || h.Client.Unbans[0].ChatID != -100405 || h.Client.Unbans[0].UserID != 90 {
		t.Fatalf("expected captcha sweep kick cleanup, got %+v", h.Client.Unbans)
	}
	if len(h.Client.Restrictions) == 0 {
		t.Fatalf("expected auto-unmute restriction call")
	}
	foundUnmute := false
	for _, call := range h.Client.Restrictions {
		if call.ChatID == -100406 && call.UserID == 91 && call.Permissions.CanSendMessages {
			foundUnmute = true
		}
	}
	if !foundUnmute {
		t.Fatalf("expected auto-unmute restriction call, got %+v", h.Client.Restrictions)
	}
}

func domainCaptchaChallenge(id string, botID string, chatID int64, userID int64, timeoutAction string) domain.CaptchaChallenge {
	return domain.CaptchaChallenge{
		ID:            id,
		BotID:         botID,
		ChatID:        chatID,
		UserID:        userID,
		Prompt:        "Prompt",
		Answer:        "Answer",
		MessageID:     1,
		ExpiresAt:     time.Now().Add(-time.Minute),
		Status:        "pending",
		Mode:          "math",
		TimeoutAction: timeoutAction,
		FailureAction: "kick",
	}
}

func requireMarkup(t *testing.T, msg testsupport.SentMessage) *telegram.InlineKeyboardMarkup {
	t.Helper()
	if msg.Options.ReplyMarkup == nil {
		t.Fatalf("expected inline keyboard markup, got %+v", msg.Options)
	}
	return msg.Options.ReplyMarkup
}

func requireEditedMarkup(t *testing.T, msg testsupport.EditedMessage) *telegram.InlineKeyboardMarkup {
	t.Helper()
	if msg.Options.ReplyMarkup == nil {
		t.Fatalf("expected inline keyboard markup, got %+v", msg.Options)
	}
	return msg.Options.ReplyMarkup
}

func assertButton(t *testing.T, markup *telegram.InlineKeyboardMarkup, row int, col int, text string, callbackData string, url string) {
	t.Helper()
	if len(markup.InlineKeyboard) <= row || len(markup.InlineKeyboard[row]) <= col {
		t.Fatalf("expected button at row %d col %d, got %+v", row, col, markup.InlineKeyboard)
	}
	button := markup.InlineKeyboard[row][col]
	if button.Text != text || button.CallbackData != callbackData || button.URL != url {
		t.Fatalf("unexpected button at row %d col %d: got %+v", row, col, button)
	}
}
