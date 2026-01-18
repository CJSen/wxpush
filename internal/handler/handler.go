package handler

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"wxpush/internal/config"
	"wxpush/internal/params"
	"wxpush/internal/storage"
	"wxpush/internal/web"
	"wxpush/internal/wechat"

	"github.com/gofrs/uuid"
)

type Handler struct {
	cfg    config.Config
	client *wechat.Client
	store  *storage.Store
}

var singleSegRe = regexp.MustCompile(`^/([^/]+)/?$`)

func New(cfg config.Config) (*Handler, error) {
	store, err := storage.NewSQLite(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	return &Handler{
		// 依赖集中注入，便于统一管理
		cfg:    cfg,
		client: wechat.NewClient(),
		store:  store,
	}, nil
}

func (h *Handler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	// 根路径既支持主页，也支持 /{token} 测试页
	path := r.URL.Path
	singleSeg := singleSegRe.FindStringSubmatch(path)

	if r.Method == http.MethodGet && (path == "/" || path == "/index.html") {
		writeHTML(w, web.RenderHomePage())
		return
	}

	if len(singleSeg) == 2 && singleSeg[1] != "wxsend" && singleSeg[1] != "index.html" && singleSeg[1] != "detail" && singleSeg[1] != "msg" {
		rawToken := singleSeg[1]
		if rawToken != h.cfg.APIToken {
			http.Error(w, "Invalid token", http.StatusForbidden)
			return
		}
		sanitized := sanitizeToken(rawToken)
		writeHTML(w, web.RenderTestPage(sanitized))
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func (h *Handler) HandleWxSend(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleSendGet(w, r)
	case http.MethodPost:
		h.handleSendPost(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) HandleMsg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	h.handleMessagePage(w, r)
}

func (h *Handler) handleSendGet(w http.ResponseWriter, r *http.Request) {
	p := params.GetQueryParams(r)
	h.handleSendWithParams(w, r, p)
}

func (h *Handler) handleSendPost(w http.ResponseWriter, r *http.Request) {

	p, err := params.GetBodyParams(r)
	if err != nil {
		http.Error(w, "Failed to parse request", http.StatusBadRequest)
		return
	}
	h.handleSendWithParams(w, r, p)
}

func (h *Handler) handleSendWithParams(w http.ResponseWriter, r *http.Request, p map[string]string) {
	// 统一处理 GET/POST 的参数与认证
	content := p["content"]
	title := p["title"]
	requestToken := p["token"]
	if requestToken == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			authHeader = r.Header.Get("authorization")
		}
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				requestToken = parts[1]
			} else {
				requestToken = authHeader
			}
		}
	}

	if content == "" || title == "" || requestToken == "" {
		http.Error(w, "Missing required parameters: content, title, token", http.StatusBadRequest)
		return
	}

	if requestToken != h.cfg.APIToken {
		http.Error(w, "Invalid token", http.StatusForbidden)
		return
	}

	appid := orDefault(p["appid"], h.cfg.AppID)
	secret := orDefault(p["secret"], h.cfg.Secret)
	useridStr := orDefault(p["userid"], h.cfg.UserID)
	templateID := orDefault(p["template_id"], h.cfg.TemplateID)
	baseURL := orDefault(p["base_url"], h.cfg.BaseURL)

	if appid == "" || secret == "" || useridStr == "" || templateID == "" {
		http.Error(w, "Missing required config: appid, secret, userid, template_id", http.StatusInternalServerError)
		return
	}

	userList := splitUsers(useridStr)
	accessToken, err := h.client.GetStableToken(appid, secret)
	if err != nil || accessToken == "" {
		http.Error(w, "Failed to get access token", http.StatusInternalServerError)
		return
	}

	successCount := 0
	firstError := ""
	for _, userid := range userList {
		msgID, err := uuid.NewV7()
		if err != nil {
			if firstError == "" {
				firstError = "failed to generate msg_id"
			}
			continue
		}
		tokenID, err := uuid.NewV7()
		if err != nil {
			if firstError == "" {
				firstError = "failed to generate token_id"
			}
			continue
		}
		msg := storage.Message{
			MsgID:      msgID.String(),
			TokenID:    tokenID.String(),
			Title:      title,
			Content:    content,
			UserID:     userid,
			TemplateID: templateID,
			BaseURL:    baseURL,
			CreatedAt:  time.Now(),
		}
		if err := h.store.InsertMessage(msg); err != nil {
			if firstError == "" {
				firstError = err.Error()
			}
			continue
		}
		targetURL := buildMessageURL(baseURL, msgID.String(), tokenID.String())
		log.Printf("prepare send userid=%s msg_id=%s url=%s", userid, msgID.String(), targetURL)
		resp, err := h.client.SendMessage(accessToken, userid, templateID, targetURL, title, content)
		if err != nil {
			if firstError == "" {
				firstError = err.Error()
			}
			continue
		}
		if resp.ErrMsg == "ok" {
			successCount++
		} else if firstError == "" {
			firstError = resp.ErrMsg
		}
	}

	if successCount > 0 {
		fmt.Fprintf(w, "Successfully sent messages to %d user(s). First response: ok", successCount)
		return
	}
	if firstError == "" {
		firstError = "Unknown error"
	}
	http.Error(w, fmt.Sprintf("Failed to send messages. First error: %s", firstError), http.StatusInternalServerError)
}

func (h *Handler) handleMessagePage(w http.ResponseWriter, r *http.Request) {
	msgID := r.URL.Query().Get("msg_id")
	tokenID := r.URL.Query().Get("token_id")
	if msgID == "" || tokenID == "" {
		http.Error(w, "Missing msg_id or token_id", http.StatusNotFound)
		return
	}
	msg, err := h.store.GetMessage(msgID, tokenID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid msg_id or token_id", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to load message", http.StatusInternalServerError)
		return
	}
	page := web.RenderMessagePage(web.MessagePageData{
		MsgID:     msg.MsgID,
		Title:     msg.Title,
		Content:   msg.Content,
		UserID:    msg.UserID,
		CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
	})
	writeHTML(w, page)
}

func splitUsers(input string) []string {
	// 使用 | 分隔多个用户
	parts := strings.Split(input, "|")
	users := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			users = append(users, trimmed)
		}
	}
	return users
}

func orDefault(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func sanitizeToken(token string) string {
	// 输出到 HTML 前做最小化转义
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return replacer.Replace(token)
}

func writeHTML(w http.ResponseWriter, html string) {
	// 统一输出 HTML 响应
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, html)
}

func buildMessageURL(baseURL, msgID, tokenID string) string {
	// 生成带 msg_id/token_id 的跳转地址
	target := baseURL
	if target == "" {
		target = "/detail"
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return target
	}
	q := parsed.Query()
	q.Set("msg_id", msgID)
	q.Set("token_id", tokenID)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
