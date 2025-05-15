package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"
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
		panic(err)
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
			cards_bytes := renderCards()
			RenderedMap["cards"] = cards_bytes

			RenderedMap["top_tag"] = RenderPageTemplate("top_tag", map[string][]byte{
				"tag_name": []byte("Go"),
			})
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
