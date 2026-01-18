package params

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func GetQueryParams(r *http.Request) map[string]string {
	params := map[string]string{}
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}

func GetBodyParams(r *http.Request) (map[string]string, error) {
	params := map[string]string{}

	contentType := strings.ToLower(r.Header.Get("content-type"))
	bodyParams := map[string]string{}

	// 根据 Content-Type 解析请求体
	switch {
	case strings.Contains(contentType, "application/json"):
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return params, err
		}
		log.Printf("request raw body length=%d", len(body))
		bodyParams = parseJSONBody(body)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		if err := r.ParseForm(); err != nil {
			return params, err
		}
		for key, values := range r.PostForm {
			if len(values) > 0 {
				bodyParams[key] = values[0]
			}
		}
	case strings.Contains(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return params, err
		}
		for key, values := range r.MultipartForm.Value {
			if len(values) > 0 {
				bodyParams[key] = values[0]
			}
		}
	default:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return params, err
		}
		bodyParams = parseJSONBody(body)
		if len(bodyParams) == 0 && len(body) > 0 {
			bodyParams["content"] = string(body)
		}
	}

	for k, v := range bodyParams {
		params[k] = v
	}
	return params, nil
}

func parseJSONBody(body []byte) map[string]string {
	result := map[string]string{}
	if len(body) == 0 {
		return result
	}

	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		log.Printf("parseJSONBody error: %v body=%s", err, string(body))
		return result
	}

	switch v := raw.(type) {
	case string:
		result["content"] = v
	case map[string]any:
		if params, ok := v["params"].(map[string]any); ok {
			return mapFromAny(params)
		}
		if data, ok := v["data"].(map[string]any); ok {
			return mapFromAny(data)
		}
		return mapFromAny(v)
	}

	return result
}

func mapFromAny(input map[string]any) map[string]string {
	result := map[string]string{}
	for k, v := range input {
		switch val := v.(type) {
		case string:
			result[k] = val
		case fmt.Stringer:
			result[k] = val.String()
		default:
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}
