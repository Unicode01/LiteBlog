package main

type articleJsonStruct struct {
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	ContentHTML string            `json:"content_html"`
	Author      string            `json:"author"`
	Edit_Date   string            `json:"edit_date"`
	Pub_Date    string            `json:"pub_date"`
	ExtraFlags  map[string]string `json:"extra_flags"`
	Comments    []struct {
		Author   string `json:"author"`
		Email    string `json:"email"`
		Content  string `json:"content"`
		Pub_Date string `json:"pub_date"`
		ID       string `json:"id"`
		ReplyTo  string `json:"reply_to"`
	} `json:"comments"`
}
