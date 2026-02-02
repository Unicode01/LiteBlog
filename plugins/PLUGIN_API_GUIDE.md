# LiteBlog 插件 API 参考

## 概述

LiteBlog 插件系统支持 **gRPC** 和 **JavaScript** 两种插件类型。

---

## 核心 API

### 文章操作

#### GetArticle - 获取文章
**参数:**
- `article_id` (string): 文章ID

**返回:**
- `success` (bool): 是否成功
- `article` (json): 文章JSON数据
- `error` (string): 错误信息（失败时）

```javascript
var result = GetArticle([buildArg("article_id", "article123")])
result = parseArgs(result)
if (result.success) {
    var article = JSON.parse(result.article)
}
```

#### AddArticle - 创建文章
**参数:**
- `title` (string): 标题
- `content` (string): Markdown内容
- `content_html` (string): HTML内容
- `author` (string): 作者
- `extra_flags` (json): 额外标记（可选）

**返回:**
- `success` (bool): 是否成功
- `article_id` (string): 新文章ID
- `error` (string): 错误信息（失败时）

```javascript
var result = AddArticle([
    buildArg("title", "标题"),
    buildArg("content", "# 内容"),
    buildArg("content_html", "<h1>内容</h1>"),
    buildArg("author", "作者")
])
```

#### EditArticle - 编辑文章
**参数:**
- `article_id` (string): 文章ID（必需）
- `title` (string): 标题（可选）
- `content` (string): 内容（可选）
- `content_html` (string): HTML（可选）
- `author` (string): 作者（可选）
- `extra_flags` (json): 标记（可选）

**返回:**
- `success` (bool): 是否成功

**注意:** 只更新提供的字段

#### DeleteArticle - 删除文章
**参数:**
- `article_id` (string): 文章ID

**返回:**
- `success` (bool): 是否成功

#### GetAllArticleIDs - 获取文章ID列表
**返回:**
- `success` (bool): 是否成功
- `article_ids` (json): ID数组

---

### 评论操作

#### GetComments - 获取评论
**参数:**
- `article_id` (string): 文章ID

**返回:**
- `success` (bool): 是否成功
- `comments` (json): 评论数组
- `error` (string): 错误信息

#### AddComment - 添加评论
**参数:**
- `article_id` (string): 文章ID
- `author` (string): 作者
- `email` (string): 邮箱（可选）
- `content` (string): 内容
- `reply_to` (string): 回复的评论ID（可选）
- `subscribed` (bool): 是否订阅（可选）

**返回:**
- `success` (bool): 是否成功
- `comment_id` (string): 新评论ID

```javascript
var result = AddComment([
    buildArg("article_id", "firstArticle"),
    buildArg("author", "张三"),
    buildArg("content", "很棒的文章！"),
    buildArg("email", "test@example.com")
])
```

#### DeleteComment - 删除评论
**参数:**
- `article_id` (string): 文章ID
- `comment_id` (string): 评论ID

**返回:**
- `success` (bool): 是否成功

---

### 卡片操作

#### GetAllCards - 获取所有卡片
**返回:**
- `success` (bool): 是否成功
- `cards` (json): 卡片数组

#### GetCard - 获取单个卡片
**参数:**
- `card_id` (string): 卡片ID

**返回:**
- `success` (bool): 是否成功
- `card` (json): 卡片数据

#### AddCard - 添加卡片
**参数:**
- `card` (json): 卡片数据对象

**返回:**
- `success` (bool): 是否成功
- `card_id` (string): 新卡片ID

```javascript
var result = AddCard([
    buildArg("card", {
        "card_title": "新卡片",
        "card_description": "描述",
        "card_link": "/articles/test",
        "template": "card_template_classical",
        "order": "1"
    }, "json")
])
```

#### EditCard - 编辑卡片
**参数:**
- `card` (json): 卡片数据（必须包含id）

**返回:**
- `success` (bool): 是否成功

#### DeleteCard - 删除卡片
**参数:**
- `card_id` (string): 卡片ID

**返回:**
- `success` (bool): 是否成功

---

### 认证与配置

#### VerifyToken - 验证Token
**参数:**
- `token` (string): 访问令牌

**返回:**
- `valid` (bool): 是否有效

```javascript
var result = VerifyToken([buildArg("token", token)])
result = parseArgs(result)
if (result.valid) {
    // Token有效
}
```

#### GetConfig - 读取配置
**参数:**
- `key` (string): 配置键（如 "server_config"）

**返回:**
- `success` (bool): 是否成功
- `value` (json): 配置值

---

### 工具

#### Log - 记录日志
**参数:**
- `level` (int): 日志级别（0=DEBUG, 1=INFO, 2=WARNING, 3=ERROR）
- `message` (string): 消息
- `plugin_name` (string): 插件名（可选）

```javascript
Log([
    buildArg("level", 1, "int"),
    buildArg("message", "操作完成"),
    buildArg("plugin_name", pluginName)
])
```

---

### 路由管理

#### AddHook - 添加路由钩子
**参数:**
- `class` (string): 固定为 `"onRequest"`
- `name` (string): 路由模式
  - 精确匹配: `/api/users`
  - 参数匹配: `/api/users/:id`
  - 通配符匹配: `/files/*path`
- `callback` (string): 回调函数名

```javascript
// 精确匹配
AddHook([
    buildArg("class", "onRequest"),
    buildArg("name", "/api/endpoint"),
    buildArg("callback", handlerName)
])

// 参数匹配
AddHook([
    buildArg("class", "onRequest"),
    buildArg("name", "/api/articles/:id"),
    buildArg("callback", handlerName)
])
```

**钩子回调接收的参数:**
- `path` (string): 请求路径
- `method` (string): HTTP方法
- `headers` (json): 请求头
- `ip` (string): 客户端IP
- `traceID` (string): 追踪ID
- `params` (json): 路由参数（参数化路由）
- `body` (bytes): 请求体

**钩子回调返回值:**
- `statusCode` (int): HTTP状态码
- `header` (json): 响应头
- `body` (bytes): 响应体

#### DeleteHook - 删除钩子
**参数:**
- `class` (string): `"onRequest"`
- `name` (string): 路由模式

---

## 完整示例

### JavaScript 插件

```javascript
function Init() {
    log(1, "插件初始化")
    
    namespace = genNamespace()
    handler = injectNamespace(namespace, "handler", handleRequest)
    registerMethods([handler])
    
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/api/myendpoint"),
        buildArg("callback", handler)
    ])
}

function handleRequest(args) {
    args = parseArgs(args)
    
    // 验证token（如需要）
    if (!verifyToken(args.headers)) {
        return errorResponse(401, "Unauthorized")
    }
    
    // 获取文章列表
    var result = GetAllArticleIDs([])
    result = parseArgs(result)
    
    if (result.success) {
        var ids = JSON.parse(result.article_ids)
        return jsonResponse(200, {articles: ids})
    }
    
    return errorResponse(500, "Error")
}

function verifyToken(headers) {
    var token = ""
    if (headers && headers["Authorization"]) {
        token = headers["Authorization"][0]
    }
    if (!token) return false
    
    var result = VerifyToken([buildArg("token", token)])
    result = parseArgs(result)
    return result.valid
}

function jsonResponse(statusCode, data) {
    return [
        buildArg("statusCode", statusCode, "int"),
        buildArg("header", {"Content-Type": "application/json"}, "json"),
        buildArg("body", JSON.stringify(data))
    ]
}

function errorResponse(statusCode, message) {
    return jsonResponse(statusCode, {error: message})
}

Init()
```

### 路由分发示例

```javascript
// 根据HTTP方法分发请求
function routeArticles(args) {
    args = parseArgs(args)
    
    if (args.method === "GET") {
        return handleList(args)
    } else if (args.method === "POST") {
        return handleCreate(args)
    } else {
        return errorResponse(405, "方法不允许")
    }
}
```

---

## 配置

在 `configs/config.json` 中启用插件：

```json
{
  "plugins_config": {
    "enabled": true,
    "grpc_config": {
      "enabled": true,
      "listener_address": "127.0.0.1:9090",
      "access_key": "your-key"
    },
    "js_config": {
      "enabled": true,
      "plugin_dir": "plugins",
      "init_delay": 2
    }
  }
}
```

---

## 安全最佳实践

1. **始终验证Token**（敏感操作）
2. **验证输入**（检查必需字段、长度限制）
3. **错误处理**（使用 try-catch 捕获异常）
4. **记录日志**（使用 Log API）

---

## 测试

```bash
# 启动服务
./LiteBlog

# 测试端点
curl http://localhost:8080/api/myendpoint

# 带token的POST请求
curl -X POST http://localhost:8080/api/myendpoint \
  -H "Authorization: your-token" \
  -H "Content-Type: application/json" \
  -d '{"title": "测试"}'
```

---

## 常见问题

**Q: Token在哪里配置？**  
A: 在 `configs/config.json` 的 `access_config.access_token`

**Q: 插件未加载？**  
A: 检查 `plugins_config.enabled` 是否为 true，查看 `liteblog.log`

**Q: 如何传递Token？**  
A: HTTP请求头：`Authorization: Bearer your-token` 或 `Authorization: your-token`

**Q: 只读模式下能创建文章吗？**  
A: 不能。只读模式下所有写操作都会失败

---

**更多示例:** 查看 `plugins/article-manager.js` 和 `plugins/test.js`
