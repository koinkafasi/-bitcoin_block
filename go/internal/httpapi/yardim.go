package httpapi

import (
	"html/template"
	"net/http"
	"strings"

	"theblock/site/internal/webui/templkit"
)

func (s *Server) handleYardim(w http.ResponseWriter, r *http.Request) {
	d := s.baseView("tr", "/admin", "Yardım")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)

	v := templkit.YardimView{
		Kisayollar:  yardimKisayollari(),
		Ipuclari:    yardimIpuclari(),
		SSS:         yardimSSS(),
		Baglantilar: yardimBaglantilari(),
	}
	var sb strings.Builder
	if err := templkit.YardimPanel(v).Render(r.Context(), &sb); err != nil {
		s.logger.Error("yardım paneli", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	d.SlotHTML = template.HTML(sb.String())
	s.render(w, r, "admin_yardim", d)
}

func yardimKisayollari() []templkit.YardimKisayol {
	return []templkit.YardimKisayol{
		{Tus: "Ctrl K", Aciklama: "Komut paleti — içerik, kategori, medya ve sayfa arar"},
		{Tus: "?", Aciklama: "Kısayol penceresi"},
		{Tus: "Esc", Aciklama: "Paleti veya pencereyi kapat"},
		{Tus: "↑ ↓", Aciklama: "Palette ve listelerde gezin"},
		{Tus: "Enter", Aciklama: "Seçili sonucu aç"},
		{Tus: "Ctrl Z", Aciklama: "Editörde geri al"},
		{Tus: "Ctrl Shift Z", Aciklama: "Editörde yinele"},
		{Tus: "Ctrl S", Aciklama: "İçerik formunu kaydet"},
		{Tus: "Ctrl E", Aciklama: "Editörde odak modunu aç/kapat"},
	}
}

func yardimIpuclari() []templkit.YardimMadde {
	return []templkit.YardimMadde{
		{
			Soru: "Klavye ile her yeri bulun", Ekran: "Genel", Ikon: "search",
			Cevap: "Ctrl+K her ekrandan komut paletini açar. Palet yalnız sayfa adı değil, içerik başlığı, kategori, etiket, medya dosyası ve kişi de arar. Son kullandığınız sonuçlar boş sorguda listenin başında çıkar.",
		},
		{
			Soru: "Filtre çubuğu tek satırda", Ekran: "İçerikler", Ikon: "filter",
			Cevap: "Tür, kategori, tarih aralığı, ses durumu, sıralama ve tamlık eşiği aynı ızgarada. Sağ kenarlar hizalı olduğu için hangi alanın hangi sütunda olduğu kaybolmuyor. “temizle” bağlantısı tüm filtreleri sıfırlar.",
		},
		{
			Soru: "Uzun listede başlık sabit kalır", Ekran: "İçerikler", Ikon: "table",
			Cevap: "Liste kendi içinde kaydırılır; sütun başlıkları yukarıda yapışık kalır. Sayfa değil tablo kaydığı için filtre ve toplu işlem çubuğu gözden kaybolmaz.",
		},
		{
			Soru: "Yayın önceliği ne demek?", Ekran: "İçerikler", Ikon: "chart-bar",
			Cevap: "Dört eksenin birleşimi: trend, tazelik, güven ve tamlık. “Öncelik” sütununda büyük sayı toplam puandır; T/Z kısaltmaları trend ve tazelik puanlarını gösterir. Sıralama listesinden “Yayın önceliği” seçildiğinde bu puan kullanılır.",
		},
		{
			Soru: "Yayınlama kapısı neyi engeller?", Ekran: "İçerikler", Ikon: "shield-check",
			Cevap: "YMYL (sağlık, finans, hukuk) içerikte uzman ve kaynak alanları doldurulmadan yayınlanmaz. Ayrıca alaka puanı eşiğin altındaysa yayın reddedilir. Toplu yayınlama da aynı kapıya tabidir.",
		},
		{
			Soru: "Blokları sürükleyerek düzenleyin", Ekran: "Yeni içerik", Ikon: "drag-drop",
			Cevap: "Editörde bir bloğun solunda, üzerine gelince tutamaç belirir. Tutamaçtan tutup sürükleyin; bloklar arasında bırakma çizgisi görünür. Klavyeyle taşımak için bloğu seçip Ctrl+↑ / Ctrl+↓ kullanın.",
		},
		{
			Soru: "AI önerisi hangi alanı doldurur?", Ekran: "Yeni içerik", Ikon: "sparkles",
			Cevap: "Başlık, spot ve SEO alanlarının yanındaki yıldız düğmesi o alanın önerisini üretir. SEO düğmeleri başlık ve spottan beslenir; önce onları doldurun. Üretim sürerken düğme devre dışı kalır, sonuç geldiğinde alana yazılır ve geri alınabilir.",
		},
		{
			Soru: "Otomatik kaydet ne zaman çalışır?", Ekran: "Genel", Ikon: "device-floppy",
			Cevap: "Üst bardaki anahtar açıkken, kaydet düğmesi olan formlar yazmayı bıraktığınızdan 800 ms sonra kendiliğinden kaydedilir. Parola, dosya yükleme ve silme formları bilinçli olarak kapsam dışıdır.",
		},
		{
			Soru: "Gazete sayfasına blok ekleyin", Ekran: "Gazete", Ikon: "news",
			Cevap: "Sol paletten blok türüne tıklayın; tuvalde sürükleyerek yerleştirin. Sağ panelde katmanlar ve özellikler var. “Otomatik Doldur” manşet haberlerinden sayfayı kurar; “Şablon olarak Kaydet” düzeni yeniden kullanmanızı sağlar.",
		},
		{
			Soru: "Medya dosyasını yeniden kullanın", Ekran: "Medya", Ikon: "photo",
			Cevap: "Bir dosyanın alt metni, başlığı ve açıklaması aranabilir. Aynı dosyayı içerikte tekrar kullanmak için “URL kopyala” yerine dosya seçiciden seçin; böylece kırık yol oluşmaz.",
		},
	}
}

func yardimSSS() []templkit.YardimMadde {
	return []templkit.YardimMadde{
		{
			Soru:  "Kaydettiğim içerik neden hemen sitede görünmüyor?",
			Ekran: "İçerikler",
			Cevap: "Durum “yayında” değilse site tarafında görünmez. Ayrıca önbellek nedeniyle en fazla bir dakika gecikme olabilir. Planlanan içerik publish_at zamanı gelince yayına girer.",
		},
		{
			Soru:  "Bir içeriği silmek yerine arşivlemek daha mı iyi?",
			Ekran: "İçerikler",
			Cevap: "Evet. Arşivlenen içerik adresini korur; 301 yönlendirme gerekmez ve bağlantı değeri kaybolmaz. Silme yalnız gerçekten yanlış oluşturulmuş içerik için kullanılmalı.",
		},
		{
			Soru:  "Kategori silince içerikler ne olur?",
			Ekran: "Kategoriler",
			Cevap: "İçerikler silinmez, kategorisiz kalır. “Dağıtım” ekranındaki “Öksüz içerik” listesinden görüp toplu olarak yeni kategoriye taşıyabilirsiniz.",
		},
		{
			Soru:  "Slug değiştirirsem eski adres ne olur?",
			Ekran: "Yeni içerik",
			Cevap: "Eski adres 404 verir. Bu yüzden “Taşıma ve 301” ekranından eski yolu yeni yola yönlendirin; toplu içe aktarma da destekleniyor.",
		},
		{
			Soru:  "Çeviri içeriği ayrı içerik mi?",
			Ekran: "Çeviriler",
			Cevap: "Evet, ayrı bir kayıt ama translation_of alanıyla aslına bağlı. Böylece dil değiştiricide karşılığı bulunabiliyor. Çeviri yayınlanmadan asıl içerik yayında kalabilir.",
		},
		{
			Soru:  "Rolüm bir ekranı açmıyor, neden?",
			Ekran: "Genel",
			Cevap: "Rotalar role göre korunur (Muhabir, Editör, Yönetici…). Gazete modülü ayrıca kendi rol tablosunu kullanır: gazete erişimi olmayan kullanıcı sayıları göremez. Yetki değişikliği için yöneticinize başvurun.",
		},
		{
			Soru:  "Toplu işlem neden bazı satırlarda çalışmıyor?",
			Ekran: "İçerikler",
			Cevap: "Toplu yayınlama tek tek yayınlamayla aynı kapıya tabidir. Kapıyı geçemeyen satırlar atlanır ve sonuç özetinde kaç içeriğin yayınlandığı yazar.",
		},
		{
			Soru:  "Verilerimi nasıl dışa aktarırım?",
			Ekran: "Raporlar",
			Cevap: "Raporlar ekranında CSV dışa aktarma var. Medya listesi de CSV verir. Gazete sayıları PDF olarak üretilir. Toplu içerik taşıma için “Taşıma” ekranındaki örnek CSV biçimini kullanın.",
		},
	}
}

func yardimBaglantilari() []templkit.YardimBaglanti {
	return []templkit.YardimBaglanti{
		{Ad: "İçerikler", Href: "/admin/icerikler", Ikon: "file-text", Not: "Filtre, toplu işlem, yayın önceliği"},
		{Ad: "Yeni içerik", Href: "/admin/icerikler/yeni", Ikon: "file-plus", Not: "Blok editörü, AI önerileri, ses"},
		{Ad: "Kategoriler ve etiketler", Href: "/admin/kategoriler", Ikon: "folder", Not: "Menü, sıralama, birleştirme"},
		{Ad: "Medya", Href: "/admin/dosyalar", Ikon: "photo", Not: "Alt metin, kategori, toplu işlem"},
		{Ad: "Gazete", Href: "/admin/gazete", Ikon: "news", Not: "Sayı üretimi, şablonlar, aboneler"},
		{Ad: "Ayarlar", Href: "/admin/ayarlar", Ikon: "settings", Not: "Site, e-posta, yedekleme"},
		{Ad: "Geri bildirim", Href: "/admin/geri-bildirim", Ikon: "message-dots", Not: "Hata bildirimi ve istek gönderin"},
	}
}
