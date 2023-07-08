package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/google/uuid"
)

// ////////////////////////////////////////////////////////////////////官方api请求格式
type APIRequest struct {
	Messages  []api_message `json:"messages"`
	Stream    bool          `json:"stream"`
	Model     string        `json:"model"`
	PluginIDs []string      `json:"plugin_ids"`
}

type api_message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// /////////////////////////////////////////////////////////////////////非官方api请求格式
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
		HistoryAndTrainingDisabled: !enable_history, //是否保存对话历史
	}
}

func (c *ChatGPTRequest) AddMessage(role string, content string) {
	c.Messages = append(c.Messages, Chatgpt_message{
		ID:      uuid.New(),
		Author:  chatgpt_author{Role: role},
		Content: chatgpt_content{ContentType: "text", Parts: []string{content}},
	})
}

// /////////////////////////////////////////////////////////////////////伪浏览器客户端
var (
	jar     = tls_client.NewCookieJar()
	options = []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(360),
		tls_client.WithClientProfile(tls_client.Safari_Ipad_15_6),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar), // create cookieJar instance and pass it as argument
		// Disable SSL verification
		tls_client.WithInsecureSkipVerify(),
	}
	client, _ = tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	// http_proxy        = os.Getenv("http_proxy")
	// API_REVERSE_PROXY = os.Getenv("API_REVERSE_PROXY")
)
var http_proxy string

// /////////////////////////////////////////////////////////////////////无服务器函数/////////////////////////////////////////////////////////////////////////////

func Handler(w http.ResponseWriter, r *http.Request) { //对下游请求r的响应w

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	// w.Header().Set("Connection", "keep-alive")
	accessToken := r.Header.Get("Authorization")
	if accessToken != "" {
		customAccessToken := strings.Replace(accessToken, "Bearer ", "", 1)
		// Check if customAccessToken starts with sk-
		if strings.HasPrefix(customAccessToken, "eyJhbGciOiJSUzI1NiI") {
			accessToken = customAccessToken
		}
	}

	var original_request APIRequest

	err := json.NewDecoder(r.Body).Decode(&original_request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Request must be proper JSON",
				"type":    "invalid_request_error",
				"param":   nil,
				"code":    err.Error(),
			},
		}
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// 转换官方API请求为chatGPTAPI请求（backend-api）
	translated_request := ConvertAPIRequest(original_request)

	response, err := POSTconversation(translated_request, accessToken) //向上游发起请求
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorResponse := map[string]interface{}{
			"error": "error sending request",
		}
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	defer response.Body.Close()
	if Handle_request_error(w, response) { //向下游发回应错误
		return
	}

	///////////////////////////////////////////////////////////////////////对上游返回的有效响应进行处理
	var full_response string
	var continue_info *ContinueInfo
	var response_part string

	response_part, continue_info = responseHandler(&w, response, original_request.Stream)

	full_response = response_part
	// println(full_response)
	if continue_info == nil {
		return
		// break
	}

	if !original_request.Stream { //完成非流回复
		response := NewChatCompletion(full_response) //以官方格式回复
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain") //完成流式回复
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}

}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func ConvertAPIRequest(api_request APIRequest) ChatGPTRequest { ///官方api请求转非官方请求
	chatgpt_request := NewChatGPTRequest()
	if strings.HasPrefix(api_request.Model, "gpt-3.5") {
		chatgpt_request.Model = "text-davinci-002-render-sha"
	}
	if strings.HasPrefix(api_request.Model, "gpt-4") {
		chatgpt_request.Model = api_request.Model
		chatgpt_request.ArkoseToken = generate_random_hex(17) + "|r=ap-southeast-1|meta=3|meta_width=300|metabgclr=transparent|metaiconclr=%23555555|guitextcolor=%23000000|pk=35536E1E-65B4-4D96-9D97-6ADB7EFF8147|at=40|sup=1|rid=" + strconv.Itoa(randint(1, 99)) + "|ag=101|cdn_url=https%3A%2F%2Ftcr9i.chat.openai.com%2Fcdn%2Ffc|lurl=https%3A%2F%2Faudio-ap-southeast-1.arkoselabs.com|surl=https%3A%2F%2Ftcr9i.chat.openai.com|smurl=https%3A%2F%2Ftcr9i.chat.openai.com%2Fcdn%2Ffc%2Fassets%2Fstyle-manager"
	}
	if api_request.Model == "gpt-4" {
		chatgpt_request.Model = "gpt-4-mobile"
	}
	if api_request.PluginIDs != nil {
		chatgpt_request.PluginIDs = api_request.PluginIDs
		chatgpt_request.Model = "gpt-4-plugins"
	}

	for _, api_message := range api_request.Messages {
		if api_message.Role == "system" {
			api_message.Role = "critic"
		}
		chatgpt_request.AddMessage(api_message.Role, api_message.Content)
	}
	return chatgpt_request
}
func generate_random_hex(length int) string {
	const charset = "0123456789abcdef"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func randint(min int, max int) int {
	return rand.Intn(max-min) + min
}

func POSTconversation(message ChatGPTRequest, access_token string) (*fhttp.Response, error) { ///发送非官方请求

	if http_proxy != "" {
		client.SetProxy(http_proxy) //调试时使用!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	}
	apiUrl := "https://chat.openai.com/backend-api/conversation"

	// JSONify the body and add it to the request
	body_json, err := json.Marshal(message)
	if err != nil {
		return &fhttp.Response{}, err
	}

	request, err := fhttp.NewRequest(http.MethodPost, apiUrl, bytes.NewBuffer(body_json))
	if err != nil {
		return &fhttp.Response{}, err
	}

	request.Header.Set("Content-Type", "application/json") //"text/event-stream"
	request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36")
	request.Header.Set("Accept", "*/*")
	if access_token != "" {
		request.Header.Set("Authorization", "Bearer "+access_token)
	}
	if err != nil {
		return &fhttp.Response{}, err
	}

	response, err := client.Do(request)
	return response, err
}

func Handle_request_error(w http.ResponseWriter, resp *fhttp.Response) bool { //错误处理
	if resp.StatusCode != 200 {
		// Try read response body as JSON
		var error_response map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&error_response)
		if err != nil {
			// Read response body
			body, _ := io.ReadAll(resp.Body)

			w.WriteHeader(fhttp.StatusInternalServerError)
			_, err := w.Write([]byte(body))
			if err != nil {
				return true
			}
			return true
		}

		w.WriteHeader(resp.StatusCode)
		_, err1 := w.Write([]byte(error_response["detail"].(string)))
		if err1 != nil {
			return true
		}
		return true
	}
	return false
}

type ContinueInfo struct {
	ConversationID string `json:"conversation_id"`
	ParentID       string `json:"parent_id"`
}
type StringStruct struct {
	Text string `json:"text"`
}

func responseHandler(w *http.ResponseWriter, response *fhttp.Response, stream bool) (string, *ContinueInfo) {
	writer := bufio.NewWriterSize((*w), 512)
	max_tokens := false
	var err error
	if stream {
		// Response content type is text/event-stream
		(*w).Header().Set("Content-Type", "text/event-stream")
	} else {
		// Response content type is application/json
		(*w).Header().Set("Content-Type", "application/json")
	}
	var finish_reason string
	var previous_text StringStruct
	var original_response ChatGPTResponse
	var isRole = true
	// 使用 bufio 读取一行数据
	scanner := bufio.NewScanner(response.Body)
	line := ""
	for scanner.Scan() {
		line = scanner.Text()
		line = scanner.Text()
		line = strings.TrimPrefix(line, "data: ")
		// line = strings.TrimRight(line, "\n")

		if !strings.HasPrefix(line, "[DONE]") {

			// Parse the line as JSON

			err = json.Unmarshal([]byte(line), &original_response)
			if err != nil {
				continue
			}
			if original_response.Error != nil {
				(*w).WriteHeader(http.StatusInternalServerError)
				response := map[string]interface{}{
					"error": original_response.Error,
				}
				json.NewEncoder((*w)).Encode(response)
				return "", nil
			}
			if original_response.Message.Author.Role != "assistant" || original_response.Message.Content.Parts == nil {

				continue
			}
			if original_response.Message.Metadata.MessageType != "next" && original_response.Message.Metadata.MessageType != "continue" || original_response.Message.EndTurn != nil {
				continue
			}
			response_string := ConvertToString(&original_response, &previous_text, isRole)
			//previous_text积累每次循环的单词为一整个文本，用于非流式回复
			isRole = false
			if stream {
				_, err = writer.Write([]byte(response_string + "\n\n"))
				if err != nil {
					return "", nil
				}
				// println("\r" + response_string)
				// println("---------------------------------------------")

			}
			writer.Flush()//立即推送
 
			if original_response.Message.Metadata.FinishDetails != nil {
				if original_response.Message.Metadata.FinishDetails.Type == "max_tokens" {
					max_tokens = true
				}
				finish_reason = original_response.Message.Metadata.FinishDetails.Type
			}

		} else {
			if stream {
				final_line := StopChunk(finish_reason)
				writer.Write([]byte("data: " + final_line.String() + "\n\n"))
			}
		}

	}

	if !max_tokens {
		// return previous_text.Text, nil
		return previous_text.Text, &ContinueInfo{
			ConversationID: original_response.ConversationID,
			ParentID:       original_response.Message.ID,
		}
	}
	return previous_text.Text, &ContinueInfo{
		ConversationID: original_response.ConversationID,
		ParentID:       original_response.Message.ID,
	}

}

///////////////////////////////////////////////////////////////////////官方api响应体结构

type ChatCompletionChunk struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choices `json:"choices"`
}

func (chunk *ChatCompletionChunk) String() string {
	resp, _ := json.Marshal(chunk)
	return string(resp)
}

type Choices struct {
	Delta        Delta       `json:"delta"`
	Index        int         `json:"index"`
	FinishReason interface{} `json:"finish_reason"`
}

type Delta struct {
	Content string `json:"content,omitempty"`
	Role    string `json:"role,omitempty"`
}

func NewChatCompletionChunk(text string) ChatCompletionChunk {
	return ChatCompletionChunk{
		ID:      "chatcmpl-QXlha2FBbmROaXhpZUFyZUF3ZXNvbWUK",
		Object:  "chat.completion.chunk",
		Created: 0,
		Model:   "gpt-3.5-turbo-0301",
		Choices: []Choices{
			{
				Index: 0,
				Delta: Delta{
					Content: text,
				},
				FinishReason: nil,
			},
		},
	}
}

func StopChunk(reason string) ChatCompletionChunk {
	return ChatCompletionChunk{
		ID:      "chatcmpl-QXlha2FBbmROaXhpZUFyZUF3ZXNvbWUK",
		Object:  "chat.completion.chunk",
		Created: 0,
		Model:   "gpt-3.5-turbo-0301",
		Choices: []Choices{
			{
				Index:        0,
				FinishReason: reason,
			},
		},
	}
}

type ChatCompletion struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Usage   usage    `json:"usage"`
	Choices []Choice `json:"choices"`
}
type Msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Choice struct {
	Index        int         `json:"index"`
	Message      Msg         `json:"message"`
	FinishReason interface{} `json:"finish_reason"`
}
type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func NewChatCompletion(full_test string) ChatCompletion {
	return ChatCompletion{
		ID:      "chatcmpl-QXlha2FBbmROaXhpZUFyZUF3ZXNvbWUK",
		Object:  "chat.completion",
		Created: int64(0),
		Model:   "gpt-3.5-turbo-0301",
		Usage: usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
		Choices: []Choice{
			{
				Message: Msg{
					Content: full_test,
					Role:    "assistant",
				},
				Index: 0,
			},
		},
	}
}

// /////////////////////////////////////////////////////////////////////非官方api响应体结构
type ChatGPTResponse struct {
	Message        Message     `json:"message"`
	ConversationID string      `json:"conversation_id"`
	Error          interface{} `json:"error"`
}

type Message struct {
	ID         string      `json:"id"`
	Author     Author      `json:"author"`
	CreateTime float64     `json:"create_time"`
	UpdateTime interface{} `json:"update_time"`
	Content    Content     `json:"content"`
	EndTurn    interface{} `json:"end_turn"`
	Weight     float64     `json:"weight"`
	Metadata   Metadata    `json:"metadata"`
	Recipient  string      `json:"recipient"`
}

type Content struct {
	ContentType string   `json:"content_type"`
	Parts       []string `json:"parts"`
}

type Author struct {
	Role     string                 `json:"role"`
	Name     interface{}            `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Metadata struct {
	Timestamp     string         `json:"timestamp_"`
	MessageType   string         `json:"message_type"`
	FinishDetails *FinishDetails `json:"finish_details"`
	ModelSlug     string         `json:"model_slug"`
	Recipient     string         `json:"recipient"`
}

type FinishDetails struct {
	Type string `json:"type"`
	Stop string `json:"stop"`
}

// /////////////////////////////////////////////////////////////////////响应体 非转官
func ConvertToString(chatgpt_response *ChatGPTResponse, previous_text *StringStruct, role bool) string {
	translated_response := NewChatCompletionChunk(strings.ReplaceAll(chatgpt_response.Message.Content.Parts[0], *&previous_text.Text, ""))
	if role {
		translated_response.Choices[0].Delta.Role = chatgpt_response.Message.Author.Role
	}
	previous_text.Text = chatgpt_response.Message.Content.Parts[0]
	return "data: " + translated_response.String() + "\n\n"

}
