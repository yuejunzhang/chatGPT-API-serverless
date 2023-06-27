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


