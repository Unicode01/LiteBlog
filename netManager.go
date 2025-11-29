package main

import (
	utils "LiteBlog/utils"
	"LiteBlog/utils/firewall"
	"LiteBlog/utils/plugins"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	radix "github.com/hashicorp/go-immutable-radix"
	"github.com/microcosm-cc/bluemonday"
)

var (
	httpServer         *http.Server
	fireWall           *firewall.Firewall
	cacheManager       *utils.CacheManager
	deliverManager     *utils.DeliverManager
	notifyManager      *utils.NotifyManager
	snifferManager     *utils.SnifferManager
	notifyTriggerMap   = make(map[string]bool)
	pathTraversalRegex = regexp.MustCompile(`(?i)(\.\./|\.\.\\)|(/etc/passwd|/bin/sh|/bin/bash|/\.env)`)
	cardAPILocker      = sync.RWMutex{}
	articleAPILocker   = sync.RWMutex{}
	settingsAPILocker  = sync.RWMutex{}
	LastCommentTime    time.Time
	EncryptTokenKey    string
	LoginTokens        = make(map[string]struct {
		timeout time.Time
		genOn   time.Time
	}) // key: generatedToken, value: struct{}

	RequestHookRadixTree = radix.New() // hook map for request, when
)

// Init the network manager
// init net proxy
func InitNetManager(config *ServerConfig) error {
	ctx := context.Background()
	// init firewall
	fireWall = firewall.NewFirewall(ctx)
	// build cache manager
	cacheManager = utils.NewCacheManager(Config.CacheCfg.MaxCacheSize, Config.CacheCfg.MaxCacheItems) // 2GB cache, 1 million cache item
	// build sniffer manager
	if Config.SnifferCfg.Enabled {
		snifferManager = utils.NewSnifferManager()
	}
	// build deliver manager
	deliverManager = utils.NewDeliverManager(Config.DeliverCfg.Buffer, Config.DeliverCfg.Threads, context.Background())
	// build notification manager
	if Config.NotifyCfg.Enabled {
		switch Config.NotifyCfg.Type {
		case "smtp":
			notifyManager = utils.NewNotifyManager(
				&utils.NotifyTypeSMTP{
					SmtpServer: Config.NotifyCfg.SMTPConfig.Host,
					SmtpUser:   Config.NotifyCfg.SMTPConfig.UserName,
					SmtpPass:   Config.NotifyCfg.SMTPConfig.Password,
					FromEmail:  Config.NotifyCfg.SMTPConfig.FromAddr,
					ToEmail:    Config.NotifyCfg.SMTPConfig.ToAddrs,
				},
			)
		case "telegrambot":
			notifyManager = utils.NewNotifyManager(
				&utils.NotifyTypeTelegramBot{
					BotToken: Config.NotifyCfg.TelegramBotConfig.Token,
					ChatID:   Config.NotifyCfg.TelegramBotConfig.ChatID,
				},
			)
		}
		for _, trigger := range Config.NotifyCfg.Trigger {
			notifyTriggerMap[trigger] = true
		}
	}
	// init http server
	if config.TlsConfig.Enabled {
		// enable tls
		certificate, err := os.ReadFile(config.TlsConfig.CertFile)
		if err != nil {
			return err
		}
		key, err := os.ReadFile(config.TlsConfig.KeyFile)
		if err != nil {
			return err
		}
		tlsCert, err := tls.X509KeyPair(certificate, key)
		if err != nil {
			return err
		}
		tlsConfig := &tls.Config{
			Certificates:     []tls.Certificate{tlsCert},
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
			},
			SessionTicketsDisabled: true,
			ClientSessionCache:     nil,
			NextProtos:             []string{"h2", "http/1.1"},
			MinVersion:             tls.VersionTLS12,
		}
		httpServer = &http.Server{
			Addr:              net.JoinHostPort(config.Host, fmt.Sprint(config.Port)),
			TLSConfig:         tlsConfig,
			ReadTimeout:       10 * time.Second,
			ReadHeaderTimeout: 2 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       10 * time.Second,
		}
	} else {
		httpServer = &http.Server{
			Addr:              net.JoinHostPort(config.Host, fmt.Sprint(config.Port)),
			ReadTimeout:       10 * time.Second,
			ReadHeaderTimeout: 2 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       10 * time.Second,
		}
	}
	httpServer.Handler = http.HandlerFunc(httpHandler)
	// ensure upload directory exists
	if err := os.MkdirAll("upload", 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}
	// init file manager
	ctxFileManager := context.Background()
	if err := utils.InitFileManager(ctxFileManager); err != nil {
		return fmt.Errorf("failed to initialize file manager: %w", err)
	}
	// start auto render
	go autoRender(context.Background())
	go autoCleanLoginTokens(context.Background())
	// start http server
	var err error
	if config.TlsConfig.Enabled {
		err = httpServer.ListenAndServeTLS(config.TlsConfig.CertFile, config.TlsConfig.KeyFile)
	} else {
		err = httpServer.ListenAndServe()
	}
	return err
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	// 初始化请求上下文
	ctx := initRequestContext(w, r)
	defer ctx.logRequest()

	// 1. 插件钩子检查（最高优先级）
	if Config.PluginCfg.Enabled && handlePluginHook(ctx) {
		return
	}

	// 2. 安全检查
	if !performSecurityChecks(ctx) {
		return
	}

	// 3. 路径预处理
	preprocessPath(r)

	// 4. 设置额外的响应头
	setExtraHeaders(w)

	// 5. 路由分发 - 按优先级顺序
	if handleRoute(ctx) {
		return
	}

	// 6. 默认：静态文件服务
	handleStaticFile(ctx)
}

// requestContext 请求上下文
type requestContext struct {
	w              http.ResponseWriter
	r              *http.Request
	startTime      time.Time
	traceID        string
	ip             string
	cached         bool
	proxyWriter    *utils.ProxyResponseWriter
	originalWriter http.ResponseWriter
}

// initRequestContext 初始化请求上下文
func initRequestContext(w http.ResponseWriter, r *http.Request) *requestContext {
	ctx := &requestContext{
		w:              w,
		r:              r,
		startTime:      time.Now(),
		ip:             getRequestIP(r),
		originalWriter: w,
	}

	// 处理 traceID
	traceIDCookie, err := r.Cookie("traceID")
	if err == nil && traceIDCookie.Value != "" {
		ctx.traceID = traceIDCookie.Value
	} else {
		ctx.traceID = generateTraceID()
		http.SetCookie(w, &http.Cookie{
			Name:    "traceID",
			Value:   ctx.traceID,
			Path:    "/",
			Expires: time.Now().Add(time.Hour * 24), // 1 day
		})
	}

	// 设置 sniffer 代理
	if Config.SnifferCfg.Enabled {
		ctx.proxyWriter = snifferManager.ProxyResponseWriter(w, r)
		ctx.w = ctx.proxyWriter
	}

	return ctx
}

// logRequest 记录请求日志
func (ctx *requestContext) logRequest() {
	responseTime := time.Since(ctx.startTime)
	if ctx.proxyWriter == nil {
		utils.Log(1, fmt.Sprintf("HTTP request from %s, traceID: %s, UA: '%s', %s %s, %s, disk_cached=%t",
			ctx.ip, ctx.traceID, ctx.r.Header.Get("User-Agent"), ctx.r.Method, ctx.r.URL.Path, responseTime, ctx.cached))
	} else {
		utils.Log(1, fmt.Sprintf("HTTP request from %s, traceID: %s, UA: '%s', %s %s %d, %s, disk_cached=%t",
			ctx.ip, ctx.traceID, ctx.r.Header.Get("User-Agent"), ctx.r.Method, ctx.r.URL.Path,
			ctx.proxyWriter.StatusCode(), responseTime, ctx.cached))
	}
}

// handlePluginHook 处理插件钩子
func handlePluginHook(ctx *requestContext) bool {
	o, ok := RequestHookRadixTree.Get([]byte(ctx.r.URL.Path))
	if !ok {
		return false
	}

	f := string(o.([]byte))
	headersBytes, _ := json.Marshal(ctx.r.Header)
	args := []*plugins.Arg{
		{Name: "path", Type: "string", Data: ctx.r.URL.Path},
		{Name: "method", Type: "string", Data: ctx.r.Method},
		{Name: "headers", Type: "json", Data: string(headersBytes)},
		{Name: "ip", Type: "string", Data: ctx.ip},
		{Name: "traceID", Type: "string", Data: ctx.traceID},
	}

	result, err := pluginManager.CallPluginMethod(f, args)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to call plugin method %s, %s", f, err))
		RequestHookRadixTree, _, _ = RequestHookRadixTree.Delete([]byte(ctx.r.URL.Path))
		return false
	}

	// 解析插件响应
	var statusCode int = 200
	var body []byte
	for _, arg := range result {
		switch arg.Name {
		case "statusCode":
			statusCode = getInt_safe(arg.Data)
			if statusCode == 0 {
				utils.Log(3, fmt.Sprintf("Failed to parse plugin status code, %s", getString_safe(arg.Data)))
				statusCode = 500
			}
		case "body":
			body = getBytes_safe(arg.Data)
		case "header":
			headers := make(map[string]string)
			if err := json.Unmarshal(getBytes_safe(arg.Data), &headers); err == nil {
				for k, v := range headers {
					ctx.w.Header().Add(k, v)
				}
			} else {
				utils.Log(3, fmt.Sprintf("Failed to parse plugin header, %s", err))
			}
		}
	}

	ctx.w.WriteHeader(statusCode)
	ctx.w.Write(body)
	return true
}

// performSecurityChecks 执行安全检查
func performSecurityChecks(ctx *requestContext) bool {
	// 防火墙检查
	firewallAction, blockReason := fireWall.MatchRule(ctx.ip, ctx.r)
	if firewallAction == 1 {
		serveError(ctx.w, http.StatusForbidden, blockReason)
		return false
	}

	// 路径遍历检查
	if pathTraversalRegex.MatchString(ctx.r.URL.Path) {
		serveError(ctx.w, http.StatusForbidden, "path traversal")
		fireWall.AddRule(&firewall.Rule{
			Name:    "auto_blocked_by_path_traversal-IP-" + ctx.ip,
			Action:  1,
			Rule:    ctx.ip,
			Type:    "ipaddr",
			Timeout: time.Now().Add(time.Hour).Unix(),
			Reason:  "path traversal",
		})
		return false
	}

	return true
}

// preprocessPath 预处理路径
func preprocessPath(r *http.Request) {
	// 目录路径自动添加 index.html
	if strings.HasSuffix(r.URL.Path, "/") {
		r.URL.Path = r.URL.Path + "index.html"
	}
}

// setExtraHeaders 设置额外的响应头
func setExtraHeaders(w http.ResponseWriter) {
	for k, v := range Config.ServerCfg.ExtraHeaders {
		w.Header().Set(k, v)
	}
}

// handleRoute 路由分发处理
func handleRoute(ctx *requestContext) bool {
	// 路由表 - 按优先级顺序检查
	routes := []struct {
		prefix  string
		enabled bool
		handler func(*requestContext)
	}{
		{"/api/v1/", true, handlePublicAPI},
		{"/" + Config.AccessCfg.BackendPath + "/", Config.AccessCfg.EnableBackend, handleBackendAPI},
		{"/upload/", true, handleUploadedFile},
		{"/articles/", true, handleArticle},
	}

	for _, route := range routes {
		if route.enabled && strings.HasPrefix(ctx.r.URL.Path, route.prefix) {
			route.handler(ctx)
			return true
		}
	}

	return false
}

// handlePublicAPI 处理公共 API
func handlePublicAPI(ctx *requestContext) {
	servePublicAPI(ctx.w, ctx.r)
}

// handleBackendAPI 处理后端 API
func handleBackendAPI(ctx *requestContext) {
	serveBackend(ctx.w, ctx.r)
}

// handleUploadedFile 处理上传文件访问
func handleUploadedFile(ctx *requestContext) {
	serveUploadedFile(ctx.w, ctx.r)
}

// handleArticle 处理文章访问
func handleArticle(ctx *requestContext) {
	// 检查缓存
	if Config.CacheCfg.UseDisk {
		if f, err := cacheManager.GetCacheItem(ctx.r.URL.Path); f != nil && err == nil {
			ctx.cached = true
			ctx.w.Header().Set("X-LiteBlog-Disk-Cache", "hit")
			ctx.w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.Copy(ctx.w, f)
			f.Close()
			return
		}
	}

	// 渲染文章
	articleIDHTML := ctx.r.URL.Path[len("/articles/"):]
	filebin := renderarticle(articleIDHTML)
	if len(filebin) == 0 {
		serveError(ctx.w, http.StatusNotFound, "Article not found")
		return
	}

	ctx.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.w.Write(filebin)

	// 异步添加到缓存
	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			if err := cacheManager.AddCacheItem(ctx.r.URL.Path, bytes.NewReader(filebin), Config.CacheCfg.ExpireTime); err != nil {
				utils.Log(1, fmt.Sprintf("Failed to add cache item for %s, %s", ctx.r.URL.Path, err))
			}
		})
	}
}

// handleStaticFile 处理静态文件
func handleStaticFile(ctx *requestContext) {
	fileExt := path.Ext(ctx.r.URL.Path)
	renderExtensions := []string{".js", ".css", ".html", ".xml"}
	needRender := fileExt != "" && slices.Contains(renderExtensions, fileExt)

	// 非渲染文件直接服务
	if !needRender {
		serveStaticFileDirect(ctx)
		return
	}

	// 检查缓存
	if Config.CacheCfg.UseDisk {
		if f, err := cacheManager.GetCacheItem(ctx.r.URL.Path); f != nil && err == nil {
			ctx.cached = true
			ctx.w.Header().Set("X-LiteBlog-Disk-Cache", "hit")
			http.ServeContent(ctx.w, ctx.r, ctx.r.URL.Path, time.Now(), f)
			f.Close()
			return
		}
	}

	// 读取并渲染文件
	serveRenderedFile(ctx)
}

// serveStaticFileDirect 直接服务静态文件（不渲染）
func serveStaticFileDirect(ctx *requestContext) {
	file, err := os.Open("public" + ctx.r.URL.Path)
	if err != nil {
		serveError(ctx.w, http.StatusNotFound, "File not found")
		return
	}
	defer file.Close()

	http.ServeContent(ctx.w, ctx.r, ctx.r.URL.Path, time.Now(), file)
}

// serveRenderedFile 服务需要渲染的文件
func serveRenderedFile(ctx *requestContext) {
	file, err := os.Open("public" + ctx.r.URL.Path)
	if err != nil {
		serveError(ctx.w, http.StatusNotFound, "File not found")
		return
	}
	defer file.Close()

	// 读取并渲染
	fileBin, err := io.ReadAll(file)
	if err != nil {
		serveError(ctx.w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	fileBin = RenderTemplate(fileBin, nil)
	contentType := GetContentType(ctx.r.URL.Path)
	ctx.w.Header().Set("Content-Type", contentType)
	ctx.w.Header().Set("Content-Length", fmt.Sprint(len(fileBin)))
	ctx.w.Write(fileBin)

	// 异步添加到缓存
	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			if err := cacheManager.AddCacheItem(ctx.r.URL.Path, bytes.NewReader(fileBin), Config.CacheCfg.ExpireTime); err != nil {
				utils.Log(1, fmt.Sprintf("Failed to add cache item for %s, %s", ctx.r.URL.Path, err))
			}
		})
	}
}

// authMiddleware 鉴权中间件：验证请求方法和token
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// 读取请求体
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			serveError(w, http.StatusBadRequest, "Failed to read request body")
			return
		}
		defer r.Body.Close()

		// 解析token
		var tokenReq struct {
			Token string `json:"token"`
		}
		err = json.Unmarshal(bodyBytes, &tokenReq)
		if err != nil {
			serveError(w, http.StatusBadRequest, "Failed to parse request body")
			return
		}

		// 验证token
		if !checkToken(tokenReq.Token) {
			serveError(w, http.StatusForbidden, "Invalid token")
			return
		}

		// 恢复请求体供后续handler使用
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// 调用下一个handler
		next(w, r)
	}
}

// authMiddlewareMultipart 鉴权中间件（用于multipart/form-data请求）：验证请求方法和token
func authMiddlewareMultipart(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// 从 form data 中获取 token
		token := r.FormValue("token")
		if token == "" {
			serveError(w, http.StatusBadRequest, "Token is required")
			return
		}

		// 验证token
		if !checkToken(token) {
			serveError(w, http.StatusForbidden, "Invalid token")
			return
		}

		// 调用下一个handler
		next(w, r)
	}
}

func serveBackend(w http.ResponseWriter, r *http.Request) {
	backendPrefix := "/" + Config.AccessCfg.BackendPath + "/"
	// enter backend
	backendUrl := "/" + r.URL.Path[len(backendPrefix):]
	// fmt.Printf("Enter backend url: %s\n", backendUrl)
	switch backendUrl {
	case "/edit_order":
		authMiddleware(backendHandler_edit_order)(w, r)
		return
	case "/delete_card":
		authMiddleware(backendHandler_delete_card)(w, r)
		return
	case "/add_card":
		authMiddleware(backendHandler_add_card)(w, r)
		return
	case "/add_article":
		authMiddleware(backendHandler_add_article)(w, r)
		return
	case "/edit_article":
		authMiddleware(backendHandler_edit_article)(w, r)
		return
	case "/get_article":
		authMiddleware(backendHandler_get_article)(w, r)
		return
	case "/get_all_article_id":
		authMiddleware(backendHandler_get_all_article_id)(w, r)
		return
	case "/delete_article":
		authMiddleware(backendHandler_delete_article)(w, r)
		return
	case "/get_card":
		authMiddleware(backendHandler_get_card)(w, r)
		return
	case "/get_all_cards":
		authMiddleware(backendHandler_get_all_cards)(w, r)
		return
	case "/edit_card":
		authMiddleware(backendHandler_edit_card)(w, r)
		return
	case "/delete_comment":
		authMiddleware(backendHandler_delete_comment)(w, r)
		return
	case "/get_custom_settings":
		authMiddleware(backendHandler_get_custom_settings)(w, r)
		return
	case "/edit_custom_settings":
		authMiddleware(backendHandler_edit_custom_settings)(w, r)
		return
	case "/upload_file":
		authMiddlewareMultipart(backendHandler_upload_file)(w, r)
		return
	case "/login":
		backendHandler_login(w, r)
		return
	default:
		serveError(w, http.StatusNotFound, "Backend API not found")
		return
	}
}

func servePublicAPI(w http.ResponseWriter, r *http.Request) {
	api_path := r.URL.Path[len("/api/v1"):]
	switch api_path {
	case "/add_comment":
		public_api_add_comment(w, r)
		return
	case Config.SnifferCfg.PublicProvider:
		public_api_get_sniffer_info(w, r)
		return
	default:
		serveError(w, http.StatusNotFound, "API not found")
		return
	}
}

func serveError(w http.ResponseWriter, statusCode int, message string) {
	errorPages := map[int][]byte{
		400: []byte("400 Bad Request"),
		401: []byte("401 Unauthorized"),
		403: []byte("403 Forbidden"),
		404: []byte("404 Not Found"),
		500: []byte("500 Internal Server Error"),
	}
	// open error page
	f, err := os.OpenFile(fmt.Sprintf("public/%d.html", statusCode), os.O_RDONLY, 0)
	if err != nil {
		w.WriteHeader(statusCode)
		w.Write(errorPages[statusCode])
		return
	}
	defer f.Close()
	w.WriteHeader(statusCode)
	io.Copy(w, f)
}

func backendHandler_edit_order(w http.ResponseWriter, r *http.Request) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	type orderrequest struct {
		Changes []struct {
			ID    string `json:"cardID"`
			Order int    `json:"order"`
		} `json:"changes"`
	}
	var req orderrequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// read cards data
	type cards struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardsData cards
	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDWR, 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to open cards.json")
		return
	}
	defer cardFile.Close()
	// decode json
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode cards.json")
		return
	}

	// update order
	cardMap := make(map[string]map[string]string)
	for _, card := range cardsData.Cards {
		cardMap[card["id"]] = card
	}
	for _, change := range req.Changes {
		cardMap[change.ID]["order"] = fmt.Sprint(change.Order) // set new order
		// for i, card := range cardsData.Cards {
		// 	if card["id"] == change.ID {
		// 		cardsData.Cards[i]["order"] = fmt.Sprint(change.Order)
		// 		// fmt.Printf("Update card %s order to %d\n", change.ID, change.Order)
		// 		break
		// 	}
		// }
	}
	// write back
	if _, err := cardFile.Seek(0, 0); err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to seek cards.json")
		return
	}
	if err := cardFile.Truncate(0); err != nil { // directly clear file, will be replace as rename later
		serveError(w, http.StatusInternalServerError, "Failed to truncate cards.json")
		return
	}
	jsonEncoder := json.NewEncoder(cardFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode cards.json")
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/index.html")
		}
	})
}

func backendHandler_delete_card(w http.ResponseWriter, r *http.Request) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	type cardrequest struct {
		ID string `json:"cardID"`
	}
	var req cardrequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// delete card
	type cards struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardsData cards
	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDWR, 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to open cards.json")
		return
	}
	defer cardFile.Close()
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode cards.json")
		return
	}
	newCards := make([]map[string]string, 0)
	for _, card := range cardsData.Cards {
		if card["id"] != req.ID {
			newCards = append(newCards, card)
		}
	}
	cardsData.Cards = newCards
	if _, err := cardFile.Seek(0, 0); err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to seek cards.json")
		return
	}
	if err := cardFile.Truncate(0); err != nil { // directly clear file, will be replace as rename later
		serveError(w, http.StatusInternalServerError, "Failed to truncate cards.json")
		return
	}
	jsonEncoder := json.NewEncoder(cardFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode cards.json")
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/index.html")
		}
	})
}

func backendHandler_add_card(w http.ResponseWriter, r *http.Request) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	type cardrequest struct {
		CardJson map[string]string `json:"card"`
	}
	var req cardrequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	if Config.ContentAdvisorCfg.Enabled && Config.ContentAdvisorCfg.FilterCard {
		// sanitize input, use bluemonday to prevent XSS attack
		// NewPolicy() creates a new policy with the default settings.
		p := bluemonday.NewPolicy()
		for k, v := range req.CardJson {
			req.CardJson[k] = p.Sanitize(v)
		}
	}
	// add card
	type cards struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardsData cards
	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDWR, 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to open cards.json")
		return
	}
	defer cardFile.Close()
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode cards.json")
		return
	}
	newCard := req.CardJson
	newCard["id"] = generateTraceID()
	for {
		isUnique := true
		// 检查整个列表
		for _, card := range cardsData.Cards {
			if card["id"] == newCard["id"] {
				isUnique = false
				break // 发现重复立即跳出
			}
		}

		if isUnique {
			break // 唯一则退出
		}
		// 不唯一时生成新ID
		newCard["id"] = generateTraceID()
	}

	cardsData.Cards = append(cardsData.Cards, newCard)
	if _, err := cardFile.Seek(0, 0); err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to seek cards.json")
		return
	}
	if err := cardFile.Truncate(0); err != nil { // directly clear file, will be replace as rename later
		serveError(w, http.StatusInternalServerError, "Failed to truncate cards.json")
		return
	}
	jsonEncoder := json.NewEncoder(cardFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode cards.json")
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/index.html")
		}
	})
}

func backendHandler_get_card(w http.ResponseWriter, r *http.Request) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	type cardrequest struct {
		ID string `json:"cardID"`
	}
	var req cardrequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// get card
	type cards struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardsData cards
	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDONLY, 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to open cards.json")
		return
	}
	defer cardFile.Close()
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode cards.json")
		return
	}
	for _, card := range cardsData.Cards {
		if card["id"] == req.ID {
			// response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			jsonEncoder := json.NewEncoder(w)
			jsonEncoder.Encode(card) // no error will be returned as string-string map
			return
		}
	}
	serveError(w, http.StatusNotFound, "Card not found")
}

func backendHandler_get_all_cards(w http.ResponseWriter, r *http.Request) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()
	// get card
	type cards struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardsData cards
	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDONLY, 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to open cards.json")
		return
	}
	defer cardFile.Close()
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode cards.json")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	jsonEncoder.Encode(cardsData.Cards) // no error will be returned as string-string map
}

func backendHandler_edit_card(w http.ResponseWriter, r *http.Request) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()

	type cardrequest struct {
		CardJson map[string]string `json:"card"`
	}
	var req cardrequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	if Config.ContentAdvisorCfg.Enabled && Config.ContentAdvisorCfg.FilterCard {
		// sanitize input, use bluemonday to prevent XSS attack
		// NewPolicy() creates a new policy with the default settings.
		p := bluemonday.NewPolicy()
		for k, v := range req.CardJson {
			req.CardJson[k] = p.Sanitize(v)
		}
	}
	// update card
	type cards struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardsData cards
	cardFile, err := os.OpenFile("configs/cards.json", os.O_RDWR, 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to open cards.json")
		return
	}
	defer cardFile.Close()
	err = json.NewDecoder(cardFile).Decode(&cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode cards.json")
		return
	}
	for i, card := range cardsData.Cards {
		if card["id"] == req.CardJson["id"] {
			cardsData.Cards[i] = req.CardJson
			break
		}
	}
	if _, err := cardFile.Seek(0, 0); err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to seek cards.json")
		return
	}
	if err := cardFile.Truncate(0); err != nil { // directly clear file, will be replace as rename later
		serveError(w, http.StatusInternalServerError, "Failed to truncate cards.json")
		return
	}
	jsonEncoder := json.NewEncoder(cardFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(cardsData)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode cards.json")
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/index.html")
		}
	})
}

func backendHandler_add_article(w http.ResponseWriter, r *http.Request) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	type articlerequest struct {
		Article struct {
			Title       string            `json:"title"`
			Content     string            `json:"content"`
			ContentHTML string            `json:"content_html"`
			Author      string            `json:"author"`
			ExtraFlags  map[string]string `json:"extra_flags"`
		} `json:"article"`
	}
	var req articlerequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	if Config.ContentAdvisorCfg.Enabled && Config.ContentAdvisorCfg.FilterArticle {
		// sanitize input, use bluemonday to prevent XSS attack
		// NewPolicy() creates a new policy with the default settings.
		p := bluemonday.NewPolicy()
		pcontent := bluemonday.UGCPolicy()
		req.Article.Title = p.Sanitize(req.Article.Title)
		req.Article.ContentHTML = pcontent.Sanitize(req.Article.ContentHTML)
		req.Article.Author = p.Sanitize(req.Article.Author)
		for k, v := range req.Article.ExtraFlags {
			req.Article.ExtraFlags[k] = p.Sanitize(v)
		}
	}
	// add article
	// generate article id
	articleID := generateTraceID()
	// check if article id is unique
	// get all article ids
	articleDir, err := os.ReadDir("configs/articles")
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to read articles directory")
		return
	}
	articleIDList := make([]string, 0)
	for _, file := range articleDir {
		if !file.IsDir() {
			articleID := file.Name()[:len(file.Name())-5] // remove ".json"
			articleIDList = append(articleIDList, articleID)
		}
	}
	for {
		isUnique := !slices.Contains(articleIDList, articleID) // => contains => true => not unique => false
		if isUnique {
			break // 唯一则退出
		}
		// 不唯一时生成新ID
		articleID = generateTraceID()
	}
	articleJsonPath := "configs/articles/" + articleID + ".json"
	articleJson := articleJsonStruct{
		Title:       req.Article.Title,
		Content:     req.Article.Content,
		ContentHTML: req.Article.ContentHTML,
		Author:      req.Article.Author,
		Edit_Date:   time.Now().Format("2006-01-02 15:04:05"),
		Pub_Date:    time.Now().Format("2006-01-02 15:04:05"),
		ExtraFlags:  req.Article.ExtraFlags,
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
	ArticleFile, err := os.OpenFile(articleJsonPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to create article file")
		return
	}
	defer ArticleFile.Close()
	jsonEncoder := json.NewEncoder(ArticleFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(articleJson)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode article json")
		return
	}
	// response
	type Response struct {
		ArticleID string `json:"article_id"`
	}
	response := Response{
		ArticleID: articleID,
	}
	jsonEncoder = json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonEncoder.Encode(response) // no error will be returned as string
}

func backendHandler_edit_article(w http.ResponseWriter, r *http.Request) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	type articlerequest struct {
		Article struct {
			ID          string            `json:"article_id"`
			Title       string            `json:"title"`
			Content     string            `json:"content"`
			ContentHTML string            `json:"content_html"`
			Author      string            `json:"author"`
			ExtraFlags  map[string]string `json:"extra_flags"`
		} `json:"article"`
	}
	var req articlerequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	if Config.ContentAdvisorCfg.Enabled && Config.ContentAdvisorCfg.FilterArticle {
		// sanitize input, use bluemonday to prevent XSS attack
		// NewPolicy() creates a new policy with the default settings.
		p := bluemonday.NewPolicy()
		pcontent := bluemonday.UGCPolicy()
		req.Article.Title = p.Sanitize(req.Article.Title)
		req.Article.ContentHTML = pcontent.Sanitize(req.Article.ContentHTML)
		req.Article.Author = p.Sanitize(req.Article.Author)
		for k, v := range req.Article.ExtraFlags {
			req.Article.ExtraFlags[k] = p.Sanitize(v)
		}
	}
	// update article
	if isValidID(req.Article.ID) {
		serveError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	articleJsonPath := "configs/articles/" + req.Article.ID + ".json"
	articleFile, err := os.OpenFile(articleJsonPath, os.O_RDWR, 0644)
	if err != nil {
		serveError(w, http.StatusNotFound, "Article not found")
		return
	}
	defer articleFile.Close()
	var articleJson articleJsonStruct
	err = json.NewDecoder(articleFile).Decode(&articleJson)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode article json")
		return
	}
	articleJson.Title = req.Article.Title
	articleJson.Content = req.Article.Content
	articleJson.ContentHTML = req.Article.ContentHTML
	articleJson.Author = req.Article.Author
	articleJson.ExtraFlags = req.Article.ExtraFlags
	articleJson.Edit_Date = time.Now().Format("2006-01-02 15:04:05")
	if _, err := articleFile.Seek(0, 0); err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to seek article file")
		return
	}
	if err := articleFile.Truncate(0); err != nil { // directly clear file, will be replace as rename later
		serveError(w, http.StatusInternalServerError, "Failed to truncate article file")
		return
	}
	jsonEncoder := json.NewEncoder(articleFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(articleJson)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode article json")
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/articles/" + req.Article.ID)
		}
	})
}

func backendHandler_delete_article(w http.ResponseWriter, r *http.Request) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	type articlerequest struct {
		ID string `json:"article_id"`
	}
	var req articlerequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// check if ID is valid
	if isValidID(req.ID) {
		serveError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	// delete article
	articleJsonPath := "configs/articles/" + req.ID + ".json"
	err = os.Remove(articleJsonPath)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to delete article file")
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/articles/" + req.ID)
		}
	})
}

func backendHandler_get_article(w http.ResponseWriter, r *http.Request) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	type articlerequest struct {
		ID string `json:"article_id"`
	}
	var req articlerequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// check if ID is valid
	if isValidID(req.ID) {
		serveError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	// get article
	articleJsonPath := "configs/articles/" + req.ID + ".json"
	articleFile, err := os.OpenFile(articleJsonPath, os.O_RDONLY, 0644)
	if err != nil {
		serveError(w, http.StatusNotFound, "Article not found")
		return
	}
	defer articleFile.Close()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, articleFile)
}

func backendHandler_get_all_article_id(w http.ResponseWriter, r *http.Request) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()
	// get all articles
	articleDir, err := os.ReadDir("configs/articles")
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to read articles directory")
		return
	}
	articleIDs := make([]string, 0)
	for _, file := range articleDir {
		if !file.IsDir() {
			articleID := file.Name()[:len(file.Name())-5] // remove ".json"
			articleIDs = append(articleIDs, articleID)
		}
	}
	jsonEncoder := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	jsonEncoder.Encode(articleIDs) // no error will be returned as []string
}

func backendHandler_delete_comment(w http.ResponseWriter, r *http.Request) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()

	type commentrequest struct {
		ArticleID string `json:"article_id"`
		CommentID string `json:"comment_id"`
	}
	var req commentrequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// check if article ID is valid
	if isValidID(req.ArticleID) {
		serveError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	// delete comment
	articleJsonPath := "configs/articles/" + req.ArticleID + ".json"
	articleFile, err := os.OpenFile(articleJsonPath, os.O_RDWR, 0644)
	if err != nil {
		serveError(w, http.StatusNotFound, "Article not found")
		return
	}
	defer articleFile.Close()
	var articleJson articleJsonStruct
	err = json.NewDecoder(articleFile).Decode(&articleJson)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode article json")
		return
	}
	foundComment := false
	for i, comment := range articleJson.Comments {
		if comment.ID == req.CommentID {
			articleJson.Comments = append(articleJson.Comments[:i], articleJson.Comments[i+1:]...)
			foundComment = true
			break
		}
	}
	if !foundComment {
		serveError(w, http.StatusNotFound, "Comment not found")
		return
	}
	if _, err := articleFile.Seek(0, 0); err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to seek article file")
		return
	}
	if err := articleFile.Truncate(0); err != nil { // directly clear file, will be replace as rename later
		serveError(w, http.StatusInternalServerError, "Failed to truncate article file")
		return
	}
	jsonEncoder := json.NewEncoder(articleFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(articleJson)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode article json")
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/articles/" + req.ArticleID)
		}
	})
}

func backendHandler_get_custom_settings(w http.ResponseWriter, r *http.Request) {
	settingsAPILocker.Lock()
	defer settingsAPILocker.Unlock()
	Output := make(map[string]any)
	// get global settings
	NewMap := make(map[string]any)
	blackList := []string{"cf_site_key", "comment_check_type", "google_site_key"}
	for k, v := range GlobalMap {
		inBlackList := slices.Contains(blackList, k)
		if !inBlackList {
			NewMap[k] = string(v)
		}
	}
	Output["global_settings"] = NewMap
	// set custom settings
	// set custom script field
	script, err := os.ReadFile("public/js/inject.js")
	if err == nil {
		Output["custom_script"] = string(script)
	} else {
		Output["custom_script"] = ""
	}
	// set custom style field
	style, err := os.ReadFile("public/css/customizestyle.css")
	if err == nil {
		Output["custom_style"] = string(style)
	} else {
		Output["custom_style"] = ""
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	err = jsonEncoder.Encode(Output)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
}

func backendHandler_edit_custom_settings(w http.ResponseWriter, r *http.Request) {
	settingsAPILocker.Lock()
	defer settingsAPILocker.Unlock()

	type customsettingsrequest struct {
		CustomSettings struct {
			GlobalSettings map[string]string `json:"global_settings"`
			CustomScript   string            `json:"custom_script"`
			CustomStyle    string            `json:"custom_style"`
		} `json:"custom_settings"`
	}
	var req customsettingsrequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// write to file
	globalFile, err := os.OpenFile("configs/global.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to open global.json")
		return
	}
	defer globalFile.Close()
	if _, err := globalFile.Seek(0, 0); err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to seek global.json")
		return
	}
	if err := globalFile.Truncate(0); err != nil { // directly clear file, will be replace as rename later
		serveError(w, http.StatusInternalServerError, "Failed to truncate global.json")
		return
	}
	jsonEncoder := json.NewEncoder(globalFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(req.CustomSettings.GlobalSettings)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode global settings")
		return
	}
	// update custom script
	err = os.WriteFile("public/js/inject.js", []byte(req.CustomSettings.CustomScript), 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to write custom script")
		return
	}
	// update custom style
	err = os.WriteFile("public/css/customizestyle.css", []byte(req.CustomSettings.CustomStyle), 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to write custom style")
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	// clear the cache
	deliverManager.AddTask(func() {
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/css/customizestyle.css")
			cacheManager.DelCacheItem("/js/inject.js")
		}
	})
}

func backendHandler_login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type loginRequest struct {
		AccessToken string `json:"access_token"`
	}
	var req loginRequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check access token
	if !checkToken(req.AccessToken) {
		serveError(w, http.StatusForbidden, "Invalid access token")
		return
	}
	// generate token
	token := generateTraceID() + generateTraceID() + generateTraceID() + generateTraceID() // 4*16
	LoginTokens[token] = struct {
		timeout time.Time
		genOn   time.Time
	}{
		timeout: time.Now().Add(time.Hour * 3), // 3 hour
		genOn:   time.Now(),
	} // add token to the map
	type loginResponse struct {
		Token   string `json:"token"`
		Timeout int64  `json:"timeout"`
	}
	Output := loginResponse{
		Token:   token,
		Timeout: time.Now().Add(time.Hour * 3).Unix(), // 3 hour
	}
	// response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	err = jsonEncoder.Encode(Output)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
}

func public_api_add_comment(w http.ResponseWriter, r *http.Request) {
	articleAPILocker.Lock()
	defer articleAPILocker.Unlock()
	if !Config.CommentCfg.Enabled {
		serveError(w, http.StatusForbidden, "Comment function is not enabled")
		return
	}
	if LastCommentTime.Add(time.Second * time.Duration(Config.CommentCfg.MinSecondsBetweenComments)).After(time.Now()) { // check if the last comment is too frequent
		serveError(w, http.StatusForbidden, "Too frequent comments")
		return
	}
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type commentRequest struct {
		Verify_token string `json:"verify_token"`
		Article_id   string `json:"article_id"`
		Content      string `json:"content"`
		Author       string `json:"author"`
		Email        string `json:"email"`
		Subscribed   bool   `json:"subscribed"`
		ReplyTo      string `json:"reply_to"`
	}
	var req commentRequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// check if text is empty
	if req.Content == "" {
		serveError(w, http.StatusBadRequest, "Text is empty")
		return
	}
	// check text length
	if Config.CommentCfg.MaxTextLength != 0 && len(req.Content) > Config.CommentCfg.MaxTextLength {
		serveError(w, http.StatusBadRequest, "Text length exceeds the limit")
		return
	}
	// check the email address
	if !isAvailableEmailAddress(req.Email) {
		serveError(w, http.StatusBadRequest, "Invalid email address")
		return
	}
	// check if article id is valid
	if isValidID(req.Article_id) {
		s := "Invalid article ID: " + req.Article_id
		serveError(w, http.StatusBadRequest, s)
		return
	}
	// check if the verify token is correct
	pass := false
	switch Config.CommentCfg.Type {
	case "cloudflare_turnstile":
		pass = CFVerifyCheck(req.Verify_token, getRequestIP(r))
	case "google_recaptcha":
		pass = GoogleVerifyCheck(req.Verify_token, getRequestIP(r))
	}
	if !pass {
		serveError(w, http.StatusForbidden, "Invalid verify token")
		return
	}

	if Config.ContentAdvisorCfg.Enabled && Config.ContentAdvisorCfg.FilterComment {
		// sanitize input, use bluemonday to prevent XSS attack
		// UGCPolicy() creates a policy suitable for user generated content (comments, forum posts, etc.)
		// It allows common HTML tags like <p>, <a>, <strong>, <em>, <ul>, <li>, <code>, <pre>, <blockquote>, etc.
		p := bluemonday.UGCPolicy()
		req.Content = p.Sanitize(req.Content)
		// Use strict policy for non-content fields
		pStrict := bluemonday.StrictPolicy()
		req.ReplyTo = pStrict.Sanitize(req.ReplyTo)
		req.Email = pStrict.Sanitize(req.Email)
		req.Author = pStrict.Sanitize(req.Author)
	}
	// add comment
	articleJsonPath := "configs/articles/" + req.Article_id + ".json"

	articleFile, err := os.OpenFile(articleJsonPath, os.O_RDWR, 0644)
	if err != nil {
		serveError(w, http.StatusNotFound, "Article not found")
		return
	}
	var articleJson articleJsonStruct
	jsonDecoder = json.NewDecoder(articleFile)
	err = jsonDecoder.Decode(&articleJson)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to decode article json")
		return
	}
	commentID := generateTraceID()
	for {
		isUnique := true
		// 检查整个列表

		for _, comment := range articleJson.Comments {
			if comment.ID == commentID {
				isUnique = false
				break // 发现重复立即跳出
			}
		}

		if isUnique {
			break // 唯一则退出
		}
		// 不唯一时生成新ID
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
		Author:     req.Author,
		Email:      req.Email,
		Content:    req.Content,
		ID:         commentID,
		Subscribed: req.Subscribed,
		Pub_Date:   time.Now().Format("2006-01-02 15:04:05"),
		ReplyTo:    req.ReplyTo,
	})
	if _, err := articleFile.Seek(0, 0); err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to seek article file")
		return
	}
	if err := articleFile.Truncate(0); err != nil { // directly clear file, will be replace as rename later
		serveError(w, http.StatusInternalServerError, "Failed to truncate article file")
		return
	}
	jsonEncoder := json.NewEncoder(articleFile)
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(articleJson)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode article json")
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	// set last comment time
	LastCommentTime = time.Now()
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/articles/" + req.Article_id)
		}
	})
	// check trigger
	if Config.NotifyCfg.Enabled {
		if notifyTriggerMap["receive_comment"] {
			deliverManager.AddTask(func() {
				// build message
				message := "Article ID: " + req.Article_id + "\n"
				message += "Article Title: " + articleJson.Title + "\n"
				message += "Author: " + req.Author + "\n"
				message += "Email: " + req.Email + " " + fmt.Sprintf("(Subscribed: %t)", req.Subscribed) + "\n"
				message += "Content: " + req.Content + "\n"
				message += "Reply To: " + req.ReplyTo + "\n"
				message += "Link: " + Config.ServerCfg.URLOrigin + "/articles/" + req.Article_id + "#comment-" + commentID + "\n"
				// send message
				err := notifyManager.Notify("New Comment Received", message)
				if err != nil {
					fmt.Printf("Failed to send notification, %s\n", err)
				}
			})
		}
		if notifyTriggerMap["subscribed_comment_reply"] {
			deliverManager.AddTask(func() {
				// check if the comment is a reply and the author is subscribed
				for _, comment := range articleJson.Comments {
					if comment.ID == req.ReplyTo && comment.Subscribed {
						// build message
						message := "Article ID: " + req.Article_id + "\n"
						message += "Article Title: " + articleJson.Title + "\n"
						message += "Author: " + req.Author + "\n"
						message += "Content: " + req.Content + "\n"
						message += "Reply To: " + req.ReplyTo + "\n"
						message += "Link: " + Config.ServerCfg.URLOrigin + "/articles/" + req.Article_id + "#comment-" + commentID + "\n"
						// send message
						err := notifyManager.Notify("New Comment Reply Received", message)
						if err != nil {
							fmt.Printf("Failed to send notification, %s\n", err)
						}
						break
					}
				}
			})
		}
	}
}

func public_api_get_sniffer_info(w http.ResponseWriter, r *http.Request) {
	if !Config.SnifferCfg.Enabled {
		serveError(w, http.StatusForbidden, "Sniffer function is not enabled")
		return
	}
	if r.Method != "GET" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	ret := make(map[string]any)
	// check params
	params := r.URL.Query()
	path := "/index.html"
	if len(params["path"]) > 0 {
		path = params.Get("path")
	}
	ret["count"] = snifferManager.PathRequestCount(path)
	ret["all_requests"] = snifferManager.AllRequestCount()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	err := jsonEncoder.Encode(ret)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
}

// backendHandler_upload_file 处理文件上传
func backendHandler_upload_file(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 限制上传文件大小为32MB
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)

	// 解析multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	// 获取有效期参数
	expiryDaysStr := r.FormValue("expiry_days")
	if expiryDaysStr == "" {
		expiryDaysStr = "7" // 默认7天
	}

	// 获取上传的文件
	file, header, err := r.FormFile("file")
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to get file from form")
		return
	}
	defer file.Close()

	// 获取原始文件名和扩展名
	originalFilename := header.Filename
	fileExt := filepath.Ext(originalFilename)

	// 生成哈希文件名: hash(filename + timestamp)
	timestamp := time.Now().Format("20060102150405")
	hashInput := originalFilename + timestamp
	hash := sha256.Sum256([]byte(hashInput))
	hashFilename := hex.EncodeToString(hash[:]) + fileExt

	// 检查+创建目标目录
	destDir := "upload"
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		err := os.MkdirAll(destDir, 0755)
		if err != nil {
			utils.Log(3, fmt.Sprintf("Failed to create upload directory: %s", err))
			serveError(w, http.StatusInternalServerError, "Failed to create directory")
			return
		}
	}
	// 创建目标文件
	destPath := filepath.Join(destDir, hashFilename)
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to create upload file: %s", err))
		serveError(w, http.StatusInternalServerError, "Failed to create file")
		return
	}
	defer destFile.Close()

	// 复制文件内容
	written, err := io.Copy(destFile, file)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to write upload file: %s", err))
		serveError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	// 计算过期时间
	var expiryTime int64 = 0 // 0 表示永不过期
	if expiryDaysStr != "never" {
		if days, err := strconv.ParseInt(expiryDaysStr, 10, 64); err == nil && days > 0 {
			expiryTime = time.Now().Add(time.Duration(days) * 24 * time.Hour).Unix()
		}
	}

	// 创建文件元数据
	metadata := utils.FileMetadata{
		Filename:     hashFilename,
		OriginalName: originalFilename,
		Size:         written,
		UploadTime:   time.Now().Unix(),
		ExpiryTime:   expiryTime,
		Hash:         hex.EncodeToString(hash[:]),
	}

	// 保存元数据
	if err := utils.SaveFileMetadata(hashFilename, metadata); err != nil {
		utils.Log(3, fmt.Sprintf("Failed to save metadata for %s: %s", hashFilename, err))
		// 不影响上传，继续
	}

	utils.Log(1, fmt.Sprintf("File uploaded: %s -> %s (%d bytes, expiry: %s)", originalFilename, hashFilename, written, expiryDaysStr))

	// 返回响应
	type UploadResponse struct {
		Filename     string `json:"filename"`
		OriginalName string `json:"original_name"`
		Size         int64  `json:"size"`
		URL          string `json:"url"`
		Hash         string `json:"hash"`
		ExpiryTime   int64  `json:"expiry_time"`
	}

	response := UploadResponse{
		Filename:     hashFilename,
		OriginalName: originalFilename,
		Size:         written,
		URL:          "/upload/" + hashFilename,
		Hash:         hex.EncodeToString(hash[:]),
		ExpiryTime:   expiryTime,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	err = jsonEncoder.Encode(response)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to encode upload response: %s", err))
	}
}

// serveUploadedFile 处理上传文件的公开访问
func serveUploadedFile(w http.ResponseWriter, r *http.Request) {
	// 提取文件名
	filename := r.URL.Path[len("/upload/"):]

	// 安全检查：防止路径遍历
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		serveError(w, http.StatusForbidden, "Invalid filename")
		return
	}

	// 构建文件路径
	filePath := filepath.Join("upload", filename)

	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil || fileInfo.IsDir() {
		serveError(w, http.StatusNotFound, "File not found")
		return
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to open file")
		return
	}
	defer file.Close()

	// 设置内容类型
	contentType := GetContentType(filename)
	w.Header().Set("Content-Type", contentType)

	// 设置缓存控制
	w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year

	// 提供文件内容
	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)
}
