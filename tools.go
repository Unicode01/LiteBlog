package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

type Xorshift32 struct {
	state uint32
}

func NewXorshift32(seed uint32) (*Xorshift32, error) {
	if seed == 0 {
		return nil, fmt.Errorf("seed cannot be zero")
	}
	return &Xorshift32{state: seed}, nil
}

func (x *Xorshift32) Next() uint32 {
	x.state ^= x.state << 13
	x.state ^= x.state >> 17
	x.state ^= x.state << 5
	return x.state
}

func (x *Xorshift32) Random() float64 {
	return float64(x.Next()) / float64(1<<32)
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

func GoogleVerifyCheck(responseToken, userIP string) bool {
	verifyUrl := "https://www.google.com/recaptcha/api/siteverify"
	data := url.Values{
		"secret":   {Config.CommentCfg.GoogleSecretKey},
		"response": {responseToken},
		"remoteip": {userIP},
	}
	resp, err := http.PostForm(verifyUrl, data)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	var google_response struct {
		Success     bool     `json:"success"`
		ChallengeTS string   `json:"challenge_ts"`
		Hostname    string   `json:"hostname"`
		ErrorCodes  []string `json:"error-codes"`
	}
	err = json.Unmarshal(body, &google_response)
	if err != nil {
		return false
	}
	return google_response.Success
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

func isAvailableEmailAddress(email string) bool {
	// 基础检查：空值、无@符号
	if email == "" || !strings.Contains(email, "@") {
		return false
	}

	// 分割本地部分和域名部分
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	localPart := parts[0]
	domainPart := parts[1]

	// 1. 本地部分验证
	if len(localPart) < 1 || len(localPart) > 64 ||
		localPart[0] == '.' || localPart[len(localPart)-1] == '.' ||
		strings.Contains(localPart, "..") {
		return false
	}

	// 本地部分字符集验证
	localRegex := regexp.MustCompile(`^[a-zA-Z0-9!#$%&'*+\-\/=?^_{|}~]+(\.[a-zA-Z0-9!#$%&'*+\-\/=?^_{|}~]+)*$`)
	if !localRegex.MatchString(localPart) {
		return false
	}

	// 2. 域名部分验证
	if len(domainPart) < 1 || len(domainPart) > 255 ||
		domainPart[0] == '-' || domainPart[len(domainPart)-1] == '-' ||
		domainPart[0] == '.' || domainPart[len(domainPart)-1] == '.' ||
		strings.Contains(domainPart, "..") {
		return false
	}

	// 域名标签分割验证
	domainLabels := strings.Split(domainPart, ".")
	labelRegex := regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9\-]*[a-zA-Z0-9])?$`)

	for _, label := range domainLabels {
		if len(label) < 1 || len(label) > 63 ||
			!labelRegex.MatchString(label) {
			return false
		}
	}

	// 顶级域名检查 (至少2个字母)
	tld := domainLabels[len(domainLabels)-1]
	tldRegex := regexp.MustCompile(`^[a-zA-Z]{2,}$`)
	return tldRegex.MatchString(tld)
}

func checkToken(token string) bool {
	// generate 5 tokens for check ( allow now ± 20s token expired )
	nowTimestamp := (time.Now().Unix() / 10)
	befTs1 := nowTimestamp - 1
	befTs2 := nowTimestamp - 2
	aftTs1 := nowTimestamp + 1
	aftTs2 := nowTimestamp + 2
	befTs1Base64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", befTs1)))
	befTs2Base64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", befTs2)))
	aftTs1Base64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", aftTs1)))
	aftTs2Base64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", aftTs2)))
	nowTsBase64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", nowTimestamp)))
	var tokens []string
	tokens = append(tokens, generateEncryptToken(Config.AccessCfg.AccessToken, EncryptTokenKey, nowTsBase64))
	tokens = append(tokens, generateEncryptToken(Config.AccessCfg.AccessToken, EncryptTokenKey, befTs1Base64))
	tokens = append(tokens, generateEncryptToken(Config.AccessCfg.AccessToken, EncryptTokenKey, aftTs1Base64))
	tokens = append(tokens, generateEncryptToken(Config.AccessCfg.AccessToken, EncryptTokenKey, befTs2Base64))
	tokens = append(tokens, generateEncryptToken(Config.AccessCfg.AccessToken, EncryptTokenKey, aftTs2Base64))
	// fmt.Printf("token: %v\n", token)
	for _, t := range tokens {
		// fmt.Printf("check token: %s|%s\n", t, token)
		if t == token {
			return true
		}
	}
	return false
}
