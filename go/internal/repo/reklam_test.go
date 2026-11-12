package repo

import "testing"

// Tartışmasız reklam: yol işareti.
func TestReklamYolIsareti(t *testing.T) {
	for _, link := range []string{
		"https://cryptoslate.com/press-release/yeni-borsa-aciliyor/",
		"https://decrypt.co/sponsored/proje-tanitimi",
		"https://example.com/sponsored-content/abc",
		"https://example.com/partner-content/abc",
		"https://example.com/advertorial/abc",
		"https://example.com/paid-post/abc",
		"https://ornek.com.tr/basin-bulteni/yeni-urun",
		"https://ornek.com.tr/sponsorlu/kampanya",
	} {
		ok, neden := ReklamIcerikMi(link, "Başlık", "Gövde")
		if !ok {
			t.Errorf("reklam sayılmalıydı: %s", link)
		} else if neden == "" {
			t.Errorf("gerekçe boş: %s", link)
		}
	}
}

// Yönlendirme (affiliate) parametresi.
func TestReklamYonlendirmeParametresi(t *testing.T) {
	for _, link := range []string{
		"https://borsa.example.com/kayit?ref=abc123",
		"https://borsa.example.com/kayit?aff=9988",
		"https://borsa.example.com/x?affiliate=zz",
		"https://borsa.example.com/x?invite=code",
		"https://borsa.example.com/x?referral=k",
	} {
		if ok, _ := ReklamIcerikMi(link, "Başlık", ""); !ok {
			t.Errorf("yönlendirme parametresi yakalanmalıydı: %s", link)
		}
	}
}

// `utm_campaign` İŞARETLENMEMELİ — meşru haber bağlantılarında da var.
func TestUtmCampaignReklamDegil(t *testing.T) {
	link := "https://www.theblock.co/news/markets/2026-11-05-bitcoin?utm_campaign=rss&utm_source=feed"
	if ok, neden := ReklamIcerikMi(link, "Bitcoin 80.000 dolara geriledi", ""); ok {
		t.Errorf("utm_campaign reklam sayılmamalı (gerekçe: %s)", neden)
	}
}

// Açık sponsor ifşası — başlıkta ve gövde başında.
func TestSponsorIfsasi(t *testing.T) {
	if ok, _ := ReklamIcerikMi("https://x.com/a", "Sponsored post: Yeni cüzdan", ""); !ok {
		t.Error("başlıktaki sponsor ifşası yakalanmalıydı")
	}
	govde := "This article is sponsored by Acme Labs. Acme bugün yeni bir ürün duyurdu."
	if ok, _ := ReklamIcerikMi("https://x.com/a", "Acme yeni ürün duyurdu", govde); !ok {
		t.Error("gövde başındaki sponsor ifşası yakalanmalıydı")
	}
	if ok, _ := ReklamIcerikMi("https://x.com/a", "Başlık", "Bu icerik sponsorludur. Devamı..."); !ok {
		t.Error("Türkçe sponsor ifşası yakalanmalıydı")
	}
}

// Teşvik kalıbı YALNIZ başlıkta aranır.
func TestTesvikKalibi(t *testing.T) {
	for _, b := range []string{
		"Guaranteed returns with this new protocol",
		"Use code BONUS50 for your deposit",
		"Claim your free tokens now",
		"The next 100x gem is here",
		"Garantili kazanc firsati",
		"Promosyon kodu ile hemen kazan",
	} {
		if ok, _ := ReklamIcerikMi("https://x.com/a", b, ""); !ok {
			t.Errorf("teşvik kalıbı yakalanmalıydı: %q", b)
		}
	}
}

// ---- SAKIN İŞARETLEMEYECEKLER ----

// Ölçümde geniş bir desene takılan GERÇEK haber. Regresyon kapısı.
func TestGercekHaberIsaretlenmez_KuzeyKore(t *testing.T) {
	baslik := "North Korea reportedly rewards top hackers with Pyongyang apartments after Binance heist"
	link := "https://cryptobriefing.com/north-korea-hackers-perks-binance-heist/"
	if ok, neden := ReklamIcerikMi(link, baslik, baslik); ok {
		t.Errorf("gerçek haber reklam sayıldı (gerekçe: %s)", neden)
	}
}

func TestAirdropPresaleHaberiIsaretlenmez(t *testing.T) {
	for _, b := range []string{
		"Zcash airdrop tarihi açıklandı",
		"Arbitrum presale sonuçları yayımlandı",
		"Yeni token sale SEC incelemesinde",
		"Airdrop avcıları 12 milyon dolar kazandı",
		"Solana ICO döneminin yatırımcıları ne durumda",
	} {
		if ok, neden := ReklamIcerikMi("https://decrypt.co/haber/x", b, ""); ok {
			t.Errorf("gerçek haber reklam sayıldı: %q (gerekçe: %s)", b, neden)
		}
	}
}

// "basın bülteni" / "press release" GÖVDEDE geçmesi haberi reklam yapmaz.
func TestGovdedekiBasinBulteniIfadesiIsaretlenmez(t *testing.T) {
	govde := "Şirket, konuyla ilgili basın bülteninde değerlendirmede bulundu. " +
		"According to a press release, the exchange will expand to three markets."
	if ok, neden := ReklamIcerikMi("https://www.theblock.co/news/business/x", "Borsa üç pazara açılıyor", govde); ok {
		t.Errorf("gövdedeki bülten ifadesi reklam sayılmamalı (gerekçe: %s)", neden)
	}
}

// "ortaklık" ifadesi gerçek haberde geçer.
func TestOrtaklikHaberiIsaretlenmez(t *testing.T) {
	for _, b := range []string{
		"Samsung Wallet, Solana ile ortaklıkla USDC hesapları açtı",
		"Samsung Wallet launches USDC accounts in partnership with Solana",
		"Schroders, Centrifuge ile tokenize fon başlattı",
	} {
		if ok, neden := ReklamIcerikMi("https://www.theblock.co/news/business/x", b, b); ok {
			t.Errorf("ortaklık haberi reklam sayıldı: %q (gerekçe: %s)", b, neden)
		}
	}
}

func TestGercekFeedOrnekleriIsaretlenmez(t *testing.T) {
	ornekler := []struct{ link, baslik string }{
		{"https://www.theblock.co/news/defi/2026-11-06-zcashs-nu7-goes-live-435",
			"Zcash's major network upgrade NU7 goes live on mainnet"},
		{"https://www.theblock.co/news/markets/2026-11-05-ark-invest-buys-coinbase",
			"Ark Invest buys more Coinbase as shares plunge 10% in wider market rout"},
		{"https://decrypt.co/382476/openai-ai-data-centers-rebrand-protect",
			"OpenAI Wants to Rebrand AI Data Centers, Use Them to 'Protect the Grid'"},
		{"https://cryptobriefing.com/blackrock-bitcoin-etf-clients-sell-105m/",
			"BlackRock Bitcoin ETF clients pull $104.97 million from the fund"},
		{"https://cryptoslate.com/bitcoins-rate-cut-relief-stalls-as-service-inflation-flares/",
			"Bitcoin's rate cut relief stalls as service inflation flares"},
		{"https://cryptobriefing.com/cory-fields-quantum-landmines-bitcoin-theft/",
			"Cory Fields proposes quantum landmines to deter Bitcoin theft"},
	}
	for _, o := range ornekler {
		if ok, neden := ReklamIcerikMi(o.link, o.baslik, o.baslik); ok {
			t.Errorf("gerçek feed öğesi reklam sayıldı: %q (gerekçe: %s)", o.baslik, neden)
		}
	}
}

func TestAlakaFiltresiDegil(t *testing.T) {
	for _, b := range []string{
		"Oil prices set for 12% weekly drop despite tanker attacks in Gulf",
		"Meta, Google and TikTok sue California AG to block teen social media law",
		"Anthropic's annualized revenue run rate tops $80 billion",
	} {
		if ok, neden := ReklamIcerikMi("https://cryptobriefing.com/x/", b, ""); ok {
			t.Errorf("kripto dışı ama meşru haber reklam sayıldı: %q (gerekçe: %s)", b, neden)
		}
	}
}
