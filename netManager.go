package main

import (
	utils "LiteBlog/utils"
	"LiteBlog/utils/firewall"
	"LiteBlog/utils/plugins"
	"bufio"
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
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

	RequestHookRadixTree = radix.New() // hook map for request (精确匹配)

	// 参数化路由钩子列表，支持 :param 和 *wildcard
	parameterizedHooks        = make([]*ParameterizedHook, 0)
	parameterizedHooksIndex   = make(map[int][]*ParameterizedHook) // 长度索引：key=段数, value=该段数的钩子列表
	parameterizedHooksLock    sync.RWMutex
	parameterizedHooksVersion uint64 // 版本号，用于检测变化

	// 路由监听列表（仅观察，不拦截）
	routeListeners        = make([]*RouteListener, 0)
	routeListenersMap     = make(map[string][]*RouteListener) // key: pattern, value: 监听器列表
	routeListenersIndex   = make(map[int][]*RouteListener)    // 长度索引：key=段数, value=该段数的监听器列表
	routeListenersLock    sync.RWMutex
	routeListenersVersion uint64 // 使用原子操作的版本号，用于快照机制

	// 路径匹配缓存（LRU缓存）
	hookMatchCache     *utils.LRUCache // 缓存 path -> (*ParameterizedHook, map[string]string)
	listenerMatchCache *utils.LRUCache // 缓存 path -> []*listenerMatch
)

// ParameterizedHook 参数化路由钩子
type ParameterizedHook struct {
	Pattern     string   // 原始模式，如 /api/users/:id
	Callback    string   // 回调方法名
	Segments    []string // 分段后的路径
	ParamIdx    []int    // 参数位置索引（:param）
	WildcardIdx int      // 通配符位置索引（*wildcard），-1 表示无通配符
}

// RouteListener 路由监听（仅观测，不修改响应）
type RouteListener struct {
	Pattern    string
	Callback   string
	Hook       *ParameterizedHook
	Phase      listenPhase
	Priority   int    // 优先级，数值越大优先级越高，默认 0
	paramsJSON []byte // 缓存序列化后的参数，减少重复序列化
}

type listenerMatch struct {
	listener   *RouteListener
	params     map[string]string
	paramsJSON []byte // 预序列化的参数，避免重复序列化
}

type listenPhase uint8

const (
	listenPhaseRequest listenPhase = 1 << iota
	listenPhaseResponse
)

func (p listenPhase) String() string {
	switch p {
	case listenPhaseRequest:
		return "request"
	case listenPhaseResponse:
		return "response"
	case listenPhaseRequest | listenPhaseResponse:
		return "both"
	default:
		return "unknown"
	}
}

func parseListenPhase(v string) listenPhase {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "request":
		return listenPhaseRequest
	case "response":
		return listenPhaseResponse
	case "both", "":
		return listenPhaseRequest | listenPhaseResponse
	default:
		return listenPhaseRequest | listenPhaseResponse
	}
}

// splitPathSegments 统一的路径分段逻辑，避免重复分割带来的开销
func splitPathSegments(path string) []string {
	path = strings.TrimPrefix(path, "/")
	return strings.Split(path, "/")
}

// parseRoutePattern 解析路由模式
func parseRoutePattern(pattern string) *ParameterizedHook {
	hook := &ParameterizedHook{
		Pattern:     pattern,
		Segments:    make([]string, 0),
		ParamIdx:    make([]int, 0),
		WildcardIdx: -1,
	}

	// 去掉开头的 /
	pattern = strings.TrimPrefix(pattern, "/")

	segments := strings.Split(pattern, "/")
	for i, seg := range segments {
		hook.Segments = append(hook.Segments, seg)
		if strings.HasPrefix(seg, ":") {
			hook.ParamIdx = append(hook.ParamIdx, i)
		} else if strings.HasPrefix(seg, "*") {
			hook.WildcardIdx = i
			break // 通配符后面的都忽略
		}
	}

	return hook
}

// matchRouteSegments 在已分段的路径上匹配，减少每次匹配时的字符串分割
func (h *ParameterizedHook) matchRouteSegments(pathSegments []string) (bool, map[string]string) {
	params := make(map[string]string)

	// 如果有通配符，路径段数可以 >= 模式段数
	// 否则必须相等
	if h.WildcardIdx >= 0 {
		if len(pathSegments) < h.WildcardIdx {
			return false, nil
		}
	} else {
		if len(pathSegments) != len(h.Segments) {
			return false, nil
		}
	}

	for i, seg := range h.Segments {
		if i >= len(pathSegments) {
			return false, nil
		}

		if strings.HasPrefix(seg, ":") {
			// 参数段，提取参数值
			paramName := seg[1:]
			params[paramName] = pathSegments[i]
		} else if strings.HasPrefix(seg, "*") {
			// 通配符段，匹配剩余所有路径
			paramName := seg[1:]
			if paramName == "" {
				paramName = "wildcard"
			}
			params[paramName] = strings.Join(pathSegments[i:], "/")
			return true, params
		} else {
			// 精确匹配段
			if seg != pathSegments[i] {
				return false, nil
			}
		}
	}

	return true, params
}

// isParameterizedRoute 检查是否为参数化路由
func isParameterizedRoute(pattern string) bool {
	return strings.Contains(pattern, ":") || strings.Contains(pattern, "*")
}

// AddParameterizedHook 添加参数化路由钩子
func AddParameterizedHook(pattern, callback string) {
	parameterizedHooksLock.Lock()
	defer parameterizedHooksLock.Unlock()

	hook := parseRoutePattern(pattern)
	hook.Callback = callback
	parameterizedHooks = append(parameterizedHooks, hook)

	// 更新长度索引
	segmentCount := len(hook.Segments)
	if hook.WildcardIdx >= 0 {
		// 通配符路由，索引键为通配符位置（最小段数）
		segmentCount = hook.WildcardIdx
	}
	parameterizedHooksIndex[segmentCount] = append(parameterizedHooksIndex[segmentCount], hook)

	// 增加版本号并清空缓存
	atomic.AddUint64(&parameterizedHooksVersion, 1)
	if hookMatchCache != nil {
		hookMatchCache.Clear()
	}
}

// RemoveParameterizedHook 移除参数化路由钩子
func RemoveParameterizedHook(pattern string) {
	parameterizedHooksLock.Lock()
	defer parameterizedHooksLock.Unlock()

	for i, hook := range parameterizedHooks {
		if hook.Pattern == pattern {
			// 从主列表删除
			parameterizedHooks = append(parameterizedHooks[:i], parameterizedHooks[i+1:]...)

			// 从索引中删除
			segmentCount := len(hook.Segments)
			if hook.WildcardIdx >= 0 {
				segmentCount = hook.WildcardIdx
			}
			if hooks, exists := parameterizedHooksIndex[segmentCount]; exists {
				dst := hooks[:0]
				for _, h := range hooks {
					if h.Pattern != pattern {
						dst = append(dst, h)
					}
				}
				if len(dst) == 0 {
					delete(parameterizedHooksIndex, segmentCount)
				} else {
					parameterizedHooksIndex[segmentCount] = dst
				}
			}

			// 增加版本号并清空缓存
			atomic.AddUint64(&parameterizedHooksVersion, 1)
			if hookMatchCache != nil {
				hookMatchCache.Clear()
			}
			return
		}
	}
}

// registerRouteListener 添加路由监听（仅观测，不拦截）
func registerRouteListener(pattern, callback string, phase listenPhase) {
	registerRouteListenerWithPriority(pattern, callback, phase, 0)
}

// registerRouteListenerWithPriority 添加带优先级的路由监听
func registerRouteListenerWithPriority(pattern, callback string, phase listenPhase, priority int) {
	routeListenersLock.Lock()
	defer routeListenersLock.Unlock()

	listener := &RouteListener{
		Pattern:  pattern,
		Callback: callback,
		Hook:     parseRoutePattern(pattern),
		Phase:    phase,
		Priority: priority,
	}

	// 添加到切片
	routeListeners = append(routeListeners, listener)

	// 添加到 map 以加速查找和删除
	routeListenersMap[pattern] = append(routeListenersMap[pattern], listener)

	// 添加到长度索引
	segmentCount := len(listener.Hook.Segments)
	if listener.Hook.WildcardIdx >= 0 {
		segmentCount = listener.Hook.WildcardIdx
	}
	routeListenersIndex[segmentCount] = append(routeListenersIndex[segmentCount], listener)

	// 按优先级排序（优先级高的在前）
	sort.Slice(routeListeners, func(i, j int) bool {
		return routeListeners[i].Priority > routeListeners[j].Priority
	})
	listeners := routeListenersMap[pattern]
	sort.Slice(listeners, func(i, j int) bool {
		return listeners[i].Priority > listeners[j].Priority
	})
	// 索引中的监听器也需要排序
	indexListeners := routeListenersIndex[segmentCount]
	sort.Slice(indexListeners, func(i, j int) bool {
		return indexListeners[i].Priority > indexListeners[j].Priority
	})

	// 增加版本号并清空缓存
	atomic.AddUint64(&routeListenersVersion, 1)
	if listenerMatchCache != nil {
		listenerMatchCache.Clear()
	}
}

// removeRouteListener 删除路由监听
func removeRouteListener(pattern, callback string) {
	routeListenersLock.Lock()
	defer routeListenersLock.Unlock()

	// 先找到要删除的监听器以获取段数（用于更新索引）
	var removedSegmentCounts []int
	for _, l := range routeListeners {
		if l.Pattern == pattern && (callback == "" || l.Callback == callback) {
			segmentCount := len(l.Hook.Segments)
			if l.Hook.WildcardIdx >= 0 {
				segmentCount = l.Hook.WildcardIdx
			}
			removedSegmentCounts = append(removedSegmentCounts, segmentCount)
		}
	}

	// 从 map 中删除
	if listeners, exists := routeListenersMap[pattern]; exists {
		if callback == "" {
			// 删除整个 pattern 的所有监听器
			delete(routeListenersMap, pattern)
		} else {
			// 删除特定 callback 的监听器
			dst := listeners[:0]
			for _, l := range listeners {
				if l.Callback != callback {
					dst = append(dst, l)
				}
			}
			if len(dst) == 0 {
				delete(routeListenersMap, pattern)
			} else {
				routeListenersMap[pattern] = dst
			}
		}
	}

	// 从切片中删除
	dst := routeListeners[:0]
	for _, l := range routeListeners {
		if l.Pattern == pattern && (callback == "" || l.Callback == callback) {
			continue
		}
		dst = append(dst, l)
	}
	routeListeners = dst

	// 从索引中删除
	for _, segmentCount := range removedSegmentCounts {
		if indexListeners, exists := routeListenersIndex[segmentCount]; exists {
			dst := indexListeners[:0]
			for _, l := range indexListeners {
				if l.Pattern == pattern && (callback == "" || l.Callback == callback) {
					continue
				}
				dst = append(dst, l)
			}
			if len(dst) == 0 {
				delete(routeListenersIndex, segmentCount)
			} else {
				routeListenersIndex[segmentCount] = dst
			}
		}
	}

	// 增加版本号并清空缓存
	atomic.AddUint64(&routeListenersVersion, 1)
	if listenerMatchCache != nil {
		listenerMatchCache.Clear()
	}
}

// matchParameterizedHook 匹配参数化路由钩子（带缓存和索引优化）
func matchParameterizedHook(path string) (*ParameterizedHook, map[string]string) {
	// 1. 尝试从缓存获取
	if hookMatchCache != nil {
		if cached := hookMatchCache.Get(path); cached != nil {
			result := cached.(hookMatchResult)
			return result.hook, result.params
		}
	}

	// 2. 快速路径：检查是否有钩子
	if len(parameterizedHooks) == 0 {
		return nil, nil
	}

	parameterizedHooksLock.RLock()
	defer parameterizedHooksLock.RUnlock()

	// 3. 仅做一次分段，避免在每个钩子上重复 Split 带来的开销
	pathSegments := splitPathSegments(path)
	segLen := len(pathSegments)

	// 4. 使用长度索引加速匹配（先精确匹配，再通配符匹配）
	// 精确匹配：段数相同的钩子
	if hooks, exists := parameterizedHooksIndex[segLen]; exists {
		for _, hook := range hooks {
			if hook.WildcardIdx >= 0 {
				continue // 跳过通配符钩子
			}
			if matched, params := hook.matchRouteSegments(pathSegments); matched {
				// 缓存结果
				if hookMatchCache != nil {
					hookMatchCache.Set(path, hookMatchResult{hook, params})
				}
				return hook, params
			}
		}
	}

	// 通配符匹配：段数 <= 路径段数的钩子
	for indexSegCount, hooks := range parameterizedHooksIndex {
		if indexSegCount > segLen {
			continue // 通配符位置超过路径段数，跳过
		}
		for _, hook := range hooks {
			if hook.WildcardIdx < 0 {
				continue // 已在精确匹配中处理
			}
			if matched, params := hook.matchRouteSegments(pathSegments); matched {
				// 缓存结果
				if hookMatchCache != nil {
					hookMatchCache.Set(path, hookMatchResult{hook, params})
				}
				return hook, params
			}
		}
	}

	// 5. 未匹配，缓存空结果
	if hookMatchCache != nil {
		hookMatchCache.Set(path, hookMatchResult{nil, nil})
	}
	return nil, nil
}

// hookMatchResult 缓存结构
type hookMatchResult struct {
	hook   *ParameterizedHook
	params map[string]string
}

// matchRouteListeners 匹配路由监听列表（使用缓存、索引和快照机制）
func matchRouteListeners(path string) []*listenerMatch {
	// 1. 尝试从缓存获取
	if listenerMatchCache != nil {
		if cached := listenerMatchCache.Get(path); cached != nil {
			return cached.([]*listenerMatch)
		}
	}

	// 2. 快速路径：检查是否有监听器
	if len(routeListeners) == 0 {
		return nil
	}

	// 3. 快照机制：快速复制索引，然后释放锁
	routeListenersLock.RLock()
	indexCopy := make(map[int][]*RouteListener)
	for k, v := range routeListenersIndex {
		indexCopy[k] = v
	}
	routeListenersLock.RUnlock()

	// 4. 在锁外进行匹配操作
	pathSegments := splitPathSegments(path)
	segLen := len(pathSegments)
	matches := make([]*listenerMatch, 0)

	// 5. 使用长度索引优化匹配（精确匹配 + 通配符匹配）
	// 精确匹配
	if listeners, exists := indexCopy[segLen]; exists {
		for _, listener := range listeners {
			if listener.Hook.WildcardIdx >= 0 {
				continue // 跳过通配符监听器
			}
			if matched, params := listener.Hook.matchRouteSegments(pathSegments); matched {
				var paramsJSON []byte
				if len(params) > 0 {
					paramsJSON, _ = json.Marshal(params)
				}
				matches = append(matches, &listenerMatch{
					listener:   listener,
					params:     params,
					paramsJSON: paramsJSON,
				})
			}
		}
	}

	// 通配符匹配
	for indexSegCount, listeners := range indexCopy {
		if indexSegCount > segLen {
			continue
		}
		for _, listener := range listeners {
			if listener.Hook.WildcardIdx < 0 {
				continue // 已在精确匹配中处理
			}
			if matched, params := listener.Hook.matchRouteSegments(pathSegments); matched {
				var paramsJSON []byte
				if len(params) > 0 {
					paramsJSON, _ = json.Marshal(params)
				}
				matches = append(matches, &listenerMatch{
					listener:   listener,
					params:     params,
					paramsJSON: paramsJSON,
				})
			}
		}
	}

	// 6. 缓存结果
	if listenerMatchCache != nil {
		listenerMatchCache.Set(path, matches)
	}

	return matches
}

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

	// 初始化路径匹配缓存（LRU，容量1000，热点路径加速）
	hookMatchCache = utils.NewLRUCache(1000)
	listenerMatchCache = utils.NewLRUCache(1000)

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

	// 预先匹配并准备路由监听（仅观测，不拦截）
	setupRouteListeners(ctx)
	if len(ctx.listeners) > 0 {
		// 响应阶段的监听必须在请求结束后执行
		defer dispatchRouteListeners(ctx, listenPhaseResponse)
		// 请求阶段立即触发
		dispatchRouteListeners(ctx, listenPhaseRequest)
	}

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
	listeners      []*listenerMatch
	requestBody    []byte
	listenWriter   *listenResponseWriter
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

// setupRouteListeners 根据路由匹配监听器，准备请求/响应捕获
func setupRouteListeners(ctx *requestContext) {
	if !Config.PluginCfg.Enabled {
		return
	}

	matches := matchRouteListeners(ctx.r.URL.Path)
	if len(matches) == 0 {
		return
	}

	// 检查是否需要捕获请求体
	captureRequestBody := Config.PluginCfg.RouteListenerConfig.CaptureRequestBody
	maxRequestBodySize := Config.PluginCfg.RouteListenerConfig.MaxRequestBodySize
	if maxRequestBodySize == 0 {
		maxRequestBodySize = 10 * 1024 * 1024 // 默认 10MB
	}

	// 按需捕获请求体
	if captureRequestBody {
		// 使用 LimitReader 限制读取大小，防止内存溢出
		limitedReader := io.LimitReader(ctx.r.Body, maxRequestBodySize+1) // +1 用于检测是否超限
		bodyBytes, err := io.ReadAll(limitedReader)
		ctx.r.Body.Close()

		if err == nil && int64(len(bodyBytes)) <= maxRequestBodySize {
			// 只有在没有超过限制时才缓存
			ctx.r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			ctx.requestBody = bodyBytes
		} else {
			// 超过限制或读取错误，不缓存请求体
			ctx.r.Body = io.NopCloser(bytes.NewReader(nil))
			if err != nil {
				utils.Log(2, fmt.Sprintf("Failed to read request body: %v", err))
			} else {
				utils.Log(2, fmt.Sprintf("Request body too large (>%d bytes), skipped capture", maxRequestBodySize))
			}
		}
	} else {
		// 不捕获请求体时，仍需关闭原 Body
		ctx.r.Body = io.NopCloser(bytes.NewReader(nil))
	}

	// 检查是否需要捕获响应体
	captureResponseBody := Config.PluginCfg.RouteListenerConfig.CaptureResponseBody
	maxResponseBodySize := Config.PluginCfg.RouteListenerConfig.MaxResponseBodySize
	if maxResponseBodySize == 0 {
		maxResponseBodySize = 10 * 1024 * 1024 // 默认 10MB
	}

	// 包装响应写入器以捕获响应信息
	lrw := newListenResponseWriter(ctx.w, captureResponseBody, maxResponseBodySize)
	ctx.listenWriter = lrw
	ctx.w = lrw

	ctx.listeners = matches
}

// listenResponseWriter 用于捕获响应头/体/状态码
type listenResponseWriter struct {
	http.ResponseWriter
	statusCode     int
	body           bytes.Buffer
	captureBody    bool  // 是否捕获响应体
	maxBodySize    int64 // 最大响应体大小
	bodySize       int64 // 已写入的响应体大小
	bodySizeExceed bool  // 响应体是否超过限制
}

func newListenResponseWriter(w http.ResponseWriter, captureBody bool, maxBodySize int64) *listenResponseWriter {
	return &listenResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		captureBody:    captureBody,
		maxBodySize:    maxBodySize,
	}
}

func (lrw *listenResponseWriter) Header() http.Header {
	return lrw.ResponseWriter.Header()
}

func (lrw *listenResponseWriter) WriteHeader(statusCode int) {
	lrw.statusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func (lrw *listenResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}

	// 只在需要捕获且未超限时写入缓冲区
	if lrw.captureBody && !lrw.bodySizeExceed {
		newSize := lrw.bodySize + int64(len(b))
		if newSize <= lrw.maxBodySize {
			lrw.body.Write(b)
			lrw.bodySize = newSize
		} else {
			// 超过限制，记录标记并清空已缓存的内容以释放内存
			lrw.bodySizeExceed = true
			lrw.body.Reset()
			utils.Log(2, fmt.Sprintf("Response body exceeded limit (%d bytes), stopped capturing", lrw.maxBodySize))
		}
	}

	return lrw.ResponseWriter.Write(b)
}

func (lrw *listenResponseWriter) StatusCode() int {
	if lrw.statusCode == 0 {
		return http.StatusOK
	}
	return lrw.statusCode
}

func (lrw *listenResponseWriter) BodyBytes() []byte {
	if lrw.bodySizeExceed {
		return nil // 超过限制时返回 nil
	}
	return lrw.body.Bytes()
}

func (lrw *listenResponseWriter) HeaderClone() map[string][]string {
	dst := make(map[string][]string)
	for k, v := range lrw.Header() {
		cp := make([]string, len(v))
		copy(cp, v)
		dst[k] = cp
	}
	return dst
}

// Hijack 透传
func (lrw *listenResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("hijacker not supported")
}

// Flush 透传
func (lrw *listenResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Push 透传
func (lrw *listenResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := lrw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// dispatchRouteListeners 触发监听回调（优化：减少重复序列化）
func dispatchRouteListeners(ctx *requestContext, phase listenPhase) {
	if len(ctx.listeners) == 0 {
		return
	}

	// 预序列化通用数据，避免在循环中重复序列化
	headersBytes, _ := json.Marshal(ctx.r.Header)
	phaseStr := phase.String()

	var respHeadersBytes []byte
	var respBody []byte
	statusCode := 0

	if phase&listenPhaseResponse != 0 && ctx.listenWriter != nil {
		statusCode = ctx.listenWriter.StatusCode()
		respHeadersBytes, _ = json.Marshal(ctx.listenWriter.HeaderClone())
		respBody = ctx.listenWriter.BodyBytes()
	}

	for _, match := range ctx.listeners {
		if match.listener.Phase&phase == 0 {
			continue
		}

		args := []*plugins.Arg{
			{Name: "path", Type: "string", Data: ctx.r.URL.Path},
			{Name: "method", Type: "string", Data: ctx.r.Method},
			{Name: "headers", Type: "json", Data: string(headersBytes)},
			{Name: "body", Type: "[]byte", Data: ctx.requestBody},
			{Name: "traceID", Type: "string", Data: ctx.traceID},
			{Name: "phase", Type: "string", Data: phaseStr},
		}

		// 使用预序列化的参数，避免重复序列化
		if len(match.paramsJSON) > 0 {
			args = append(args, &plugins.Arg{
				Name: "params",
				Type: "json",
				Data: string(match.paramsJSON),
			})
		}

		if phase&listenPhaseResponse != 0 {
			args = append(args,
				&plugins.Arg{Name: "statusCode", Type: "int", Data: statusCode},
				&plugins.Arg{Name: "responseHeaders", Type: "json", Data: string(respHeadersBytes)},
				&plugins.Arg{Name: "responseBody", Type: "[]byte", Data: respBody},
			)
		}

		if _, err := pluginManager.CallPluginMethod(match.listener.Callback, args); err != nil {
			utils.Log(3, fmt.Sprintf("Failed to call listener %s for %s: %v", match.listener.Callback, ctx.r.URL.Path, err))
		}
	}
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
	var callbackName string
	var routeParams map[string]string

	// 1. 先尝试精确匹配
	o, ok := RequestHookRadixTree.Get([]byte(ctx.r.URL.Path))
	if ok {
		callbackName = string(o.([]byte))
	} else {
		// 2. 再尝试参数化路由匹配
		hook, params := matchParameterizedHook(ctx.r.URL.Path)
		if hook == nil {
			return false
		}
		callbackName = hook.Callback
		routeParams = params
	}

	headersBytes, _ := json.Marshal(ctx.r.Header)
	args := []*plugins.Arg{
		{Name: "path", Type: "string", Data: ctx.r.URL.Path},
		{Name: "method", Type: "string", Data: ctx.r.Method},
		{Name: "headers", Type: "json", Data: string(headersBytes)},
		{Name: "ip", Type: "string", Data: ctx.ip},
		{Name: "traceID", Type: "string", Data: ctx.traceID},
	}

	// 添加路由参数
	if len(routeParams) > 0 {
		paramsBytes, _ := json.Marshal(routeParams)
		args = append(args, &plugins.Arg{
			Name: "params",
			Type: "json",
			Data: string(paramsBytes),
		})
	}

	result, err := pluginManager.CallPluginMethod(callbackName, args)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to call plugin method %s, %s", callbackName, err))
		// 仅对精确匹配的路由进行删除
		if routeParams == nil {
			RequestHookRadixTree, _, _ = RequestHookRadixTree.Delete([]byte(ctx.r.URL.Path))
		}
		return false
	}

	// 解析插件响应
	var statusCode int = 200
	var body []byte
	for _, arg := range result {
		switch arg.Name {
		case "statusCode":
			statusCode = utils.GetIntSafe(arg.Data)
			if statusCode == 0 {
				utils.Log(3, fmt.Sprintf("Failed to parse plugin status code, %s", utils.GetStringSafe(arg.Data)))
				statusCode = 500
			}
		case "body":
			body = utils.GetBytesSafe(arg.Data)
		case "header":
			headers := make(map[string]string)
			if err := json.Unmarshal(utils.GetBytesSafe(arg.Data), &headers); err == nil {
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
	GlobalMapLocker.RLock()
	scriptPathByte, ok := GlobalMap["CustomScript"]
	GlobalMapLocker.RUnlock()
	if !ok || scriptPathByte == nil {
		scriptPathByte = []byte("public/js/inject.js")
	}
	scriptPath := string(scriptPathByte)
	script, err := os.ReadFile(scriptPath)
	if err == nil {
		Output["custom_script"] = string(script)
	} else {
		Output["custom_script"] = ""
	}
	// set custom style field
	GlobalMapLocker.RLock()
	stylePathByte, ok := GlobalMap["CustomStyle"]
	GlobalMapLocker.RUnlock()
	if !ok || stylePathByte == nil {
		stylePathByte = []byte("public/css/customizestyle.css")
	}
	stylePath := string(stylePathByte)
	style, err := os.ReadFile(stylePath)
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
	GlobalMapLocker.RLock()
	scriptPathByte, ok := GlobalMap["CustomScript"]
	GlobalMapLocker.RUnlock()
	if !ok || scriptPathByte == nil {
		scriptPathByte = []byte("public/js/inject.js")
	}
	scriptPath := string(scriptPathByte)
	err = os.WriteFile(scriptPath, []byte(req.CustomSettings.CustomScript), 0644)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to write custom script")
		return
	}
	// update custom style
	GlobalMapLocker.RLock()
	stylePathByte, ok := GlobalMap["CustomStyle"]
	GlobalMapLocker.RUnlock()
	if !ok || stylePathByte == nil {
		stylePathByte = []byte("public/css/customizestyle.css")
	}
	stylePath := string(stylePathByte)
	err = os.WriteFile(stylePath, []byte(req.CustomSettings.CustomStyle), 0644)
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
			cacheManager.DelCacheItem(stylePath)
			cacheManager.DelCacheItem(scriptPath)
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
