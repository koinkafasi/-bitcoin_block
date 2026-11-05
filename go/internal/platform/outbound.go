package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PostJSON verilen URL'e JSON POST eder ve durum kodunu + gövdeyi döndürür.
func PostJSON(ctx context.Context, urlStr string, body any, headers map[string]string) (int, []byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	return PostRaw(ctx, urlStr, "application/json", b, headers)
}

// outboundClient dış çağrılar için ortak istemci: her istek için üst sınırlı
// süre ve yönlendirme sayısı (http.DefaultClient'ın ikisi de yoktur).
var outboundClient = &http.Client{
	Timeout: 25 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("çok fazla yönlendirme")
		}
		return nil
	},
}

// PostRaw gövdeyi olduğu gibi gönderir.
func PostRaw(ctx context.Context, urlStr, contentType string, body []byte, headers map[string]string) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "theblock/1.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := outboundClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, rb, nil
}

// SendTelegram Telegram Bot API ile mesaj gönderir.
func SendTelegram(ctx context.Context, token, chatID, text string) error {
	if token == "" || chatID == "" {
		return fmt.Errorf("telegram token/chat eksik")
	}
	url := "https://api.telegram.org/bot" + token + "/sendMessage"
	code, body, err := PostJSON(ctx, url, map[string]any{"chat_id": chatID, "text": text, "disable_web_page_preview": false}, nil)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("telegram HTTP %d: %s", code, truncate(string(body), 200))
	}
	return nil
}

// SubmitIndexNow IndexNow (Bing/Yandex) uçlarına URL listesi bildirir.
func SubmitIndexNow(ctx context.Context, endpoint, host, key, keyLocation string, urls []string) (int, error) {
	if endpoint == "" {
		endpoint = "https://api.indexnow.org/indexnow"
	}
	body := map[string]any{"host": host, "key": key, "urlList": urls}
	if keyLocation != "" {
		body["keyLocation"] = keyLocation
	}
	code, rb, err := PostJSON(ctx, endpoint, body, nil)
	if err != nil {
		return 0, err
	}
	if code >= 300 {
		return code, fmt.Errorf("indexnow HTTP %d: %s", code, truncate(string(rb), 200))
	}
	return code, nil
}
