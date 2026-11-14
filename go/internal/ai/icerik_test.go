package ai

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func sessizKayitci() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// evrenYanitiDusunmeDolu, 14 Kasım 2026'da evren/deepseek-v4.1-flash'ten
// ÖLÇÜLEN gerçek yanıt biçimidir: content null, çıktı reasoning'de,
// finish_reason "length". Biçimi elle sadeleştirmiyoruz; testin değeri
// sağlayıcının gerçekten döndürdüğü şeye dayanmasından geliyor.
const evrenYanitiDusunmeDolu = `{
  "id": "4bc5a150-c202-4f51-9536-df9971e549e8",
  "object": "chat.completion",
  "model": "deepseek-v4.1-flash",
  "choices": [{
    "index": 0,
    "finish_reason": "length",
    "message": {
      "role": "assistant",
      "refusal": null,
      "content": null,
      "reasoning": "We need answer Turkish. User asks to write news. Need JSON."
    }
  }],
  "usage": {
    "prompt_tokens": 45,
    "completion_tokens": 200,
    "completion_tokens_details": {"reasoning_tokens": 200}
  }
}`

func testSaglayici(t *testing.T, govde string, durum int) *httpProvider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(durum)
		_, _ = w.Write([]byte(govde))
	}))
	t.Cleanup(srv.Close)
	return &httpProvider{
		name: "test", baseURL: srv.URL, apiKey: "k", model: "m",
		client: srv.Client(), logger: sessizKayitci(),
	}
}

// Asıl gerileme testi: düşünen model yanıtı eskiden BOŞ BAŞARI dönüyordu.
// Hata düşmediği için hiçbir günlükte görünmüyor, iş birkaç adım sonra
// "geçerli JSON değil" diye ölüyordu ve sebep yanlış yeri gösteriyordu.
func TestCompleteDusunenModelBosIcerikteHataVerir(t *testing.T) {
	p := testSaglayici(t, evrenYanitiDusunmeDolu, http.StatusOK)
	resp, err := p.Complete(context.Background(), Request{
		Messages: []Msg{{Role: RoleUser, Content: "yaz"}}, MaxTokens: 200,
	})
	if err == nil {
		t.Fatalf("boş içerik hata vermedi; Text=%q (eski sessiz kayıp geri geldi)", resp.Text)
	}
	if !errors.Is(err, ErrBosIcerik) {
		t.Fatalf("ErrBosIcerik beklendi, gelen: %v", err)
	}
	// Hata mesajı ÇÖZÜMÜ söylemeli; yoksa okuyan yine max_tokens'ı
	// büyütmeye çalışır ve o yol 204 sn sürüp zaman aşımına düşer.
	if !strings.Contains(err.Error(), "OMNIROUTE_REASONING_EFFORT") {
		t.Fatalf("hata mesajı çözümü söylemiyor: %v", err)
	}
}

func TestCompleteKesikCiktiHataVerir(t *testing.T) {
	govde := `{"choices":[{"finish_reason":"length","message":{"content":"{\"title\":\"ya"}}],` +
		`"usage":{"prompt_tokens":10,"completion_tokens":4000}}`
	p := testSaglayici(t, govde, http.StatusOK)
	_, err := p.Complete(context.Background(), Request{
		Messages: []Msg{{Role: RoleUser, Content: "yaz"}}, MaxTokens: 4000,
	})
	if !errors.Is(err, ErrKesikIcerik) {
		t.Fatalf("ErrKesikIcerik beklendi, gelen: %v", err)
	}
}

// Sohbet yolu etkileşimli düz metindir: yarım cümle, hiç yanıt
// olmamasından iyidir. Kesilme orada hata DEĞİL uyarıdır.
func TestChatKesikMetniDondurur(t *testing.T) {
	govde := `{"choices":[{"finish_reason":"length","message":{"content":"Yarim cum"}}],` +
		`"usage":{"completion_tokens":5}}`
	p := testSaglayici(t, govde, http.StatusOK)
	out, err := p.Chat(context.Background(), []Msg{{Role: RoleUser, Content: "selam"}})
	if err != nil {
		t.Fatalf("sohbet yolunda kesilme hata vermemeli: %v", err)
	}
	if out != "Yarim cum" {
		t.Fatalf("metin beklendi, gelen: %q", out)
	}
}

// Boş içerik her yolda hatadır: boş metni başarı saymak hiçbir yerde
// doğru değil.
func TestChatBosIcerikteHataVerir(t *testing.T) {
	p := testSaglayici(t, evrenYanitiDusunmeDolu, http.StatusOK)
	_, err := p.Chat(context.Background(), []Msg{{Role: RoleUser, Content: "selam"}})
	if !errors.Is(err, ErrBosIcerik) {
		t.Fatalf("ErrBosIcerik beklendi, gelen: %v", err)
	}
}

// Kota sınıflandırması Chat'e eklenmişti ama hat CompleteStrict
// kullanıyor. Bu test o yolu bağlar: 429 sıradan hata olmamalı, yoksa
// iş `failed` işaretlenip kaybediliyor.
func TestCompleteKotaHatasiniAyirtEder(t *testing.T) {
	p := testSaglayici(t, `{"error":"rate limit"}`, http.StatusTooManyRequests)
	_, err := p.Complete(context.Background(), Request{
		Messages: []Msg{{Role: RoleUser, Content: "yaz"}},
	})
	if !errors.Is(err, ErrKotaDoldu) {
		t.Fatalf("ErrKotaDoldu beklendi, gelen: %v", err)
	}
}

// CompleteStrict kota tükenmişken HİÇ çağrı yapmamalı.
func TestCompleteStrictKotaTukendiyseCagriYapmaz(t *testing.T) {
	var cagri int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cagri++
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limit"}`))
	}))
	defer srv.Close()

	c := &Client{logger: sessizKayitci(), kota: NewKotaIzleyici()}
	c.providers = append(c.providers, &httpProvider{
		name: "test", baseURL: srv.URL, apiKey: "k", model: "m",
		client: srv.Client(), logger: sessizKayitci(),
	})

	req := Request{Messages: []Msg{{Role: RoleUser, Content: "yaz"}}}
	if _, err := c.CompleteStrict(context.Background(), req); !errors.Is(err, ErrKotaDoldu) {
		t.Fatalf("ilk çağrı ErrKotaDoldu vermeli, gelen: %v", err)
	}
	if cagri != 1 {
		t.Fatalf("ilk turda 1 çağrı beklendi, oldu: %d", cagri)
	}
	// İkinci çağrı kapıya takılmalı: sağlayıcıya gitmemeli.
	if _, err := c.CompleteStrict(context.Background(), req); !errors.Is(err, ErrKotaDoldu) {
		t.Fatalf("ikinci çağrı da ErrKotaDoldu vermeli, gelen: %v", err)
	}
	if cagri != 1 {
		t.Fatalf("kota tükenmişken ikinci HTTP çağrısı yapıldı (%d); kota kapısı çalışmıyor", cagri)
	}
}

// İki yeniden-deneme kapısı birbirine karışmamalı: içerik hatası zaman
// aşımı değildir, zaman aşımı da içerik hatası değildir. Karışırsa
// aieditor yanlış dalı seçer.
func TestIcerikHatasiVeZamanAsimiAyriKalir(t *testing.T) {
	if ZamanAsimiHatasi(ErrBosIcerik) {
		t.Fatal("boş içerik zaman aşımı sayılmamalı")
	}
	if ZamanAsimiHatasi(ErrKesikIcerik) {
		t.Fatal("kesik içerik zaman aşımı sayılmamalı")
	}
	if IcerikHatasi(context.DeadlineExceeded) {
		t.Fatal("zaman aşımı içerik hatası sayılmamalı")
	}
	if IcerikHatasi(ErrKotaDoldu) {
		t.Fatal("kota hatası içerik hatası sayılmamalı: iş kuyrukta beklemeli")
	}
	if !IcerikHatasi(ErrBosIcerik) || !IcerikHatasi(ErrKesikIcerik) {
		t.Fatal("IcerikHatasi kendi hatalarını tanımıyor")
	}
}

// Ölçülen gerçek değer: düşünme açıkken aynı istek 204.2 sn sürdü,
// HTTP sınırımız 180 sn. Yani "bütçeyi büyüt" yolu zaman aşımına düşer;
// hata mesajının düşünmeyi kapatmayı önermesi bu yüzden şart.
func TestDusunmeAcikOlcumuSinirinUstunde(t *testing.T) {
	const olculenDusunmeli = 204 * time.Second
	if aiHTTPTimeout() >= olculenDusunmeli {
		t.Fatalf("HTTP sınırı %s, düşünme açıkken ölçülen süre %s: artık "+
			"sığıyor. ErrBosIcerik mesajı çözüm olarak 'düşünmeyi kapat' "+
			"diyor; sınır bu kadar büyüdüyse 'max_tokens'ı artır' da "+
			"geçerli bir yol oldu ve öneri gözden geçirilmeli.",
			aiHTTPTimeout(), olculenDusunmeli)
	}
}
