package web

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"
)

const (
	testPagePath = "web/test.html"
	homePagePath = "web/home.html"
)

var messagePagePath = "web/message.html"

func SetMessagePagePath(path string) {
	fmt.Println("MessageHtml is:", path)
	if strings.TrimSpace(path) == "" {
		return
	}
	messagePagePath = path
}

func RenderTestPage(token string) string {
	html := readFile(testPagePath)
	return strings.ReplaceAll(html, "{{TOKEN}}", token)
}

func RenderHomePage() string {
	return readFile(homePagePath)
}

type MessagePageData struct {
	MsgID     string
	Title     string
	Content   string
	UserID    string
	CreatedAt string
}

func RenderMessagePage(data MessagePageData) string {
	html := readFile(messagePagePath)
	if data.CreatedAt == "" {
		data.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	}
	tpl, err := template.New("message").Parse(html)
	if err != nil {
		return html
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return html
	}
	return buf.String()
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
