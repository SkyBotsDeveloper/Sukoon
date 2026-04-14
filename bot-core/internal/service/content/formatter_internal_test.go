package content

import (
	"testing"

	"sukoon/bot-core/internal/telegram"
)

func TestRenderStoredPayloadAddsRulesButtonAndPreviewFlags(t *testing.T) {
	result, err := renderStoredPayload(
		"Press below {rules:same} https://example.com {preview:top}",
		`[[{"text":"Docs","url":"https://misssukoon.vercel.app/"}]]`,
		telegram.User{ID: 1, FirstName: "Test"},
		telegram.Chat{ID: -1001, Title: "Chat"},
		"Be nice.",
		"sukoonbot",
		-1001,
	)
	if err != nil {
		t.Fatalf("renderStoredPayload() error = %v", err)
	}
	if result.DisableWebPagePreview || !result.EnableWebPagePreview || !result.ShowPreviewAboveText {
		t.Fatalf("expected preview to be enabled above text, got %+v", result)
	}
	if result.ReplyMarkup == nil || len(result.ReplyMarkup.InlineKeyboard) != 1 || len(result.ReplyMarkup.InlineKeyboard[0]) != 2 {
		t.Fatalf("expected rules button on same row, got %+v", result.ReplyMarkup)
	}
	if result.ReplyMarkup.InlineKeyboard[0][1].Text != "Rules" {
		t.Fatalf("expected rules button, got %+v", result.ReplyMarkup.InlineKeyboard[0][1])
	}
}

func TestParseFilterReplyPayloadSupportsMediaCaptions(t *testing.T) {
	text, buttonsJSON, mediaType, mediaFileID, err := parseFilterReplyPayload(&telegram.Message{
		MessageID: 55,
		Chat:      telegram.Chat{ID: -1001, Title: "Chat"},
		Caption:   "Spoiler drop {mediaspoiler}",
		Photo: []telegram.PhotoSize{
			{FileID: "thumb"},
			{FileID: "full"},
		},
	})
	if err != nil {
		t.Fatalf("parseFilterReplyPayload() error = %v", err)
	}
	if text != "Spoiler drop {mediaspoiler}" || buttonsJSON != "[]" {
		t.Fatalf("unexpected text/buttons: %q %q", text, buttonsJSON)
	}
	if mediaType != "photo" || mediaFileID != "full" {
		t.Fatalf("unexpected media payload: %s %s", mediaType, mediaFileID)
	}
}
