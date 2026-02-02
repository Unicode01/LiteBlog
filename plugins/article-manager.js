// LiteBlog 插件示例：文章管理插件
// 此插件提供了一套RESTful API用于管理文章

function Init() {
    log(1, "文章管理插件初始化开始")
    log(1, "插件ID: " + pluginId)
    log(1, "插件名称: " + pluginName)

    // 生成命名空间
    namespace = genNamespace()

    // 注册路由分发器
    articlesRouter = injectNamespace(namespace, "articlesRouter", routeArticles)
    articleRouter = injectNamespace(namespace, "articleRouter", routeArticle)

    // 注册方法到插件管理器
    registerMethods([articlesRouter, articleRouter])

    // 添加路由钩子
    // /api/plugin/articles - 列表(GET) 和 创建(POST)
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/api/plugin/articles"),
        buildArg("callback", articlesRouter)
    ])

    // /api/plugin/articles/:id - 获取(GET)、更新(PUT)、删除(DELETE)
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/api/plugin/articles/:id"),
        buildArg("callback", articleRouter)
    ])

    log(1, "文章管理插件初始化完成")
    log(1, "可用路由:")
    log(1, "  GET    /api/plugin/articles      - 获取文章列表")
    log(1, "  POST   /api/plugin/articles      - 创建文章 (需要token)")
    log(1, "  GET    /api/plugin/articles/:id  - 获取单个文章")
    log(1, "  PUT    /api/plugin/articles/:id  - 更新文章 (需要token)")
    log(1, "  DELETE /api/plugin/articles/:id  - 删除文章 (需要token)")
}

// 路由分发器：/api/plugin/articles
function routeArticles(args) {
    try {
        var parsed = parseArgs(args)

        if (parsed.method === "GET") {
            return handleListArticles(args)
        } else if (parsed.method === "POST") {
            return handleCreateArticle(args)
        } else {
            return errorResponse(405, "方法不允许")
        }
    } catch (e) {
        log(3, "[文章管理] 路由错误: " + e.message)
        return errorResponse(500, "内部错误")
    }
}

// 路由分发器：/api/plugin/articles/:id
function routeArticle(args) {
    try {
        var parsed = parseArgs(args)

        if (parsed.method === "GET") {
            return handleGetArticle(args)
        } else if (parsed.method === "PUT" || parsed.method === "PATCH") {
            return handleUpdateArticle(args)
        } else if (parsed.method === "DELETE") {
            return handleDeleteArticle(args)
        } else {
            return errorResponse(405, "方法不允许")
        }
    } catch (e) {
        log(3, "[文章管理] 路由错误: " + e.message)
        return errorResponse(500, "内部错误")
    }
}

// 辅助函数：验证Token
function verifyToken(headers) {
    var token = ""

    // 尝试从Authorization头获取token
    if (headers && headers["Authorization"]) {
        var authHeader = headers["Authorization"][0]
        // 支持 "Bearer <token>" 格式
        if (authHeader.indexOf("Bearer ") === 0) {
            token = authHeader.substring(7)
        } else {
            token = authHeader
        }
    }

    if (!token) {
        return false
    }

    var result = VerifyToken([buildArg("token", token)])
    result = parseArgs(result)
    return result.valid
}

// 辅助函数：返回JSON响应
function jsonResponse(statusCode, data) {
    return [
        buildArg("statusCode", statusCode, "int"),
        buildArg("header", { "Content-Type": "application/json; charset=utf-8" }, "json"),
        buildArg("body", JSON.stringify(data, null, 2))
    ]
}

// 辅助函数：返回错误响应
function errorResponse(statusCode, message) {
    return jsonResponse(statusCode, {
        success: false,
        error: message
    })
}

// 处理获取文章列表
function handleListArticles(args) {
    args = parseArgs(args)

    log(1, "[文章管理] 获取文章列表请求: " + args.path)

    // 获取所有文章ID
    var result = GetAllArticleIDs([])
    result = parseArgs(result)

    if (!result.success) {
        log(3, "[文章管理] 获取文章ID列表失败: " + result.error)
        return errorResponse(500, "获取文章列表失败")
    }

    var articleIds = result.article_ids  // parseArgs 已经解析了 JSON
    var articles = []

    // 获取每篇文章的基本信息
    for (var i = 0; i < articleIds.length; i++) {
        var articleId = articleIds[i]
        var articleResult = GetArticle([buildArg("article_id", articleId)])
        articleResult = parseArgs(articleResult)

        if (articleResult.success) {
            var article = articleResult.article  // parseArgs 已经解析了 JSON
            // 返回完整文章内容
            article.id = articleId
            articles.push(article)
        }
    }

    log(1, "[文章管理] 成功返回 " + articles.length + " 篇文章")

    return jsonResponse(200, {
        success: true,
        count: articles.length,
        articles: articles
    })
}

// 处理获取单个文章
function handleGetArticle(args) {
    args = parseArgs(args)

    var params = args.params || {}
    var articleId = params.id || ""

    log(1, "[文章管理] 获取文章请求: " + articleId)

    if (!articleId) {
        return errorResponse(400, "缺少文章ID")
    }

    // 获取文章
    var result = GetArticle([buildArg("article_id", articleId)])
    result = parseArgs(result)

    if (!result.success) {
        log(2, "[文章管理] 文章未找到: " + articleId)
        return errorResponse(404, "文章未找到")
    }

    var article = result.article  // parseArgs 已经解析了 JSON

    log(1, "[文章管理] 成功返回文章: " + article.title)

    return jsonResponse(200, {
        success: true,
        article: article
    })
}

// 处理创建文章
function handleCreateArticle(args) {
    args = parseArgs(args)

    log(1, "[文章管理] 创建文章请求")

    // 验证token
    if (!verifyToken(args.headers)) {
        log(2, "[文章管理] Token验证失败")
        return errorResponse(401, "未授权：Token无效或缺失")
    }

    // 解析请求体
    var requestBody
    try {
        if (!args.body || args.body.length === 0) {
            return errorResponse(400, "请求体为空")
        }
        var bodyStr = typeof args.body === 'string' ? args.body : args.body.toString()
        requestBody = JSON.parse(bodyStr)
    } catch (e) {
        return errorResponse(400, "无效的JSON格式")
    }

    // 验证必需字段
    if (!requestBody.title || !requestBody.content || !requestBody.author) {
        return errorResponse(400, "缺少必需字段: title, content, author")
    }

    // 添加文章
    var result = AddArticle([
        buildArg("title", requestBody.title),
        buildArg("content", requestBody.content),
        buildArg("content_html", requestBody.content_html || ""),
        buildArg("author", requestBody.author),
        buildArg("extra_flags", requestBody.extra_flags || {}, "json")
    ])
    result = parseArgs(result)

    if (!result.success) {
        log(3, "[文章管理] 创建文章失败: " + result.error)
        return errorResponse(500, "创建文章失败: " + result.error)
    }

    // 记录日志
    Log([
        buildArg("level", 1, "int"),
        buildArg("message", "新文章已创建: " + result.article_id + " - " + requestBody.title),
        buildArg("plugin_name", pluginName)
    ])

    log(1, "[文章管理] 文章创建成功: " + result.article_id)

    return jsonResponse(201, {
        success: true,
        article_id: result.article_id,
        message: "文章创建成功"
    })
}

// 处理更新文章
function handleUpdateArticle(args) {
    args = parseArgs(args)

    var params = args.params || {}
    var articleId = params.id || ""

    log(1, "[文章管理] 更新文章请求: " + articleId)

    // 验证token
    if (!verifyToken(args.headers)) {
        log(2, "[文章管理] Token验证失败")
        return errorResponse(401, "未授权：Token无效或缺失")
    }

    if (!articleId) {
        return errorResponse(400, "缺少文章ID")
    }

    // 解析请求体
    var requestBody
    try {
        if (!args.body || args.body.length === 0) {
            return errorResponse(400, "请求体为空")
        }
        var bodyStr = typeof args.body === 'string' ? args.body : args.body.toString()
        requestBody = JSON.parse(bodyStr)
    } catch (e) {
        return errorResponse(400, "无效的JSON格式")
    }

    // 构建更新参数（只包含提供的字段）
    var updateArgs = [buildArg("article_id", articleId)]

    if (requestBody.title) {
        updateArgs.push(buildArg("title", requestBody.title))
    }
    if (requestBody.content) {
        updateArgs.push(buildArg("content", requestBody.content))
    }
    if (requestBody.content_html) {
        updateArgs.push(buildArg("content_html", requestBody.content_html))
    }
    if (requestBody.author) {
        updateArgs.push(buildArg("author", requestBody.author))
    }
    if (requestBody.extra_flags) {
        updateArgs.push(buildArg("extra_flags", requestBody.extra_flags, "json"))
    }

    // 更新文章
    var result = EditArticle(updateArgs)
    result = parseArgs(result)

    if (!result.success) {
        log(3, "[文章管理] 更新文章失败: " + result.error)
        return errorResponse(500, "更新文章失败: " + result.error)
    }

    // 记录日志
    Log([
        buildArg("level", 1, "int"),
        buildArg("message", "文章已更新: " + articleId),
        buildArg("plugin_name", pluginName)
    ])

    log(1, "[文章管理] 文章更新成功: " + articleId)

    return jsonResponse(200, {
        success: true,
        message: "文章更新成功"
    })
}

// 处理删除文章
function handleDeleteArticle(args) {
    args = parseArgs(args)

    var params = args.params || {}
    var articleId = params.id || ""

    log(1, "[文章管理] 删除文章请求: " + articleId)

    // 验证token
    if (!verifyToken(args.headers)) {
        log(2, "[文章管理] Token验证失败")
        return errorResponse(401, "未授权：Token无效或缺失")
    }

    if (!articleId) {
        return errorResponse(400, "缺少文章ID")
    }

    // 删除文章
    var result = DeleteArticle([buildArg("article_id", articleId)])
    result = parseArgs(result)

    if (!result.success) {
        log(3, "[文章管理] 删除文章失败: " + result.error)
        return errorResponse(500, "删除文章失败: " + result.error)
    }

    // 记录日志
    Log([
        buildArg("level", 1, "int"),
        buildArg("message", "文章已删除: " + articleId),
        buildArg("plugin_name", pluginName)
    ])

    log(1, "[文章管理] 文章删除成功: " + articleId)

    return jsonResponse(200, {
        success: true,
        message: "文章删除成功"
    })
}

// 初始化插件
Init()

