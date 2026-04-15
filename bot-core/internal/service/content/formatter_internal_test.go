package content

import (
	"strings"
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

func TestRenderStoredPayloadSupportsRoseMarkdown(t *testing.T) {
	result, err := renderStoredPayload(
		"Hello *bold* _italic_ __under__ ~gone~ ||secret|| `code`\n> quote\n[site](misssukoon.vercel.app)",
		"",
		telegram.User{ID: 1, FirstName: "Test"},
		telegram.Chat{ID: -1001, Title: "Chat"},
		"",
		"sukoonbot",
		-1001,
	)
	if err != nil {
		t.Fatalf("renderStoredPayload() error = %v", err)
	}
	for _, fragment := range []string{
		"<b>bold</b>",
		"<i>italic</i>",
		"<u>under</u>",
		"<s>gone</s>",
		`<span class="tg-spoiler">secret</span>`,
		"<code>code</code>",
		"<blockquote>quote</blockquote>",
		`<a href="https://misssukoon.vercel.app">site</a>`,
	} {
		if !strings.Contains(result.Text, fragment) {
			t.Fatalf("expected rendered markdown to contain %q, got %q", fragment, result.Text)
		}
	}
	if result.ParseMode != "HTML" {
		t.Fatalf("expected HTML parse mode, got %q", result.ParseMode)
	}
}

func TestParseStoredContentSupportsRoseButtonSyntax(t *testing.T) {
	text, buttonsJSON, err := parseStoredContent(strings.Join([]string{
		"Choose:",
		"[Google](buttonurl://google.com)",
		"[Bing](buttonurl://bing.com:same)",
		"[Note](buttonurl://#my_note)",
		"[Styled](buttonurl#primary://misssukoon.vercel.app)",
	}, "\n"))
	if err != nil {
		t.Fatalf("parseStoredContent() error = %v", err)
	}
	if text != "Choose:" {
		t.Fatalf("unexpected text: %q", text)
	}
	markup, err := buttonsFromJSONWithContext(buttonsJSON, "sukoonbot", -1001)
	if err != nil {
		t.Fatalf("buttonsFromJSONWithContext() error = %v", err)
	}
	if len(markup.InlineKeyboard) != 3 || len(markup.InlineKeyboard[0]) != 2 {
		t.Fatalf("expected :same to append to previous row, got %+v", markup.InlineKeyboard)
	}
	if markup.InlineKeyboard[0][0].URL != "https://google.com" || markup.InlineKeyboard[0][1].URL != "https://bing.com" {
		t.Fatalf("unexpected URL buttons: %+v", markup.InlineKeyboard[0])
	}
	if !strings.Contains(markup.InlineKeyboard[1][0].URL, "start=note_") || !strings.Contains(markup.InlineKeyboard[1][0].URL, "my_note") {
		t.Fatalf("expected note button deep link, got %+v", markup.InlineKeyboard[1][0])
	}
	if markup.InlineKeyboard[2][0].URL != "https://misssukoon.vercel.app" {
		t.Fatalf("expected styled button to keep URL, got %+v", markup.InlineKeyboard[2][0])
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
