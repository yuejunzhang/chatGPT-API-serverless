package chatgpt

import (
	"chatgpt-api-serverless/api/typings"
	chatgpt_types "chatgpt-api-serverless/api/typings/chatgpt"
	official_types "chatgpt-api-serverless/api/typings/official"
	"strings"
)

func ConvertToString(chatgpt_response *chatgpt_types.ChatGPTResponse, previous_text *typings.StringStruct, role bool) string {
	translated_response := official_types.NewChatCompletionChunk(strings.ReplaceAll(chatgpt_response.Message.Content.Parts[0], *&previous_text.Text, ""))
	if role {
		translated_response.Choices[0].Delta.Role = chatgpt_response.Message.Author.Role
	}
	previous_text.Text = chatgpt_response.Message.Content.Parts[0]
	return "data: " + translated_response.String() + "\n\n"

}
