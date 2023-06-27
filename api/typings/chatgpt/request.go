package chatgpt

import (
	"os"

	"github.com/google/uuid"
)

type Chatgpt_message struct {
	ID      uuid.UUID       `json:"id"`
	Author  chatgpt_author  `json:"author"`
	Content chatgpt_content `json:"content"`
}

type chatgpt_content struct {
	ContentType string   `json:"content_type"`
	Parts       []string `json:"parts"`
}

type chatgpt_author struct {
	Role string `json:"role"`
}

type ChatGPTRequest struct {
	Action                     string            `json:"action"`
	Messages                   []Chatgpt_message `json:"messages"`
	Model                      string            `json:"model"`
	ParentMessageID            string            `json:"parent_message_id,omitempty"`
	ConversationID             string            `json:"conversation_id,omitempty"`
	PluginIDs                  []string          `json:"plugin_ids,omitempty"`
	TimezoneOffsetMin          int               `json:"timezone_offset_min"`
	ArkoseToken                string            `json:"arkose_token,omitempty"`
	HistoryAndTrainingDisabled bool              `json:"history_and_training_disabled"`
	AutoContinue               bool              `json:"auto_continue"`
}

func NewChatGPTRequest() ChatGPTRequest {
	enable_history := os.Getenv("ENABLE_HISTORY") == ""
	return ChatGPTRequest{
		Action:                     "next",
		ParentMessageID:            uuid.NewString(),
		Model:                      "text-davinci-002-render-sha",
		HistoryAndTrainingDisabled: !enable_history, ///////////////////////////!!!!!!!!!!!!!!是否保存对话历史
	}
}

func (c *ChatGPTRequest) AddMessage(role string, content string) {
	c.Messages = append(c.Messages, Chatgpt_message{
		ID:      uuid.New(),
		Author:  chatgpt_author{Role: role},
		Content: chatgpt_content{ContentType: "text", Parts: []string{content}},
	})
}
