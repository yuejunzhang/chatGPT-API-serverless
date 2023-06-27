本项目是可部署在vercel上的无服务函数，实现chatGPT伪API代理（免费且无限制）

同一账号会有不同的 access-token，并且都有效

暂时没有会话ID（ConversationID）和上下文ID（ParentMessageID）支持，客户端自行管理会话与上下文。

暂未添加认证模块

会话历史保存默认允许

使用方法同官方API，区别是将请求头"Authorization"设置为"Bearer "+"<你的access-token>”

例：

curl 本项目网址/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $access-token" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "system", "content": "You are a helpful assistant."}, {"role": "user", "content": "Hello!"}]
  }'


你的access-token可以通过登录官方网页 https://chat.openai.com 后由 https://chat.openai.com/api/auth/session 获取，前提是要有官方账号。（添加认证模块后将自动获取access-token）


func main() {
	// Init()
	// 创建一个响应记录器
	ww := httptest.NewRecorder()
	// 创建一个 JSON 字符串作为请求体
	requestBody := `{    
		"model":  "gpt-3.5-turbo",
    	"messages": [{"role": "user", "content": "你是谁"}],
		"stream": true 
		}`
	// 创建一个请求对象
	rr := httptest.NewRequest("POST", "http://www.chat.openai.com", bytes.NewBuffer([]byte(requestBody)))

	rr.Header.Add("Content-Type", "application/json")
	rr.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36")
	rr.Header.Add("Authorization", "Bearer "+"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Ik1UaEVOVUpHTkVNMVFURTRNMEZCTWpkQ05UZzVNRFUxUlRVd1FVSkRNRU13UmtGRVFrRXpSZyJ9.eyJodHRwczovL2FwaS5vcGVuYWkuY29tL3Byb2ZpbGUiOnsiZW1haWwiOiJha3Vua2VqaUBnbWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZX0sImh0dHBzOi8vYXBpLm9wZW5haS5jb20vYXV0aCI6eyJ1c2VyX2lkIjoidXNlci1yUTBOejFGa2d5VkJvblZTUDk1WnpWdlIifSwiaXNzIjoiaHR0cHM6Ly9hdXRoMC5vcGVuYWkuY29tLyIsInN1YiI6Imdvb2dsZS1vYXV0aDJ8MTE1NTA3NTE1MjAzNDIwNDY0NDkzIiwiYXVkIjpbImh0dHBzOi8vYXBpLm9wZW5haS5jb20vdjEiLCJodHRwczovL29wZW5haS5vcGVuYWkuYXV0aDBhcHAuY29tL3VzZXJpbmZvIl0sImlhdCI6MTY4Nzc2NTAxMywiZXhwIjoxNjg4OTc0NjEzLCJhenAiOiJUZEpJY2JlMTZXb1RIdE45NW55eXdoNUU0eU9vNkl0RyIsInNjb3BlIjoib3BlbmlkIHByb2ZpbGUgZW1haWwgbW9kZWwucmVhZCBtb2RlbC5yZXF1ZXN0IG9yZ2FuaXphdGlvbi5yZWFkIG9yZ2FuaXphdGlvbi53cml0ZSJ9.iIJJaNbv_uWzB2MSlHrmJ293qEvLcfOWYPEO_EjuPTvjj7EF1otQMfz2ppj3_blKfvxE4zGLZhTLOdIdY278WzZ6lXvqekQ-8HbdW90peHVgqgf2Kj1kKbZkSO-aDuhoerCUxnp1KE4oBd6Hp29BFWgRyCx2senox2TM7canm___VB8p0uNGw9w1At35o2OCWNZ11oE90zvZU_SPoe-Vcx1LITZrWhCC48WjyPMGlB1ghentYDeJMTThxJbb1vFLT6tiPEHmynALO4aW6_GFAm02UkCE4CVSv5O2aLxMOK0LLu4HuQ5R6HmCn3Xedbnwrk556OnfFtwmT8IV83mkwA")
	// 调用处理函数
	// rr.Header.Add("Authorization", "Bearer "+"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Ik1UaEVOVUpHTkVNMVFURTRNMEZCTWpkQ05UZzVNRFUxUlRVd1FVSkRNRU13UmtGRVFrRXpSZyJ9.eyJodHRwczovL2FwaS5vcGVuYWkuY29tL3Byb2ZpbGUiOnsiZW1haWwiOiJha3Vua2VqaUBnbWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZX0sImh0dHBzOi8vYXBpLm9wZW5haS5jb20vYXV0aCI6eyJ1c2VyX2lkIjoidXNlci1yUTBOejFGa2d5VkJvblZTUDk1WnpWdlIifSwiaXNzIjoiaHR0cHM6Ly9hdXRoMC5vcGVuYWkuY29tLyIsInN1YiI6Imdvb2dsZS1vYXV0aDJ8MTE1NTA3NTE1MjAzNDIwNDY0NDkzIiwiYXVkIjpbImh0dHBzOi8vYXBpLm9wZW5haS5jb20vdjEiLCJodHRwczovL29wZW5haS5vcGVuYWkuYXV0aDBhcHAuY29tL3VzZXJpbmZvIl0sImlhdCI6MTY4Njk2MDg1NiwiZXhwIjoxNjg4MTcwNDU2LCJhenAiOiJUZEpJY2JlMTZXb1RIdE45NW55eXdoNUU0eU9vNkl0RyIsInNjb3BlIjoib3BlbmlkIHByb2ZpbGUgZW1haWwgbW9kZWwucmVhZCBtb2RlbC5yZXF1ZXN0IG9yZ2FuaXphdGlvbi5yZWFkIG9yZ2FuaXphdGlvbi53cml0ZSJ9.kqdFg7UlvzNxNm1ns6hVDzb6UhvmRB9cf9xbjHHlY5FXXaO_dqE1qxZPOeq_bHk081bRxgXByYukxYMk8CAf6IrXZstLP5253kfwXMTbX8XvrYENek3FKp6F_C4GboosFaRKv2G6bvVYIORJfXvXlOzSDwlaxMDn8FuT3Cc1KoaHoF_yK-rgQFCLPPqcfGlJaTHuDMz1dS_Cj40mlNHBHJC5I-Y3vgrdrHmz1_31AwVGAmMyrQanNAp8daazhVVngA7xaUNqiF-18f3bHElgfiIF0JRqD5SLmCO_OQj22HKZhRk2g2Xq0U6EmrHTyiN5RXu1VzEOjrb_7RZy2fUsgQ")

	Handler(ww, rr)
}