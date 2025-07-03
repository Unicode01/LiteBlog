package main

import (
	utils "LiteBlog/utils"
	"LiteBlog/utils/firewall"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

var (
	httpServer         *http.Server
	fireWall           *firewall.Firewall
	cacheManager       *utils.CacheManager
	deliverManager     *utils.DeliverManager
	notifyManager      *utils.NotifyManager
	notifyTriggerMap   = make(map[string]bool)
	pathTraversalRegex = regexp.MustCompile(`(?i)(\.\./|\.\.\\)|(/etc/passwd|/bin/sh|/bin/bash|/\.env)`)
	cardAPILocker      = sync.RWMutex{}
	articleAPILocker   = sync.RWMutex{}
	settingsAPILocker  = sync.RWMutex{}
	LastCommentTime    time.Time
	EncryptTokenKey    string
)

// Init the network manager
// init net proxy
func InitNetManager(config *ServerConfig) error {
	ctx := context.Background()
	// init firewall
	fireWall = firewall.NewFirewall(ctx)
	// build cache manager
	cacheManager = utils.NewCacheManager(Config.CacheCfg.MaxCacheSize, Config.CacheCfg.MaxCacheItems) // 2GB cache, 1 million cache item
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
			Certificates: []tls.Certificate{tlsCert},
			NextProtos:   []string{"h2", "http/1.1"},
			MinVersion:   tls.VersionTLS12,
		}
		httpServer = &http.Server{
			Addr:         net.JoinHostPort(config.Host, fmt.Sprint(config.Port)),
			TLSConfig:    tlsConfig,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
		}
	} else {
		httpServer = &http.Server{
			Addr:         net.JoinHostPort(config.Host, fmt.Sprint(config.Port)),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
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
		Log(1, fmt.Sprintf("HTTP request from %s, traceID: %s, UA: '%s', %s %s, %s, disk_cached=%t", IP, traceID, r.Header.Get("User-Agent"), r.Method, r.URL.Path, response_time, cached))
	}()
	firewallAction, blockReason := fireWall.MatchRule(IP, r)
	if firewallAction == 1 {
		serveError(w, http.StatusForbidden, blockReason)
		return
	}
	if pathTraversalRegex.MatchString(r.URL.Path) { // path traversal
		serveError(w, http.StatusForbidden, "path traversal")
		// add to block list
		fireWall.AddRule(&firewall.Rule{
			Name:    "auto_blocked_by_path_traversal-IP-" + IP,
			Action:  1,
			Rule:    IP,
			Type:    "ipaddr",
			Timeout: time.Now().Add(time.Hour).Unix(), // block for 1 hour
			Reason:  "path traversal",
		})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/") { // redirect to index.html
		http.Redirect(w, r, r.URL.Path+"index.html", http.StatusMovedPermanently)
		return
	}

	// set extra headers
	for k, v := range Config.ServerCfg.ExtraHeaders {
		w.Header().Set(k, v)
	}

	// check public api
	if strings.HasPrefix(r.URL.Path, "/api/v1/") { // public api
		servePublicAPI(w, r)
		return
	}

	// check backend url
	if Config.AccessCfg.EnableBackend {
		backendPrefix := "/" + Config.AccessCfg.BackendPath + "/"
		if strings.HasPrefix(r.URL.Path, backendPrefix) { // backend url
			serveBackend(w, r)
			return
		}
	}

	// check if article file
	if strings.HasPrefix(r.URL.Path, "/articles/") { // article file
		if Config.CacheCfg.UseDisk {
			// check cache
			f, err := cacheManager.GetCacheItem(r.URL.Path)
			if f != nil && err == nil { // hit cache
				cached = true
				w.Header().Set("X-LiteBlog-Disk-Cache", "hit")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				io.Copy(w, f)
				f.Close()
				return
			}
		}
		// get article file
		articleIDHTML := r.URL.Path[len("/articles/"):]
		filebin := renderarticle(articleIDHTML)
		if len(filebin) == 0 {
			serveError(w, http.StatusNotFound, "Article not found")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(filebin)
		// add to cache(using deliverManager to avoid extra delay)
		if Config.CacheCfg.UseDisk {
			deliverManager.AddTask(func() {
				err = cacheManager.AddCacheItem(r.URL.Path, bytes.NewReader(filebin), Config.CacheCfg.ExpireTime)
				if err != nil {
					Log(1, fmt.Sprintf("Failed to add cache item for %s, %s", r.URL.Path, err))
				}
			})
		}
		return
	}
	// render file
	file_ext := path.Ext(r.URL.Path)
	renderList := []string{".js", ".css", ".html", ".xml"}
	// check if file is renderable
	if file_ext == "" || !strings.Contains(strings.Join(renderList, "|"), file_ext) { // not render file
		// directly serve file
		file, err := os.OpenFile("public"+r.URL.Path, os.O_RDONLY, 0) // check file exist
		if err != nil {
			serveError(w, http.StatusNotFound, "File not found")
			return
		}
		// content_type := GetContentType(r.URL.Path)
		// w.Header().Set("Content-Type", content_type)
		// defer file.Close()

		// set cache control
		// w.Header().Set("Cache-Control", "max-age=31536000, public") // 1 year cache
		http.ServeContent(w, r, r.URL.Path, time.Now(), file) // directly serve file
		file.Close()
		// io.Copy(w, file)
		return
	}

	if Config.CacheCfg.UseDisk {
		// check cache
		f, err := cacheManager.GetCacheItem(r.URL.Path)
		if f != nil && err == nil { // hit cache
			cached = true
			w.Header().Set("X-LiteBlog-Disk-Cache", "hit")
			// content_type := GetContentType(r.URL.Path)
			// w.Header().Set("Content-Type", content_type)
			http.ServeContent(w, r, r.URL.Path, time.Now(), f) // directly serve file
			// io.Copy(w, f)
			f.Close()
			return
		}
	}

	// open file to render
	file, err := os.OpenFile("public"+r.URL.Path, os.O_RDONLY, 0) // check file exist
	if err != nil {
		serveError(w, http.StatusNotFound, "File not found")
		return
	}
	defer file.Close()

	// render template
	fileBin, err := io.ReadAll(file)
	if err != nil {
		serveError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}
	fileBin = RenderTemplate(fileBin, nil)
	content_type := GetContentType(r.URL.Path)
	w.Header().Set("Content-Type", content_type)
	w.Header().Set("Content-Length", fmt.Sprint(len(fileBin)))
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
	case "/add_article":
		backendHandler_add_article(w, r)
		return
	case "/edit_article":
		backendHandler_edit_article(w, r)
		return
	case "/get_article":
		backendHandler_get_article(w, r)
		return
	case "/get_all_article_id":
		backendHandler_get_all_article_id(w, r)
		return
	case "/delete_article":
		backendHandler_delete_article(w, r)
		return
	case "/get_card":
		backendHandler_get_card(w, r)
		return
	case "/get_all_cards":
		backendHandler_get_all_cards(w, r)
		return
	case "/edit_card":
		backendHandler_edit_card(w, r)
		return
	case "/delete_comment":
		backendHandler_delete_comment(w, r)
		return
	case "/get_custom_settings":
		backendHandler_get_custom_settings(w, r)
		return
	case "/edit_custom_settings":
		backendHandler_edit_custom_settings(w, r)
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
	Log(1, fmt.Sprintf("Serve error: %d, %s", statusCode, message))
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	jsonDecoder := json.NewDecoder(r.Body)
	type orderrequest struct {
		Token   string `json:"token"`
		Changes []struct {
			ID    string `json:"cardID"`
			Order int    `json:"order"`
		} `json:"changes"`
	}
	var req orderrequest
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	jsonDecoder := json.NewDecoder(r.Body)
	type cardrequest struct {
		Token string `json:"token"`
		ID    string `json:"cardID"`
	}
	var req cardrequest
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type cardrequest struct {
		Token    string            `json:"token"`
		CardJson map[string]string `json:"card"`
	}
	var req cardrequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	jsonDecoder = json.NewDecoder(cardFile)
	err = jsonDecoder.Decode(&cardsData)
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type cardrequest struct {
		Token string `json:"token"`
		ID    string `json:"cardID"`
	}
	var req cardrequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type cardrequest struct {
		Token string `json:"token"`
	}
	var req cardrequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	jsonEncoder.Encode(cardsData.Cards) // no error will be returned as string-string map
}

func backendHandler_edit_card(w http.ResponseWriter, r *http.Request) {
	cardAPILocker.Lock()
	defer cardAPILocker.Unlock()
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type cardrequest struct {
		Token    string            `json:"token"`
		CardJson map[string]string `json:"card"`
	}
	var req cardrequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	jsonDecoder = json.NewDecoder(cardFile)
	err = jsonDecoder.Decode(&cardsData)
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type articlerequest struct {
		Token   string `json:"token"`
		Article struct {
			Title       string            `json:"title"`
			Content     string            `json:"content"`
			ContentHTML string            `json:"content_html"`
			Author      string            `json:"author"`
			ExtraFlags  map[string]string `json:"extra_flags"`
		} `json:"article"`
	}
	var req articlerequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type articlerequest struct {
		Token   string `json:"token"`
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
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	jsonDecoder = json.NewDecoder(articleFile)
	err = jsonDecoder.Decode(&articleJson)
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type articlerequest struct {
		Token string `json:"token"`
		ID    string `json:"article_id"`
	}
	var req articlerequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type articlerequest struct {
		Token string `json:"token"`
		ID    string `json:"article_id"`
	}
	var req articlerequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type articlerequest struct {
		Token string `json:"token"`
	}
	var req articlerequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
		return
	}
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type commentrequest struct {
		Token     string `json:"token"`
		ArticleID string `json:"article_id"`
		CommentID string `json:"comment_id"`
	}
	var req commentrequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
	jsonDecoder = json.NewDecoder(articleFile)
	err = jsonDecoder.Decode(&articleJson)
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type tokenrequest struct {
		Token string `json:"token"`
	}
	var req tokenrequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
		return
	}
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
	if r.Method != "POST" {
		serveError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type customsettingsrequest struct {
		Token          string `json:"token"`
		CustomSettings struct {
			GlobalSettings map[string]string `json:"global_settings"`
			CustomScript   string            `json:"custom_script"`
			CustomStyle    string            `json:"custom_style"`
		} `json:"custom_settings"`
	}
	var req customsettingsrequest
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&req)
	if err != nil {
		serveError(w, http.StatusBadRequest, "Failed to parse request body")
		// fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if !checkToken(req.Token) {
		serveError(w, http.StatusForbidden, "Invalid token")
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
		// NewPolicy() creates a new policy with the default settings.
		p := bluemonday.NewPolicy()
		req.Content = p.Sanitize(req.Content)
		req.Author = p.Sanitize(req.Author)
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
