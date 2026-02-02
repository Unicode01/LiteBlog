package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	utils "LiteBlog/utils"
	"LiteBlog/utils/plugins"
)

var (
	pluginManager = plugins.NewPluginManager()
)

func InitPlugins() {
	cfg := Config.PluginCfg

	// 注册公共方法（所有加载器共享）
	methodsMap := map[string]func(args []*plugins.Arg) ([]*plugins.Arg, error){
		// 渲染数据管理
		"AddRenderMap":    AddRenderMap,
		"GetRenderMap":    GetRenderMap,
		"DeleteRenderMap": DeleteRenderMap,
		// 路由钩子管理
		"AddHook":             AddHook,
		"DeleteHook":          DeleteHook,
		"AddRouteListener":    AddRouteListener,
		"DeleteRouteListener": DeleteRouteListener,
		// 文章操作API
		"GetArticle":       PluginGetArticle,
		"AddArticle":       PluginAddArticle,
		"EditArticle":      PluginEditArticle,
		"DeleteArticle":    PluginDeleteArticle,
		"GetAllArticleIDs": PluginGetAllArticleIDs,
		// 评论操作API
		"GetComments":   PluginGetComments,
		"AddComment":    PluginAddComment,
		"DeleteComment": PluginDeleteComment,
		// 卡片操作API
		"GetAllCards": PluginGetAllCards,
		"GetCard":     PluginGetCard,
		"AddCard":     PluginAddCard,
		"EditCard":    PluginEditCard,
		"DeleteCard":  PluginDeleteCard,
		// Token验证API
		"VerifyToken": PluginVerifyToken,
		// 配置读取API
		"GetConfig": PluginGetConfig,
		// 日志API
		"Log": PluginLog,
	}
	pluginManager.RegisterMethods(methodsMap)

	// 注册 gRPC 插件加载器
	if cfg.GRPCConfig.Enabled {
		grpcLoader := &plugins.LoaderTypeGRPC{
			ListenerAddress: cfg.GRPCConfig.ListenerAddress,
			AccessKey:       cfg.GRPCConfig.AccessKey,
		}
		// 设置命令超时时间
		if cfg.GRPCConfig.CommandTimeout > 0 {
			grpcLoader.CommandTimeout = time.Duration(cfg.GRPCConfig.CommandTimeout) * time.Second
		}
		loaderId, err := pluginManager.RegisterLoader(grpcLoader)
		if err != nil {
			utils.Log(3, fmt.Sprintf("Register gRPC plugin loader failed: %s", err))
		} else {
			accessKeyStatus := "disabled"
			if cfg.GRPCConfig.AccessKey != "" {
				accessKeyStatus = "enabled"
			}
			utils.Log(1, fmt.Sprintf("gRPC plugin loader registered, id: %s, address: %s, access_key: %s", loaderId, cfg.GRPCConfig.ListenerAddress, accessKeyStatus))
		}
	} else {
		utils.Log(1, "gRPC plugin loader is disabled")
	}

	// 注册 JavaScript 插件加载器
	if cfg.JSConfig.Enabled {
		jsLoader := &plugins.LoaderTypeJS{
			PluginDir: cfg.JSConfig.PluginDir,
			InitDelay: time.Duration(cfg.JSConfig.InitDelay) * time.Second,
		}
		loaderId, err := pluginManager.RegisterLoader(jsLoader)
		if err != nil {
			utils.Log(3, fmt.Sprintf("Register JavaScript plugin loader failed: %s", err))
		} else {
			utils.Log(1, fmt.Sprintf("JavaScript plugin loader registered, id: %s, plugin_dir: %s", loaderId, cfg.JSConfig.PluginDir))
		}
	} else {
		utils.Log(1, "JavaScript plugin loader is disabled")
	}
}

// plugin interface:
func AddRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var class, key string
	var data []byte

	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = utils.GetStringSafe(arg.Data)
		case "key":
			key = utils.GetStringSafe(arg.Data)
		case "data":
			data = utils.GetBytesSafe(arg.Data)
		}
	}

	// 参数验证
	if class == "" {
		return nil, errors.New("missing required parameter: class")
	}
	if key == "" {
		return nil, errors.New("missing required parameter: key")
	}

	switch class {
	case "rendered":
		RenderedMapLocker.Lock()
		RenderedMap[key] = data
		RenderedMapLocker.Unlock()
	case "global":
		GlobalMapLocker.Lock()
		GlobalMap[key] = data
		GlobalMapLocker.Unlock()
	default:
		return nil, fmt.Errorf("unknown class: %s (expected: rendered, global)", class)
	}

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// plugin interface:
func GetRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var class, key string

	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = utils.GetStringSafe(arg.Data)
		case "key":
			key = utils.GetStringSafe(arg.Data)
		}
	}

	// 参数验证
	if class == "" {
		return nil, errors.New("missing required parameter: class")
	}
	if key == "" {
		return nil, errors.New("missing required parameter: key")
	}

	var data []byte
	var found bool

	switch class {
	case "rendered":
		RenderedMapLocker.RLock()
		data, found = RenderedMap[key]
		RenderedMapLocker.RUnlock()
	case "global":
		GlobalMapLocker.RLock()
		data, found = GlobalMap[key]
		GlobalMapLocker.RUnlock()
	default:
		return nil, fmt.Errorf("unknown class: %s (expected: rendered, global)", class)
	}

	if !found {
		return []*plugins.Arg{
			{Name: "found", Type: "bool", Data: false},
		}, nil
	}

	return []*plugins.Arg{
		{Name: "found", Type: "bool", Data: true},
		{Name: "data", Type: "bytes", Data: data},
	}, nil
}

// plugin interface:
func DeleteRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var class, key string

	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = utils.GetStringSafe(arg.Data)
		case "key":
			key = utils.GetStringSafe(arg.Data)
		}
	}

	// 参数验证
	if class == "" {
		return nil, errors.New("missing required parameter: class")
	}
	if key == "" {
		return nil, errors.New("missing required parameter: key")
	}

	switch class {
	case "rendered":
		RenderedMapLocker.Lock()
		delete(RenderedMap, key)
		RenderedMapLocker.Unlock()
	case "global":
		GlobalMapLocker.Lock()
		delete(GlobalMap, key)
		GlobalMapLocker.Unlock()
	default:
		return nil, fmt.Errorf("unknown class: %s (expected: rendered, global)", class)
	}

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// plugin interface:
// Hook class like onRequest ...
// 支持的路由格式：
//   - 精确匹配: /api/users
//   - 参数匹配: /api/users/:id （:前缀表示单段参数）
//   - 通配符匹配: /api/*path （*前缀表示匹配剩余所有路径）
func AddHook(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var hookName, hookClass, callbackName string

	for _, arg := range args {
		switch arg.Name {
		case "name":
			hookName = utils.GetStringSafe(arg.Data)
		case "class":
			hookClass = utils.GetStringSafe(arg.Data)
		case "callback":
			callbackName = utils.GetStringSafe(arg.Data)
		}
	}

	// 参数验证
	if hookName == "" {
		return nil, errors.New("missing required parameter: name")
	}
	if hookClass == "" {
		return nil, errors.New("missing required parameter: class")
	}
	if callbackName == "" {
		return nil, errors.New("missing required parameter: callback")
	}

	switch hookClass {
	case "onRequest":
		// 检查是否为参数化路由（包含 : 或 *）
		if isParameterizedRoute(hookName) {
			utils.Log(2, fmt.Sprintf("add parameterized request hook: %s -> %s", hookName, callbackName))
			AddParameterizedHook(hookName, callbackName)
		} else {
			utils.Log(2, fmt.Sprintf("add request hook: %s -> %s", hookName, callbackName))
			RequestHookRadixTree, _, _ = RequestHookRadixTree.Insert([]byte(hookName), []byte(callbackName))
		}
	default:
		return nil, fmt.Errorf("unknown hook class: %s (expected: onRequest)", hookClass)
	}

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// plugin interface:
func DeleteHook(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var hookName, hookClass string

	for _, arg := range args {
		switch arg.Name {
		case "name":
			hookName = utils.GetStringSafe(arg.Data)
		case "class":
			hookClass = utils.GetStringSafe(arg.Data)
		}
	}

	// 参数验证
	if hookName == "" {
		return nil, errors.New("missing required parameter: name")
	}
	if hookClass == "" {
		return nil, errors.New("missing required parameter: class")
	}

	switch hookClass {
	case "onRequest":
		// 检查是否为参数化路由
		if isParameterizedRoute(hookName) {
			utils.Log(2, fmt.Sprintf("delete parameterized request hook: %s", hookName))
			RemoveParameterizedHook(hookName)
		} else {
			utils.Log(2, fmt.Sprintf("delete request hook: %s", hookName))
			RequestHookRadixTree, _, _ = RequestHookRadixTree.Delete([]byte(hookName))
		}
	default:
		return nil, fmt.Errorf("unknown hook class: %s (expected: onRequest)", hookClass)
	}

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// AddRouteListener 注册路由监听（仅观测，不拦截），支持 request/response/both
// 参数：
//   - route: 路由模式（支持精确/:param/*wildcard）
//   - callback: 插件回调方法名
//   - phase: request/response/both，默认 both
//   - priority: 优先级（可选），数值越大优先级越高，默认 0
func AddRouteListener(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var route, callback, phaseStr string
	var priority int

	for _, arg := range args {
		switch arg.Name {
		case "route":
			route = utils.GetStringSafe(arg.Data)
		case "callback":
			callback = utils.GetStringSafe(arg.Data)
		case "phase":
			phaseStr = utils.GetStringSafe(arg.Data)
		case "priority":
			priority = utils.GetIntSafe(arg.Data)
		}
	}

	if route == "" {
		return nil, errors.New("missing required parameter: route")
	}
	if callback == "" {
		return nil, errors.New("missing required parameter: callback")
	}

	phase := parseListenPhase(phaseStr)
	registerRouteListenerWithPriority(route, callback, phase, priority)

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// DeleteRouteListener 删除路由监听
// 参数：
//   - route: 路由模式
//   - callback:（可选）指定回调名，仅删除匹配项
func DeleteRouteListener(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var route, callback string

	for _, arg := range args {
		switch arg.Name {
		case "route":
			route = utils.GetStringSafe(arg.Data)
		case "callback":
			callback = utils.GetStringSafe(arg.Data)
		}
	}

	if route == "" {
		return nil, errors.New("missing required parameter: route")
	}

	removeRouteListener(route, callback)
	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// 注意: 类型安全转换函数已移至 utils 包
// 使用: utils.GetStringSafe(), utils.GetBytesSafe(), utils.GetIntSafe()

// ============== 文章操作API ==============

// PluginGetArticle 获取文章内容
// 参数:
//   - article_id: 文章ID
//
// 返回:
//   - success: bool - 是否成功
//   - article: json - 文章内容（JSON格式）
//   - error: string - 错误信息（如果失败）
func PluginGetArticle(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var articleID string

	for _, arg := range args {
		if arg.Name == "article_id" {
			articleID = utils.GetStringSafe(arg.Data)
		}
	}

	if articleID == "" {
		return nil, errors.New("missing required parameter: article_id")
	}

	// 验证ID格式
	if !isValidID(articleID) {
		return nil, errors.New("invalid article_id format")
	}

	// 读取文章文件
	articleJsonPath := "configs/articles/" + articleID + ".json"
	articleData, err := os.ReadFile(articleJsonPath)
	if err != nil {
		return []*plugins.Arg{
			{Name: "success", Type: "bool", Data: false},
			{Name: "error", Type: "string", Data: "article not found"},
		}, nil
	}

	// 解析JSON验证
	var articleJson articleJsonStruct
	err = json.Unmarshal(articleData, &articleJson)
	if err != nil {
		return []*plugins.Arg{
			{Name: "success", Type: "bool", Data: false},
			{Name: "error", Type: "string", Data: "failed to parse article"},
		}, nil
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
		{Name: "article", Type: "json", Data: string(articleData)},
	}, nil
}

// PluginAddArticle 添加新文章
// 参数:
//   - title: string - 文章标题
//   - content: string - 文章内容（Markdown）
//   - content_html: string - 文章HTML内容
//   - author: string - 作者名称
//   - extra_flags: json - 额外标记（可选）
//
// 返回:
//   - success: bool - 是否成功
//   - article_id: string - 新文章ID
//   - error: string - 错误信息（如果失败）
func PluginAddArticle(args []*plugins.Arg) ([]*plugins.Arg, error) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	// 检查只读模式
	if Config.AccessCfg.ReadOnly {
		return nil, errors.New("read-only mode enabled")
	}

	var title, content, contentHTML, author string
	var extraFlags map[string]string

	for _, arg := range args {
		switch arg.Name {
		case "title":
			title = utils.GetStringSafe(arg.Data)
		case "content":
			content = utils.GetStringSafe(arg.Data)
		case "content_html":
			contentHTML = utils.GetStringSafe(arg.Data)
		case "author":
			author = utils.GetStringSafe(arg.Data)
		case "extra_flags":
			jsonData := utils.GetBytesSafe(arg.Data)
			json.Unmarshal(jsonData, &extraFlags)
		}
	}

	if title == "" || content == "" || author == "" {
		return nil, errors.New("missing required parameters: title, content, or author")
	}

	// 生成文章ID
	articleID := generateTraceID()

	// 检查ID唯一性
	articleDir, err := os.ReadDir("configs/articles")
	if err != nil {
		return nil, errors.New("failed to read articles directory")
	}

	articleIDList := make([]string, 0)
	for _, file := range articleDir {
		if !file.IsDir() {
			articleIDList = append(articleIDList, file.Name()[:len(file.Name())-5])
		}
	}

	for _, id := range articleIDList {
		if id == articleID {
			articleID = generateTraceID()
		}
	}

	// 创建文章结构
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	newArticle := articleJsonStruct{
		Title:       title,
		Content:     content,
		ContentHTML: contentHTML,
		Author:      author,
		Pub_Date:    currentTime,
		Edit_Date:   currentTime,
		ExtraFlags:  extraFlags,
		Comments: make([]struct {
			Author     string `json:"author"`
			Email      string `json:"email"`
			Content    string `json:"content"`
			Pub_Date   string `json:"pub_date"`
			ID         string `json:"id"`
			Subscribed bool   `json:"subscribed"`
			ReplyTo    string `json:"reply_to"`
		}, 0),
	}

	// 保存文章
	articleJsonPath := "configs/articles/" + articleID + ".json"
	articleFile, err := os.Create(articleJsonPath)
	if err != nil {
		return nil, errors.New("failed to create article file")
	}
	defer articleFile.Close()

	jsonEncoder := json.NewEncoder(articleFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(newArticle)
	if err != nil {
		os.Remove(articleJsonPath)
		return nil, errors.New("failed to encode article json")
	}

	// 清理缓存
	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			cacheManager.DelCacheItem("/index.html")
		})
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
		{Name: "article_id", Type: "string", Data: articleID},
	}, nil
}

// PluginEditArticle 编辑文章
// 参数:
//   - article_id: string - 文章ID
//   - title: string - 文章标题
//   - content: string - 文章内容（Markdown）
//   - content_html: string - 文章HTML内容
//   - author: string - 作者名称
//   - extra_flags: json - 额外标记（可选）
//
// 返回:
//   - success: bool - 是否成功
//   - error: string - 错误信息（如果失败）
func PluginEditArticle(args []*plugins.Arg) ([]*plugins.Arg, error) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	// 检查只读模式
	if Config.AccessCfg.ReadOnly {
		return nil, errors.New("read-only mode enabled")
	}

	var articleID, title, content, contentHTML, author string
	var extraFlags map[string]string

	for _, arg := range args {
		switch arg.Name {
		case "article_id":
			articleID = utils.GetStringSafe(arg.Data)
		case "title":
			title = utils.GetStringSafe(arg.Data)
		case "content":
			content = utils.GetStringSafe(arg.Data)
		case "content_html":
			contentHTML = utils.GetStringSafe(arg.Data)
		case "author":
			author = utils.GetStringSafe(arg.Data)
		case "extra_flags":
			jsonData := utils.GetBytesSafe(arg.Data)
			json.Unmarshal(jsonData, &extraFlags)
		}
	}

	if articleID == "" {
		return nil, errors.New("missing required parameter: article_id")
	}

	if !isValidID(articleID) {
		return nil, errors.New("invalid article_id format")
	}

	// 读取现有文章
	articleJsonPath := "configs/articles/" + articleID + ".json"
	articleFile, err := os.OpenFile(articleJsonPath, os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.New("article not found")
	}
	defer articleFile.Close()

	var articleJson articleJsonStruct
	err = json.NewDecoder(articleFile).Decode(&articleJson)
	if err != nil {
		return nil, errors.New("failed to decode article json")
	}

	// 更新字段（只更新提供的字段）
	if title != "" {
		articleJson.Title = title
	}
	if content != "" {
		articleJson.Content = content
	}
	if contentHTML != "" {
		articleJson.ContentHTML = contentHTML
	}
	if author != "" {
		articleJson.Author = author
	}
	if extraFlags != nil {
		articleJson.ExtraFlags = extraFlags
	}
	articleJson.Edit_Date = time.Now().Format("2006-01-02 15:04:05")

	// 保存更新
	if _, err := articleFile.Seek(0, 0); err != nil {
		return nil, errors.New("failed to seek article file")
	}
	if err := articleFile.Truncate(0); err != nil {
		return nil, errors.New("failed to truncate article file")
	}

	jsonEncoder := json.NewEncoder(articleFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(articleJson)
	if err != nil {
		return nil, errors.New("failed to encode article json")
	}

	// 清理缓存
	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			cacheManager.DelCacheItem("/articles/" + articleID)
		})
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
	}, nil
}

// PluginDeleteArticle 删除文章
// 参数:
//   - article_id: string - 文章ID
//
// 返回:
//   - success: bool - 是否成功
//   - error: string - 错误信息（如果失败）
func PluginDeleteArticle(args []*plugins.Arg) ([]*plugins.Arg, error) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	// 检查只读模式
	if Config.AccessCfg.ReadOnly {
		return nil, errors.New("read-only mode enabled")
	}

	var articleID string
	for _, arg := range args {
		if arg.Name == "article_id" {
			articleID = utils.GetStringSafe(arg.Data)
		}
	}

	if articleID == "" {
		return nil, errors.New("missing required parameter: article_id")
	}

	if !isValidID(articleID) {
		return nil, errors.New("invalid article_id format")
	}

	// 删除文章文件
	articleJsonPath := "configs/articles/" + articleID + ".json"
	err := os.Remove(articleJsonPath)
	if err != nil {
		return nil, errors.New("failed to delete article file")
	}

	// 清理缓存
	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			cacheManager.DelCacheItem("/articles/" + articleID)
		})
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
	}, nil
}

// PluginGetAllArticleIDs 获取所有文章ID列表
// 返回:
//   - success: bool - 是否成功
//   - article_ids: json - 文章ID列表（JSON数组）
//   - error: string - 错误信息（如果失败）
func PluginGetAllArticleIDs(args []*plugins.Arg) ([]*plugins.Arg, error) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	articleDir, err := os.ReadDir("configs/articles")
	if err != nil {
		return nil, errors.New("failed to read articles directory")
	}

	articleIDs := make([]string, 0)
	for _, file := range articleDir {
		if !file.IsDir() && len(file.Name()) > 5 {
			articleID := file.Name()[:len(file.Name())-5]
			articleIDs = append(articleIDs, articleID)
		}
	}

	articleIDsJSON, err := json.Marshal(articleIDs)
	if err != nil {
		return nil, errors.New("failed to marshal article IDs")
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
		{Name: "article_ids", Type: "json", Data: string(articleIDsJSON)},
	}, nil
}

// ============== Token验证API ==============

// PluginVerifyToken 验证访问令牌
// 参数:
//   - token: string - 要验证的令牌
//
// 返回:
//   - valid: bool - 令牌是否有效
func PluginVerifyToken(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var token string

	for _, arg := range args {
		if arg.Name == "token" {
			token = utils.GetStringSafe(arg.Data)
		}
	}

	if token == "" {
		return []*plugins.Arg{
			{Name: "valid", Type: "bool", Data: false},
		}, nil
	}

	// 验证token（使用统一的鉴权方式，支持登录令牌和时间戳加密令牌）
	isValid := checkToken(token)

	return []*plugins.Arg{
		{Name: "valid", Type: "bool", Data: isValid},
	}, nil
}

// ============== 配置读取API ==============

// PluginGetConfig 读取配置项
// 参数:
//   - key: string - 配置键名（支持点号分隔的路径，如 "server_config.port"）
//
// 返回:
//   - success: bool - 是否成功
//   - value: string - 配置值
//   - error: string - 错误信息（如果失败）
func PluginGetConfig(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var key string

	for _, arg := range args {
		if arg.Name == "key" {
			key = utils.GetStringSafe(arg.Data)
		}
	}

	if key == "" {
		return nil, errors.New("missing required parameter: key")
	}

	// 将配置序列化为JSON
	configJSON, err := json.Marshal(Config)
	if err != nil {
		return nil, errors.New("failed to marshal config")
	}

	// 解析为map以支持路径查询
	var configMap map[string]interface{}
	err = json.Unmarshal(configJSON, &configMap)
	if err != nil {
		return nil, errors.New("failed to unmarshal config")
	}

	// 简单的键查询（支持一级键）
	value, exists := configMap[key]
	if !exists {
		return []*plugins.Arg{
			{Name: "success", Type: "bool", Data: false},
			{Name: "error", Type: "string", Data: "config key not found"},
		}, nil
	}

	// 将值转换为JSON字符串
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return nil, errors.New("failed to marshal config value")
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
		{Name: "value", Type: "json", Data: string(valueJSON)},
	}, nil
}

// ============== 日志API ==============

// PluginLog 记录日志
// 参数:
//   - level: int - 日志级别（0=DEBUG, 1=INFO, 2=WARNING, 3=ERROR）
//   - message: string - 日志消息
//   - plugin_name: string - 插件名称（可选，用于标识日志来源）
//
// 返回:
//   - success: bool - 是否成功
func PluginLog(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var level int
	var message, pluginName string

	for _, arg := range args {
		switch arg.Name {
		case "level":
			level = utils.GetIntSafe(arg.Data)
		case "message":
			message = utils.GetStringSafe(arg.Data)
		case "plugin_name":
			pluginName = utils.GetStringSafe(arg.Data)
		}
	}

	if message == "" {
		return nil, errors.New("missing required parameter: message")
	}

	// 添加插件标识
	logMessage := message
	if pluginName != "" {
		logMessage = fmt.Sprintf("[Plugin:%s] %s", pluginName, message)
	} else {
		logMessage = fmt.Sprintf("[Plugin] %s", message)
	}

	// 记录日志
	utils.Log(level, logMessage)

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
	}, nil
}

// ============== 评论操作API ==============

// PluginGetComments 获取文章的评论列表
// 参数:
//   - article_id: string - 文章ID
//
// 返回:
//   - success: bool - 是否成功
//   - comments: json - 评论列表
//   - error: string - 错误信息
func PluginGetComments(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var articleID string

	for _, arg := range args {
		if arg.Name == "article_id" {
			articleID = utils.GetStringSafe(arg.Data)
		}
	}

	if articleID == "" {
		return nil, errors.New("missing required parameter: article_id")
	}

	if !isValidID(articleID) {
		return nil, errors.New("invalid article_id format")
	}

	articleJsonPath := "configs/articles/" + articleID + ".json"
	articleData, err := os.ReadFile(articleJsonPath)
	if err != nil {
		return []*plugins.Arg{
			{Name: "success", Type: "bool", Data: false},
			{Name: "error", Type: "string", Data: "article not found"},
		}, nil
	}

	var articleJson articleJsonStruct
	err = json.Unmarshal(articleData, &articleJson)
	if err != nil {
		return []*plugins.Arg{
			{Name: "success", Type: "bool", Data: false},
			{Name: "error", Type: "string", Data: "failed to parse article"},
		}, nil
	}

	commentsJSON, err := json.Marshal(articleJson.Comments)
	if err != nil {
		return nil, errors.New("failed to marshal comments")
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
		{Name: "comments", Type: "json", Data: string(commentsJSON)},
	}, nil
}

// PluginAddComment 添加评论
// 参数:
//   - article_id: string - 文章ID
//   - author: string - 作者
//   - email: string - 邮箱（可选）
//   - content: string - 内容
//   - reply_to: string - 回复的评论ID（可选）
//   - subscribed: bool - 是否订阅（可选）
//
// 返回:
//   - success: bool - 是否成功
//   - comment_id: string - 新评论ID
func PluginAddComment(args []*plugins.Arg) ([]*plugins.Arg, error) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	if Config.AccessCfg.ReadOnly {
		return nil, errors.New("read-only mode enabled")
	}

	var articleID, author, email, content, replyTo string
	var subscribed bool

	for _, arg := range args {
		switch arg.Name {
		case "article_id":
			articleID = utils.GetStringSafe(arg.Data)
		case "author":
			author = utils.GetStringSafe(arg.Data)
		case "email":
			email = utils.GetStringSafe(arg.Data)
		case "content":
			content = utils.GetStringSafe(arg.Data)
		case "reply_to":
			replyTo = utils.GetStringSafe(arg.Data)
		case "subscribed":
			if b, ok := arg.Data.(bool); ok {
				subscribed = b
			}
		}
	}

	if articleID == "" || author == "" || content == "" {
		return nil, errors.New("missing required parameters")
	}

	if !isValidID(articleID) {
		return nil, errors.New("invalid article_id format")
	}

	articleJsonPath := "configs/articles/" + articleID + ".json"
	articleFile, err := os.OpenFile(articleJsonPath, os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.New("article not found")
	}
	defer articleFile.Close()

	var articleJson articleJsonStruct
	err = json.NewDecoder(articleFile).Decode(&articleJson)
	if err != nil {
		return nil, errors.New("failed to decode article")
	}

	commentID := generateTraceID()
	for {
		unique := true
		for _, comment := range articleJson.Comments {
			if comment.ID == commentID {
				unique = false
				break
			}
		}
		if unique {
			break
		}
		commentID = generateTraceID()
	}

	articleJson.Comments = append(articleJson.Comments, struct {
		Author     string `json:"author"`
		Email      string `json:"email"`
		Content    string `json:"content"`
		Pub_Date   string `json:"pub_date"`
		ID         string `json:"id"`
		Subscribed bool   `json:"subscribed"`
		ReplyTo    string `json:"reply_to"`
	}{
		Author:     author,
		Email:      email,
		Content:    content,
		ID:         commentID,
		Subscribed: subscribed,
		Pub_Date:   time.Now().Format("2006-01-02 15:04:05"),
		ReplyTo:    replyTo,
	})

	if _, err := articleFile.Seek(0, 0); err != nil {
		return nil, errors.New("failed to seek file")
	}
	if err := articleFile.Truncate(0); err != nil {
		return nil, errors.New("failed to truncate file")
	}

	jsonEncoder := json.NewEncoder(articleFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(articleJson)
	if err != nil {
		return nil, errors.New("failed to encode article")
	}

	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			cacheManager.DelCacheItem("/articles/" + articleID)
		})
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
		{Name: "comment_id", Type: "string", Data: commentID},
	}, nil
}

// PluginDeleteComment 删除评论
// 参数:
//   - article_id: string - 文章ID
//   - comment_id: string - 评论ID
//
// 返回:
//   - success: bool - 是否成功
func PluginDeleteComment(args []*plugins.Arg) ([]*plugins.Arg, error) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	if Config.AccessCfg.ReadOnly {
		return nil, errors.New("read-only mode enabled")
	}

	var articleID, commentID string

	for _, arg := range args {
		switch arg.Name {
		case "article_id":
			articleID = utils.GetStringSafe(arg.Data)
		case "comment_id":
			commentID = utils.GetStringSafe(arg.Data)
		}
	}

	if articleID == "" || commentID == "" {
		return nil, errors.New("missing required parameters")
	}

	if !isValidID(articleID) {
		return nil, errors.New("invalid article_id format")
	}

	articleJsonPath := "configs/articles/" + articleID + ".json"
	articleFile, err := os.OpenFile(articleJsonPath, os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.New("article not found")
	}
	defer articleFile.Close()

	var articleJson articleJsonStruct
	err = json.NewDecoder(articleFile).Decode(&articleJson)
	if err != nil {
		return nil, errors.New("failed to decode article")
	}

	found := false
	for i, comment := range articleJson.Comments {
		if comment.ID == commentID {
			articleJson.Comments = append(articleJson.Comments[:i], articleJson.Comments[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return []*plugins.Arg{
			{Name: "success", Type: "bool", Data: false},
			{Name: "error", Type: "string", Data: "comment not found"},
		}, nil
	}

	if _, err := articleFile.Seek(0, 0); err != nil {
		return nil, errors.New("failed to seek file")
	}
	if err := articleFile.Truncate(0); err != nil {
		return nil, errors.New("failed to truncate file")
	}

	jsonEncoder := json.NewEncoder(articleFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(articleJson)
	if err != nil {
		return nil, errors.New("failed to encode article")
	}

	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			cacheManager.DelCacheItem("/articles/" + articleID)
		})
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
	}, nil
}

// ============== 卡片操作API ==============

// PluginGetAllCards 获取所有卡片
func PluginGetAllCards(args []*plugins.Arg) ([]*plugins.Arg, error) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	type cards struct {
		Cards []map[string]string `json:"cards"`
	}

	cardData, err := os.ReadFile("configs/cards.json")
	if err != nil {
		return nil, errors.New("failed to read cards.json")
	}

	var cardsData cards
	err = json.Unmarshal(cardData, &cardsData)
	if err != nil {
		return nil, errors.New("failed to parse cards.json")
	}

	cardsJSON, err := json.Marshal(cardsData.Cards)
	if err != nil {
		return nil, errors.New("failed to marshal cards")
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
		{Name: "cards", Type: "json", Data: string(cardsJSON)},
	}, nil
}

// PluginGetCard 获取单个卡片
// 参数:
//   - card_id: string - 卡片ID
func PluginGetCard(args []*plugins.Arg) ([]*plugins.Arg, error) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	var cardID string

	for _, arg := range args {
		if arg.Name == "card_id" {
			cardID = utils.GetStringSafe(arg.Data)
		}
	}

	if cardID == "" {
		return nil, errors.New("missing required parameter: card_id")
	}

	type cards struct {
		Cards []map[string]string `json:"cards"`
	}

	cardData, err := os.ReadFile("configs/cards.json")
	if err != nil {
		return nil, errors.New("failed to read cards.json")
	}

	var cardsData cards
	err = json.Unmarshal(cardData, &cardsData)
	if err != nil {
		return nil, errors.New("failed to parse cards.json")
	}

	for _, card := range cardsData.Cards {
		if card["id"] == cardID {
			cardJSON, err := json.Marshal(card)
			if err != nil {
				return nil, errors.New("failed to marshal card")
			}
			return []*plugins.Arg{
				{Name: "success", Type: "bool", Data: true},
				{Name: "card", Type: "json", Data: string(cardJSON)},
			}, nil
		}
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: false},
		{Name: "error", Type: "string", Data: "card not found"},
	}, nil
}

// PluginAddCard 添加卡片
// 参数:
//   - card: json - 卡片数据
func PluginAddCard(args []*plugins.Arg) ([]*plugins.Arg, error) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	if Config.AccessCfg.ReadOnly {
		return nil, errors.New("read-only mode enabled")
	}

	var cardData map[string]string

	for _, arg := range args {
		if arg.Name == "card" {
			jsonData := utils.GetBytesSafe(arg.Data)
			json.Unmarshal(jsonData, &cardData)
		}
	}

	if cardData == nil {
		return nil, errors.New("missing required parameter: card")
	}

	type cards struct {
		Cards []map[string]string `json:"cards"`
	}

	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.New("failed to open cards.json")
	}
	defer cardFile.Close()

	var cardsData cards
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		return nil, errors.New("failed to decode cards.json")
	}

	cardID := generateTraceID()
	for {
		unique := true
		for _, card := range cardsData.Cards {
			if card["id"] == cardID {
				unique = false
				break
			}
		}
		if unique {
			break
		}
		cardID = generateTraceID()
	}

	cardData["id"] = cardID
	cardsData.Cards = append(cardsData.Cards, cardData)

	if _, err := cardFile.Seek(0, 0); err != nil {
		return nil, errors.New("failed to seek file")
	}
	if err := cardFile.Truncate(0); err != nil {
		return nil, errors.New("failed to truncate file")
	}

	jsonEncoder := json.NewEncoder(cardFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(cardsData)
	if err != nil {
		return nil, errors.New("failed to encode cards.json")
	}

	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			cacheManager.DelCacheItem("/index.html")
		})
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
		{Name: "card_id", Type: "string", Data: cardID},
	}, nil
}

// PluginEditCard 编辑卡片
// 参数:
//   - card: json - 卡片数据（必须包含id）
func PluginEditCard(args []*plugins.Arg) ([]*plugins.Arg, error) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	if Config.AccessCfg.ReadOnly {
		return nil, errors.New("read-only mode enabled")
	}

	var cardData map[string]string

	for _, arg := range args {
		if arg.Name == "card" {
			jsonData := utils.GetBytesSafe(arg.Data)
			json.Unmarshal(jsonData, &cardData)
		}
	}

	if cardData == nil || cardData["id"] == "" {
		return nil, errors.New("missing card or card id")
	}

	type cards struct {
		Cards []map[string]string `json:"cards"`
	}

	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.New("failed to open cards.json")
	}
	defer cardFile.Close()

	var cardsData cards
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		return nil, errors.New("failed to decode cards.json")
	}

	found := false
	for i, card := range cardsData.Cards {
		if card["id"] == cardData["id"] {
			cardsData.Cards[i] = cardData
			found = true
			break
		}
	}

	if !found {
		return []*plugins.Arg{
			{Name: "success", Type: "bool", Data: false},
			{Name: "error", Type: "string", Data: "card not found"},
		}, nil
	}

	if _, err := cardFile.Seek(0, 0); err != nil {
		return nil, errors.New("failed to seek file")
	}
	if err := cardFile.Truncate(0); err != nil {
		return nil, errors.New("failed to truncate file")
	}

	jsonEncoder := json.NewEncoder(cardFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(cardsData)
	if err != nil {
		return nil, errors.New("failed to encode cards.json")
	}

	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			cacheManager.DelCacheItem("/index.html")
		})
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
	}, nil
}

// PluginDeleteCard 删除卡片
// 参数:
//   - card_id: string - 卡片ID
func PluginDeleteCard(args []*plugins.Arg) ([]*plugins.Arg, error) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	if Config.AccessCfg.ReadOnly {
		return nil, errors.New("read-only mode enabled")
	}

	var cardID string

	for _, arg := range args {
		if arg.Name == "card_id" {
			cardID = utils.GetStringSafe(arg.Data)
		}
	}

	if cardID == "" {
		return nil, errors.New("missing required parameter: card_id")
	}

	type cards struct {
		Cards []map[string]string `json:"cards"`
	}

	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.New("failed to open cards.json")
	}
	defer cardFile.Close()

	var cardsData cards
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		return nil, errors.New("failed to decode cards.json")
	}

	found := false
	for i, card := range cardsData.Cards {
		if card["id"] == cardID {
			cardsData.Cards = append(cardsData.Cards[:i], cardsData.Cards[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return []*plugins.Arg{
			{Name: "success", Type: "bool", Data: false},
			{Name: "error", Type: "string", Data: "card not found"},
		}, nil
	}

	if _, err := cardFile.Seek(0, 0); err != nil {
		return nil, errors.New("failed to seek file")
	}
	if err := cardFile.Truncate(0); err != nil {
		return nil, errors.New("failed to truncate file")
	}

	jsonEncoder := json.NewEncoder(cardFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(cardsData)
	if err != nil {
		return nil, errors.New("failed to encode cards.json")
	}

	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			cacheManager.DelCacheItem("/index.html")
		})
	}

	return []*plugins.Arg{
		{Name: "success", Type: "bool", Data: true},
	}, nil
}
