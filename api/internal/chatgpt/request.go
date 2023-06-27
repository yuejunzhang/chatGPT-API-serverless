package chatgpt

import (
	"bufio"
	"bytes"
	"chatgpt-api-serverless/api/typings"
	chatgpt_types "chatgpt-api-serverless/api/typings/chatgpt"
	"encoding/json"

	// "fmt"
	"io"
	http0 "net/http"

	// "math/rand"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"

	// "github.com/gin-gonic/gin"
	// "github.com/joho/godotenv"

	chatgpt_response_converter "chatgpt-api-serverless/api/conversion/response/chatgpt"

	// chatgpt "freechatgpt/internal/chatgpt"

	official_types "chatgpt-api-serverless/api/typings/official"
)

var proxies []string

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

// func init() {
// 	err := godotenv.Load(".env")
// 	if err != nil {
// 		// 处理加载环境变量文件失败的错误
// 	}
// 	// Check for proxies.txt
// 	if _, err := os.Stat("proxies.txt"); err == nil {
// 		// Each line is a proxy, put in proxies array
// 		file, _ := os.Open("proxies.txt")
// 		defer file.Close()
// 		scanner := bufio.NewScanner(file)
// 		for scanner.Scan() {
// 			// Split line by :
// 			proxy := scanner.Text()
// 			proxy_parts := strings.Split(proxy, ":")
// 			if len(proxy_parts) == 2 {
// 				proxy = "socks5://" + proxy
// 			} else if len(proxy_parts) == 4 {
// 				proxy = "socks5://" + proxy_parts[2] + ":" + proxy_parts[3] + "@" + proxy_parts[0] + ":" + proxy_parts[1]
// 			} else {
// 				continue
// 			}
// 			proxies = append(proxies, proxy)
// 		}
// 	}
// }

// func random_int(min int, max int) int {
// 	return min + rand.Intn(max-min)
// }

func POSTconversation(message chatgpt_types.ChatGPTRequest, access_token string) (*http.Response, error) {
	if http_proxy != "" && len(proxies) == 0 {
		client.SetProxy(http_proxy)
	}
	// // Take random proxy from proxies.txt
	// if len(proxies) > 0 {
	// 	client.SetProxy(proxies[random_int(0, len(proxies))])
	// }
	client.SetProxy("http://127.0.0.1:7890") //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!11
	apiUrl := "https://chat.openai.com/backend-api/conversation"
	// if API_REVERSE_PROXY != "" {
	// 	apiUrl = API_REVERSE_PROXY
	// }

	// JSONify the body and add it to the request
	body_json, err := json.Marshal(message)
	if err != nil {
		return &http.Response{}, err
	}

	request, err := http.NewRequest(http.MethodPost, apiUrl, bytes.NewBuffer(body_json))
	if err != nil {
		return &http.Response{}, err
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
		return &http.Response{}, err
	}
	// fmt.Printf("%+v\n", request)

	response, err := client.Do(request)
	return response, err
}

// Returns whether an error was handled
// func Handle_request_error(c *gin.Context, response *http.Response) bool {
// 	if response.StatusCode != 200 {
// 		// Try read response body as JSON
// 		var error_response map[string]interface{}
// 		err := json.NewDecoder(response.Body).Decode(&error_response)
// 		if err != nil {
// 			// Read response body
// 			body, _ := io.ReadAll(response.Body)
// 			c.JSON(500, gin.H{"error": gin.H{
// 				"message": "Unknown error",
// 				"type":    "internal_server_error",
// 				"param":   nil,
// 				"code":    "500",
// 				"details": string(body),
// 			}})
// 			return true
// 		}
// 		c.JSON(response.StatusCode, gin.H{"error": gin.H{
// 			"message": error_response["detail"],
// 			"type":    response.Status,
// 			"param":   nil,
// 			"code":    "error",
// 		}})
// 		return true
// 	}
// 	return false
// }

func Handle_request_error(w http0.ResponseWriter, resp *http.Response) bool {
	if resp.StatusCode != 200 {
		// Try read response body as JSON
		var error_response map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&error_response)
		if err != nil {
			// Read response body
			body, _ := io.ReadAll(resp.Body)
			// http.Error(w, string(body), http.StatusInternalServerError)
			w.WriteHeader(http0.StatusInternalServerError) ////////////////////////////////////////////////////
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

type ContinueInfo struct {
	ConversationID string `json:"conversation_id"`
	ParentID       string `json:"parent_id"`
}

// func Handler(c *gin.Context, response *http.Response, token string, translated_request chatgpt_types.ChatGPTRequest, stream bool) (string, *ContinueInfo) {
// 	max_tokens := false

// 	// Create a bufio.Reader from the response body
// 	reader := bufio.NewReader(response.Body)

// 	// Read the response byte by byte until a newline character is encountered
// 	if stream {
// 		// Response content type is text/event-stream
// 		c.Header("Content-Type", "text/event-stream")
// 	} else {
// 		// Response content type is application/json
// 		c.Header("Content-Type", "application/json")
// 	}
// 	var finish_reason string
// 	var previous_text typings.StringStruct
// 	var original_response chatgpt_types.ChatGPTResponse
// 	var isRole = true
// 	for {
// 		line, err := reader.ReadString('\n')
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			return "", nil
// 		}
// 		if len(line) < 6 {
// 			continue
// 		}
// 		// Remove "data: " from the beginning of the line
// 		line = line[6:]
// 		// Check if line starts with [DONE]
// 		if !strings.HasPrefix(line, "[DONE]") {
// 			// Parse the line as JSON

// 			err = json.Unmarshal([]byte(line), &original_response)
// 			if err != nil {
// 				continue
// 			}
// 			if original_response.Error != nil {
// 				c.JSON(500, gin.H{"error": original_response.Error})
// 				return "", nil
// 			}
// 			if original_response.Message.Author.Role != "assistant" || original_response.Message.Content.Parts == nil {
// 				continue
// 			}
// 			if original_response.Message.Metadata.MessageType != "next" && original_response.Message.Metadata.MessageType != "continue" || original_response.Message.EndTurn != nil {
// 				continue
// 			}
// 			response_string := chatgpt_response_converter.ConvertToString(&original_response, &previous_text, isRole)
// 			isRole = false
// 			if stream {
// 				_, err = c.Writer.WriteString(response_string)
// 				if err != nil {
// 					return "", nil
// 				}
// 			}
// 			// Flush the response writer buffer to ensure that the client receives each line as it's written
// 			c.Writer.Flush()

// 			if original_response.Message.Metadata.FinishDetails != nil {
// 				if original_response.Message.Metadata.FinishDetails.Type == "max_tokens" {
// 					max_tokens = true
// 				}
// 				finish_reason = original_response.Message.Metadata.FinishDetails.Type
// 			}

// 		} else {
// 			if stream {
// 				final_line := official_types.StopChunk(finish_reason)
// 				c.Writer.WriteString("data: " + final_line.String() + "\n\n")
// 			}
// 		}
// 	}
// 	if !max_tokens {
// 		return previous_text.Text, nil
// 	}
// 	return previous_text.Text, &ContinueInfo{
// 		ConversationID: original_response.ConversationID,
// 		ParentID:       original_response.Message.ID,
// 	}
// }

func Handler(w *http0.ResponseWriter, response *http.Response, token string, translated_request chatgpt_types.ChatGPTRequest, stream bool) (string, *ContinueInfo) {
	max_tokens := false

	// Create a bufio.Reader from the response body
	reader := bufio.NewReader(response.Body) //响应体有n行数据，每一行是回复内容（n个字）的逐字递增，
	// body, err := ioutil.ReadAll(response.Body)
	// if err != nil {
	// 	panic(err)
	// }
	// print(string(body))

	// Read the response byte by byte until a newline character is encountered逐字节读取响应，直到遇到换行符为止
	if stream {
		// Response content type is text/event-stream
		(*w).Header().Set("Content-Type", "text/event-stream")
	} else {
		// Response content type is application/json
		(*w).Header().Set("Content-Type", "application/json")
	}
	var finish_reason string
	var previous_text typings.StringStruct
	var original_response chatgpt_types.ChatGPTResponse
	var isRole = true
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil
		}
		if len(line) < 6 {
			continue
		}
		// Remove "data: " from the beginning of the line
		line = line[6:]
		// Check if line starts with [DONE]
		if !strings.HasPrefix(line, "[DONE]") {
			// Parse the line as JSON

			err = json.Unmarshal([]byte(line), &original_response)
			if err != nil {
				continue
			}
			if original_response.Error != nil {
				(*w).WriteHeader(http.StatusInternalServerError)
				// json.NewEncoder(w).Encode(gin.H{"error": original_response.Error})
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
			response_string := chatgpt_response_converter.ConvertToString(&original_response, &previous_text, isRole)
			//previous_text积累每次循环的单词为一整个文本，用于非流式回复
			isRole = false
			if stream {
				_, err = (*w).Write([]byte("data: " + response_string + "\n\n"))
				if err != nil {
					return "", nil
				}
			}
			// Flush the response writer buffer to ensure that the client receives each line as it's written
			// print(response_string) //////////////////////////////////////////////////////

			// println("\r" + previous_text.Text) //////////////////////////////////////////////////////

			(*w).(http.Flusher).Flush()

			if original_response.Message.Metadata.FinishDetails != nil {
				if original_response.Message.Metadata.FinishDetails.Type == "max_tokens" {
					max_tokens = true
				}
				finish_reason = original_response.Message.Metadata.FinishDetails.Type
			}

		} else {
			if stream {
				final_line := official_types.StopChunk(finish_reason)
				(*w).Write([]byte("data: " + final_line.String() + "\n\n"))
			}
		}
	}

	if !max_tokens {
		// return previous_text.Text, nil ////////////////////////////////////////////////
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

func GETengines() (interface{}, int, error) {
	url := "https://api.openai.com/v1/models"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+os.Getenv("OFFICIAL_API_KEY"))
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	var result interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, resp.StatusCode, nil
}
