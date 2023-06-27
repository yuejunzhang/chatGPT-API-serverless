package main

import (
	// "bytes"
	"encoding/json"
	"fmt"
	"net/http"

	// "net/http/httptest"
	"strings"

	"os"
	// "sync"

	// tls_client "github.com/bogdanfinn/tls-client"
	// "github.com/gin-gonic/gin"

	official_types "chatgpt-api-serverless/api/typings/official"

	chatgpt_request_converter "chatgpt-api-serverless/api/conversion/requests/chatgpt"
	chatgpt "chatgpt-api-serverless/api/internal/chatgpt"
	// "github.com/joho/godotenv"
)

var ( // 全局环境变量量
	initialized bool
	// mutex          sync.Mutex
	openaiEmail    string
	openaiPassword string
	httpProxy      string
	accessToken    string
)

// func Init() {
// 	err := godotenv.Load("../.env")
// 	if err != nil {
// 		return
// 	}
// 	// 读取环境变量并保存为全局变量
// openaiEmail = os.Getenv("OPENAI_EMAIL")
// openaiPassword = os.Getenv("OPENAI_PASSWORD")
// httpProxy = os.Getenv("http_proxy")
// accessToken = os.Getenv("ACCESS_TOKEN")

// 	// 标记为已初始化
// 	initialized = true
// 	// if accessToken != "" {
// 	// 	initialized = true
// 	// } else {
// 	// 	initialized = false
// 	// }
// }

func Handler(w http.ResponseWriter, r *http.Request) { //对下游的请求r进行响应w
	// 使用互斥锁保护对全局变量的读取和写入操作
	// mutex.Lock()
	// defer mutex.Unlock()
	// 	// 读取环境变量并保存为全局变量
	// openaiEmail = os.Getenv("OPENAI_EMAIL")
	// openaiPassword = os.Getenv("OPENAI_PASSWORD")
	httpProxy = os.Getenv("http_proxy")
	// accessToken = os.Getenv("ACCESS_TOKEN")

	// 进行处理////////////////////////////////////////////////////////////////////////
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	accessToken := r.Header.Get("Authorization")
	if accessToken != "" {
		parts := strings.Fields(accessToken)
		if len(parts) >= 2 {
			accessToken = parts[1]
		}
	}
	// func nightmare(c *gin.Context) {///////////////////////////////////////////////////////////////////
	var original_request official_types.APIRequest

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
	//

	// authHeader := c.GetHeader("Authorization")
	// token := ACCESS_TOKENS.GetToken()
	// if authHeader != "" {
	// 	customAccessToken := strings.Replace(authHeader, "Bearer ", "", 1)
	// 	// Check if customAccessToken starts with sk-
	// 	if strings.HasPrefix(customAccessToken, "eyJhbGciOiJSUzI1NiI") {
	// 		token = customAccessToken
	// 	}
	// }
	// Convert the chat request to a ChatGPT request转换官方API请求为chatGPTAPI请求（backend-api）
	// original_request.Stream = false 这并不有效
	translated_request := chatgpt_request_converter.ConvertAPIRequest(original_request)

	response, err := chatgpt.POSTconversation(translated_request, accessToken) //向上游发起请求
	if err != nil {                                                            //向上游发起请求出错
		w.WriteHeader(http.StatusInternalServerError)
		errorResponse := map[string]interface{}{
			"error": "error sending request",
		}
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	defer response.Body.Close()
	if chatgpt.Handle_request_error(w, response) { //向下游发回应错误
		return
	}
	//对上游返回的有效响应进行处理 w-----------------------------------------------------------------
	var full_response string
	// for i := 2; i > 0; i-- {
	var continue_info *chatgpt.ContinueInfo
	var response_part string
	response_part, continue_info = chatgpt.Handler(&w, response, accessToken, translated_request, original_request.Stream) /////////////////////////////////////////////////////////
	full_response = response_part                                                                                          /////////////////////////////////////////////////////////
	println(full_response)
	if continue_info == nil {
		// break
		// os.Setenv("ConversationID", "")
	}

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
		response := official_types.NewChatCompletion(full_response) //完成非流回复
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain") //完成流式回复
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}

	// }

}
