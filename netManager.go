package main

import (
	"LiteBlog/firewall"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
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
	deliverManager     *DeliverManager
	pathTraversalRegex = regexp.MustCompile(`(?i)(\.\./|\.\.\\)|(/etc/passwd|/bin/sh|/bin/bash)`)
)

// Init the network manager
// init net proxy
func InitNetManager(config *ServerConfig) error {
	// init firewall
	fireWall = firewall.NewFirewall()
	// build cache manager
	cacheManager = NewCacheManager(Config.CacheCfg.MaxCacheSize, Config.CacheCfg.MaxCacheItems) // 2GB cache, 1 million cache item
	// build deliver manager
	deliverManager = NewDeliverManager(Config.DeliverCfg.Buffer, Config.DeliverCfg.Threads, context.Background())
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
		Log(1, fmt.Sprintf("HTTP request from %s, traceID: %s, method: %s %s, %s, disk_cached=%t", IP, traceID, r.Method, r.URL.Path, response_time, cached))
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
	// check if file is renderable
	if file_ext == "" || !strings.Contains(strings.Join(renderList, "|"), file_ext) { // not render file
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
		io.Copy(w, file) // directly serve file
		return
	}

	if Config.CacheCfg.UseDisk {
		// check cache
		f, err := cacheManager.GetCacheItem(r.URL.Path)
		if f != nil && err == nil { // hit cache
			cached = true
			w.Header().Set("X-LiteBlog-Disk-Cache", "hit")
			content_type := GetContentType(r.URL.Path)
			w.Header().Set("Content-Type", content_type)
			io.Copy(w, f)
			f.Close()
			return
		}
	}

	// open file to render
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
	// add to cache(using deliverManager to avoid extra delay)
	if Config.CacheCfg.UseDisk {
		deliverManager.AddTask(func() {
			err = cacheManager.AddCacheItem(r.URL.Path, bytes.NewReader(fileBin), Config.CacheCfg.ExpireTime)
			if err != nil {
				Log(1, fmt.Sprintf("Failed to add cache item for %s, %s", r.URL.Path, err))
			}
		})
	}

}

func serveBackend(w http.ResponseWriter, r *http.Request) {
	backendPrefix := "/" + Config.AccessCfg.BackendPath + "/"
	// enter backend
	backendUrl := "/" + r.URL.Path[len(backendPrefix):]
	// fmt.Printf("Enter backend url: %s\n", backendUrl)
	switch backendUrl {
	case "/edit_order":
		backendHandler_edit_order(w, r)
		return
	case "/delete_card":
		backendHandler_delete_card(w, r)
		return
	case "/add_card":
		backendHandler_add_card(w, r)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
		return
	}
}

func backendHandler_edit_order(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type orderrequest struct {
		Token   string `json:"token"`
		Changes []struct {
			ID    string `json:"cardID"`
			Order int    `json:"order"`
		} `json:"changes"`
	}
	var req orderrequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != Config.AccessCfg.AccessToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// update order
	for _, change := range req.Changes {
		// update order
		type cards struct {
			Cards []map[string]string `json:"cards"`
		}
		var cardsData cards
		cardsDataBin, err := os.ReadFile("configs/cards.json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = json.Unmarshal(cardsDataBin, &cardsData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for i, card := range cardsData.Cards {
			if card["id"] == change.ID {
				cardsData.Cards[i]["order"] = fmt.Sprint(change.Order)
				// fmt.Printf("Update card %s order to %d\n", change.ID, change.Order)
				break
			}
		}
		cardsDataBin, err = json.MarshalIndent(cardsData, "", "    ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = os.WriteFile("configs/cards.json", cardsDataBin, 0644)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func backendHandler_delete_card(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type cardrequest struct {
		Token string `json:"token"`
		ID    string `json:"cardID"`
	}
	var req cardrequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != Config.AccessCfg.AccessToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// delete card
	type cards struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardsData cards
	cardsDataBin, err := os.ReadFile("configs/cards.json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(cardsDataBin, &cardsData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	newCards := make([]map[string]string, 0)
	for _, card := range cardsData.Cards {
		if card["id"] != req.ID {
			newCards = append(newCards, card)
		}
	}
	cardsData.Cards = newCards
	cardsDataBin, err = json.MarshalIndent(cardsData, "", "    ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.WriteFile("configs/cards.json", cardsDataBin, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func backendHandler_add_card(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type cardrequest struct {
		Token    string            `json:"token"`
		CardJson map[string]string `json:"card"`
	}
	var req cardrequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != Config.AccessCfg.AccessToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// add card
	type cards struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardsData cards
	cardsDataBin, err := os.ReadFile("configs/cards.json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(cardsDataBin, &cardsData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	newCard := req.CardJson
	newCard["id"] = generateTraceID()
	cardsData.Cards = append(cardsData.Cards, newCard)
	cardsDataBin, err = json.MarshalIndent(cardsData, "", "    ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.WriteFile("configs/cards.json", cardsDataBin, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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
