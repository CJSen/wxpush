# WXPush

一个极简的微信模板消息推送服务，提供 HTTP 接口与可视化测试页。通过 Webhook 请求发送模板消息，并生成带鉴权的消息详情页。

## 仓库介绍

本项目使用 Go 实现，内置 SQLite 存储与简单页面渲染，适合自托管的微信模板消息推送场景。

## 功能介绍

- HTTP 接口发送微信模板消息（支持 GET/POST）
- 自动获取稳定版 access_token
- 多用户发送（`userid` 使用 `|` 分隔）
- 生成消息详情页链接（`/msg`）
- SQLite 持久化消息记录
- 内置测试页面（`/{APIToken}`）

## 快速开始

1. 复制配置文件并填写参数：

```bash
cp config.yaml.example config.yaml
```

2. 启动服务：

```bash
go run main.go
```

3. 访问首页与测试页：

- 首页：`http://localhost:8080/`
- 测试页：`http://localhost:8080/{APIToken}`

## 配置介绍

配置文件路径：`config.yaml`

```yaml
APIToken: your_token
AppID: your_appid
Secret: your_secret
UserID: openid1|openid2
TemplateID: your_template_id
BaseURL: http://localhost:8080/msg
DBPath: wxpush.db
Port: 8080
```

字段说明：

- `APIToken`：接口鉴权 token
- `AppID` / `Secret`：微信公众号 AppID 与 Secret
- `UserID`：默认接收用户 OpenID，多用户用 `|` 分隔
- `TemplateID`：模板消息 ID
- `BaseURL`：消息详情页基础地址，建议配置为 `/msg`
- `DBPath`：SQLite 数据库路径，默认 `data/wxpush.db`
- `Port`：服务端口，默认 `8080`

## 接口使用介绍

### 发送消息

`GET /wxsend` 或 `POST /wxsend`

必填参数：

- `title`：消息标题
- `content`：消息内容
- `token`：鉴权 token

可选参数（不传则使用配置文件默认值）：

- `appid`
- `secret`
- `userid`
- `template_id`
- `base_url`

鉴权方式：

- 参数 `token`（query 或 body）
- 或在请求头 `Authorization` 中携带（支持 `Bearer token` 或直接 token）

示例（JSON）：

```bash
curl -X POST http://localhost:8080/wxsend \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_token" \
  -d '{
    "title": "告警标题",
    "content": "告警内容",
    "userid": "openid1|openid2"
  }'
```

示例（表单）：

```bash
curl -X POST http://localhost:8080/wxsend \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "title=告警标题&content=告警内容&token=your_token"
```

### 消息详情页

`GET /msg?msg_id=xxx&token_id=yyy`

该链接由服务在发送模板消息时生成并写入模板消息跳转地址。

## 数据存储

SQLite 数据库记录消息详情与生成的 `msg_id` / `token_id`，用于消息详情页鉴权与展示。

## 目录结构

```
.
├── main.go
├── config.yaml.example
├── internal
│   ├── config
│   ├── handler
│   ├── params
│   ├── storage
│   ├── web
│   └── wechat
└── web
    ├── home.html
    ├── test.html
    └── message.html
```

## 注意事项

- `BaseURL` 建议配置为外部可访问的 `/msg` 地址，确保模板消息跳转可用。
- `APIToken` 相当于接口口令，请妥善保管。
