package httpapi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// cdGirisOlustur hero görünümünü gerçek veriden kurar.
func (s *Server) cdGirisOlustur(ctx context.Context) templkit.CDGiris {
	var g templkit.CDGiris
	v := repo.PiyasaGirisVeri(ctx, s.pool)
	if !v.Hazir {
		return g
	}

	g.Hazir = true
	g.Eyebrow = "Kripto dünyasından en güncel bilgiler"
	// Başlık üç parça: vurgu kısmı altın gradyanla basılıyor (kaynak
	// tasarımdaki <em> ile aynı işlev).
	g.BaslikOnce = "Türkiye'nin En Güvenilir "
	g.BaslikVurgu = "Dijital Varlık Bilgi"
	g.BaslikSonra = " Platformu"
	g.Aciklama = "Bitcoin, Ethereum, altcoinler ve blockchain dünyasından anlık haberler, " +
		"derinlemesine analizler ve güvenilir veriler."
	// CTA'lar GERÇEK sayfalara gider (kaynak tasarımda #ch-latest gibi
	// boş çapalara gidiyordu).
	g.Cta1Ad, g.Cta1Href = "Son Haberleri Keşfet", "/latest-crypto-news"
	g.Cta2Ad, g.Cta2Href = "Piyasaları İncele", "/price"
	g.Maddeler = []templkit.CDGirisMadde{
		{No: "01", Metin: "Anlık Piyasa Verisi"},
		{No: "02", Metin: "Derinlemesine Analiz"},
		{No: "03", Metin: "Doğrulanmış Kaynak"},
	}
	g.Guncelleme = cdYasMetni(v.Cekildi)
	g.Cekildi = v.Cekildi

	g.BTCFiyat = repo.FiyatBicim(v.BTCFiyat)
	g.BTCDegisim = repo.DegisimBicim(v.BTC24)
	g.BTCUp = v.BTC24 >= 0
	g.BTCYol, g.BTCAlan = v.BTCYol, v.BTCAlan
	g.Hacim = cdKisaDolar(v.Hacim24)
	if v.BTCDominan > 0 {
		g.Dominans = strconv.FormatFloat(v.BTCDominan, 'f', 1, 64) + "%"
	}

	if v.ETHFiyat > 0 {
		g.ETHFiyat = repo.FiyatBicim(v.ETHFiyat)
		g.ETHDegisim = repo.DegisimBicim(v.ETH24)
		g.ETHUp = v.ETH24 >= 0
	}
	if v.SOLFiyat > 0 {
		g.SOLFiyat = repo.FiyatBicim(v.SOLFiyat)
		g.SOLDegisim = repo.DegisimBicim(v.SOL24)
		g.SOLUp = v.SOL24 >= 0
	}

	if v.FNG > 0 {
		g.FNGVar = true
		g.FNGDeger = v.FNG
		g.FNGEtiket = cdFNGEtiket(v.FNG)
		g.FNGUp = v.FNG >= 50
		g.FNGDun, g.FNGHafta = v.FNGDun, v.FNGHafta
		g.Endeks = strconv.Itoa(v.FNG)
		g.EndeksUp = v.FNG >= 50
	}
	if v.RSI > 0 {
		g.RSIVar = true
		g.RSI = int(v.RSI)
		g.RSIEtiket = cdRSIEtiket(g.RSI)
		g.RSIUp = g.RSI >= 30 && g.RSI <= 70
	}
	return g
}

func cdKisaDolar(f float64) string {
	switch {
	case f <= 0:
		return ""
	case f >= 1e12:
		return "$" + repo.SayiBicim(f/1e12, 1) + " Tn"
	case f >= 1e9:
		return "$" + repo.SayiBicim(f/1e9, 1) + " Mr"
	case f >= 1e6:
		return "$" + repo.SayiBicim(f/1e6, 1) + " Mn"
	default:
		return "$" + repo.SayiBicim(f, 0)
	}
}

func cdFNGEtiket(deger int) string {
	switch {
	case deger < 25:
		return "Aşırı Korku"
	case deger < 50:
		return "Korku"
	case deger < 75:
		return "Açgözlülük"
	default:
		return "Aşırı Açgözlülük"
	}
}

// cdRSIEtiket RSI değerini Türkçe sınıfa çevirir.
func cdRSIEtiket(rsi int) string {
	switch {
	case rsi < 30:
		return "Aşırı Satım"
	case rsi > 70:
		return "Aşırı Alım"
	default:
		return "Nötr"
	}
}

func cdYasMetni(unix int64) string {
	if unix <= 0 {
		return ""
	}
	fark := time.Since(time.Unix(unix, 0))
	switch {
	case fark < time.Minute:
		return "AZ ÖNCE"
	case fark < time.Hour:
		return fmt.Sprintf("%d DK ÖNCE", int(fark.Minutes()))
	case fark < 24*time.Hour:
		return fmt.Sprintf("%d SA ÖNCE", int(fark.Hours()))
	default:
		return strings.ToUpper(fmt.Sprintf("%d gün önce", int(fark.Hours()/24)))
	}
}
