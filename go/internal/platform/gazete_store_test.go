package platform

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- SecretBox ----

func TestSecretBoxRoundtrip(t *testing.T) {
	box, err := NewSecretBox([]byte("test-master-key"))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("minio-secret-key-123")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sealed, "minio-secret-key-123") {
		t.Fatalf("şifreli değer düz metin sızdırıyor: %s", sealed)
	}
	plain, err := box.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "minio-secret-key-123" {
		t.Fatalf("açılan değer uyuşmuyor: %q", plain)
	}
	if v, _ := box.Seal(""); v != "" {
		t.Fatalf("boş değer şifrelenmemeli, dönen: %q", v)
	}
}

func TestSecretBoxTamper(t *testing.T) {
	box, _ := NewSecretBox([]byte("k"))
	sealed, _ := box.Seal("gizli")
	bad := sealed[:len(sealed)-2] + "xx"
	if _, err := box.Open(bad); err == nil {
		t.Fatal("bozuk şifreli değer açılmamalı")
	}
}

// ---- LocalStore ----

func TestLocalStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	ls := &LocalStore{Root: dir}
	key := "gazete/site-1/2026-10-25/gazete.pdf"
	url, err := ls.Put(key, []byte("PDFDATA"), "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if url != "/"+key {
		t.Fatalf("url: %s", url)
	}
	got, err := ls.Get(key)
	if err != nil || string(got) != "PDFDATA" {
		t.Fatalf("get: %v %q", err, got)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(key))); err != nil {
		t.Fatalf("dosya yok: %v", err)
	}
	if err := ls.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := ls.Get(key); !os.IsNotExist(err) {
		t.Fatalf("silinmiş dosya okunmamalı: %v", err)
	}
	// Var olmayan silme sessiz geçer.
	if err := ls.Delete(key); err != nil {
		t.Fatal(err)
	}
}

// ---- S3Store imzalama ----

// AWS dokümantasyonundaki imzalama zinciri vektörü (string-to-sign → signature).
func TestSigV4Chain(t *testing.T) {
	const stringToSign = "AWS4-HMAC-SHA256\n20150830T123600Z\n20150830/us-east-1/iam/aws4_request\nf536975d06c0309214f805bb90ccff089219ecd68b2577efef23edd43b7e1a59"
	const want = "5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7"
	secret := "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	kDate := hmacSHA256([]byte("AWS4"+secret), "20150830")
	kRegion := hmacSHA256(kDate, "us-east-1")
	kService := hmacSHA256(kRegion, "iam")
	kSigning := hmacSHA256(kService, "aws4_request")
	got := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))
	if got != want {
		t.Fatalf("imza zinciri uyuşmuyor\nalan: %s\n istenen: %s", got, want)
	}
}

func TestS3StoreSignedRequests(t *testing.T) {
	var sawAuth, sawHash string
	var sawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawHash = r.Header.Get("x-amz-content-sha256")
		sawPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &S3Store{
		Endpoint: srv.URL, Bucket: "site2-medya", Region: "auto",
		AccessKeyID: "AK", Secret: "SK",
	}
	if _, err := s.Put("gazete/site-2/x.pdf", []byte("hello"), "application/pdf"); err != nil {
		t.Fatal(err)
	}
	if sawPath != "/site2-medya/gazete/site-2/x.pdf" {
		t.Fatalf("yol: %s", sawPath)
	}
	if !strings.HasPrefix(sawAuth, "AWS4-HMAC-SHA256 Credential=AK/") {
		t.Fatalf("Authorization: %s", sawAuth)
	}
	if !strings.Contains(sawAuth, "SignedHeaders=") || !strings.Contains(sawAuth, "Signature=") {
		t.Fatalf("Authorization eksik: %s", sawAuth)
	}
	if want := sha256HexBytes([]byte("hello")); sawHash != want {
		t.Fatalf("payload hash: %s != %s", sawHash, want)
	}
}

func TestGazeteKey(t *testing.T) {
	if got := GazeteKey(2, "2026-10-25", "gazete.pdf"); got != "gazete/site-2/2026-10-25/gazete.pdf" {
		t.Fatalf("GazeteKey: %s", got)
	}
}

func TestNewObjectStore(t *testing.T) {
	if _, err := NewObjectStore(StoreConfig{Provider: "local"}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := NewObjectStore(StoreConfig{Provider: "r2"}, t.TempDir()); err == nil {
		t.Fatal("eksik yapılandırmayla r2 kurulmamalı")
	}
	st, err := NewObjectStore(StoreConfig{
		Provider: "r2", Endpoint: "https://acct.r2.cloudflarestorage.com",
		Bucket: "b", AccessKeyID: "a", Secret: "s",
	}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.(*S3Store); !ok {
		t.Fatalf("r2 S3Store olmalı: %T", st)
	}
	if _, err := NewObjectStore(StoreConfig{Provider: "ftp"}, t.TempDir()); err == nil {
		t.Fatal("bilinmeyen sağlayıcı reddedilmeli")
	}
}
