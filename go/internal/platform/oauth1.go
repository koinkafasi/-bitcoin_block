// package platform — OAuth 1.0a (X/Twitter status/update). HMAC-SHA1, tek kullanım.
package platform

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// oauth1Header X v1.1 isteği için Authorization başlığı üretir.
func oauth1Header(method, baseURL string, params map[string]string, consumerKey, consumerSecret, token, tokenSecret string) string {
	p := make(map[string]string, len(params)+7)
	for k, v := range params {
		p[k] = v
	}
	nonce := randNonce(16)
	p["oauth_consumer_key"] = consumerKey
	p["oauth_nonce"] = nonce
	p["oauth_signature_method"] = "HMAC-SHA1"
	p["oauth_timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	p["oauth_token"] = token
	p["oauth_version"] = "1.0"

	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var qParts []string
	for _, k := range keys {
		qParts = append(qParts, oauthEscape(k)+"="+oauthEscape(p[k]))
	}
	sigBase := method + "&" + oauthEscape(baseURL) + "&" + oauthEscape(strings.Join(qParts, "&"))
	signingKey := oauthEscape(consumerSecret) + "&" + oauthEscape(tokenSecret)
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(sigBase))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	p["oauth_signature"] = sig

	// Authorization başlığında oauth_*'lar (token varsa oauth_callback yok)
	var head []string
	orders := []string{"oauth_consumer_key", "oauth_nonce", "oauth_signature", "oauth_signature_method", "oauth_timestamp", "oauth_token", "oauth_version"}
	for _, k := range orders {
		if v, ok := p[k]; ok {
			head = append(head, k+"=\""+oauthEscape(v)+"\"")
		}
	}
	return "OAuth " + strings.Join(head, ", ")
}

func oauthEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func randNonce(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	t := make([]byte, base64.RawStdEncoding.EncodedLen(len(b)))
	base64.RawStdEncoding.Encode(t, b)
	return string(t)
}
