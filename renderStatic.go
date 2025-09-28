package main

import (
	utils "LiteBlog/utils"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// this func used to generate static files, which can be served by nginx or other web server.
// And backend will be disabled.
func RenderStatic() {
	// render everything
	render_start_time := time.Now()
	// render cards
	cards_bytes := renderCardsStatic()
	RenderedMap["cards"] = cards_bytes

	// render context menu
	RenderedMap["context_menu_html"] = RenderPageTemplate("context_menu", map[string][]byte{})

	// render bottom bar
	RenderedMap["bottom_bar"] = RenderPageTemplate("bottom_bar", map[string][]byte{})

	// render RSS
	if Config.RenderCfg.Render.RssFeed {
		RenderedMap["rss_feed"] = renderRSSFeedStatic()
	}

	// render sitemap
	if Config.RenderCfg.Render.SiteMap {
		RenderedMap["site_map"] = renderSiteMap()
	}

	// fmt.Printf("card rendered\n")
	render_end_time := time.Now()
	render_time := render_end_time.Sub(render_start_time)
	if render_time > 1*time.Second {
		utils.Log(3, "render time too long: "+render_time.String())
	}
	// done render
	// render static files
	renderdir := "public/"
	outputDir := "static/"
	// rm output dir
	os.RemoveAll(outputDir)
	renderList := []string{".js", ".css", ".html", ".xml"}
	os.Mkdir(outputDir, 0755)
	// search all files in renderdir and render them to static files
	err := filepath.Walk(renderdir, func(Fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		destPath := filepath.Join(outputDir, Fpath)
		destDir := filepath.Dir(destPath)
		file_ext := filepath.Ext(Fpath)
		// check if in renderList
		if file_ext == "" || !strings.Contains(strings.Join(renderList, "|"), file_ext) {
			// no render
			// direct copy file to static file
			utils.Log(1, "direct copy file: "+Fpath)
			inFile, err := os.OpenFile(Fpath, os.O_RDONLY, 0644)
			if err != nil {
				panic(err)
			}
			defer inFile.Close()
			// check if output file exist
			if _, err := os.Stat(destPath); err == nil {
				// file exist, delete
				os.Remove(destPath)
				return nil
			}
			if _, err := os.Stat(destDir); err != nil {
				os.MkdirAll(destDir, 0755)
			}
			// open output file
			outFile, err := os.Create(destPath)
			if err != nil {
				panic(err)
			}
			defer outFile.Close()
			// copy file
			_, err = io.Copy(outFile, inFile)
			if err != nil {
				panic(err)
			}
		} else {
			// render file to static file
			NeedRender, err := os.ReadFile(Fpath)
			if err != nil {
				return err
			}
			utils.Log(1, "render static file: "+Fpath)
			fileBin := RenderTemplate(NeedRender, nil)
			// check if output file exist
			if _, err := os.Stat(destPath); err == nil {
				// file exist, delete
				os.Remove(destPath)
			}
			if _, err := os.Stat(destDir); err != nil {
				os.MkdirAll(destDir, 0755)
			}
			// open output file
			outFile, err := os.Create(destPath)
			if err != nil {
				panic(err)
			}
			defer outFile.Close()
			// write file
			_, err = outFile.Write(fileBin)
			if err != nil {
				panic(err)
			}
			return nil
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	// render articles
	articlesDir := "configs/articles/"
	outputDir = "static/public/articles/"
	articlesList, err := os.ReadDir(articlesDir)
	if err != nil {
		panic(err)
	}
	for _, article := range articlesList {
		if article.IsDir() {
			continue
		}
		articleName := article.Name()
		// articlePath := articlesDir + articleName
		articleID := articleName[:len(articleName)-len(".json")]
		utils.Log(1, "render article: "+articleName)
		// build article
		fileBin := renderarticle(articleID)
		// check if output file exist
		if _, err := os.Stat(outputDir + articleID + ".html"); err == nil {
			// file exist, delete
			os.Remove(outputDir + articleID + ".html")
		}
		// create output dir
		output_dir := path.Dir(outputDir + articleID + ".html")
		if _, err := os.Stat(output_dir); err != nil {
			os.MkdirAll(output_dir, 0755)
		}
		// open output file
		outFile, err := os.Create(outputDir + articleID + ".html")
		if err != nil {
			panic(err)
		}
		defer outFile.Close()
		// write file
		_, err = outFile.Write(fileBin)
		if err != nil {
			panic(err)
		}
	}
}

func renderCardsStatic() []byte {
	type Card_Config struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardcfg Card_Config
	card_config_filepath := "configs/cards.json"
	card_file, err := os.ReadFile(card_config_filepath)
	if err != nil {
		utils.Log(3, "error reading card config file: "+err.Error())
		return []byte("")
	}
	err = json.Unmarshal(card_file, &cardcfg)
	if err != nil {
		utils.Log(3, "error parsing card config file: "+err.Error())
		return []byte("")
	}
	cards_bytes := []byte("")
	for _, card := range cardcfg.Cards {
		card_opt := map[string][]byte{}
		for k, v := range card {
			if k == "card_link" && (strings.HasPrefix(v, "/articles/") || strings.HasPrefix(v, "articles/")) {
				v = v + ".html"
			}
			card_opt[k] = []byte(v)
		}
		cb := RenderPageTemplate(card["template"], card_opt)
		cards_bytes = append(cards_bytes, cb...)
	}

	return cards_bytes
}

func renderRSSFeedStatic() []byte {
	type Card_Config struct {
		Cards []map[string]string `json:"cards"`
	}
	var cardcfg Card_Config
	card_config_filepath := "configs/cards.json"
	card_file, err := os.ReadFile(card_config_filepath)
	if err != nil {
		utils.Log(3, "error reading card config file: "+err.Error())
		return []byte("")
	}
	err = json.Unmarshal(card_file, &cardcfg)
	if err != nil {
		utils.Log(3, "error parsing card config file: "+err.Error())
		return []byte("")
	}
	rss_posts := []byte("")

	cards := cardcfg.Cards
	// sort
	for i, j := 0, len(cards)-1; i < j; i, j = i+1, j-1 {
		cards[i], cards[j] = cards[j], cards[i] // reverse order
	}
	for _, card := range cards {
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
		card_link = card_link + ".html"
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
