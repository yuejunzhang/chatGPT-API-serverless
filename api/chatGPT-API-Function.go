package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/google/uuid"
	// "github.com/joho/godotenv"
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
		HistoryAndTrainingDisabled: !enable_history, //!!!!!!!!!!!!!!是否保存对话历史
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
	client, _         = tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	http_proxy        = os.Getenv("http_proxy")
	API_REVERSE_PROXY = os.Getenv("API_REVERSE_PROXY")
)

// /////////////////////////////////////////////////////无服务器函数/////////////////////////////////////////////////////////////////////////////

func Handler(w http.ResponseWriter, r *http.Request) { //对下游的请求r进行响应w
	// httpProxy := os.Getenv("http_proxy")
	// accessToken = os.Getenv("ACCESS_TOKEN")

	// 进行处理////////////////////////////////////////////////////////////////////////
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	accessToken := r.Header.Get("Authorization")
	if accessToken != "" {
		customAccessToken := strings.Replace(accessToken, "Bearer ", "", 1)
		// Check if customAccessToken starts with sk-
		if strings.HasPrefix(customAccessToken, "eyJhbGciOiJSUzI1NiI") {
			accessToken = customAccessToken
		}
		// parts := strings.Fields(accessToken)
		// if len(parts) >= 2 {
		// 	accessToken = parts[1]
		//
	}
	// func nightmare(c *gin.Context) {///////////////////////////////////////////////////////////////////
	var original_request APIRequest

	err := json.NewDecoder(r.Body).Decode(&original_request) //尝试解析下游请求为官方API请求
	if err != nil {                                          //下游API请求不符合官方格式
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

	// Convert the chat request to a ChatGPT request转换官方API请求为chatGPTAPI请求（backend-api）
	// original_request.Stream = false 这并不有效
	translated_request := ConvertAPIRequest(original_request)

	response, err := POSTconversation(translated_request, accessToken) //向上游发起请求
	if err != nil {                                                    //向上游发起请求出错
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
	//对上游返回的有效响应进行处理 w-----------------------------------------------------------------
	print(response.Body) ////////////////////////////////////////////////////////////////////////////////////////////////
	// var full_response string
	// // for i := 2; i > 0; i-- {
	// var continue_info *chatgpt.ContinueInfo
	// var response_part string
	// response_part, continue_info = chatgpt.Handler(&w, response, accessToken, translated_request, original_request.Stream) /////////////////////////////////////////////////////////
	// full_response = response_part                                                                                          /////////////////////////////////////////////////////////
	// println(full_response)
	// if continue_info == nil {
	// 	// break
	// 	// os.Setenv("ConversationID", "")
	// }

	// println("continue_info.ConversationID:" + continue_info.ConversationID)
	// println("continue_info.ParentID:" + continue_info.ParentID)
	// println("Continuing conversation") //连续的会话
	// translated_request = chatgpt_request_converter.ConvertAPIRequest(original_request)
	// // translated_request.Action = "continue"
	// translated_request.Action = "next"//next--进行下一句，continue--在同一句中继续
	// translated_request.ConversationID = continue_info.ConversationID //ConversationID 会话ID--用于同一会话标识
	// translated_request.ParentMessageID = continue_info.ParentID      //上一条消息的ID--形成ID链用于上下文关联
	// response, err = chatgpt.POSTconversation(translated_request, accessToken)
	// if err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	errorResponse := map[string]interface{}{
	// 		"error": "error sending request",
	// 	}
	// 	json.NewEncoder(w).Encode(errorResponse)
	// 	return
	// }
	// defer response.Body.Close()
	// if chatgpt.Handle_request_error(w, response) { //向下游发回应错误
	// 	return
	// }
	// }

	if !original_request.Stream {
		// response := official_types.NewChatCompletion(full_response) //完成非流回复
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain") //完成流式回复
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}

	// }

} //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

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
	// if http_proxy != "" && len(proxies) == 0 {
	// 	client.SetProxy(http_proxy)
	// }

	// client.SetProxy("http://127.0.0.1:7890") //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!11
	apiUrl := "https://chat.openai.com/backend-api/conversation"
	// if API_REVERSE_PROXY != "" {
	// 	apiUrl = API_REVERSE_PROXY
	// }

	// JSONify the body and add it to the request
	body_json, err := json.Marshal(message)
	if err != nil {
		return &fhttp.Response{}, err
	}

	request, err := fhttp.NewRequest(http.MethodPost, apiUrl, bytes.NewBuffer(body_json))
	if err != nil {
		return &fhttp.Response{}, err
	}
	// // Clear cookies
	// if os.Getenv("PUID") != "" {
	// 	request.Header.Set("Cookie", "_puid="+os.Getenv("PUID")+";")
	// }
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36")
	request.Header.Set("Accept", "*/*")
	if access_token != "" {
		request.Header.Set("Authorization", "Bearer "+access_token)
	}
	if err != nil {
		return &fhttp.Response{}, err
	}
	// fmt.Printf("%+v\n", request)

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
			// http.Error(w, string(body), http.StatusInternalServerError)
			w.WriteHeader(fhttp.StatusInternalServerError) ////////////////////////////////////////////////////
			_, err := w.Write([]byte(body))
			if err != nil {
				return true
			}
			return true
		}
		// http.Error(w, error_response["detail"].(string), resp.StatusCode)
		w.WriteHeader(resp.StatusCode)
		_, err1 := w.Write([]byte(error_response["detail"].(string))) //////////////////////////////////
		if err1 != nil {
			return true
		}
		return true
	}
	return false
}
