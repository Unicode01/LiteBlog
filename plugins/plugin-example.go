package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcloader "LiteBlog/utils/plugins/gRPCLoader"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// PluginClient gRPC插件客户端
type PluginClient struct {
	conn      *grpc.ClientConn
	client    grpcloader.PluginServiceClient
	id        string
	accessKey string
}

// NewPluginClient 创建插件客户端
func NewPluginClient(serverAddr string, accessKey string) (*PluginClient, error) {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	client := grpcloader.NewPluginServiceClient(conn)
	return &PluginClient{
		conn:      conn,
		client:    client,
		accessKey: accessKey,
	}, nil
}

// Initialize 初始化插件
func (c *PluginClient) Initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if c.accessKey != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "access-key", c.accessKey)
	}

	resp, err := c.client.Initialize(ctx, &grpcloader.Empty{})
	if err != nil {
		return fmt.Errorf("initialize failed: %v", err)
	}

	c.id = resp.Id
	log.Printf("初始化成功 ID: %s, Version: %s", resp.Id, resp.Version)
	return nil
}

func (c *PluginClient) withIDContext(ctx context.Context) context.Context {
	if c.id == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "id", c.id)
}

// RegisterMethods 注册插件方法
func (c *PluginClient) RegisterMethods(methods []*grpcloader.MethodDef) error {
	ctx, cancel := context.WithTimeout(c.withIDContext(context.Background()), 5*time.Second)
	defer cancel()

	resp, err := c.client.RegisterPluginMethods(ctx, &grpcloader.RegisterMethodsRequest{
		Methods: methods,
	})
	if err != nil {
		return err
	}

	if resp != nil && len(resp.Methods) > 0 {
		log.Printf("服务器可用方法: %d 个", len(resp.Methods))
	}
	return nil
}

// CallServerMethod 调用服务器方法
func (c *PluginClient) CallServerMethod(method string, args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	ctx, cancel := context.WithTimeout(c.withIDContext(context.Background()), 10*time.Second)
	defer cancel()

	resp, err := c.client.CallServerMethod(ctx, &grpcloader.CallMethod{
		Method: method,
		Args:   args,
	})
	if err != nil {
		return nil, err
	}
	return resp.Args, nil
}

// NewCommandStream 创建命令流
func (c *PluginClient) NewCommandStream() (grpcloader.PluginService_NewCommandStreamClient, error) {
	ctx := c.withIDContext(context.Background())
	return c.client.NewCommandStream(ctx)
}

// Unload 卸载插件
func (c *PluginClient) Unload() error {
	ctx, cancel := context.WithTimeout(c.withIDContext(context.Background()), 5*time.Second)
	defer cancel()
	_, err := c.client.UnLoad(ctx, &grpcloader.Verify{Id: c.id})
	return err
}

// Close 关闭连接
func (c *PluginClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// RequestInfo 请求信息
type RequestInfo struct {
	Path    string
	Method  string
	Headers map[string][]string
	IP      string
	TraceID string
	Params  map[string]string
	Body    []byte
}

func parseRequestArgs(args []*grpcloader.Arg) *RequestInfo {
	info := &RequestInfo{Params: make(map[string]string)}
	for _, arg := range args {
		switch arg.Name {
		case "path":
			info.Path = string(arg.Arg)
		case "method":
			info.Method = string(arg.Arg)
		case "headers":
			json.Unmarshal(arg.Arg, &info.Headers)
		case "ip":
			info.IP = string(arg.Arg)
		case "traceID":
			info.TraceID = string(arg.Arg)
		case "params":
			json.Unmarshal(arg.Arg, &info.Params)
		case "body":
			info.Body = arg.Arg
		}
	}
	return info
}

func jsonResponse(statusCode int, body []byte) []*grpcloader.Arg {
	return []*grpcloader.Arg{
		{Name: "statusCode", Type: "int", Arg: []byte(fmt.Sprintf("%d", statusCode))},
		{Name: "header", Type: "json", Arg: []byte(`{"Content-Type": "application/json"}`)},
		{Name: "body", Type: "[]byte", Arg: body},
	}
}

func errorResponse(statusCode int, message string) []*grpcloader.Arg {
	response := map[string]interface{}{"success": false, "error": message}
	responseJSON, _ := json.Marshal(response)
	return jsonResponse(statusCode, responseJSON)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("用法: plugin-example <server-address> [access-key]")
	}
	serverAddr := os.Args[1]
	accessKey := ""
	if len(os.Args) >= 3 {
		accessKey = os.Args[2]
	}

	client, err := NewPluginClient(serverAddr, accessKey)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if err := client.Initialize(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	// 注册插件方法
	methods := []*grpcloader.MethodDef{
		{Name: "Plugin.RouteHandler"},   // 路由处理示例
		{Name: "Plugin.ArticleStats"},   // 文章统计
		{Name: "Plugin.CommentManager"}, // 评论管理
		{Name: "Plugin.CardManager"},    // 卡片管理
	}
	if err := client.RegisterMethods(methods); err != nil {
		log.Printf("方法注册失败: %v", err)
	}

	// 创建命令流
	stream, err := client.NewCommandStream()
	if err != nil {
		log.Printf("命令流创建失败: %v", err)
	} else {
		go handleCommandStream(stream, client)
	}

	// 注册路由钩子
	registerHooks(client)

	// 测试 API
	log.Println("\n=== API 测试 ===")
	testAPIs(client)

	log.Println("\n插件运行中，按 Ctrl+C 退出...")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-c

	log.Println("正在关闭...")
	cleanup(client)
}

func registerHooks(client *PluginClient) {
	hooks := []struct {
		path     string
		callback string
		desc     string
	}{
		{"/api/demo/welcome", "Plugin.RouteHandler", "欢迎页面"},
		{"/api/demo/stats", "Plugin.ArticleStats", "文章统计"},
		{"/api/demo/comments/:article_id", "Plugin.CommentManager", "评论管理"},
		{"/api/demo/cards", "Plugin.CardManager", "卡片管理"},
	}

	log.Println("\n注册路由钩子:")
	for _, hook := range hooks {
		args := []*grpcloader.Arg{
			{Name: "class", Type: "string", Arg: []byte("onRequest")},
			{Name: "callback", Type: "string", Arg: []byte(hook.callback)},
			{Name: "name", Type: "string", Arg: []byte(hook.path)},
		}
		if _, err := client.CallServerMethod("AddHook", args); err != nil {
			log.Printf("  ✗ %s - 失败: %v", hook.path, err)
		} else {
			log.Printf("  ✓ %s - %s", hook.path, hook.desc)
		}
	}
}

func testAPIs(client *PluginClient) {
	// 测试文章 API
	log.Println("\n1. 文章 API:")
	result, _ := client.CallServerMethod("GetAllArticleIDs", []*grpcloader.Arg{})
	for _, arg := range result {
		if arg.Name == "article_ids" {
			var ids []string
			json.Unmarshal(arg.Arg, &ids)
			log.Printf("   文章数量: %d", len(ids))
		}
	}

	// 测试评论 API
	log.Println("\n2. 评论 API:")
	result, _ = client.CallServerMethod("GetComments", []*grpcloader.Arg{
		{Name: "article_id", Type: "string", Arg: []byte("firstArticle")},
	})
	for _, arg := range result {
		if arg.Name == "comments" {
			var comments []map[string]interface{}
			json.Unmarshal(arg.Arg, &comments)
			log.Printf("   评论数量: %d", len(comments))
		}
	}

	// 测试卡片 API
	log.Println("\n3. 卡片 API:")
	result, _ = client.CallServerMethod("GetAllCards", []*grpcloader.Arg{})
	for _, arg := range result {
		if arg.Name == "cards" {
			var cards []map[string]string
			json.Unmarshal(arg.Arg, &cards)
			log.Printf("   卡片数量: %d", len(cards))
		}
	}

	// 测试日志 API
	log.Println("\n4. 日志 API:")
	client.CallServerMethod("Log", []*grpcloader.Arg{
		{Name: "level", Type: "int", Arg: []byte("1")},
		{Name: "message", Type: "string", Arg: []byte("插件测试完成")},
		{Name: "plugin_name", Type: "string", Arg: []byte("PluginExample")},
	})
}

func cleanup(client *PluginClient) {
	hooks := []string{
		"/api/demo/welcome",
		"/api/demo/stats",
		"/api/demo/comments/:article_id",
		"/api/demo/cards",
	}

	for _, hook := range hooks {
		args := []*grpcloader.Arg{
			{Name: "class", Type: "string", Arg: []byte("onRequest")},
			{Name: "name", Type: "string", Arg: []byte(hook)},
		}
		client.CallServerMethod("DeleteHook", args)
	}
	client.Unload()
}

// ========== 命令流处理 ==========

func handleCommandStream(stream grpcloader.PluginService_NewCommandStreamClient, client *PluginClient) {
	for {
		cmd, err := stream.Recv()
		if err != nil {
			log.Printf("命令流错误: %v", err)
			return
		}

		var args []*grpcloader.Arg
		var handlerErr error

		switch cmd.Command {
		case "Plugin.RouteHandler":
			args, handlerErr = onRouteHandler(cmd.Args)
		case "Plugin.ArticleStats":
			args, handlerErr = onArticleStats(cmd.Args, client)
		case "Plugin.CommentManager":
			args, handlerErr = onCommentManager(cmd.Args, client)
		case "Plugin.CardManager":
			args, handlerErr = onCardManager(cmd.Args, client)
		case "heartbeat":
			args = []*grpcloader.Arg{
				{Name: "status", Type: "int", Arg: []byte("200")},
			}
		default:
			log.Printf("未知命令: %s", cmd.Command)
			args = errorResponse(400, "未知命令")
		}

		if handlerErr != nil {
			log.Printf("处理 %s 失败: %v", cmd.Command, handlerErr)
			continue
		}

		stream.Send(&grpcloader.Command{
			CommandId: cmd.CommandId,
			Command:   "return",
			Args:      args,
		})
	}
}

// ========== 路由处理器 ==========

func onRouteHandler(args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	info := parseRequestArgs(args)
	log.Printf("[RouteHandler] %s %s from %s", info.Method, info.Path, info.IP)

	response := map[string]interface{}{
		"message": "Welcome to LiteBlog Plugin!",
		"path":    info.Path,
		"method":  info.Method,
	}
	responseJSON, _ := json.Marshal(response)
	return jsonResponse(200, responseJSON), nil
}

// ========== 文章统计 ==========

func onArticleStats(args []*grpcloader.Arg, client *PluginClient) ([]*grpcloader.Arg, error) {
	log.Println("[ArticleStats] 获取文章统计")

	// 获取所有文章
	result, err := client.CallServerMethod("GetAllArticleIDs", []*grpcloader.Arg{})
	if err != nil {
		return errorResponse(500, "获取文章列表失败"), nil
	}

	var articleIDs []string
	for _, arg := range result {
		if arg.Name == "article_ids" {
			json.Unmarshal(arg.Arg, &articleIDs)
		}
	}

	stats := map[string]interface{}{
		"total_articles": len(articleIDs),
		"total_comments": 0,
		"articles":       []map[string]interface{}{},
	}

	// 统计每篇文章
	for _, articleID := range articleIDs {
		articleResult, _ := client.CallServerMethod("GetArticle", []*grpcloader.Arg{
			{Name: "article_id", Type: "string", Arg: []byte(articleID)},
		})

		for _, arg := range articleResult {
			if arg.Name == "article" {
				var article map[string]interface{}
				json.Unmarshal(arg.Arg, &article)

				// 获取评论数
				commentsResult, _ := client.CallServerMethod("GetComments", []*grpcloader.Arg{
					{Name: "article_id", Type: "string", Arg: []byte(articleID)},
				})

				commentCount := 0
				for _, carg := range commentsResult {
					if carg.Name == "comments" {
						var comments []interface{}
						json.Unmarshal(carg.Arg, &comments)
						commentCount = len(comments)
						stats["total_comments"] = stats["total_comments"].(int) + commentCount
					}
				}

				stats["articles"] = append(stats["articles"].([]map[string]interface{}), map[string]interface{}{
					"id":            articleID,
					"title":         article["title"],
					"author":        article["author"],
					"comment_count": commentCount,
				})
			}
		}
	}

	statsJSON, _ := json.Marshal(stats)
	return jsonResponse(200, statsJSON), nil
}

// ========== 评论管理 ==========

func onCommentManager(args []*grpcloader.Arg, client *PluginClient) ([]*grpcloader.Arg, error) {
	info := parseRequestArgs(args)
	articleID := info.Params["article_id"]

	log.Printf("[CommentManager] %s 文章: %s", info.Method, articleID)

	if articleID == "" {
		return errorResponse(400, "缺少文章ID"), nil
	}

	// 获取评论
	result, err := client.CallServerMethod("GetComments", []*grpcloader.Arg{
		{Name: "article_id", Type: "string", Arg: []byte(articleID)},
	})
	if err != nil {
		return errorResponse(500, "获取评论失败"), nil
	}

	var comments []map[string]interface{}
	for _, arg := range result {
		if arg.Name == "comments" {
			json.Unmarshal(arg.Arg, &comments)
		}
	}

	response := map[string]interface{}{
		"success":       true,
		"article_id":    articleID,
		"comment_count": len(comments),
		"comments":      comments,
	}
	responseJSON, _ := json.Marshal(response)
	return jsonResponse(200, responseJSON), nil
}

// ========== 卡片管理 ==========

func onCardManager(args []*grpcloader.Arg, client *PluginClient) ([]*grpcloader.Arg, error) {
	info := parseRequestArgs(args)
	log.Printf("[CardManager] %s", info.Method)

	// 获取所有卡片
	result, err := client.CallServerMethod("GetAllCards", []*grpcloader.Arg{})
	if err != nil {
		return errorResponse(500, "获取卡片失败"), nil
	}

	var cards []map[string]string
	for _, arg := range result {
		if arg.Name == "cards" {
			json.Unmarshal(arg.Arg, &cards)
		}
	}

	// 只返回摘要信息
	cardSummary := []map[string]string{}
	for _, card := range cards {
		cardSummary = append(cardSummary, map[string]string{
			"id":       card["id"],
			"title":    card["card_title"],
			"template": card["template"],
			"order":    card["order"],
		})
	}

	response := map[string]interface{}{
		"success": true,
		"count":   len(cardSummary),
		"cards":   cardSummary,
	}
	responseJSON, _ := json.Marshal(response)
	return jsonResponse(200, responseJSON), nil
}
