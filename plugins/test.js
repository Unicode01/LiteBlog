function Init() {
    log(1, "loaderVersion:" + loaderVersion)
    log(1, "pluginId:" + pluginId)
    log(1, "pluginName:" + pluginName)
    log(1, "pluginDirPath:" + pluginDirPath)
    log(1, "publicMethods:" + getPublicMethods())
    namespace = genNamespace()
    n = injectNamespace(namespace, "test", welcomeHook)
    // register public Methdos
    // 导出公共方法,以便后续调用
    registerMethods([
        n
    ])
    registerMethods([
        n
    ])
    AddHook([
        buildArg("class","onRequest"),
        buildArg("name", "/welcome"),
        buildArg("callback", n)
    ])
}

function welcomeHook(args) {
    args = parseArgs(args)
    // for (var i = 0; i < args.length; i++) {
    //     let arg = parseArg(args[i])
    //     log(1, "argname:" + arg.Name + " argtype:" + arg.Type + " argdata:" + bytesToString(arg.Data))
    // }
    // log(1, "args:" + JSON.stringify(args))
    rt = [
        buildArg("statusCode", 200, "int"),
        buildArg("header", {
            "Content-Type": "text/html" ,
            "Server": "LiteBlog"
        }, "json"),
        buildArg("body", "<h1>Welcome to LiteBlog</h1>")
    ]
    return rt
}

Init()