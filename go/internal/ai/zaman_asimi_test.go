package ai

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

// Canlıda görülen hata metninin TAM şekli. 14 Kasım 2026'da 136 başarısız
// işin 128'i bu hatayla ve tam 240 saniyede düştü (2 × 120 sn).
func TestZamanAsimiHatasiCanliOrnek(t *testing.T) {
	// http.Client'in Client.Timeout dolduğunda ürettiği sarmalın eşi.
	ic := &url.Error{
		Op:  "Post",
		URL: "http://omniroute.omniroute.svc.cluster.local:20128/v1/chat/completions",
		Err: fmt.Errorf("%w (Client.Timeout exceeded while awaiting headers)", context.DeadlineExceeded),
	}
	// aieditor'ün sardığı iki katman da üstüne binsin.
	sarmal := fmt.Errorf("AI sağlayıcısına ulaşılamadı: %w", ic)
	if !ZamanAsimiHatasi(sarmal) {
		t.Error("canlıdaki zaman aşımı hatası tanınmadı — yedek deneme boşa gidecek")
	}
}

func TestZamanAsimiHatasiBaglamSuresi(t *testing.T) {
	if !ZamanAsimiHatasi(context.DeadlineExceeded) {
		t.Error("bağlam süresi tanınmadı")
	}
	if !ZamanAsimiHatasi(context.Canceled) {
		t.Error("iptal tanınmadı")
	}
}

// netZamanAsimi: net.Error arayüzünü Timeout()=true ile karşılar.
type netZamanAsimi struct{}

func (netZamanAsimi) Error() string { return "okuma süresi doldu" }
func (netZamanAsimi) Timeout() bool { return true }
func (netZamanAsimi) Temporary() bool {
	return true
}

func TestZamanAsimiHatasiAgKatmani(t *testing.T) {
	var _ net.Error = netZamanAsimi{}
	if !ZamanAsimiHatasi(fmt.Errorf("sarmal: %w", netZamanAsimi{})) {
		t.Error("ağ katmanı zaman aşımı tanınmadı")
	}
}

// BİÇİM REDDİ zaman aşımı DEĞİL: JSON modunu desteklemeyen uç 4xx döner ve
// yedek deneme tam bu durum için var — yanlışlıkla engellenmemeli.
func TestZamanAsimiHatasiBicimReddiDegil(t *testing.T) {
	// httpProvider'ın 4xx için ürettiği hatanın eşi.
	bicim := fmt.Errorf("AI uç noktası %d: %s", 400,
		`{"error":{"message":"response_format is not supported"}}`)
	if ZamanAsimiHatasi(bicim) {
		t.Error("biçim reddi zaman aşımı sayıldı — JSON modu yedeği devre dışı kalır")
	}
}

func TestZamanAsimiHatasiDigerleri(t *testing.T) {
	if ZamanAsimiHatasi(nil) {
		t.Error("nil zaman aşımı sayıldı")
	}
	if ZamanAsimiHatasi(errors.New("boş yanıt")) {
		t.Error("alakasız hata zaman aşımı sayıldı")
	}
	if ZamanAsimiHatasi(ErrKotaDoldu) {
		t.Error("kota hatası zaman aşımı sayıldı")
	}
}
