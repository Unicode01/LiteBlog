package main

import (
	"LiteBlog/firewall"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

var (
	httpServer         *http.Server
	fireWall           *firewall.Firewall
	cacheManager       *CacheManager
	pathTraversalRegex = regexp.MustCompile(`(?i)(\.\./|\.\.\\)|(/etc/passwd|/bin/sh|/bin/bash)`)
)

// Init the network manager
// init net proxy
func InitNetManager(config *ServerConfig) error {
	// init firewall
	fireWall = firewall.NewFirewall()
	// build cache manager
	cacheManager = NewCacheManager(Config.CacheCfg.MaxCacheSize, Config.CacheCfg.MaxCacheItems) // 2GB cache, 1 million cache item
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
			Certificates: []tls.Certificate{tlsCert},
			NextProtos:   []string{"http/1.1"},
			MinVersion:   tls.VersionTLS12,
		}
		httpServer = &http.Server{
			Addr:      net.JoinHostPort(config.Host, fmt.Sprint(config.Port)),
			TLSConfig: tlsConfig,
		}
	} else {
		httpServer = &http.Server{
			Addr: net.JoinHostPort(config.Host, fmt.Sprint(config.Port)),
		}
	}
	httpServer.Handler = http.HandlerFunc(httpHandler)
	// start auto render
	go autoRender(context.Background())
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
	response_start_time := time.Now()
	// traceID := generateTraceID()
	traceID := ""
	IP := getRequestIP(r)
	traceIDCookie, err := r.Cookie("traceID")
	if err == nil {
		traceID = traceIDCookie.Value
	}
	if traceIDCookie == nil || traceIDCookie.Value == "" {
		traceID = generateTraceID()
		http.SetCookie(w, &http.Cookie{
			Name:    "traceID",
			Value:   traceID,
			Path:    "/",
			Expires: time.Now().Add(time.Hour * 24), // 1 day
		})
	}
	cached := false
	defer func() {
		response_end_time := time.Now()
		response_time := response_end_time.Sub(response_start_time)
		Log(1, fmt.Sprintf("HTTP request from %s, traceID: %s, method: %s %s, %s, cached=%t", IP, traceID, r.Method, r.URL.Path, response_time, cached))
	}()

	if fireWall.MatchRule(IP) == 1 { // block ip
		w.WriteHeader(http.StatusForbidden)
		f, err := os.Open("public/403.html")
		if err != nil {
			w.Write([]byte("403 Forbidden | You have been blocked by the firewall."))
			return
		}
		io.Copy(w, f)
		f.Close()
		return
	}
	if pathTraversalRegex.MatchString(r.URL.Path) { // path traversal
		w.WriteHeader(http.StatusForbidden)
		f, err := os.Open("public/403.html")
		if err != nil {
			w.Write([]byte("403 Forbidden"))
			return
		}
		io.Copy(w, f)
		f.Close()
		// add to block list
		fireWall.AddRule(&firewall.Rule{Action: 1, Rule: IP, Timeout: time.Now().Add(time.Hour).Unix()})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/") { // redirect to index.html
		http.Redirect(w, r, r.URL.Path+"index.html", http.StatusFound)
		return
	}

	// check backend url
	backendPrefix := "/" + Config.AccessCfg.BackendPath + "/"
	if strings.HasPrefix(r.URL.Path, backendPrefix) { // backend url
		serveBackend(w, r)
		return
	}

	// render file
	file_ext := path.Ext(r.URL.Path)
	renderList := []string{".js", ".css", ".html"}

	file, err := os.OpenFile("public"+r.URL.Path, os.O_RDONLY, 0) // check file exist
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		f, err := os.Open("public/404.html")
		if err != nil {
			w.Write([]byte("404 Not Found"))
			return
		}
		io.Copy(w, f)
		f.Close()
		return
	}
	defer file.Close()
	if file_ext == "" || !strings.Contains(strings.Join(renderList, "|"), file_ext) { // not render file
		io.Copy(w, file) // directly serve file
		return
	}
	// check cache
	f, err := cacheManager.GetCacheItem(r.URL.Path)
	if f != nil && err == nil { // hit cache
		cached = true
		w.Header().Set("X-LiteBlog-Cache", "hit")
		content_type := GetContentType(r.URL.Path)
		w.Header().Set("Content-Type", content_type)
		io.Copy(w, f)
		return
	}
	// render template
	fileBin, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	fileBin = RenderTemplate(fileBin, nil)
	content_type := GetContentType(r.URL.Path)
	w.Header().Set("Content-Type", content_type)
	w.Write(fileBin)
	// add to cache
	reader := bytes.NewReader(fileBin)
	err = cacheManager.AddCacheItem(r.URL.Path, reader, Config.CacheCfg.ExpireTime)
	if err != nil {
		Log(1, fmt.Sprintf("Failed to add cache item for %s, %s", r.URL.Path, err))
	}
}

func serveBackend(w http.ResponseWriter, r *http.Request) {
	backendPrefix := "/" + Config.AccessCfg.BackendPath + "/"
	// enter backend
	backendUrl := "/" + r.URL.Path[len(backendPrefix):]
	fmt.Printf("Enter backend url: %s\n", backendUrl)
	backendPath := "backend" + backendUrl
	backendFile, err := os.OpenFile(backendPath, os.O_RDONLY, 0)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		f, err := os.Open("public/404.html")
		if err != nil {
			w.Write([]byte("404 Not Found"))
			return
		}
		io.Copy(w, f)
		f.Close()
		return
	}
	defer backendFile.Close()
	backendBin, err := io.ReadAll(backendFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	backendBin = RenderTemplate(backendBin, nil)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(backendBin)
}

func GetContentType(filename string) string {
	var contentTypeMap = map[string]string{
		".md":       "text/markdown; charset=utf-8",
		".markdown": "text/markdown; charset=utf-8",
		".woff":     "font/woff",
		".woff2":    "font/woff2",
		".svg":      "image/svg+xml",
		".csv":      "text/csv; charset=utf-8",
		".avi":      "video/x-msvideo",
		".ics":      "text/calendar",
	}
	ext := path.Ext(filename)
	ext = strings.ToLower(ext)

	// 检查自定义类型
	if ct, ok := contentTypeMap[ext]; ok {
		return ct
	}

	// 查询标准MIME类型
	ct := mime.TypeByExtension(ext)
	if ct != "" {
		return ct
	}

	// 默认返回二进制流类型
	return "application/octet-stream"
}

func getRequestIP(r *http.Request) string {
	ip := r.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = r.Header.Get("X-Real-Ip")
	}
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	return ip
}

func generateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
