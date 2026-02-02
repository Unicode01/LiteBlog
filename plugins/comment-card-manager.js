// 评论和卡片管理插件示例

function Init() {
    log(1, "评论卡片管理插件初始化")
    
    namespace = genNamespace()
    
    // 评论相关
    commentsHandler = injectNamespace(namespace, "comments", handleComments)
    addCommentHandler = injectNamespace(namespace, "addComment", handleAddComment)
    deleteCommentHandler = injectNamespace(namespace, "deleteComment", handleDeleteComment)
    
    // 卡片相关
    cardsHandler = injectNamespace(namespace, "cards", handleCards)
    cardHandler = injectNamespace(namespace, "card", handleCard)
    
    registerMethods([
        commentsHandler, addCommentHandler, deleteCommentHandler,
        cardsHandler, cardHandler
    ])
    
    // 评论路由
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/api/plugin/articles/:id/comments"),
        buildArg("callback", commentsHandler)
    ])
    
    // 卡片路由
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/api/plugin/cards"),
        buildArg("callback", cardsHandler)
    ])
    
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/api/plugin/cards/:id"),
        buildArg("callback", cardHandler)
    ])
    
    log(1, "可用路由:")
    log(1, "  GET    /api/plugin/articles/:id/comments - 获取评论")
    log(1, "  POST   /api/plugin/articles/:id/comments - 添加评论 (需要token)")
    log(1, "  DELETE /api/plugin/articles/:id/comments - 删除评论 (需要token)")
    log(1, "  GET    /api/plugin/cards                 - 获取所有卡片")
    log(1, "  POST   /api/plugin/cards                 - 创建卡片 (需要token)")
    log(1, "  GET    /api/plugin/cards/:id             - 获取单个卡片")
    log(1, "  PUT    /api/plugin/cards/:id             - 更新卡片 (需要token)")
    log(1, "  DELETE /api/plugin/cards/:id             - 删除卡片 (需要token)")
}

// 辅助函数
function verifyToken(headers) {
    var token = ""
    if (headers && headers["Authorization"]) {
        var authHeader = headers["Authorization"][0]
        if (authHeader.indexOf("Bearer ") === 0) {
            token = authHeader.substring(7)
        } else {
            token = authHeader
        }
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
        buildArg("body", JSON.stringify(data, null, 2))
    ]
}

function errorResponse(statusCode, message) {
    return jsonResponse(statusCode, {success: false, error: message})
}

// ========== 评论处理 ==========

function handleComments(args) {
    try {
        var parsed = parseArgs(args)
        var params = parsed.params || {}
        var articleId = params.id || ""
        
        if (parsed.method === "GET") {
            return handleGetComments(args, articleId)
        } else if (parsed.method === "POST") {
            return handleAddComment(args, articleId)
        } else if (parsed.method === "DELETE") {
            return handleDeleteComment(args, articleId)
        } else {
            return errorResponse(405, "方法不允许")
        }
    } catch (e) {
        log(3, "评论路由错误: " + e.message)
        return errorResponse(500, "内部错误")
    }
}

function handleGetComments(args, articleId) {
    args = parseArgs(args)
    
    if (!articleId) {
        return errorResponse(400, "缺少文章ID")
    }
    
    var result = GetComments([buildArg("article_id", articleId)])
    result = parseArgs(result)
    
    if (!result.success) {
        return errorResponse(404, result.error || "获取评论失败")
    }
    
    return jsonResponse(200, {
        success: true,
        comments: result.comments
    })
}

function handleAddComment(args, articleId) {
    args = parseArgs(args)
    
    if (!verifyToken(args.headers)) {
        return errorResponse(401, "未授权")
    }
    
    if (!articleId) {
        return errorResponse(400, "缺少文章ID")
    }
    
    var body
    try {
        if (!args.body || args.body.length === 0) {
            return errorResponse(400, "请求体为空")
        }
        body = JSON.parse(args.body.toString())
    } catch (e) {
        return errorResponse(400, "无效的JSON")
    }
    
    if (!body.author || !body.content) {
        return errorResponse(400, "缺少必需字段")
    }
    
    var commentArgs = [
        buildArg("article_id", articleId),
        buildArg("author", body.author),
        buildArg("content", body.content)
    ]
    
    if (body.email) commentArgs.push(buildArg("email", body.email))
    if (body.reply_to) commentArgs.push(buildArg("reply_to", body.reply_to))
    if (body.subscribed) commentArgs.push(buildArg("subscribed", body.subscribed))
    
    var result = AddComment(commentArgs)
    result = parseArgs(result)
    
    if (!result.success) {
        return errorResponse(500, "添加评论失败")
    }
    
    log(1, "评论已添加: " + result.comment_id)
    
    return jsonResponse(201, {
        success: true,
        comment_id: result.comment_id
    })
}

function handleDeleteComment(args, articleId) {
    args = parseArgs(args)
    
    if (!verifyToken(args.headers)) {
        return errorResponse(401, "未授权")
    }
    
    var body
    try {
        body = JSON.parse(args.body.toString())
    } catch (e) {
        return errorResponse(400, "无效的JSON")
    }
    
    if (!body.comment_id) {
        return errorResponse(400, "缺少评论ID")
    }
    
    var result = DeleteComment([
        buildArg("article_id", articleId),
        buildArg("comment_id", body.comment_id)
    ])
    result = parseArgs(result)
    
    if (!result.success) {
        return errorResponse(404, result.error || "删除失败")
    }
    
    log(1, "评论已删除: " + body.comment_id)
    
    return jsonResponse(200, {success: true})
}

// ========== 卡片处理 ==========

function handleCards(args) {
    try {
        var parsed = parseArgs(args)
        
        if (parsed.method === "GET") {
            return handleGetAllCards(args)
        } else if (parsed.method === "POST") {
            return handleAddCard(args)
        } else {
            return errorResponse(405, "方法不允许")
        }
    } catch (e) {
        log(3, "卡片路由错误: " + e.message)
        return errorResponse(500, "内部错误")
    }
}

function handleCard(args) {
    try {
        var parsed = parseArgs(args)
        
        if (parsed.method === "GET") {
            return handleGetCard(args)
        } else if (parsed.method === "PUT" || parsed.method === "PATCH") {
            return handleEditCard(args)
        } else if (parsed.method === "DELETE") {
            return handleDeleteCard(args)
        } else {
            return errorResponse(405, "方法不允许")
        }
    } catch (e) {
        log(3, "卡片路由错误: " + e.message)
        return errorResponse(500, "内部错误")
    }
}

function handleGetAllCards(args) {
    args = parseArgs(args)
    
    var result = GetAllCards([])
    result = parseArgs(result)
    
    if (!result.success) {
        return errorResponse(500, "获取卡片失败")
    }
    
    return jsonResponse(200, {
        success: true,
        count: result.cards.length,
        cards: result.cards
    })
}

function handleGetCard(args) {
    args = parseArgs(args)
    
    var params = args.params || {}
    var cardId = params.id || ""
    
    if (!cardId) {
        return errorResponse(400, "缺少卡片ID")
    }
    
    var result = GetCard([buildArg("card_id", cardId)])
    result = parseArgs(result)
    
    if (!result.success) {
        return errorResponse(404, result.error || "卡片未找到")
    }
    
    return jsonResponse(200, {
        success: true,
        card: result.card
    })
}

function handleAddCard(args) {
    args = parseArgs(args)
    
    if (!verifyToken(args.headers)) {
        return errorResponse(401, "未授权")
    }
    
    var body
    try {
        if (!args.body) {
            return errorResponse(400, "请求体为空")
        }
        body = JSON.parse(args.body.toString())
    } catch (e) {
        return errorResponse(400, "无效的JSON")
    }
    
    var result = AddCard([buildArg("card", body, "json")])
    result = parseArgs(result)
    
    if (!result.success) {
        return errorResponse(500, "创建卡片失败")
    }
    
    log(1, "卡片已创建: " + result.card_id)
    
    return jsonResponse(201, {
        success: true,
        card_id: result.card_id
    })
}

function handleEditCard(args) {
    args = parseArgs(args)
    
    if (!verifyToken(args.headers)) {
        return errorResponse(401, "未授权")
    }
    
    var params = args.params || {}
    var cardId = params.id || ""
    
    if (!cardId) {
        return errorResponse(400, "缺少卡片ID")
    }
    
    var body
    try {
        body = JSON.parse(args.body.toString())
    } catch (e) {
        return errorResponse(400, "无效的JSON")
    }
    
    body.id = cardId
    
    var result = EditCard([buildArg("card", body, "json")])
    result = parseArgs(result)
    
    if (!result.success) {
        return errorResponse(404, result.error || "更新失败")
    }
    
    log(1, "卡片已更新: " + cardId)
    
    return jsonResponse(200, {success: true})
}

function handleDeleteCard(args) {
    args = parseArgs(args)
    
    if (!verifyToken(args.headers)) {
        return errorResponse(401, "未授权")
    }
    
    var params = args.params || {}
    var cardId = params.id || ""
    
    if (!cardId) {
        return errorResponse(400, "缺少卡片ID")
    }
    
    var result = DeleteCard([buildArg("card_id", cardId)])
    result = parseArgs(result)
    
    if (!result.success) {
        return errorResponse(404, result.error || "删除失败")
    }
    
    log(1, "卡片已删除: " + cardId)
    
    return jsonResponse(200, {success: true})
}

Init()

