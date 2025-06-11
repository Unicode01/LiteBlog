package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
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
		// clean key
		key = bytes.TrimSpace(key)
		key = bytes.ReplaceAll(key, []byte(" "), []byte(""))
		key = bytes.ReplaceAll(key, []byte("\n"), []byte(""))
		l2Index := bytes.Index(key, []byte(":"))
		value := []byte("")
		if l2Index != -1 {
			l2key := key[:l2Index]
			switch string(l2key) {
			case "global":
				GlobalMapLocker.RLock()
				value = GlobalMap[string(key[l2Index+1:])]
				GlobalMapLocker.RUnlock()
			case "rendered":
				RenderedMapLocker.RLock()
				value = RenderedMap[string(key[l2Index+1:])]
				RenderedMapLocker.RUnlock()
			case "file":
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
			if EncryptTokenKey == "" {
				if Config.AccessCfg.RandomKey {
					RandSeedBytes := make([]byte, 4)
					_, err := rand.Read(RandSeedBytes)
					if err != nil {
						xorshift, _ := NewXorshift32(uint32(time.Now().UnixNano()))
						value := xorshift.Next()
						RandSeedBytes[0] = byte(value >> 24)
						RandSeedBytes[1] = byte(value >> 16)
						RandSeedBytes[2] = byte(value >> 8)
						RandSeedBytes[3] = byte(value)
					}
					RandSeed := uint32(binary.LittleEndian.Uint32(RandSeedBytes))
					xorshift, _ := NewXorshift32(RandSeed)
					EncryptTokenKey = ""
					for i := 0; i < 32; i++ { // 32 bytes key
						value := xorshift.Next() % 256 // 0-255, get a random byte
						EncryptTokenKey += fmt.Sprintf("%02x", value)
					}
					RenderedMap["token_encrypt_key"] = []byte(EncryptTokenKey)
					// fmt.Printf("token encrypt key: %s\n", EncryptTokenKey)
				} else {
					newToken := sha256.Sum256([]byte(Config.AccessCfg.BackendPath + Config.AccessCfg.AccessToken))
					RenderedMap["token_encrypt_key"] = []byte(fmt.Sprintf("%x", newToken))
					EncryptTokenKey = fmt.Sprintf("%x", newToken)
				}
			}
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
		if comment.ReplyTo != "" {
			for _, c := range articlecfg.Comments {
				if c.ID == comment.ReplyTo {
					comment.Author = comment.Author + " (@" + c.Author + ")"
					break
				}
			}
		}
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
	basicRenderMap := map[string][]byte{
		"article_title":     []byte(articlecfg.Title),
		"article_author":    []byte(articlecfg.Author),
		"article_content":   []byte(article_html),
		"article_date":      []byte(articlecfg.Pub_Date),
		"article_edit_date": []byte(articlecfg.Edit_Date),
		"comments":          comments_html,
	}
	extraFlagsList := articlecfg.ExtraFlags
	for k, v := range extraFlagsList {
		if basicRenderMap[k] != nil {
			continue
		}
		basicRenderMap[k] = []byte(v)
	}
	// render article
	rendered_article_html := RenderPageTemplate("article", basicRenderMap)
	return rendered_article_html
}
