package httpapi

import (
	"context"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/auth"
	"theblock/site/internal/i18n"
	"theblock/site/internal/piyasa"
	"theblock/site/internal/platform"
	"theblock/site/internal/quality"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ---- Ortak görünüm ----

type flashMsg struct {
	Kind string
	Text string
}

type langLink struct {
	Label  string
	Code   string // ISO dil kodu (hreflang / lang nitelikleri için)
	URL    string
	Active bool
}

type rozet struct {
	Label string
	Class string
}

type tmplData struct {
	Lang       string
	LangBase   string
	HomeURL    string
	Title      string
	SiteName   string
	SiteSlogan string
	// Sayfa bazlı SEO başlığı/açıklaması ve sosyal kart alanları.
	MetaTitle string
	MetaDesc  string
	Robots    string
	OGType    string
	OGImage   string
	// OGImageGenis: OG görseli gerçekten geniş formatlı mı (haber kapağı ya da
	// panelden yüklenen marka görseli). Kurumsal ikon yedeğine düşüldüğünde
	// false olur; twitter:card o zaman summary_large_image iddia etmez.
	OGImageGenis bool
	// BreadcrumbJSONLD sayfa tipine göre kırıntı yolu yapısal verisi.
	BreadcrumbJSONLD template.JS
	// PersonJSONLD /yazar/{slug} WebPage+Person yapısal verisi (CoinDesk).
	PersonJSONLD template.JS
	// PageCategory/ContentCategory: CoinDesk sayfa sınıflandırması. Boşsa
	// hiç basılmaz (theblock şablonları bu alanları kullanmıyor).
	PageCategory    string
	ContentCategory string
	Theme           string
	Year            string
	RTL             bool
	Locales         []i18n.Locale
	NavCategories   []repo.Category
	Menu            []MenuRow
	MenuDesign      string
	CatNavDesign    string
	MenuBgCSS       template.CSS
	MenuEdit        *menuEditData
	RSSList         *rssListData
	AISource        *aiSourceData
	TabCatsEdit     *tabCatsEditData
	Header          *HeaderConfig
	HeaderEdit      *headerEditData
	Footer          *FooterConfig
	FooterEdit      *footerEditData
	Brand           *BrandConfig
	BrandEdit       *brandEditData
	SiteBg          template.CSS // public body arka planı (boşsa varsayılan ana-bg.svg)
	DesignCSS       template.CSS // public ölçü/kenarlık token'ları (:root)
	// GA4Measurement doğrulanmış GA4 ölçüm kimliğidir (G-XXXX). Boşsa public
	// sayfa hiçbir üçüncü taraf izleme kodu basmaz — bkz. ga4.go.
	GA4Measurement string
	// GVerification Search Console site doğrulama kodudur (meta etiketi).
	// Panelden girilir; boşsa hiç basılmaz.
	GVerification string
	// AdsPublisher AdSense yayıncı kimliğidir (ca-pub-…). Panelden girilir;
	// boşsa hiç basılmaz (bkz. adsPublisher).
	AdsPublisher string
	// Scripts sayfaya özel ek betikler (admin). Ağır paketler —
	// editor.js, manset.js — yalnız ihtiyaç duyan sayfada yüklenir;
	// bkz. adminassets.go.
	Scripts    []string
	PageLayout *pageLayoutView // sayfaya özel sütun şablonu
	User       *auth.User
	Flash      *flashMsg
	AI         *aiEditorView

	// Public üye oturumu (header simgesi + üyelik sayfaları).
	MemberLoggedIn bool
	MemberName     string
	MemberInitial  string
	MemberURL      string
	// MemberStaff üye editoryal personel mi (yönetici/editör). Profil
	// simgesi artık herkese /hesabim'e gider; personel için yönetim paneli
	// kısayolu hesap sayfasında ve çekmecede gösterilir.
	MemberStaff bool
	// UyeMode/… üyelik form sayfaları (giriş, kayıt, şifre sıfırlama).
	UyeMode       string
	UyeRedirect   string
	UyeError      string
	UyeInfo       string
	UyeOK         bool
	UyeName       string
	UyeToken      string
	UyeFormTS     string
	GoogleEnabled bool
	// UyeKayitAcik: kayıt kapalıysa giriş sayfasındaki "Hemen kayıt olun"
	// bağlantısı çizilmez (tıklayan okuyucu boş bir "kayıtlar kapalı"
	// sayfasına düşüyordu).
	UyeKayitAcik bool
	// Hesap merkezi (/hesabim) verisi + header bildirim zili.
	Hesap        *hesapData
	MemberUnread int
	// Abonelik/fiyat sayfası verisi.
	Abonelik *abonelikData

	Preview bool
	// RevisionDiff içerik sürüm karşılaştırması.
	RevisionDiff []revisionDiffLine
	Home         *homeData
	Article      *articleData
	Category     *categoryData
	// Yazar / editör sayfaları (Faz 51).
	People         []repo.Person
	Person         *repo.Person
	PersonContents []repo.Content
	PersonTotal    int
	PersonPage     int
	PersonHasPrev  bool
	PersonHasNext  bool
	PersonPrevPage int
	PersonNextPage int
	PeoplePage     bool
	PersonDetail   bool
	PersonPanel    bool
	AdminPeople    []repo.Person
	EditPerson     *repo.Person
	EditAvatarID   int64
	PersonMedia    []repo.Media
	Page           *pageData
	// Iletisim yönetim ekranı verisi (Faz 145).
	Iletisim  *contactAdminData
	Dash      *dashData
	Contents  *contentsData
	Form      *formData
	Cats      *catsData
	Mahrets   *mahretsData
	Media     *mediaData
	Reports   *reportsData
	Yorumlar  *commentsData
	Tags      *tagsData
	Ceviriler *cevirilerData
	AISohbet  *sohbetData
	Ayar      *settingsData
	Manset    *mansetData
	Bot       *botData
	Siteler   *sitesData
	Kumeler   *kumelerData
	Servis    *servicesData
	Hava      []WeatherS
	NamazList []NamazS
	Imsakiye  []repo.NamazVakit
	Eczane    *repo.EczaneSehir
	Borsa     *repo.BorsaData
	// Hisse "Tüm Hisseler" sayfası (arama + A-Z + tablo).
	Hisse *hisseSayfa
	// Kripto Para modülü + /borsa/kripto sayfası (istek başına bir kez).
	Kripto *repo.KriptoData
	// Kur Çevirici modülü için kur tablosu (istek başına bir kez).
	KurTablo *repo.KurTablosu
	// KriptoSayfa /borsa/kripto sayfası görünümü.
	KriptoSayfa *kriptoSayfa
	// Hisseler modülü: BIST hisse fiyatları (önbellekten; istek başına bir kez).
	Hisseler *repo.HisseData
	Fikstur  *repo.FiksturData
	Puan     *repo.StandingsData
	// Fikstür sayfası (hafta bazlı)
	Standings    []repo.StandingRow
	FxMeta       repo.FiksturMeta
	FxRound      repo.FixtureRound
	FxRounds     []int
	Leagues      []repo.League
	CurLeague    repo.League
	Astro        *repo.AstrolojiData
	Cities       []repo.City
	Arama        []repo.Content
	Akis         *feedData
	Query        string
	AMPCanonical string
	// Canonical, sayfanın kendi (dil-farkında) mutlak adresidir; yazı
	// sayfalarında <link rel="canonical"> olarak basılır.
	Canonical string
	// Hreflangs sayfa başına basılan <link rel="alternate" hreflang>
	// etiketleri. Yalnız GERÇEKTEN çevirisi/içeriği olan diller listelenir.
	Hreflangs []langLink
	// I18nJSON istemci tarafında (JS ile üretilen metinler için) kullanılan
	// çeviri sözlüğüdür. Şablon <script> içine basar; app.js okur.
	I18nJSON template.JS
	// GoogleSourceURL, Google News'te siteyi "kaynak" ekleme bağlantısıdır:
	// https://www.google.com/preferences/source?q=<site>  (Faz 20)
	//
	// template.URL ile işaretlenir; html/template normalde URL bağlamındaki
	// değerleri kaçış uygular ve ":" ile "/" işaretleri %3A/%2F biçimine
	// dönüşür. Buradaki adres sunucu yapılandırması/host bilgisinden türer
	// (kullanıcı girdisi değildir), bu yüzden güvenli kabul edilir.
	GoogleSourceURL template.URL
	TOTPSecret      string
	TOTPEnabled     bool
	Members         []repo.MemberRow
	Digest          *digestData
	Community       *communityData
	Feedback        *feedbackData
	Assistant       *assistantData
	Ads             *adsData
	Tasima          *tasimaData
	Seo             *seoData
	Bekleyen        *pendingData
	Builder         *builderData
	Blocks          []homeBlock
	// PageTexts sayfa şablonu metin override'larının çözümlenmiş hali
	// (Faz 142). Anahtar → metin; şablon `index .PageTexts "..."` ile okur.
	PageTexts map[string]string
	// HasBodyBlock yazı sayfasında görünür "yazi_govde" bloğu var mı
	// (Faz 142). True ise sabit gövde basılmaz; gövdeyi blok taşır.
	HasBodyBlock bool
	// CatRows kategori gövdesinin görsel satırları (bağımsız modüller).
	CatRows        []catRow
	EtkinlikPanel  *eventsData
	Etkinlik       []repo.Event
	AnketPanel     *pollsData
	Anketler       []pollView
	Bulten         *bultenData
	GazetelerPanel *newspapersData
	Gazete         *gazeteViewData
	GazetePublic   *gazetePublicView
	// Admin görünüm/ marka kiti (Faz A).
	Gorunum      *gorunumView
	GorunumFonts []string
	AdminCSS     template.CSS
	AdminHNav    bool
	// Tercihler kullanıcı arayüz tercihleri (Faz 173, A-3): JSON belgesi.
	// admin_head içinde <meta> olarak gömülür; istemci localStorage
	// önbelleğini bununla tohumlar.
	Tercihler      string
	SlotHTML       template.HTML
	PageHead       template.HTML
	Gazeteler      []repo.NewspaperPage
	GazeteTarih    string
	GazeteTarihler []string
	FilmlerPanel   *filmsData
	Filmler        []repo.Film
	FilmHafta      string
	FilmHaftalar   []string
	IlanlarPanel   *noticesData
	Ilanlar        []repo.OfficialNotice
	IlanTarih      string
	IlanTarihler   []string
	// Resmî ilan filtreleri ve tek ilan detayı.
	// IlanTarih boşsa liste tüm tarihleri kapsıyordur; başlıkta tek bir gün
	// göstermek yanıltıcı olacağı için şablon buna bakar.
	IlanKurumlar []string
	IlanSehirler []string
	IlanFiltre   noticeFilterView
	Ilan         *repo.OfficialNotice
	Adsense      *adsenseData
	Backlink     *backlinkData
	Paylasim     *shareData
	Indexleme    *indexingData
	Push         *pushData
	Sites        []repo.Site
	CurSite      *repo.Site

	Embed   bool         // ?embed=1: yönetim kabuğu (menü/üst bar) gizlenir, iframe içine gömülür
	Ikonlar *ikonlarData // İkon kütüphanesi galerisi (/admin/ikonlar)

	// Toplu AI ile yeniden yazım paneli (Faz 204).
	Yeniden *yenidenData

	// Piyasa Kur & Borsa modülü göstergeleri (önbellekten; istek başına bir
	// kez okunur, aynı sayfadaki birden çok modül paylaşır).
	Piyasa *piyasa.Sonuc

	tBundle *i18n.Bundle
	tLang   string
}

// T sözlük çevirisi: yerelleştirme anahtarını mevcut dile göre döndürür.
func (d *tmplData) T(key string) string {
	if d.tBundle == nil {
		return key
	}
	return d.tBundle.T(d.tLang, key)
}

type homeData struct {
	// SiteID blok verisi kurulurken kullanılan site (kategori sekmeleri gibi
	// blok bazlı sorgular için).
	SiteID        int64
	Breaking      []repo.Content
	Lead          []repo.Content
	Recent        []repo.Content
	Slides        []repo.Content
	TopRead       []repo.Content
	GalleryPhotos []repo.Content
	GalleryVideos []repo.Content
	CatSections   []catSection
	// TabCatsSelected: sekme kategorileri yönetimden seçildiyse blok
	// "count" değeri kırpma yapmaz (seçim ne ise o kadar sekme çıkar).
	TabCatsSelected bool
	// CatBlocks: haberpi ana sayfa panel blokları (kategori başına 1 büyük + 4 kart)
	CatBlocks []catBlock
	// catCache aynı sayfadaki kategori sekmeli blokların sorgularını paylaşır.
	catCache map[string][]catSection
	// catItemsCache blok bazında seçilen kategorilerin içeriklerini sayfa
	// içinde bir kez çeker (aynı kategoriyi kullanan bloklar paylaşır).
	catItemsCache map[string][]repo.Content
	Categories    []repo.Category
	Trends        []repo.CatTrend
	BreakingLen   int
	FilterName    string
	Hava          *WeatherS
	Kurlar        []repo.Kur
	KurDemo       bool
	JSONLD        template.JS
}

// catSection kategori sekmeli bölüm için (ad + slug + içerikler).
type catSection struct {
	Name  string
	Slug  string
	Items []repo.Content
	// Pages sekme içi sayfalama: her sayfa 1 büyük + (count-1) küçük haber.
	// Oklar 5'erli (blok "count" değeri kadar) sonraki gruba geçer.
	Pages [][]repo.Content
}

// catBlock haberpi tarzı kategori paneli: renkli etiket + 1 büyük kart + 4 kart.
// Renk kategoriye göre belirlenir (haberpi: Güncel yeşil, kalanlar kırmızı).
type catBlock struct {
	Name  string
	Slug  string
	Tone  string // "yesil" | "kirmizi"
	Duzen string // "iki" | "mozaik" | "buyuk"
	Lead  repo.Content
	Items []repo.Content
}

// commentTreeNode yorum listesinde kök yorum + altındaki yanıtlar (Faz 36).
type commentTreeNode struct {
	C       repo.Comment
	Replies []repo.Comment
}

type articleData struct {
	Content repo.Content
	HTML    template.HTML
	Rozets  []rozet
	Links   []langLink
	// Hreflangs sayfa başındaki <link rel="alternate" hreflang> etiketleri
	// (mutlak adresler + x-default). Çeviri yoksa boştur.
	Hreflangs    []langLink
	Related      []repo.Content
	Fields       repo.ContentFields
	Comments     []repo.Comment
	CommentCount int
	CanComment   bool
	// CommentTree yorumları kök + yanıt zincirleriyle sıralı tutar (Faz 36).
	CommentTree []commentTreeNode
	Hits        int64
	CalcJSON    template.JS
	Locked      bool
	FreeHTML    template.HTML
	// RequiredTier kilitli içerik için gereken katman adı (ör. "Gümüş").
	RequiredTier string
	AMPCanonical string
	JSONLD       template.JS
	Tags         []string
	ReadingMin   int
	// Sağ sütun widget'ları (haberpi makale düzeni)
	SideLive   []repo.Content // canlı yayın / son video
	SidePick   []repo.Content // "İlginizi Çekebilir"
	SideTrend  []repo.Content // "Trend Haberler"
	SideLatest []repo.Content // "Son Haberler"
	// RelatedVideo yazının gövde sonuna basılan "ÖNERİLEN VİDEO" kartı (Faz 12).
	RelatedVideo *repo.Content
	// NextContent yazının bitiminde "SIRADAKİ HABER" bandı (yayın sırasına göre sonraki).
	NextContent *repo.Content
	// PrevContent yazının bitiminde "ÖNCEKİ HABER" bandı (yayın sırasına göre önceki) (Faz 14).
	PrevContent *repo.Content
	// CatBadges kategori id -> ad/slug. Önerilen video kartı ve komşu haber
	// bandlarında kategori rozeti basmak için (Faz 16).
	CatBadges map[int64]CatBadge
	// ShareBtns kanal paylaşım düğmeleri (Faz 44). Href'ler Go'da QueryEscape'li
	// kurgulanır ve template.URL olarak verilir — yoksa html/template href içinde
	// urlq çıktısını yeniden %-kodlayıp (%253A) bağlantıyı bozardı.
	ShareBtns []shareLink
	ShareURL  string
	// RelatedVideoSure önerilen videonun süresi (mm:ss); boş = bilinmiyor (Faz 17).
	RelatedVideoSure string
	// CatFollowed oturumlu kullanıcı yazının kategorisini takip ediyor mu (kırıntı yolu + düğmesi).
	CatFollowed bool
	// Bookmarked oturumlu kullanıcı yazıyı kaydettirmiş mi (görsel üstü kaydet düğmesi).
	Bookmarked bool
	// BIK, Basın İlan Kurumu tebliğinin içerik sayfalarında aradığı meta
	// değerleri (bik.go). Kurum uyumu otomatik teknik kontrolle ölçüyor.
	BIK bikMeta
	// Author / Editor yazı künyesi (başlık üstü ve alt yazar kutusu).
	Author *repo.PersonRef
	Editor *repo.PersonRef
	// BreadcrumbJSONLD yazı sayfası BreadcrumbList yapısal verisi.
	BreadcrumbJSONLD template.JS
}

// shareLink tek kanal paylaşım düğmesi.
type revisionDiffLine struct {
	Kind string
	Text string
}

type shareLink struct {
	Label string
	Icon  string
	// IconSVG marka ikonunun inline SVG'sidir (Material Symbols yerine).
	IconSVG template.HTML
	Href    template.URL
	Title   string
}

// CatBadge bir kategori rozetidir (ad + kategori bağlantısı için slug).
type CatBadge struct {
	Name string
	Slug string
}

type categoryData struct {
	Title string
	Slug  string
	Items []repo.Content
	// H1 / CrumbTitle sayfa metni override'larından gelir (Faz 142).
	// H1 başlık metnidir (ör. "Ekonomi Gündemi"); CrumbTitle kırıntı yolunda
	// HER ZAMAN gerçek kategori adını taşır. AllText "Tümü" sekmesi,
	// EmptyTitle/EmptyText boş kategori kutusunun metinleridir.
	H1         string
	CrumbTitle string
	AllText    string
	EmptyTitle string
	EmptyText  string
	// View liste görünümü şablonu (Faz 9): klasik | liste | izgara | mozaik
	View string
	// Moduler kategori gövdesinde govde zonlu blok var mı (Faz 10); true ise
	// sabit cat_liste yerine govde blokları render edilir.
	Moduler bool
	// sayfalama (haberpi: kırmızı haplar + "Sonraki →")
	Page    int
	PerPage int
	HasPrev bool
	HasNext bool
	// Hiyerarşi: alt kategori sayfasında üst kategori ve kardeş sekmeler.
	ParentName string
	ParentSlug string
	Tabs       []repo.Category
	AllURL     string
	// Intro üst kategori sayfasında H1 altında gösterilen kısa SEO metni.
	Intro string
	// Kategori gövdesi manşet düzeni (kategori_liste bloğu):
	//   klasik = 1 büyük + 2 yan manşet
	//   portre = 5 slaytlı dikey sol manşet + 2 yan manşet
	HeroStyle string
	HeroCount int
	HeroWidth int
	HeroRatio string
	ListCols  int
	// CatSlotClass bağımsız kategori modülünün satır yuvasıdır
	// (cat-slot-left / cat-slot-center / cat-slot-right). RailStyle seçimli
	// manşet kartlarının görünümüdür (overlay / yatay).
	CatSlotClass string
	RailStyle    string
	// HideBadge kategori_liste kartlarındaki kategori etiketini görsel olarak
	// gizler; etiket metni SEO/erişilebilirlik için DOM'da kalır.
	HideBadge bool
	// CardClass / CardCSS paylaşımlı kart stili (Bloomberg vb.) — kategori
	// ızgarasındaki kartlara uygulanan sınıf ve renk değişkenleri.
	CardClass string
	CardCSS   template.CSS
	// Kategori manşet slider'ı (portre/üçlü) görünümü: slider modülündeki
	// başlık arkası tasarımı, sayı göstergesi çerçevesi ve gölge seçenekleri.
	// HeroOverlayStyle başlık arkası katmanının CSS'idir (renk/geçiş,
	// saydamlık, yükseklik, bulanıklık). HeroTitleStyle başlık yazı stili,
	// HeroPagerStyle sayı göstergesinin boyut/rengi (CSS değişkenleri).
	HeroOverlayStyle template.CSS
	HeroTitleStyle   template.CSS
	HeroPagerStyle   template.CSS
	HeroPagerShape   string
	HeroPagerPos     string
	HeroShadow       bool
	// Akış vitrini (coinhit/enerji modeli): ilk 4 haber 4'lü kart ızgarası,
	// kalanı 3 sütun × 4 kartlık "enerji" bloklarıyla akar. FlowBatches her
	// blok için [sütun][kart] biçiminde hazır kartları taşır.
	FlowBatches [][][]akisCard
	FlowNext    int  // bir sonraki yüklemenin kategorideki mutlak sırası
	FlowLimit   int  // akıştaki toplam haber sınırı (manşet + kartlar dahil)
	FlowPer     int  // her yüklemede eklenecek haber sayısı
	FlowAuto    bool // kaydırınca otomatik yükleme
	FlowMore    bool // yüklenecek haber kaldı mı
	// AkisHero akışın üstündeki manşet alanını (portre/üçlü/klasik) gösterir.
	AkisHero bool
	// FlowFontStyle tüm akış kartlarının (ilk 4 kart + enerji blokları) yazı
	// ailesini tek CSS değişkeniyle ayarlar (--ak-font). Boşsa coinhit
	// varsayılanı (serif) kullanılır.
	FlowFontStyle template.CSS
}

// akisCard akış vitrinindeki tek kart: içerik + enerji düzeni kart tipi.
// tip5 = kare görsel + üstte taşan beyaz başlık kutusu, tip2 = solda görsel,
// tip4 = sağda görsel, tip1 = yalnız metin (coinhit referansı).
type akisCard struct {
	C         repo.Content
	Type      string
	LangBase  string
	HideBadge bool
}

type feedbackData struct {
	Items   []repo.Feedback
	Counts  map[string]int
	Status  string
	Logs    []repo.AssistantLog
	AsstCnt int
}

type assistantData struct {
	ContextTitle string
	ContextBody  string
	Enabled      bool
}

type communityData struct {
	Bookmarks     []repo.Content
	Notifications []repo.Notification
	Follows       []repo.Follow
	Leaderboard   []repo.UserPoints
	Definitions   []repo.Achievement
	Points        *repo.UserPoints
	Unread        int
}

type digestData struct {
	Counts   map[string]int
	FormPath string
	Done     bool
	DoneMsg  string
	Day      *repo.Digest
	Digests  []repo.Digest
	Archive  []repo.Digest
	Items    []repo.Content
	Base     string
	AudioURL string
}

type pageData struct {
	Title string
	HTML  template.HTML
	// HasBodyBlock görünür "sayfa_govde" bloğu varsa true olur; o zaman
	// sabit gövde basılmaz, gövdeyi blok taşır (Faz 143).
	HasBodyBlock bool
	// Contact iletişim sayfasında form + iletişim kartlarını basar (Faz 145).
	Contact *contactView
	// Flash iletişim formunun gönderim sonucu (ok/error).
	FlashKind string
	FlashText string
}

type dashData struct {
	Stats       []stat
	Types       []typeOpt
	Recents     []repo.Content
	RecHits     map[int64]int64
	TopRead     []repo.Content
	TopHits     map[int64]int64
	Week        []weekBar
	Hits        int64
	Revenue     float64
	RPM         float64
	Subs        int
	FeedbackNew int
	SearchOK    bool
	SiteName    string
	Days        int
	Sites       []repo.SiteStat
	MultiSite   bool
	// Redizayn (Faz B): otomasyon izleme, aktivite akışı, başlangıç listesi.
	Auto     []autoRow
	Activity []repo.LogRow
	Onboard  []onboardStep
	Backup   string
}

// autoRow otomasyon izleme satırı (izlenebilir otomasyon paneli).
type autoRow struct {
	Icon  string
	Name  string
	State string // ok | warn | danger | ""
	Info  string
}

// onboardStep başlangıç kontrol listesi adımı.
type onboardStep struct {
	Text string
	Done bool
	Link string
	CTA  string
}

type weekBar struct {
	Label string
	Count int
	Pct   int
}

type stat struct {
	Label string
	Value string
	Link  string
	Icon  string
	Tone  string // "", ok, warn, danger
	Spark string // sparkline polyline noktaları (bos = yok)
}

// pendingData panel üst şeridindeki bekleyen iş sayaçları.
type pendingData struct {
	Comments int
	Feedback int
	Drafts   int
	Trend    int
}

// Total bekleyen toplam iş.
func (p *pendingData) Total() int {
	if p == nil {
		return 0
	}
	return p.Comments + p.Feedback + p.Trend
}

type typeOpt struct {
	Key   string
	Label string
}

// catOpt kategori seçenekleri (video seçicideki kategori filtresi — Faz 25).
type catOpt struct {
	ID   int64
	Name string
}

type statusTab struct {
	Code   string
	Label  string
	Count  int
	Active bool
	// Href durum + filtreli sekme adresi (Faz 30). handler'da QueryEscape'li
	// kurgulanır; template.URL — yoksa html/template href'teki &'i %26 yapıp
	// filtreleri sekme değişince kaybettirirdi.
	Href template.URL
}

// contentRowView, liste satırı partial'ının ihtiyaç duyduğu içerik + türetilmiş
// alanları taşır (HTMX satır/panel yenilemesi için).
type contentRowView struct {
	Content repo.Content
	Yayin   string
	Hit     int64
	// Diller bu makalenin dil sürümleridir (Faz 192). Liste satırında küçük
	// etiketler olarak basılır; etikete tıklamak o dilin editörünü açar.
	Diller []repo.DilSurum
}

type contentsData struct {
	Items    []repo.Content
	Rows     []contentRowView
	Statuses []statusTab
	Status   string
	Q        string
	Total    int
	// Categories toplu "kategori taşı" seçimi için (Faz 26).
	Categories []repo.Category
	// Faz 30 — liste filtreleri: tip, kategori, tarih (son N gün) + sekme
	// linklerine eklenecek filtre sorgusu (TabURL).
	Tip        string
	Kategori   int64
	Gun        int // 0 = tümü; 7/14/30…
	Types      []repo.ContentType
	KategoriAd string
	// Faz 47 — özel tarih aralığı (YYYY-MM-DD; yankılanır).
	Baslangic string
	Bitis     string
	// Faz 126 — ses/röportaj filtresi (var|bekliyor|basarisiz).
	SesFiltre string
	// Triyaj (U1): liste sıralaması ve AI skor eşiği. Sıralama repo'da
	// beyaz listeden geçer; burada ham tutulur (form durumu için).
	Sirala  string
	MinSkor string
	// Eksenler listelenen içeriklerin yayın önceliği eksenleridir
	// (spec 11 §5c). Ayrı sorgudan gelir; contentCols paylaşımlı olduğu
	// için oraya sütun eklemek tüm çağıranları etkilerdi.
	Eksenler map[int64]repo.IcerikEksen
	// Faz 48 — kaydedilmiş liste görünümleri (filtre ön ayarları).
	Views        []ListView
	CurrentQS    string // durum+tip+kategori+gun+baslangic+bitis+q (kaydet formu için)
	CurrentDurum string
	// Faz 42 — liste sütunları: yayınlanma/tarih görüntüsü + okunma (batch).
	Yayinlar map[int64]string
	Hits     map[int64]int64
	// Faz 192 — dil filtresi (?dil=en) ve kök dil (temsilci satır dili).
	DilFiltre string
	KokDil    string
	// TumuHref "Tümü" sekmesi — filtreleri korur (Faz 30).
	TumuHref template.URL
	// Faz 31 — sayfalama (filtrelerle uyumlu): durum+tip+kategori+gun+q+sayfa.
	Sayfa       int
	ToplamSayfa int
	PrevHref    template.URL
	NextHref    template.URL
	Pages       []pagerItem
}

// pagerItem sayfalama çubuğundaki tek öğe (sayı ya da "…").
// ListView kaydedilmiş bir filtre görünümü (Faz 48).
type ListView struct {
	Ad string
	QS string
}

type pagerItem struct {
	Label  string
	Href   template.URL
	Active bool
	Dots   bool
}

type mediaOption struct {
	ID       int64
	Filename string
	Selected bool
	ThumbURL string
	Alt      string
	Caption  string
	Title    string
}

type mediaGroup struct {
	Label string
	Year  int
	Month int
	Items []repo.Media
}

type mediaSlot struct {
	Name    string // form alanı adı
	Help    string
	Options []mediaOption
}

type aiActionView struct {
	ID          string
	Label       string
	Group       string
	Description string
	Icon        string
	Research    bool
}

type formData struct {
	Content          repo.Content
	IsNew            bool
	TypeKey          string
	Types            []typeOpt
	Categories       []repo.Category
	CategorySelected int64
	Mahrets          []repo.Mahret
	MahretSelected   int64
	Clusters         []repo.Cluster
	ClusterSelected  int64
	TagsText         string
	DetailText       string
	DetailJSON       string
	MediaSlots       []mediaSlot
	Media            []repo.Media
	Rozets           []formRozet
	Langs            []langLink
	// Faz 190 — dil satırlarının durumu (yok/taslak/yayinda) ve çeviri bağlamı.
	LangRows     []templkit.IcerikLang
	CeviriSayisi int
	// Çeviri düzenleniyorsa kaynak metin (karşılaştırma için).
	KaynakBaslik    string
	KaynakSpot      string
	KaynakGovdeHTML string
	SEO             repo.SEOResult
	SEOText         string
	TitleLen        int
	PublishAtValue  string
	Sources         []repo.ContentSource
	IsYMYL          bool
	YMYLDurumu      string
	YMYLUzman       string
	OrigIndex       int
	// Yazar / editör ataması (Faz 51).
	People         []repo.Person
	AuthorSelected int64
	EditorSelected int64
	// Sesli haber (TTS) durumu.
	AudioURL        string
	AudioStatus     string
	AudioSeconds    float64
	HesaplayiciSema string
	Fields          repo.ContentFields
	// Önerilen Video (Faz 12b): video seçici listesi + mevcut elle bağlı id.
	Videos         []repo.Content
	RelatedVideoID int64
	// Video süresi (Faz 17): düzenlenen içeriğin süresi (mm:ss) ve seçicideki
	// her video için süre (id -> mm:ss). Boş = bilinmiyor.
	VideoSure    string
	VideoSureMap map[int64]string
	// VideoCats video seçicideki kategori filtresi (Faz 25): video listesinde
	// geçen kategoriler (yinelenmeden, ada göre sıralı).
	VideoCats []catOpt
	// TrendSuggestions editörde etikete eklenebilecek güncel trend terimleri.
	TrendSuggestions []repo.TrendTermRow
	// AIActions seçili metin menüsündeki 11 eylemdir.
	AIActions []aiActionView
	// Audio içeriğe bağlı private röportaj kayıtları (iş + transkript).
	Audio []repo.ContentAudio
	// Kalite ölçüm raporu (Faz 115); yeni içerikte ölçülmez.
	Kalite quality.Report
	// KaliteChecked raporun hesaplanıp hesaplanmadığı.
	KaliteChecked bool
}

type formRozet struct {
	Code string
	On   bool
}

type catsData struct {
	Items   []repo.Category
	Parents []repo.Category // üst kategoriler (alt kategori seçimi için)
	Presets []categoryPreset
}

type mahretsData struct {
	Items []repo.Mahret
}

type mediaData struct {
	Items      []repo.Media
	Groups     []mediaGroup
	Categories []repo.MediaFacet
	Periods    []repo.MediaPeriod
	Years      []int
	Query      string
	Category   string
	Year       string
	Month      string
	Page       int
	Pages      int
	PerPage    int
	Total      int
	Storage    string
	HasPrev    bool
	HasNext    bool
}

type reportsData struct {
	MostRead   []repo.HitRow
	AIUsage    []repo.UsageRow
	MostJSON   template.JS
	Revenue    []repo.RevenueRow
	RPMDefault float64
	// Faz 37 — yorum + spam istatistikleri.
	MostCommented []repo.HitRow
	Yorum         repo.CommentDurum
	BotTrapTotal  int64
	BotTrap24     int64
	BlockedIPs    int64
}

type commentsData struct {
	Items    []repo.CommentRow
	Tab      string
	Pending  int
	Approved int
	// Ara yorum listesindeki metin araması (Faz 168).
	Ara string
}

// ---- Şablon işlevleri ----

var trMediaMonths = [...]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}

func humanBytes(n int64) string {
	if n <= 0 {
		return ""
	}
	if n >= 1<<30 {
		return fmt.Sprintf("%.2f GB", float64(n)/(1<<30))
	}
	if n >= 1<<20 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
	if n >= 1<<10 {
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func mediaMonthLabel(year, month int) string {
	if month < 1 || month > 12 {
		return ""
	}
	return trMediaMonths[month-1] + " " + itoa(year)
}

// gazeteTarih ISO tarihi ("2026-11-02") Türkçe uzun biçime çevirir
// ("2 Kasım 2026"). Boş/geçersiz değer boş döner — modül tarih satırını
// o zaman hiç basmaz.
func gazeteTarih(iso string) string {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(iso))
	if err != nil {
		return ""
	}
	return itoa(t.Day()) + " " + trMediaMonths[int(t.Month())-1] + " " + itoa(t.Year())
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// yaziURL bir içeriğin KANONİK adresini üretir.
		//
		// NEDEN: eski şablonlar `/{slug}-{id}` bağlantısı basıyordu; o adres
		// 301 ile kanonik adrese gidiyor. İç bağlantıların her biri bir
		// yönlendirme sıçraması demek: tarama bütçesi boşa gider, kullanıcı
		// bir tur fazla bekler. Yeni şablonlar zaten makaleURL kullanıyor;
		// bu işlev eski şablonların da aynı adresi basmasını sağlar.
		"yaziURL": makaleURL,
		"formatTime": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Local().Format("02.01.2006 15:04")
		},
		// gazeteTarih "2026-11-02" → "2 Kasım 2026". Gazete manşetleri
		// modülünde tarih Türkçe okunur biçimde gösterilir; ISO tarih
		// okuyucuya yabancı kalıyordu.
		"gazeteTarih": gazeteTarih,
		"orStr": func(a, b any) string {
			as := anyStr(a)
			if as != "" {
				return as
			}
			return anyStr(b)
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		// quoteTime alıntı kaynağının saniyesini mm:ss biçiminde gösterir.
		// 0 ms "0:00" olmalı: boş etiket kaynağın ilk saniyede olduğunu
		// gizlerdi.
		"quoteTime": func(ms int64) string {
			if ms <= 0 {
				return "0:00"
			}
			return audioTimeLabel(ms)
		},
		// odd: şablonda sıra numarası tek mi? (GASTRO kartlarında başlık
		// konumunu dönüşümlü yapmak için — Habertürk birebir.)
		"odd": func(n int) bool { return n%2 != 0 },
		// catHeroEnd manşet düzeninin tükettiği haber sayısını döndürür;
		// akış vitrini manşetten sonra başlar.
		"catHeroEnd": catHeroEnd,
		// piyasaNum Kur & Borsa modülünde fiyatı Türkçe biçimde yazar
		// (11.151,03). Şablonda printf ile yapılamaz: binlik ayırıcı ve
		// ondalık virgül gerekir.
		"piyasaNum": piyasaNum,
		// piyasaChg yüzde değişimi işaretli yazar (+0,05 / -0,15).
		"piyasaChg": piyasaChg,
		// hisseYon fiyat yönünü CSS sınıfına çevirir (up | down | flat).
		"hisseYon": hisseYon,
		// hssOkFor Hisseler modülünün yön göstergesi (kb_ok, hss- önekiyle).
		"hssOkFor": hssOkFor,
		// hisseYonInt yönü ok çizimi sözleşmesine (-1/0/1) çevirir.
		"hisseYonInt": hisseYonInt,
		// ilkHarf metnin ilk harfini büyük döndürür (kripto ikonu yedeği).
		"ilkHarf": ilkHarf,
		// paraKisa büyük tutarları kısaltmayla yazar (12,3 b / 45,6 mn).
		"paraKisa": paraKisa,
		// aramaNorm arama için metni sadeleştirir (Türkçe duyarsız).
		"aramaNorm": hisseAramaNorm,
		// kbArrow Kur & Borsa modülünde ok tasarımını + yönü taşır.
		"kbArrow": kbArrowFor,
		// takimKisa puan şeridinde takım kısa kodunu üretir (GS, ASF…).
		"takimKisa": takimKisa,
		"sizeText": func(p *int64) string {
			if p == nil || *p <= 0 {
				return ""
			}
			return humanBytes(*p)
		},
		"dims": func(w, h *int) string {
			if w == nil || h == nil || *w <= 0 || *h <= 0 {
				return ""
			}
			return fmt.Sprintf("%d×%d", *w, *h)
		},
		"mmss": func(sec float64) string {
			if sec < 0 {
				sec = 0
			}
			t := int(sec + 0.5)
			return itoa(t/60) + ":" + fmt.Sprintf("%02d", t%60)
		},
		"safe": func(s string) template.HTML { return template.HTML(s) },
		"mediaThumbURL": func(id *int64) string {
			if id == nil {
				return "/media/none.svg"
			}
			return "/media/thumb/" + itoa64(*id)
		},
		"slice": func(args ...string) []string { return args },
		"hasStr": func(m map[string][]string, key, val string) bool {
			for _, v := range m[key] {
				if v == val {
					return true
				}
			}
			return false
		},
		"eventWhen": eventWhen,
		"agoTR":     agoTR,
		// noticeSlug resmî ilanın kalıcı adres parçasını üretir. Şablon
		// katmanı için burada tanımlı; NewServer'daki FuncMap de aynı
		// işlevi kullanır.
		"noticeSlug": noticeSlug,
		// catBadge kategori id'sine karşılık gelen rozeti döndürür (Faz 16).
		// İçerik alanı *int64 olduğu için şablon dizinleme yapamaz; bu
		// işlev işaretçiyi çözer ve bilinmeyen kategori için boş değer verir.
		"catBadge": func(m map[int64]CatBadge, id *int64) CatBadge {
			if m == nil || id == nil {
				return CatBadge{}
			}
			return m[*id]
		},
		// catCols kategori listesini iki dengeli sütuna böler. 14 kategori
		// 7 + 7; 7 üst kategori ise 4 + 3 olur (tek sütun görünmesin).
		"catCols": func(cats []repo.Category, per int) [2][]repo.Category {
			if per <= 0 {
				per = 7
			}
			var out [2][]repo.Category
			if len(cats) == 0 {
				return out
			}
			if len(cats) <= per {
				sol := (len(cats) + 1) / 2
				out[0] = cats[:sol]
				out[1] = cats[sol:]
				return out
			}
			// Dengeli böl: sol = ceil(n/2), sağ = kalan (en fazla per).
			sol := (len(cats) + 1) / 2
			if sol > per {
				sol = per
			}
			out[0] = cats[:sol]
			out[1] = cats[sol:]
			return out
		},
		// sureSec video süresini "3:42" biçiminde yazar; bilinmiyorsa boş döner
		// (Faz 21). Kart şablonları `{{with sureSec .VideoSeconds}}` ile
		// yalnız süresi bilinen videolarda rozet basar.
		"sureSec": func(p *int) string {
			if p == nil {
				return ""
			}
			return platform.FormatDurationTR(*p)
		},
		// withView bloğun kendi liste görünümünü (varsa) tmplData kopyasına
		// uygular — global kategori_view ayarını bu blok için geçersiz kılar
		// (Faz 11). Kopya kullanılır; paylaşılan d'ye mutasyon yapılmaz.
		"withView": func(d *tmplData, v string) *tmplData {
			if d == nil || d.Category == nil || v == "" {
				return d
			}
			if !validListView(v) {
				return d
			}
			c := *d.Category
			c.View = v
			nd := *d
			nd.Category = &c
			return &nd
		},
		// withCatBlock kategori_liste bloğunun görünüm + manşet/gövde
		// seçeneklerini kategori verisinin kopyasına uygular. Böylece aynı
		// kategori sayfasında farklı listeler farklı düzenlerle basılabilir.
		"withCatBlock": func(d *tmplData, b *homeBlock) *tmplData {
			if d == nil || d.Category == nil || b == nil {
				return d
			}
			c := *d.Category
			// Bağımsız kategori modülleri kendi haber aralıklarını taşır.
			// Akış modülü tam listeyi kullanır (AJAX ile devam eder).
			if b.CatItems != nil && b.Type != "kategori_akis" {
				c.Items = b.CatItems
			}
			if b.Type == "kategori_akis" {
				c.View = "akis"
			} else if validListView(b.ListView) {
				c.View = b.ListView
			} else if b.CatHeroStyle == "portre" || b.CatHeroStyle == "uclu" {
				// Portre/üçlü manşet, klasik listenin varyantlarıdır. Blokta
				// açıkça başka bir görünüm seçilmediyse genel kategori
				// görünümü (ör. izgara) bu bloğu ezmesin.
				c.View = "klasik"
			}
			c.HeroStyle = b.CatHeroStyle
			c.HeroCount = b.CatHeroCount
			c.HeroWidth = b.CatHeroWidth
			c.HeroRatio = b.CatHeroRatio
			c.ListCols = b.CatListCols
			c.CatSlotClass = b.RowSlotClass
			c.RailStyle = b.RailStyle
			c.HideBadge = b.HideBadge
			// Paylaşımlı kart stili: kategori ızgarası + manşet kartları.
			c.CardClass = ""
			if b.CardStyle != "" {
				c.CardClass = "cs-scope cs-" + b.CardStyle
			}
			c.CardCSS = b.CardCSS
			// Manşet slider'ı görünümü: slider modülündeki başlık arkası,
			// gölge ve sayı göstergesi ayarları kategori manşetine taşınır.
			c.HeroOverlayStyle = b.OverlayStyle
			c.HeroTitleStyle = b.TitleStyle
			c.HeroShadow = b.HeroShadow
			c.HeroPagerShape = b.PagerShape
			if c.HeroPagerShape != "square" && c.HeroPagerShape != "outline" {
				c.HeroPagerShape = "round"
			}
			c.HeroPagerPos = b.PagerPos
			switch c.HeroPagerPos {
			case "left", "center", "right":
			default:
				c.HeroPagerPos = "right"
			}
			var pagerVars []string
			if b.PagerSize > 0 {
				pagerVars = append(pagerVars, "--hero-dot-size:"+px(b.PagerSize))
			}
			if b.PagerColor != "" {
				pagerVars = append(pagerVars, "--pager-color:"+b.PagerColor)
			}
			if len(pagerVars) > 0 {
				c.HeroPagerStyle = template.CSS(strings.Join(pagerVars, ";"))
			}
			if c.View == "akis" {
				c.AkisHero = b.AkisHero
				// Akış kartları tek yazı ailesi kullanır (sayfa/kategori
				// bazında değiştirilebilir; boşsa coinhit serif'i).
				if fam, ok := titleFontFamilies[b.AkisFont]; ok {
					c.FlowFontStyle = template.CSS("--ak-font:" + fam + ";")
				}
				applyAkisFlow(&c, b, d.LangBase)
			}
			nd := *d
			nd.Category = &c
			return &nd
		},
	}
}

// validListView blok liste görünümünün bilinen bir değer olup olmadığını söyler.
func validListView(v string) bool {
	switch v {
	case "klasik", "liste", "izgara", "mozaik", "akis":
		return true
	}
	return false
}

// akisColumnTypes enerji vitrininin sütun kart tipleri (coinhit referansı):
// her sütunda 4 kart; 1. ve 3. sütunda büyük kare kart başta.
var akisColumnTypes = [][]string{
	{"type5", "type2", "type1", "type4"},
	{"type1", "type4", "type5", "type2"},
	{"type5", "type2", "type1", "type4"},
}

// akisBatchSize bir enerji bloğunun kart sayısı (3 sütun × 4 kart).
const akisBatchSize = 12

// catHeroEnd kategori manşet düzeninin (portre/üçlü/klasik) tükettiği haber
// sayısını döndürür; akış vitrini bu kaydırdan sonra başlar. Manşet
// gösterilemiyorsa (yetersiz içerik) 0 döner.
//
//	portre = 5 slayt + 2 yan manşet (heroCount+2)
//	uclu   = 5 slayt + 3 sol + 2 sağ manşet (heroCount+5)
//	klasik = 1 büyük + 2 yan manşet (3)
func catHeroEnd(c *categoryData, n int) int {
	if c == nil {
		return 0
	}
	hc := c.HeroCount
	if hc < 2 {
		hc = 5
	}
	switch c.HeroStyle {
	case "uclu":
		if n > hc+4 {
			return hc + 5
		}
	case "portre":
		if n > hc+1 {
			return hc + 2
		}
	default:
		if n >= 3 {
			return 3
		}
	}
	return 0
}

// applyAkisFlow kategori akış vitrini verisini hazırlar: ilk 4 haber ayrı
// basılır, kalanı 12'lik enerji bloklarına bölünür ve yüklenecek sıradaki
// haberin mutlak konumu hesaplanır. Manşet alanı açıksa manşet haberleri
// akıştan önce tüketilir (üstte portre/üçlü manşet + 4'lü kart satırı).
func applyAkisFlow(c *categoryData, b *homeBlock, langBase string) {
	if c == nil {
		return
	}
	limit := b.AkisLimit
	if limit < 8 {
		limit = 60
	}
	per := b.AkisPer
	if per < akisBatchSize {
		per = akisBatchSize
	}
	c.FlowLimit = limit
	c.FlowPer = per
	c.FlowAuto = b.AkisAuto

	base := (c.Page - 1) * c.PerPage
	if base < 0 {
		base = 0
	}
	have := len(c.Items)
	end := limit
	if end > have {
		end = have
	}
	// Manşet + ilk 4 kart akıştan önce basılır; akış bundan sonra başlar.
	// Bağımsız kategori_akis modülünde başlangıç, modülün item_offset
	// değerinden (AkisStart) gelir; 4'lü ızgara ayrı bir modüldür.
	start := 0
	grid := 4
	if b.AkisStartSet {
		start = b.AkisStart
		grid = 0
	} else if c.AkisHero {
		start = catHeroEnd(c, have)
	}
	gridEnd := start + grid
	if gridEnd > end {
		gridEnd = end
	}
	if gridEnd < start {
		// Akış başlangıcı bu sayfanın dışında: bu sayfada ek blok yok.
		gridEnd = end
	}
	if gridEnd < end {
		c.FlowBatches = buildAkisBatches(c.Items[gridEnd:end], langBase, c.HideBadge)
	}
	c.FlowNext = base + end
	if c.FlowNext > limit {
		c.FlowNext = limit
	}
	// Yüklenecek haber var mı: sınır dolmadıysa ve ya sayfada kalan ya da
	// sonraki sayfada haber varsa.
	c.FlowMore = c.FlowNext < limit && (c.FlowNext < base+have || c.HasNext)
}

// buildAkisBatches içerikleri 12'lik enerji bloklarına böler. Dönüş:
// [blok][sütun][kart]. Boş kartlar (blok dolmadıysa) ID=0 ile gelir ve
// şablonda atlanır.
func buildAkisBatches(items []repo.Content, langBase string, hideBadge bool) [][][]akisCard {
	if len(items) == 0 {
		return nil
	}
	var out [][][]akisCard
	for start := 0; start < len(items); start += akisBatchSize {
		end := start + akisBatchSize
		if end > len(items) {
			end = len(items)
		}
		chunk := items[start:end]
		block := make([][]akisCard, len(akisColumnTypes))
		for col, types := range akisColumnTypes {
			cards := make([]akisCard, 0, len(types))
			for row, typ := range types {
				idx := col*4 + row
				var c repo.Content
				if idx < len(chunk) {
					c = chunk[idx]
				}
				cards = append(cards, akisCard{
					C: c, Type: typ, LangBase: langBase, HideBadge: hideBadge,
				})
			}
			block[col] = cards
		}
		out = append(out, block)
	}
	return out
}

// agoTR haberpi biçimindeki kısa Türkçe "ne zaman" metnini üretir:
// "az önce", "12 dakika", "2 saat", "dün", "3 gün", "2 ay", "1 yıl".
func agoTR(t *time.Time) string {
	if t == nil {
		return ""
	}
	d := time.Since(t.Local())
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "az önce"
	case d < time.Hour:
		return fmt.Sprintf("%d dakika", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d saat", int(d.Hours()))
	case d < 48*time.Hour:
		return "dün"
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%d gün", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%d ay", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%d yıl", int(d.Hours()/(24*365)))
	}
}

// orStr yardımcı: *string veya string'i güvenli okur.
func anyStrPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func anyStr(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case *string:
		if t == nil {
			return ""
		}
		return *t
	default:
		if s, ok := v.(fmt.Stringer); ok {
			return s.String()
		}
		return ""
	}
}

// baseView ortak alanları doldurur.
func (s *Server) baseView(lang, langBase, title string) *tmplData {
	bundle := s.i18n
	loc := bundle.Default(lang)
	base := langBase
	if base == "" {
		base = "/" + loc
	}
	if base == "/"+loc && loc != "" && !strings.HasPrefix(base, "/"+loc+"/") {
		base = "/" + loc
	}
	homeURL := "/"
	if loc != "" && loc != "tr" {
		homeURL = "/" + loc + "/"
	}
	// Site adı ve sloganı yerelleştirilebilir (Faz 189).
	//
	// Slogan Türkçe sabitti: /en sayfasının <title>'ı "TheBlok Haber
	// Portalı — Güncel Haber Portalı" olarak çıkıyordu (ölçüldü). Çeviri
	// `translations` tablosunda `site.name` / `site.suffix` anahtarıyla
	// tutulur; yoksa Türkçe değer korunur (geriye dönük uyumlu).
	siteAdi := s.siteName
	if loc != "tr" {
		if v := bundle.T(loc, "site.name"); v != "site.name" && strings.TrimSpace(v) != "" {
			siteAdi = v
		}
	}
	slogan := siteAdi + " — " + s.localeText(loc, "site.suffix", "Güncel Haber Portalı")
	metaTitle := slogan
	// Karşılaştırma ORİJİNAL site adıyla yapılır: yerelleştirilmiş ad
	// ("TheBlok") ile karşılaştırınca ana sayfada
	// "theblock.com.tr — TheBlok" gibi çift ad çıkıyordu (ölçüldü).
	if title != "" && title != s.siteName {
		metaTitle = title + " — " + siteAdi
	}
	megaDesign, catNavDesign := s.menuDesignValues(context.Background())
	return &tmplData{
		Lang:         loc,
		LangBase:     base,
		HomeURL:      homeURL,
		Title:        title,
		SiteName:     siteAdi,
		SiteSlogan:   slogan,
		MetaTitle:    metaTitle,
		MetaDesc:     slogan,
		Robots:       "index,follow",
		I18nJSON:     bundle.JSSozluk(loc, jsI18nAnahtarlari),
		Theme:        "light",
		Year:         strconv.Itoa(time.Now().Year()),
		RTL:          bundle.RTL(loc),
		MenuDesign:   megaDesign,
		CatNavDesign: catNavDesign,
		MenuBgCSS:    s.menuBgCSS(context.Background()),
		tBundle:      bundle,
		tLang:        loc,
		Locales:      bundle.Enabled(),
	}
}

// noticeFilterView public ilan listesinde seçili filtreleri şablona taşır:
// form alanlarının seçili kalması ve "filtreyi temizle" bağlantısının
// görünmesi için.
type noticeFilterView struct {
	Tarih string
	Kurum string
	Sehir string
	Arama string
	Aktif bool // herhangi bir daraltma uygulandı mı
	Tumu  bool // tüm tarihlerde arama
}
