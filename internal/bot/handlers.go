package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api          *tgbotapi.BotAPI
	adminID      int64
	groupID      int64
	messages     map[string]string              // Мапа для зберігання текстів повідомлень
	albumBuffer  map[string][]*tgbotapi.Message // Буфер для зберігання частин альбому
	messagesLock sync.Mutex                     // М'ютекс для синхронізації доступу до мапи
}

func NewBot(token string, adminID int64, groupID int64) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:         api,
		adminID:     adminID,
		groupID:     groupID,
		messages:    make(map[string]string),
		albumBuffer: make(map[string][]*tgbotapi.Message),
	}, nil // Додаємо nil для помилки
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("Received message from %s: %s", update.Message.From.UserName, update.Message.Text)
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			log.Printf("Received callback query: %s", update.CallbackQuery.Data)
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
    if message.Chat.IsPrivate() {
        log.Printf("Message is private. Processing message from user %s", message.From.UserName)

        // Ігноруємо команду /start
        if message.Text == "/start" {
            log.Printf("Received /start command from user %s. Ignoring.", message.From.UserName)
            return
        }

        if message.MediaGroupID != "" {
            // Логування для альбомів
            log.Printf("Received message with MediaGroupID: %s, PhotoID: %s, Caption: %s",
                message.MediaGroupID,
                message.Photo[len(message.Photo)-1].FileID,
                message.Caption)

            // Зберігаємо частини альбому в буфер
            b.messagesLock.Lock()
            b.albumBuffer[message.MediaGroupID] = append(b.albumBuffer[message.MediaGroupID], message)

            // Якщо це перше повідомлення альбому, запускаємо таймер
            if len(b.albumBuffer[message.MediaGroupID]) == 1 {
                go func(mediaGroupID string) {
                    time.Sleep(3 * time.Second) // Очікуємо 3 секунди для отримання всіх частин альбому
                    b.messagesLock.Lock()
                    defer b.messagesLock.Unlock()

                    if albumMessages, ok := b.albumBuffer[mediaGroupID]; ok {
                        log.Printf("Sending album with MediaGroupID: %s to admin. Total parts: %d", mediaGroupID, len(albumMessages))
                        b.sendAlbumToAdmin(albumMessages)
                    } else {
                        log.Printf("Album with MediaGroupID: %s not found or incomplete", mediaGroupID)
                    }
                }(message.MediaGroupID)
            }
            b.messagesLock.Unlock()
        } else if len(message.Photo) > 0 {
            // Логування для окремих фото
            log.Printf("Received single photo message. PhotoID: %s, Caption: %s",
                message.Photo[len(message.Photo)-1].FileID,
                message.Caption)

            // Якщо це одне фото
            messageID := fmt.Sprintf("%d_%d", message.Chat.ID, message.MessageID)
            photo := message.Photo[len(message.Photo)-1]
            b.messagesLock.Lock()
            b.messages[messageID] = fmt.Sprintf("photo:%s|%s", photo.FileID, message.Caption)
            b.messagesLock.Unlock()
            b.sendToAdminForModeration(message)
        } else if message.Text != "" {
            // Логування для текстових повідомлень
            log.Printf("Received text message: %s", message.Text)

            // Якщо це текстове повідомлення
            messageID := fmt.Sprintf("%d_%d", message.Chat.ID, message.MessageID)
            b.messagesLock.Lock()
            b.messages[messageID] = message.Text
            b.messagesLock.Unlock()
            b.sendToAdminForModeration(message)
        } else {
            log.Printf("Unsupported message type received.")
        }
    }
}

func (b *Bot) sendToAdminForModeration(message *tgbotapi.Message) {
    messageID := fmt.Sprintf("%d_%d", message.Chat.ID, message.MessageID)

    b.messagesLock.Lock()
    messageData, exists := b.messages[messageID]
    b.messagesLock.Unlock()

    if !exists {
        log.Printf("Message ID not found: %s", messageID)
        return
    }

    moderationMsg := fmt.Sprintf("User %s (@%s) wants to publish:",
        escapeMarkdown(message.From.FirstName), escapeMarkdown(message.From.UserName))

    log.Printf("Sending moderation request for user ID: %d", message.Chat.ID)

    if strings.HasPrefix(messageData, "photo:") {
        photoParts := strings.SplitN(strings.TrimPrefix(messageData, "photo:"), "|", 2)
        photoID := photoParts[0]
        caption := ""
        if len(photoParts) > 1 {
            caption = escapeMarkdown(photoParts[1])
        }

        log.Printf("Photo Caption: %s", caption)

        media := tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(photoID))
        media.Caption = escapeMarkdown(moderationMsg) + "\n\n" + caption
        media.ParseMode = "MarkdownV2"

        mediaGroup := []interface{}{media}

        if _, err := b.api.SendMediaGroup(tgbotapi.NewMediaGroup(b.adminID, mediaGroup)); err != nil {
            log.Printf("Failed to send photo for moderation: %v", err)
            return
        }

        // Надсилаємо кнопки модерації окремим повідомленням
        inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("✅ Approve", fmt.Sprintf("approve:%s:%d:%s", messageID, message.Chat.ID, escapeMarkdown(message.From.FirstName))),
                tgbotapi.NewInlineKeyboardButtonData("❌ Reject", fmt.Sprintf("reject:%s:%d", messageID, message.Chat.ID)),
            ),
        )

        msg := tgbotapi.NewMessage(b.adminID, "Please review the photo above and choose an action:")
        msg.ReplyMarkup = inlineKeyboard
        if _, err := b.api.Send(msg); err != nil {
            log.Printf("Failed to send moderation buttons: %v", err)
        }
    } else {
        log.Printf("Sending text message to admin. User ID: %d", message.Chat.ID)

        inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("✅ Approve", fmt.Sprintf("approve:%s:%d:%s", messageID, message.Chat.ID, escapeMarkdown(message.From.FirstName))),
                tgbotapi.NewInlineKeyboardButtonData("❌ Reject", fmt.Sprintf("reject:%s:%d", messageID, message.Chat.ID)),
            ),
        )

        msgText := escapeMarkdown(moderationMsg) + "\n\n" + escapeMarkdown(messageData)
        log.Printf("Text Message: %s", msgText)

        msg := tgbotapi.NewMessage(b.adminID, msgText)
        msg.ReplyMarkup = inlineKeyboard
        msg.ParseMode = "MarkdownV2"

        if _, err := b.api.Send(msg); err != nil {
            log.Printf("Failed to send moderation message: %v", err)
        }
    }
}

func (b *Bot) sendAlbumToAdmin(messages []*tgbotapi.Message) {
    moderationMsg := fmt.Sprintf("User %s (@%s) wants to publish:",
        escapeMarkdown(messages[0].From.FirstName), escapeMarkdown(messages[0].From.UserName))

    var mediaGroup []interface{}
    for i, msg := range messages {
        photo := msg.Photo[len(msg.Photo)-1]
        caption := ""
        if i == 0 && msg.Caption != "" {
            caption = escapeMarkdown(moderationMsg) + "\n\n" + escapeMarkdown(msg.Caption)
        }

        media := tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(photo.FileID))
        if i == 0 && caption != "" {
            media.Caption = caption
            media.ParseMode = "MarkdownV2"
        }
        mediaGroup = append(mediaGroup, media)
    }

    // Надсилаємо альбом адміністратору
    if _, err := b.api.SendMediaGroup(tgbotapi.NewMediaGroup(b.adminID, mediaGroup)); err != nil {
        log.Printf("Failed to send media group to admin: %v", err)
        return
    }

    // Надсилаємо кнопки модерації окремим повідомленням
    mediaGroupID := messages[0].MediaGroupID
    inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("✅ Approve", fmt.Sprintf("approve:%s:%d:%s", mediaGroupID, messages[0].Chat.ID, escapeMarkdown(messages[0].From.FirstName))),
            tgbotapi.NewInlineKeyboardButtonData("❌ Reject", fmt.Sprintf("reject:%s:%d", mediaGroupID, messages[0].Chat.ID)),
        ),
    )

    msg := tgbotapi.NewMessage(b.adminID, "Please review the album above and choose an action:")
    msg.ReplyMarkup = inlineKeyboard
    if _, err := b.api.Send(msg); err != nil {
        log.Printf("Failed to send moderation buttons: %v", err)
    }
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
    data := callback.Data

    parts := strings.SplitN(data, ":", 4)
    if len(parts) < 3 {
        log.Printf("Failed to parse callback data: %s", data)
        return
    }

    action := parts[0]
    messageID := parts[1]
    userID, err := strconv.ParseInt(parts[2], 10, 64)
    if err != nil {
        log.Printf("Failed to parse user ID: %v", err)
        return
    }

    authorName := ""
    if len(parts) == 4 {
        authorName = parts[3]
    }

    log.Printf("Callback action: %s, Message ID: %s, User ID: %d, Author Name: %s", action, messageID, userID, authorName)

    if action == "approve" {
        log.Printf("Approving message with ID: %s for user ID: %d", messageID, userID)
        b.ApproveMessage(messageID, userID, authorName)
        b.updateModeratorMessage(callback, "✅ Post has been approved.")
    } else if action == "reject" {
        log.Printf("Rejecting message with ID: %s for user ID: %d", messageID, userID)
        b.RejectMessage(userID)
        b.updateModeratorMessage(callback, "❌ Post has been rejected.")
    }

    callbackResponse := tgbotapi.NewCallback(callback.ID, "Action processed")
    if _, err := b.api.Request(callbackResponse); err != nil {
        log.Printf("Failed to send callback response: %v", err)
    }
}

func (b *Bot) ApproveMessage(messageID string, userID int64, authorName string) {
    b.messagesLock.Lock()
    albumMessages, albumExists := b.albumBuffer[messageID]
    singleMessage, singleExists := b.messages[messageID]
    b.messagesLock.Unlock()

    if albumExists && len(albumMessages) > 0 {
        log.Printf("Publishing album with MediaGroupID: %s to group", messageID)

        caption := fmt.Sprintf("✅ *New post approved by admin:*\n\n*Author:* [%s](tg://user?id=%d)",
            escapeMarkdown(authorName), userID)

        var mediaGroup []interface{}
        for i, msg := range albumMessages {
            photo := msg.Photo[len(msg.Photo)-1]
            media := tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(photo.FileID))
            if i == 0 && msg.Caption != "" {
                media.Caption = caption + "\n\n" + escapeMarkdown(msg.Caption)
                media.ParseMode = "MarkdownV2"
            }
            mediaGroup = append(mediaGroup, media)
        }

        if _, err := b.api.SendMediaGroup(tgbotapi.NewMediaGroup(b.groupID, mediaGroup)); err != nil {
            log.Printf("Failed to send media group to group: %v", err)
            return
        }
    } else if singleExists {
        log.Printf("Publishing single message with ID: %s to group", messageID)

        if strings.HasPrefix(singleMessage, "photo:") {
            photoParts := strings.SplitN(strings.TrimPrefix(singleMessage, "photo:"), "|", 2)
            photoID := photoParts[0]
            caption := fmt.Sprintf("✅ *New post approved by admin:*\n\n*Author:* [%s](tg://user?id=%d)",
                escapeMarkdown(authorName), userID)
            if len(photoParts) > 1 && photoParts[1] != "" {
                caption += "\n\n" + escapeMarkdown(photoParts[1])
            }

            log.Printf("Photo Caption: %s", caption)

            photoMsg := tgbotapi.NewPhoto(b.groupID, tgbotapi.FileID(photoID))
            photoMsg.Caption = caption
            photoMsg.ParseMode = "MarkdownV2"

            if _, err := b.api.Send(photoMsg); err != nil {
                log.Printf("Failed to send photo to group: %v", err)
                return
            }
        } else {
            caption := fmt.Sprintf("✅ *New post approved by admin:*\n\n*Author:* [%s](tg://user?id=%d)\n\n%s",
                escapeMarkdown(authorName), userID, escapeMarkdown(singleMessage))

            log.Printf("Text Message: %s", caption)

            msg := tgbotapi.NewMessage(b.groupID, caption)
            msg.ParseMode = "MarkdownV2"

            if _, err := b.api.Send(msg); err != nil {
                log.Printf("Failed to send text message to group: %v", err)
                return
            }
        }
    } else {
        log.Printf("Message ID not found: %s", messageID)
        return
    }

    // Повідомляємо користувача про схвалення
    userMessage := "🎉 Your post has been approved and published in the group!"
    b.notifyUser(userID, userMessage)
}

func (b *Bot) RejectMessage(chatID int64) {
	// Надсилаємо повідомлення користувачу про відхилення
	userMessage := "❌ Your post has been rejected by the admin."
	b.notifyUser(chatID, userMessage)
}

func (b *Bot) notifyUser(chatID int64, message string) {
    log.Printf("Notifying user ID: %d with message: %s", chatID, message)
    msg := tgbotapi.NewMessage(chatID, message)
    if _, err := b.api.Send(msg); err != nil {
        log.Printf("Failed to send notification to user: %v", err)
    }
}

func (b *Bot) updateModeratorMessage(callback *tgbotapi.CallbackQuery, statusMessage string) {
    if callback.Message.Text != "" {
        // Оновлюємо текст і видаляємо клавіатуру в одному запиті
        originalText := callback.Message.Text
        updatedText := fmt.Sprintf("%s\n\n%s", originalText, statusMessage)

        emptyKeyboard := tgbotapi.InlineKeyboardMarkup{
            InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
        }

        editMsg := tgbotapi.NewEditMessageTextAndMarkup(
            callback.Message.Chat.ID,
            callback.Message.MessageID,
            updatedText,
            emptyKeyboard,
        )

        if _, err := b.api.Send(editMsg); err != nil {
            log.Printf("Failed to update moderator message text and remove keyboard: %v", err)
        }
    } else {
        log.Printf("Message text is empty, skipping update.")
    }
}

func escapeMarkdown(text string) string {
    replacer := strings.NewReplacer(
        "\\", "\\\\", // Екранування символа `\`
        "_", "\\_",
        "*", "\\*",
        "[", "\\[",
        "]", "\\]",
        "(", "\\(",
        ")", "\\)",
        "~", "\\~",
        "`", "\\`",
        ">", "\\>",
        "#", "\\#",
        "+", "\\+",
        "-", "\\-",
        "=", "\\=",
        "|", "\\|",
        "{", "\\{",
        "}", "\\}",
        ".", "\\.",
        "!", "\\!",
    )
    return replacer.Replace(text)
}
