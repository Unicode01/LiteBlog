function Init() {
    log(1, "loaderVersion:" + loaderVersion)
    log(1, "pluginId:" + pluginId)
    log(1, "pluginName:" + pluginName)
    log(1, "pluginDirPath:" + pluginDirPath)
    log(1, "publicMethods:" + getPublicMethods())

    namespace = genNamespace()

    // 注册处理函数
    welcomeHandler = injectNamespace(namespace, "welcome", welcomeHook)
    articleHandler = injectNamespace(namespace, "article", articleHook)
    fileHandler = injectNamespace(namespace, "file", fileHook)
    listenerHandler = injectNamespace(namespace, "listener", routeListener)

    // 注册所有方法
    registerMethods([welcomeHandler, articleHandler, fileHandler, listenerHandler])

    // 示例1: 精确匹配路由
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/js-welcome"),   // 精确匹配
        buildArg("callback", welcomeHandler)
    ])

    // 示例2: 参数化路由 - 使用 :param 匹配单个路径段
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/api/articles/:id"),  // 参数匹配
        buildArg("callback", articleHandler)
    ])

    // 示例3: 通配符路由 - 使用 *wildcard 匹配剩余所有路径
    AddHook([
        buildArg("class", "onRequest"),
        buildArg("name", "/files/*path"),  // 通配符匹配
        buildArg("callback", fileHandler)
    ])

    // 路由监听示例：监听 /js-welcome 的请求与响应
    AddRouteListener([
        buildArg("route", "/js-welcome"),
        buildArg("callback", listenerHandler),
        buildArg("phase", "both")
    ])

    log(1, "Routes registered:")
    log(1, "  - GET /js-welcome          -> welcomeHook (exact match)")
    log(1, "  - GET /api/articles/:id    -> articleHook (param match)")
    log(1, "  - GET /files/*path         -> fileHook (wildcard match)")
    log(1, "  - listen /js-welcome       -> routeListener (phase: both)")
}

// 精确匹配路由处理函数
function welcomeHook(args) {
    args = parseArgs(args)
    log(1, "[welcomeHook] path=" + args.path + ", method=" + args.method)

    return [
        buildArg("statusCode", 200, "int"),
        buildArg("header", {
            "Content-Type": "text/html",
            "Server": "LiteBlog-JS-Plugin"
        }, "json"),
        buildArg("body", "<h1>Welcome from JS Plugin!</h1><p>This is an exact match route.</p>")
    ]
}

// 参数化路由处理函数 - 处理 /api/articles/:id
function articleHook(args) {
    args = parseArgs(args)

    // 获取路由参数
    var params = args.params || {}
    var articleId = params.id || "unknown"

    log(1, "[articleHook] path=" + args.path + ", articleId=" + articleId)

    // 构建 JSON 响应
    var response = {
        success: true,
        data: {
            id: articleId,
            title: "Article " + articleId,
            content: "This is article content for ID: " + articleId,
            path: args.path,
            method: args.method,
            params: params
        }
    }

    return [
        buildArg("statusCode", 200, "int"),
        buildArg("header", {
            "Content-Type": "application/json",
            "Server": "LiteBlog-JS-Plugin"
        }, "json"),
        buildArg("body", JSON.stringify(response, null, 2))
    ]
}

// 通配符路由处理函数 - 处理 /files/*path
function fileHook(args) {
    args = parseArgs(args)

    // 获取通配符参数
    var params = args.params || {}
    var filepath = params.path || ""

    log(1, "[fileHook] path=" + args.path + ", filepath=" + filepath)

    var html = '<h1>File Server (JS Plugin)</h1>' +
        '<p>Requested file: <code>' + filepath + '</code></p>' +
        '<p>Full path: <code>' + args.path + '</code></p>' +
        '<p>Route params: <code>' + JSON.stringify(params) + '</code></p>' +
        '<p>This demonstrates wildcard route matching with *path</p>'

    return [
        buildArg("statusCode", 200, "int"),
        buildArg("header", {
            "Content-Type": "text/html",
            "Server": "LiteBlog-JS-Plugin"
        }, "json"),
        buildArg("body", html)
    ]
}

// 路由监听示例，观察请求/响应
function routeListener(args) {
    args = parseArgs(args)

    var phase = args.phase || "response"
    var status = args.statusCode || 0
    var reqBody = args.body || ""
    var respBody = args.responseBody || ""

    log(1, "[routeListener][" + phase + "] " + args.method + " " + args.path + " traceID=" + args.traceID + " status=" + status)

    if (reqBody && reqBody.length) {
        log(1, "  reqBody (" + reqBody.length + " bytes): " + reqBody.toString().substring(0, 200))
    }
    if (phase !== "request" && respBody && respBody.length) {
        log(1, "  respBody (" + respBody.length + " bytes): " + respBody.toString().substring(0, 200))
    }

    return [
        buildArg("ack", "ok")
    ]
}

Init()