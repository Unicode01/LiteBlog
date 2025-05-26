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
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

var (
	httpServer         *http.Server
	fireWall           *firewall.Firewall
	cacheManager       *CacheManager
	deliverManager     *DeliverManager
	pathTraversalRegex = regexp.MustCompile(`(?i)(\.\./|\.\.\\)|(/etc/passwd|/bin/sh|/bin/bash)`)
	LastCommentTime    time.Time
	EncryptToken       string
)

type articleJsonStruct struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentHTML string `json:"content_html"`
	Author      string `json:"author"`
	Edit_Date   string `json:"edit_date"`
	Pub_Date    string `json:"pub_date"`
	Comments    []struct {
		Author   string `json:"author"`
		Content  string `json:"content"`
		Pub_Date string `json:"pub_date"`
		ID       string `json:"id"`
	} `json:"comments"`
}

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
		Log(1, fmt.Sprintf("HTTP request from %s, traceID: %s, UA: '%s', %s %s, %s, disk_cached=%t", IP, traceID, r.Header.Get("User-Agent"), r.Method, r.URL.Path, response_time, cached))
	}()

	if fireWall.MatchRule(IP, r) == 1 {
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
		fireWall.AddRule(&firewall.Rule{Action: 1, Rule: IP, Type: "ipaddr", Timeout: time.Now().Add(time.Hour).Unix()})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/") { // redirect to index.html
		http.Redirect(w, r, r.URL.Path+"index.html", http.StatusMovedPermanently)
		return
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
	case "/add_article":
		backendHandler_add_article(w, r)
		return
	case "/edit_article":
		backendHandler_edit_article(w, r)
		return
	case "/get_article":
		backendHandler_get_article(w, r)
		return
	case "/delete_article":
		backendHandler_delete_article(w, r)
		return
	case "/get_card":
		backendHandler_get_card(w, r)
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
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
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
	if req.Token != EncryptToken {
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
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/index.html")
		}
	})
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
	if req.Token != EncryptToken {
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
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/index.html")
		}
	})
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
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
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
	deliverManager.AddTask(func() {
		// clear the cache
		if Config.CacheCfg.UseDisk {
			cacheManager.DelCacheItem("/index.html")
		}
	})
}

func backendHandler_get_card(w http.ResponseWriter, r *http.Request) {
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
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// get card
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
	for _, card := range cardsData.Cards {
		if card["id"] == req.ID {
			cardBin, err := json.Marshal(card)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(cardBin)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Card not found"))
}

func backendHandler_edit_card(w http.ResponseWriter, r *http.Request) {
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
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
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
		if card["id"] == req.CardJson["id"] {
			cardsData.Cards[i] = req.CardJson
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
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type articlerequest struct {
		Token   string `json:"token"`
		Article struct {
			Title       string `json:"title"`
			Content     string `json:"content"`
			ContentHTML string `json:"content_html"`
			Author      string `json:"author"`
		} `json:"article"`
	}
	var req articlerequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
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
	}
	// add article
	// generate article id
	articleID := generateTraceID()
	articleJsonPath := "configs/articles/" + articleID + ".json"
	articleJson := articleJsonStruct{
		Title:       req.Article.Title,
		Content:     req.Article.Content,
		ContentHTML: req.Article.ContentHTML,
		Author:      req.Article.Author,
		Edit_Date:   time.Now().Format("2006-01-02 15:04:05"),
		Pub_Date:    time.Now().Format("2006-01-02 15:04:05"),
		Comments: make([]struct {
			Author   string `json:"author"`
			Content  string `json:"content"`
			Pub_Date string `json:"pub_date"`
			ID       string `json:"id"`
		}, 0),
	}
	articleJsonBin, err := json.MarshalIndent(articleJson, "", "    ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.WriteFile(articleJsonPath, articleJsonBin, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// response
	type Response struct {
		ArticleID string `json:"article_id"`
	}
	response := Response{
		ArticleID: articleID,
	}
	responseBin, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBin)

}

func backendHandler_edit_article(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type articlerequest struct {
		Token   string `json:"token"`
		Article struct {
			ID          string `json:"article_id"`
			Title       string `json:"title"`
			Content     string `json:"content"`
			ContentHTML string `json:"content_html"`
			Author      string `json:"author"`
		} `json:"article"`
	}
	var req articlerequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
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
	}
	// update article
	articleJsonPath := "configs/articles/" + req.Article.ID + ".json"
	articleJsonBin, err := os.ReadFile(articleJsonPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var articleJson articleJsonStruct
	err = json.Unmarshal(articleJsonBin, &articleJson)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	articleJson.Title = req.Article.Title
	articleJson.Content = req.Article.Content
	articleJson.ContentHTML = req.Article.ContentHTML
	articleJson.Author = req.Article.Author
	articleJson.Edit_Date = time.Now().Format("2006-01-02 15:04:05")
	articleJsonBin, err = json.MarshalIndent(articleJson, "", "    ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.WriteFile(articleJsonPath, articleJsonBin, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type articlerequest struct {
		Token string `json:"token"`
		ID    string `json:"article_id"`
	}
	var req articlerequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// delete article
	articleJsonPath := "configs/articles/" + req.ID + ".json"
	err = os.Remove(articleJsonPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type articlerequest struct {
		Token string `json:"token"`
		ID    string `json:"article_id"`
	}
	var req articlerequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// get article
	articleJsonPath := "configs/articles/" + req.ID + ".json"
	articleJsonBin, err := os.ReadFile(articleJsonPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(articleJsonBin)
}

func backendHandler_delete_comment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type commentrequest struct {
		Token     string `json:"token"`
		ArticleID string `json:"article_id"`
		CommentID string `json:"comment_id"`
	}
	var req commentrequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// delete comment
	articleJsonPath := "configs/articles/" + req.ArticleID + ".json"
	articleJsonBin, err := os.ReadFile(articleJsonPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var articleJson articleJsonStruct
	err = json.Unmarshal(articleJsonBin, &articleJson)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusNotFound)
		return
	}
	articleJsonBin, err = json.MarshalIndent(articleJson, "", "    ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.WriteFile(articleJsonPath, articleJsonBin, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type tokenrequest struct {
		Token string `json:"token"`
	}
	var req tokenrequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	Output := make(map[string]interface{})
	// get global settings
	NewMap := make(map[string]interface{})
	for k, v := range GlobalMap {
		NewMap[k] = string(v)
	}
	Output["global_settings"] = NewMap
	// set custom settings
	// set custom script field
	script, err := os.ReadFile("public/js/inject.js")
	if err == nil {
		Output["custom_script"] = string(script)
	}
	// set custom style field
	style, err := os.ReadFile("public/css/customizestyle.css")
	if err == nil {
		Output["custom_style"] = string(style)
	}

	customSettingsBin, err := json.Marshal(Output)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(customSettingsBin)
}

func backendHandler_edit_custom_settings(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
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
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}
	// check token
	if req.Token != EncryptToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// write to file
	jsonData, err := json.MarshalIndent(req.CustomSettings.GlobalSettings, "", "    ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.WriteFile("configs/global.json", jsonData, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// update custom script
	err = os.WriteFile("public/js/inject.js", []byte(req.CustomSettings.CustomScript), 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// update custom style
	err = os.WriteFile("public/css/customizestyle.css", []byte(req.CustomSettings.CustomStyle), 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
	if !Config.CommentCfg.Enabled {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if LastCommentTime.Add(time.Second * time.Duration(Config.CommentCfg.MinSecondsBetweenComments)).After(time.Now()) { // check if the last comment is too frequent
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bodyBin, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type commentRequest struct {
		Verify_token string `json:"verify_token"`
		Article_id   string `json:"article_id"`
		Content      string `json:"content"`
		Author       string `json:"author"`
	}
	var req commentRequest
	err = json.Unmarshal(bodyBin, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Failed to parse request body, %s\n", err)
		return
	}

	// check verify token
	pass := false
	switch Config.CommentCfg.Type {
	case "cloudflare_turnstile":
		pass = CFVerifyCheck(req.Verify_token, getRequestIP(r))
	}
	if !pass {
		w.WriteHeader(http.StatusForbidden)
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

	articleJsonBin, err := os.ReadFile(articleJsonPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var articleJson articleJsonStruct
	err = json.Unmarshal(articleJsonBin, &articleJson)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	articleJson.Comments = append(articleJson.Comments, struct {
		Author   string `json:"author"`
		Content  string `json:"content"`
		Pub_Date string `json:"pub_date"`
		ID       string `json:"id"`
	}{
		Author:   req.Author,
		Content:  req.Content,
		ID:       generateTraceID(),
		Pub_Date: time.Now().Format("2006-01-02 15:04:05"),
	})
	articleJsonBin, err = json.MarshalIndent(articleJson, "", "    ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.WriteFile(articleJsonPath, articleJsonBin, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
}

func CFVerifyCheck(responseToken, userIP string) bool {
	cf_verify_url := "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	data := url.Values{
		"secret":    {Config.CommentCfg.CFSecretKey},
		"response":  {responseToken},
		"client_ip": {userIP},
		"site_key":  {Config.CommentCfg.CFSiteKey},
	}
	resp, err := http.PostForm(cf_verify_url, data)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	var cf_response struct {
		Success     bool     `json:"success"`
		ErrorsCodes []string `json:"error-codes"`
	}
	err = json.Unmarshal(body, &cf_response)
	if err != nil {
		return false
	}
	return cf_response.Success
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
