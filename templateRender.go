package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

var (
	GlobalMap         = make(map[string][]byte)
	GlobalMapLocker   = new(sync.RWMutex)
	RenderedMap       = make(map[string][]byte)
	RenderedMapLocker = new(sync.RWMutex)
)

func RenderTemplate(template []byte, ReplaceMap map[string][]byte) []byte {
	head := []byte("{{") // the key to find and replace
	end := []byte("}}")
	newTemplate := []byte("")
	start := 0
	for {
		index := bytes.Index(template[start:], head)
		if index == -1 {
			break
		}
		newTemplate = append(newTemplate, template[start:start+index]...)
		start += index + len(head)
		index = bytes.Index(template[start:], end)
		if index == -1 {
			break
		}
		key := template[start : start+index]
		l2Index := bytes.Index(key, []byte(":"))
		value := []byte("")
		if l2Index != -1 {
			l2key := key[:l2Index]
			if string(l2key) == "global" {
				GlobalMapLocker.RLock()
				value = GlobalMap[string(key[l2Index+1:])]
				GlobalMapLocker.RUnlock()
			}
			if string(l2key) == "rendered" {
				RenderedMapLocker.RLock()
				value = RenderedMap[string(key[l2Index+1:])]
				RenderedMapLocker.RUnlock()
			}
			if string(l2key) == "file" {
				valueRead, err := os.ReadFile("templates/" + string(key[l2Index+1:]) + ".html")
				if err != nil {
					Log(3, "error reading file: "+err.Error())
				}
				value = valueRead
			}
		} else {
			value = ReplaceMap[string(key)]
		}
		newTemplate = append(newTemplate, value...)
		start += index + len(end)
	}
	newTemplate = append(newTemplate, template[start:]...)
	return newTemplate
}

func RenderPageTemplate(fileRender string, mapRender map[string][]byte) []byte {
	dir := "templates"
	file := dir + "/" + fileRender + ".html"
	template, err := os.ReadFile(file)
	if err != nil {
		Log(3, "error reading template file: "+err.Error())
		return []byte("")
	}
	newTemplate := RenderTemplate(template, mapRender)
	return newTemplate
}

func autoRender(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			render_start_time := time.Now()
			// render cards
			cards_bytes := renderCards()
			RenderedMap["cards"] = cards_bytes

			// render top tag
			RenderedMap["top_tag"] = renderTopBarTags()

			// render RSS
			RenderedMap["rss_feed"] = renderRSSFeed()

			// generate token encrypt key
			newToken := sha256.Sum256([]byte(Config.AccessCfg.BackendPath + Config.AccessCfg.AccessToken))
			RenderedMap["token_encrypt_key"] = []byte(fmt.Sprintf("%x", newToken))
			EncryptTokenKey = fmt.Sprintf("%x", newToken)
			// EncryptToken = generateEncryptToken(Config.AccessCfg.AccessToken, fmt.Sprintf("%x", newToken))

			// fmt.Printf("card rendered\n")
			render_end_time := time.Now()
			render_time := render_end_time.Sub(render_start_time)
			if render_time > 1*time.Second {
				Log(3, "render time too long: "+render_time.String())
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func renderCards() []byte {
	type Card_Config struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardcfg Card_Config
	card_config_filepath := "configs/cards.json"
	card_file, err := os.ReadFile(card_config_filepath)
	if err != nil {
		Log(3, "error reading card config file: "+err.Error())
		return []byte("")
	}
	err = json.Unmarshal(card_file, &cardcfg)
	if err != nil {
		Log(3, "error parsing card config file: "+err.Error())
		return []byte("")
	}
	cards_bytes := []byte("")
	for _, card := range cardcfg.Cards {
		card_opt := map[string][]byte{}
		for k, v := range card {
			card_opt[k] = []byte(v)
		}
		cb := RenderPageTemplate(card["template"], card_opt)
		cards_bytes = append(cards_bytes, cb...)
	}

	return cards_bytes
}

func renderRSSFeed() []byte {
	type Card_Config struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardcfg Card_Config
	card_config_filepath := "configs/cards.json"
	card_file, err := os.ReadFile(card_config_filepath)
	if err != nil {
		Log(3, "error reading card config file: "+err.Error())
		return []byte("")
	}
	err = json.Unmarshal(card_file, &cardcfg)
	if err != nil {
		Log(3, "error parsing card config file: "+err.Error())
		return []byte("")
	}
	rss_posts := []byte("")
	// sort cards by order
	sort.Slice(cardcfg.Cards, func(i, j int) bool {
		return cardcfg.Cards[i]["order"] < cardcfg.Cards[j]["order"]
	})
	for _, card := range cardcfg.Cards {
		card_title := card["card_title"]
		card_description := card["card_description"]
		card_link := card["card_link"]
		if card_link == "" {
			continue
		}
		if GlobalMap["RSSLinkHead"][len(GlobalMap["RSSLinkHead"])-1] == '/' {
			GlobalMap["RSSLinkHead"] = GlobalMap["RSSLinkHead"][:len(GlobalMap["RSSLinkHead"])-1] // remove last '/'
		}
		if strings.HasPrefix(card_link, "articles/") {
			card_link = string(GlobalMap["RSSLinkHead"]) + "/" + card_link
		} else if strings.HasPrefix(card_link, "/articles/") {
			card_link = string(GlobalMap["RSSLinkHead"]) + card_link
		}

		rss_post := RenderPageTemplate("rss_post", map[string][]byte{
			"RSS_TITLE":       []byte(card_title),
			"RSS_LINK":        []byte(card_link),
			"RSS_DESCRIPTION": []byte(card_description),
		})
		rss_posts = append(rss_posts, rss_post...)
	}
	// RenderedMap["RSSPosts"] = rss_posts
	rss_feed := RenderPageTemplate("rss", map[string][]byte{
		"RSSPosts": rss_posts,
	})
	return rss_feed
}

func renderTopBarTags() []byte {
	tags := []byte("")
	tagsarry := strings.Split(string(GlobalMap["TopBarTags"]), " ")
	for _, tag := range tagsarry {
		tag_html := RenderPageTemplate("top_tag", map[string][]byte{
			"tag_name": []byte(tag),
		})
		tags = append(tags, tag_html...)
	}
	return tags
}

func renderarticle(articleID string) []byte {
	articleSavePath := "configs/articles/"
	articleSaveFile := articleSavePath + articleID + ".json"
	// open article file
	article_file, err := os.Open(articleSaveFile)
	if err != nil {
		Log(3, "error reading article file: "+err.Error())
		return []byte("")
	}
	jsonParser := json.NewDecoder(article_file)
	var articlecfg articleJsonStruct
	err = jsonParser.Decode(&articlecfg)
	if err != nil {
		Log(3, "error parsing article file: "+err.Error())
		return []byte("")
	}
	// sort comments by date
	sort.Slice(articlecfg.Comments, func(i, j int) bool {
		layout := "2006-01-02 15:04:05"

		// 解析时间
		ti, err1 := time.Parse(layout, articlecfg.Comments[i].Pub_Date)
		tj, err2 := time.Parse(layout, articlecfg.Comments[j].Pub_Date)

		// 错误处理逻辑：无效日期视为更晚的时间
		switch {
		case err1 != nil && err2 != nil:
			return false // 两者都无效时保持原顺序
		case err1 != nil:
			return false // 仅 i 无效，i 排到后面
		case err2 != nil:
			return true // 仅 j 无效，i 排到前面
		default:
			return ti.Before(tj) // 两者都有效时按时间排序
		}
	})
	// render article comments
	comments_html := []byte("")
	for _, comment := range articlecfg.Comments {
		comment_html := RenderPageTemplate("comment", map[string][]byte{
			"comment_author":  []byte(comment.Author),
			"comment_content": []byte(comment.Content),
			"comment_date":    []byte(comment.Pub_Date),
			"comment_id":      []byte(comment.ID),
		})
		comments_html = append(comments_html, comment_html...)
	}
	article_html := articlecfg.ContentHTML
	if Config.ContentAdvisorCfg.Enabled && Config.ContentAdvisorCfg.FilterArticle {
		bl := bluemonday.UGCPolicy()
		article_html = bl.Sanitize(article_html)
	}
	if articlecfg.Edit_Date != articlecfg.Pub_Date {
		articlecfg.Pub_Date = "ed. " + articlecfg.Edit_Date
	}
	// render article
	rendered_article_html := RenderPageTemplate("article", map[string][]byte{
		"article_title":   []byte(articlecfg.Title),
		"article_author":  []byte(articlecfg.Author),
		"article_content": []byte(article_html),
		"article_date":    []byte(articlecfg.Pub_Date),
		"comments":        comments_html,
	})
	return rendered_article_html
}

func generateEncryptToken(token, encryptKey string, timestampBase64 string) string {
	encryptKey = encryptKey + timestampBase64
	data := []byte(token + "|" + encryptKey)
	encoded := base64.StdEncoding.EncodeToString(data)
	// fmt.Printf("encoded: %s\n", encoded)
	tokenArray := []byte(encoded)

	getRandomChar := func(seed int) byte {
		return byte(33 + (seed % 94))
	}

	for i := 0; i < len(encryptKey); i++ {
		charCode := int(encryptKey[i])
		operation := charCode % 5

		switch operation {
		case 0:
			tokenArray = append([]byte{getRandomChar(charCode + i)}, tokenArray...)

		case 1:
			if len(tokenArray) > 0 {
				pos := (charCode * i) % len(tokenArray)
				tokenArray[pos] = getRandomChar(charCode ^ int(tokenArray[pos]))
			}

		case 2:
			insertPos := charCode % (len(tokenArray) + 1)
			char1 := getRandomChar(charCode)
			char2 := getRandomChar(charCode + 997)
			tokenArray = append(tokenArray[:insertPos], append([]byte{char1, char2}, tokenArray[insertPos:]...)...)

		case 3:
			if len(tokenArray) > 1 {
				pos1 := charCode % len(tokenArray)
				pos2 := len(tokenArray) - 1 - pos1
				tokenArray[pos1], tokenArray[pos2] = tokenArray[pos2], tokenArray[pos1]
			}

		default:
			pseudo := [...]string{"==", "=", "=A", "B="}[charCode%4]
			tokenArray = append(tokenArray, []byte(pseudo)...)
		}
	}

	finalShuffle := make([]byte, 0, len(tokenArray))
	for len(tokenArray) > 0 {
		randIndex := (len(encryptKey) * len(tokenArray)) % len(tokenArray)
		finalShuffle = append(finalShuffle, tokenArray[randIndex])
		tokenArray = append(tokenArray[:randIndex], tokenArray[randIndex+1:]...)
	}

	return string(finalShuffle)
}
