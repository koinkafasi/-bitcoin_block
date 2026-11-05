package platform

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---- Nesne depolama soyutlaması (yerel disk / Cloudflare R2 / S3) ----

// StoreConfig site bazlı nesne depolama yapılandırmasıdır (site_storage tablosu).
type StoreConfig struct {
	Provider      string // local | r2 | s3 | minio
	Endpoint      string
	Bucket        string
	Region        string
	AccessKeyID   string
	Secret        string // çözülmüş düz anahtar (yalnızca bellekte)
	PublicBaseURL string
	PathPrefix    string
}

// ObjectStore nesne tabanlı depolama arayüzü. key örn. "media/202610/abc.jpg".
type ObjectStore interface {
	Put(key string, data []byte, contentType string) (string, error)
	Get(key string) ([]byte, error)
	Delete(key string) error
	URL(key string) string
}

// NewObjectStore yapılandırmadan uygun store üretir. provider boşsa/local ise yerel disk.
func NewObjectStore(cfg StoreConfig, localDir string) (ObjectStore, error) {
	p := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch p {
	case "", "local":
		return &LocalStore{Root: localDir, PublicBaseURL: cfg.PublicBaseURL}, nil
	case "r2", "s3", "minio":
		if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKeyID == "" || cfg.Secret == "" {
			return nil, errors.New("S3 depolama için endpoint, bucket ve anahtarlar gerekli")
		}
		region := cfg.Region
		if region == "" {
			region = "auto"
		}
		return &S3Store{
			Endpoint:      strings.TrimRight(cfg.Endpoint, "/"),
			Bucket:        cfg.Bucket,
			Region:        region,
			AccessKeyID:   cfg.AccessKeyID,
			Secret:        cfg.Secret,
			PathPrefix:    strings.Trim(cfg.PathPrefix, "/"),
			PublicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
		}, nil
	default:
		return nil, fmt.Errorf("bilinmeyen depolama sağlayıcısı: %s", cfg.Provider)
	}
}

// ---- Yerel disk ----

// LocalStore {root}/{key} altına yazar (mevcut storage/media düzeniyle uyumlu).
type LocalStore struct {
	Root          string
	PublicBaseURL string
}

func (l *LocalStore) Put(key string, data []byte, _ string) (string, error) {
	p := filepath.Join(l.Root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", err
	}
	return l.URL(key), nil
}

func (l *LocalStore) Get(key string) ([]byte, error) {
	return os.ReadFile(filepath.Join(l.Root, filepath.FromSlash(key)))
}

func (l *LocalStore) Delete(key string) error {
	err := os.Remove(filepath.Join(l.Root, filepath.FromSlash(key)))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (l *LocalStore) URL(key string) string {
	if l.PublicBaseURL != "" {
		return strings.TrimRight(l.PublicBaseURL, "/") + "/" + strings.TrimLeft(key, "/")
	}
	return "/" + strings.TrimLeft(key, "/")
}

// ---- S3 uyumlu (Cloudflare R2 / AWS S3 / MinIO) — minimal SigV4 istemcisi ----

// S3Store S3 API'sini (SigV4 imzalı HTTP) kullanır; ek bağımlılık gerektirmez.
type S3Store struct {
	Endpoint      string
	Bucket        string
	Region        string
	AccessKeyID   string
	Secret        string
	PathPrefix    string
	PublicBaseURL string
	HTTPClient    *http.Client
}

func (s *S3Store) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (s *S3Store) objectKey(key string) string {
	key = strings.TrimLeft(key, "/")
	if s.PathPrefix != "" {
		key = s.PathPrefix + "/" + key
	}
	return key
}

// Put nesneyi yükler ve genel URL'ini döndürür.
func (s *S3Store) Put(key string, data []byte, contentType string) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req, err := s.newRequest(http.MethodPut, key, data, map[string]string{
		"Content-Type": contentType,
	})
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(data))
	resp, err := s.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("S3 PUT %d: %s", resp.StatusCode, string(b))
	}
	return s.URL(key), nil
}

// Get nesneyi okur.
func (s *S3Store) Get(key string) ([]byte, error) {
	req, err := s.newRequest(http.MethodGet, key, nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, os.ErrNotExist
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("S3 GET %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// Delete nesneyi siler (yoksa hata vermez).
func (s *S3Store) Delete(key string) error {
	req, err := s.newRequest(http.MethodDelete, key, nil, nil)
	if err != nil {
		return err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("S3 DELETE %d", resp.StatusCode)
	}
	return nil
}

// URL nesnenin genel erişim adresini döndürür (PublicBaseURL yoksa endpoint/bucket yolu).
func (s *S3Store) URL(key string) string {
	if s.PublicBaseURL != "" {
		return s.PublicBaseURL + "/" + s.objectKey(key)
	}
	return s.Endpoint + "/" + s.Bucket + "/" + s.objectKey(key)
}

// newRequest imzalı S3 isteği kurar (aws4_request). payload gövde içeriğidir (GET/DELETE için nil).
func (s *S3Store) newRequest(method, key string, payload []byte, extraHeaders map[string]string) (*http.Request, error) {
	u, err := url.Parse(s.Endpoint + "/" + s.Bucket + "/" + s.objectKey(key))
	if err != nil {
		return nil, err
	}
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	payloadHash := sha256HexBytes(payload)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("host", u.Host)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	// Kanonik başlıklar
	headers := map[string]string{}
	for k, v := range req.Header {
		headers[strings.ToLower(k)] = strings.TrimSpace(strings.Join(v, ","))
	}
	headers["host"] = u.Host
	var names []string
	for k := range headers {
		names = append(names, k)
	}
	sort.Strings(names)
	var canonHeaders, signedHeaders strings.Builder
	for i, n := range names {
		canonHeaders.WriteString(n + ":" + headers[n] + "\n")
		if i > 0 {
			signedHeaders.WriteString(";")
		}
		signedHeaders.WriteString(n)
	}

	canonicalRequest := strings.Join([]string{
		method,
		u.EscapedPath(),
		"", // sorgu yok
		canonHeaders.String(),
		signedHeaders.String(),
		payloadHash,
	}, "\n")

	scope := dateStamp + "/" + s.Region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex(canonicalRequest),
	}, "\n")

	kDate := hmacSHA256([]byte("AWS4"+s.Secret), dateStamp)
	kRegion := hmacSHA256(kDate, s.Region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.AccessKeyID, scope, signedHeaders.String(), signature))
	return req, nil
}

func sha256Hex(s string) string {
	return sha256HexBytes([]byte(s))
}

func sha256HexBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// GazeteKey gazete çıktıları için standart depolama anahtarı üretir.
// Örn. GazeteKey(2, "2026-10-25", "gazete.pdf") -> "gazete/site-2/2026-10-25/gazete.pdf"
func GazeteKey(siteID int64, issueDate, name string) string {
	return path.Join("gazete", fmt.Sprintf("site-%d", siteID), issueDate, name)
}
