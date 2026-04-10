package telegram

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	EditedMessage *Message       `json:"edited_message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type Message struct {
	MessageID                     int64                          `json:"message_id"`
	From                          *User                          `json:"from,omitempty"`
	ViaBot                        *User                          `json:"via_bot,omitempty"`
	SenderChat                    *Chat                          `json:"sender_chat,omitempty"`
	ForwardFromChat               *Chat                          `json:"forward_from_chat,omitempty"`
	Chat                          Chat                           `json:"chat"`
	Date                          int64                          `json:"date"`
	Text                          string                         `json:"text,omitempty"`
	Caption                       string                         `json:"caption,omitempty"`
	ReplyToMessage                *Message                       `json:"reply_to_message,omitempty"`
	Entities                      []Entity                       `json:"entities,omitempty"`
	CaptionEntities               []Entity                       `json:"caption_entities,omitempty"`
	NewChatMembers                []User                         `json:"new_chat_members,omitempty"`
	LeftChatMember                *User                          `json:"left_chat_member,omitempty"`
	PinnedMessage                 *Message                       `json:"pinned_message,omitempty"`
	NewChatTitle                  string                         `json:"new_chat_title,omitempty"`
	NewChatPhoto                  []PhotoSize                    `json:"new_chat_photo,omitempty"`
	DeleteChatPhoto               bool                           `json:"delete_chat_photo,omitempty"`
	GroupChatCreated              bool                           `json:"group_chat_created,omitempty"`
	SupergroupChatCreated         bool                           `json:"supergroup_chat_created,omitempty"`
	ChannelChatCreated            bool                           `json:"channel_chat_created,omitempty"`
	MessageAutoDeleteTimerChanged *MessageAutoDeleteTimerChanged `json:"message_auto_delete_timer_changed,omitempty"`
	VideoChatStarted              *VideoChatStarted              `json:"video_chat_started,omitempty"`
	VideoChatEnded                *VideoChatEnded                `json:"video_chat_ended,omitempty"`
	VideoChatParticipantsInvited  *VideoChatParticipantsInvited  `json:"video_chat_participants_invited,omitempty"`
	VideoChatScheduled            *VideoChatScheduled            `json:"video_chat_scheduled,omitempty"`
	ForwardOrigin                 any                            `json:"forward_origin,omitempty"`
	ExternalReply                 *ExternalReplyInfo             `json:"external_reply,omitempty"`
	IsAutomaticForward            bool                           `json:"is_automatic_forward,omitempty"`
	MediaGroupID                  string                         `json:"media_group_id,omitempty"`
	Photo                         []PhotoSize                    `json:"photo,omitempty"`
	Sticker                       *Sticker                       `json:"sticker,omitempty"`
	Animation                     *Animation                     `json:"animation,omitempty"`
	Audio                         *Audio                         `json:"audio,omitempty"`
	Voice                         *Voice                         `json:"voice,omitempty"`
	VideoNote                     *VideoNote                     `json:"video_note,omitempty"`
	Video                         *Video                         `json:"video,omitempty"`
	Document                      *Document                      `json:"document,omitempty"`
	Contact                       *Contact                       `json:"contact,omitempty"`
	Location                      *Location                      `json:"location,omitempty"`
	Poll                          *Poll                          `json:"poll,omitempty"`
	Dice                          *Dice                          `json:"dice,omitempty"`
	Game                          *Game                          `json:"game,omitempty"`
	ReplyMarkup                   *InlineKeyboardMarkup          `json:"reply_markup,omitempty"`
}

type Entity struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	URL           string `json:"url,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
	Bio      string `json:"bio,omitempty"`
}

type PhotoSize struct {
	FileID string `json:"file_id"`
}

type Sticker struct {
	FileID     string `json:"file_id"`
	SetName    string `json:"set_name,omitempty"`
	IsAnimated bool   `json:"is_animated,omitempty"`
	IsVideo    bool   `json:"is_video,omitempty"`
	IsPremium  bool   `json:"is_premium,omitempty"`
}

type Animation struct {
	FileID string `json:"file_id"`
}

type Audio struct {
	FileID string `json:"file_id"`
}

type Voice struct {
	FileID string `json:"file_id"`
}

type VideoNote struct {
	FileID string `json:"file_id"`
}

type Video struct {
	FileID string `json:"file_id"`
}

type Document struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name,omitempty"`
}

type Contact struct {
	PhoneNumber string `json:"phone_number,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

type Poll struct {
	ID string `json:"id,omitempty"`
}

type Dice struct {
	Emoji string `json:"emoji,omitempty"`
}

type Game struct{}

type ExternalReplyInfo struct {
	Chat *Chat `json:"chat,omitempty"`
}

type MessageAutoDeleteTimerChanged struct {
	MessageAutoDeleteTime int `json:"message_auto_delete_time"`
}

type VideoChatStarted struct{}

type VideoChatEnded struct {
	Duration int `json:"duration,omitempty"`
}

type VideoChatParticipantsInvited struct {
	Users []User `json:"users,omitempty"`
}

type VideoChatScheduled struct {
	StartDate int64 `json:"start_date,omitempty"`
}

type ChatAdministrator struct {
	User               User   `json:"user"`
	Status             string `json:"status"`
	IsAnonymous        bool   `json:"is_anonymous,omitempty"`
	CanDeleteMessages  bool   `json:"can_delete_messages,omitempty"`
	CanRestrictMembers bool   `json:"can_restrict_members,omitempty"`
	CanChangeInfo      bool   `json:"can_change_info,omitempty"`
	CanPinMessages     bool   `json:"can_pin_messages,omitempty"`
	CanPromoteMembers  bool   `json:"can_promote_members,omitempty"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type PromotePermissions struct {
	CanDeleteMessages  bool `json:"can_delete_messages,omitempty"`
	CanRestrictMembers bool `json:"can_restrict_members,omitempty"`
	CanChangeInfo      bool `json:"can_change_info,omitempty"`
	CanPinMessages     bool `json:"can_pin_messages,omitempty"`
	CanPromoteMembers  bool `json:"can_promote_members,omitempty"`
}

type MessageEntityMention struct {
	UserID   int64
	Username string
}

type SendMessageOptions struct {
	ReplyToMessageID      int64
	ParseMode             string
	DisableWebPagePreview bool
	ReplyMarkup           *InlineKeyboardMarkup
}

type SendPhotoOptions struct {
	ReplyToMessageID int64
	Caption          string
	ParseMode        string
	ReplyMarkup      *InlineKeyboardMarkup
}

type EditMessageTextOptions struct {
	ParseMode             string
	DisableWebPagePreview bool
	ReplyMarkup           *InlineKeyboardMarkup
}

type SetWebhookOptions struct {
	URL         string
	SecretToken string
}

type RestrictPermissions struct {
	CanSendMessages bool `json:"can_send_messages"`
}
