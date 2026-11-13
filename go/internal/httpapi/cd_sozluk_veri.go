package httpapi

// Kripto sözlüğü verisi — ÜRETİLMİŞ DOSYA, elle düzenlemeyin.
//
// Kaynak: Ghost temasının assets/js/glossary-data.js dosyası
// (ghost.bitcosar.com dışa aktarması, 13 Kasım 2026). Üreten betik:
// ~/bitcosar-betikler/sozluk_cikar.py
//
// NEDEN GO KAYNAĞI: 55 terim ~14 KB; çalışma anında dosya okumak ya da
// veritabanı sorgusu yapmak bu boyut için gereksiz karmaşıklık. Veri
// derlemeye gömülü, sayfa sorgusuz çiziliyor.

// SozlukTerimi tek bir sözlük maddesi.
type SozlukTerimi struct {
	Terim    string // İngilizce ad (birincil anahtar)
	Turkce   string // Türkçe karşılık; yoksa boş
	Aciklama string
	Kategori string // Teknoloji | Finans | Piyasa
	Seviye   string // Başlangıç | Orta | İleri
	Slug     string // sayfa içi çıpa: /sozluk#<slug>
}

// sozlukTerimleri alfabetik DEĞİL, kaynak sırasında: giriş
// seviyesinden ileriye doğru ilerliyor. Sayfa kendi sıralamasını
// uyguluyor.
var sozlukTerimleri = []SozlukTerimi{
	{Terim: "Blockchain", Turkce: "Blok Zinciri", Kategori: "Teknoloji", Seviye: "Başlangıç", Slug: "blockchain",
		Aciklama: "İşlemlerin birbirine kriptografik olarak bağlı bloklar halinde, dağıtık bir ağdaki tüm katılımcılar tarafından tutulan, geriye dönük değiştirilemeyen kayıt defteri."},
	{Terim: "Bitcoin", Turkce: "", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "bitcoin",
		Aciklama: "2009'da Satoshi Nakamoto takma adıyla tanıtılan, arzı 21 milyonla sınırlı ilk ve en büyük piyasa değerine sahip kripto para birimi."},
	{Terim: "Ethereum", Turkce: "", Kategori: "Teknoloji", Seviye: "Başlangıç", Slug: "ethereum",
		Aciklama: "Akıllı kontratları ve merkeziyetsiz uygulamaları (dApp) çalıştırabilen, kendi yerel para birimi ETH olan programlanabilir blok zinciri platformu."},
	{Terim: "Altcoin", Turkce: "", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "altcoin",
		Aciklama: "Bitcoin dışındaki tüm kripto para birimlerini tanımlamak için kullanılan genel terim."},
	{Terim: "Wallet", Turkce: "Cüzdan", Kategori: "Teknoloji", Seviye: "Başlangıç", Slug: "wallet",
		Aciklama: "Özel anahtarları saklayan ve kripto varlıkları göndermek/almak için kullanılan yazılım veya donanım aracı."},
	{Terim: "Private Key", Turkce: "Özel Anahtar", Kategori: "Teknoloji", Seviye: "Başlangıç", Slug: "private-key",
		Aciklama: "Bir kripto cüzdanının sahibini kanıtlayan ve işlemleri imzalamak için kullanılan, asla paylaşılmaması gereken gizli kod."},
	{Terim: "Public Key", Turkce: "Genel Anahtar", Kategori: "Teknoloji", Seviye: "Başlangıç", Slug: "public-key",
		Aciklama: "Özel anahtardan türetilen ve başkalarının size kripto para gönderebilmesi için paylaşılabilen adres bilgisi."},
	{Terim: "Seed Phrase", Turkce: "Kurtarma Cümlesi", Kategori: "Teknoloji", Seviye: "Başlangıç", Slug: "seed-phrase",
		Aciklama: "Bir cüzdanın tüm özel anahtarlarını yeniden oluşturmaya yarayan, genellikle 12 veya 24 kelimeden oluşan yedekleme dizisi."},
	{Terim: "Exchange", Turkce: "Borsa", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "exchange",
		Aciklama: "Kullanıcıların kripto para birimlerini fiat para veya diğer kripto paralarla alıp satabildiği platform."},
	{Terim: "Market Cap", Turkce: "Piyasa Değeri", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "market-cap",
		Aciklama: "Dolaşımdaki toplam arzın güncel fiyatla çarpımıyla hesaplanan, bir kripto varlığın toplam piyasa değeri."},
	{Terim: "ATH", Turkce: "Tüm Zamanların Zirvesi", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "ath",
		Aciklama: "\"All-Time High\" kısaltması; bir varlığın bugüne kadar ulaştığı en yüksek fiyat seviyesi."},
	{Terim: "HODL", Turkce: "", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "hodl",
		Aciklama: "\"Hold\" kelimesinin yanlış yazımından doğan, kısa vadeli fiyat dalgalanmalarına aldırmadan varlığı uzun vadede elde tutma stratejisini ifade eden topluluk terimi."},
	{Terim: "FOMO", Turkce: "Kaçırma Korkusu", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "fomo",
		Aciklama: "\"Fear Of Missing Out\"; fiyat yükselirken trene binememe korkusuyla mantıksız, plansız alım yapma dürtüsü."},
	{Terim: "FUD", Turkce: "", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "fud",
		Aciklama: "\"Fear, Uncertainty, Doubt\"; piyasada korku ve belirsizlik yaymaya yönelik asılsız veya abartılı olumsuz haber/söylenti."},
	{Terim: "Whale", Turkce: "Balina", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "whale",
		Aciklama: "Bir kripto varlığın büyük miktarını elinde bulunduran ve alım-satımlarıyla piyasa fiyatını etkileyebilen yatırımcı."},
	{Terim: "Smart Contract", Turkce: "Akıllı Kontrat", Kategori: "Teknoloji", Seviye: "Orta", Slug: "smart-contract",
		Aciklama: "Belirli koşullar sağlandığında otomatik olarak çalışan, blok zincirinde saklanan, aracıya ihtiyaç duymayan kendiliğinden yürütülen kod."},
	{Terim: "Gas", Turkce: "İşlem Ücreti", Kategori: "Teknoloji", Seviye: "Orta", Slug: "gas",
		Aciklama: "Ethereum gibi ağlarda bir işlemi veya akıllı kontrat çağrısını gerçekleştirmek için doğrulayıcılara ödenen ücret."},
	{Terim: "DeFi", Turkce: "Merkeziyetsiz Finans", Kategori: "Finans", Seviye: "Orta", Slug: "defi",
		Aciklama: "Bankalar gibi geleneksel aracılar olmadan, akıllı kontratlar üzerinden borç verme, borçlanma ve takas gibi finansal hizmetler sunan ekosistem."},
	{Terim: "NFT", Turkce: "Değiştirilemez Token", Kategori: "Teknoloji", Seviye: "Orta", Slug: "nft",
		Aciklama: "\"Non-Fungible Token\"; her biri benzersiz olan, bir dijital veya fiziksel varlığın sahipliğini blok zincirinde temsil eden token türü."},
	{Terim: "DAO", Turkce: "Merkeziyetsiz Otonom Organizasyon", Kategori: "Teknoloji", Seviye: "İleri", Slug: "dao",
		Aciklama: "Kararların merkezi bir yönetim yerine, token sahiplerinin oylamasıyla ve akıllı kontratlar aracılığıyla alındığı organizasyon yapısı."},
	{Terim: "Staking", Turkce: "", Kategori: "Finans", Seviye: "Orta", Slug: "staking",
		Aciklama: "Proof-of-Stake ağlarında, ağın güvenliğine katkıda bulunmak için kripto varlıkları kilitleyerek karşılığında ödül kazanma yöntemi."},
	{Terim: "Yield Farming", Turkce: "Getiri Çiftçiliği", Kategori: "Finans", Seviye: "İleri", Slug: "yield-farming",
		Aciklama: "DeFi protokollerine likidite sağlayarak faiz, işlem ücreti payı veya yönetişim token'ı şeklinde getiri elde etme stratejisi."},
	{Terim: "Liquidity Pool", Turkce: "Likidite Havuzu", Kategori: "Finans", Seviye: "Orta", Slug: "liquidity-pool",
		Aciklama: "Merkeziyetsiz borsalarda (DEX) takasları mümkün kılmak için kullanıcılar tarafından bir akıllı kontrata kilitlenen token çiftleri havuzu."},
	{Terim: "Impermanent Loss", Turkce: "Geçici Kayıp", Kategori: "Finans", Seviye: "İleri", Slug: "impermanent-loss",
		Aciklama: "Bir likidite havuzuna yatırılan varlıkların fiyat oranı değiştiğinde, sadece elde tutmaya kıyasla ortaya çıkan değer kaybı riski."},
	{Terim: "Slippage", Turkce: "Kayma", Kategori: "Finans", Seviye: "Orta", Slug: "slippage",
		Aciklama: "Bir işlemi başlatırken beklenen fiyat ile işlemin gerçekleştiği an oluşan fiyat arasındaki fark."},
	{Terim: "Market Order", Turkce: "Piyasa Emri", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "market-order",
		Aciklama: "Mevcut piyasa fiyatından anında gerçekleşmesi için verilen alım veya satım emri."},
	{Terim: "Limit Order", Turkce: "Limit Emir", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "limit-order",
		Aciklama: "Yalnızca belirlenen fiyata ulaşıldığında gerçekleşecek şekilde önceden verilen alım veya satım emri."},
	{Terim: "Leverage", Turkce: "Kaldıraç", Kategori: "Piyasa", Seviye: "İleri", Slug: "leverage",
		Aciklama: "Kendi sermayesinden daha büyük bir pozisyon açmak için borç alınan fonların kullanılması; hem kâr hem zararı büyütür."},
	{Terim: "Liquidation", Turkce: "Tasfiye", Kategori: "Piyasa", Seviye: "İleri", Slug: "liquidation",
		Aciklama: "Kaldıraçlı bir pozisyonda teminat, zararı karşılayamayacak seviyeye düştüğünde borsanın pozisyonu zorla kapatması."},
	{Terim: "Order Book", Turkce: "Emir Defteri", Kategori: "Piyasa", Seviye: "Orta", Slug: "order-book",
		Aciklama: "Bir borsada belirli bir varlık için bekleyen tüm alım ve satım emirlerinin fiyat seviyelerine göre listelendiği kayıt."},
	{Terim: "Market Maker", Turkce: "Piyasa Yapıcı", Kategori: "Piyasa", Seviye: "İleri", Slug: "market-maker",
		Aciklama: "Sürekli alım-satım emirleri vererek piyasaya likidite sağlayan ve alış-satış fiyat farkından (spread) kazanç elde eden katılımcı."},
	{Terim: "Proof of Work", Turkce: "İş İspatı", Kategori: "Teknoloji", Seviye: "Orta", Slug: "proof-of-work",
		Aciklama: "Bitcoin'in de kullandığı, madencilerin karmaşık matematiksel problemleri çözerek yeni blokları doğruladığı konsensüs mekanizması."},
	{Terim: "Proof of Stake", Turkce: "Hisse İspatı", Kategori: "Teknoloji", Seviye: "Orta", Slug: "proof-of-stake",
		Aciklama: "Doğrulayıcıların, kilitledikleri (stake ettikleri) varlık miktarına göre yeni blokları onaylama hakkı kazandığı, PoW'a göre çok daha az enerji tüketen konsensüs mekanizması."},
	{Terim: "Mining", Turkce: "Madencilik", Kategori: "Teknoloji", Seviye: "Orta", Slug: "mining",
		Aciklama: "Proof-of-Work ağlarında işlemleri doğrulayıp yeni blok üretme karşılığında kripto para ödülü kazanma süreci."},
	{Terim: "Node", Turkce: "Düğüm", Kategori: "Teknoloji", Seviye: "Orta", Slug: "node",
		Aciklama: "Bir blok zinciri ağına bağlı, defterin bir kopyasını tutan ve işlemleri doğrulayan bilgisayar."},
	{Terim: "Hash Rate", Turkce: "Hash Gücü", Kategori: "Teknoloji", Seviye: "İleri", Slug: "hash-rate",
		Aciklama: "Bir Proof-of-Work ağının, saniyede gerçekleştirebildiği toplam hesaplama işlem gücü; ağın güvenliğinin bir göstergesidir."},
	{Terim: "Fork", Turkce: "Çatallanma", Kategori: "Teknoloji", Seviye: "Orta", Slug: "fork",
		Aciklama: "Bir blok zincirinin protokol kurallarının değişmesiyle ikiye ayrılması; \"hard fork\" geriye uyumsuz, \"soft fork\" geriye uyumludur."},
	{Terim: "Layer 2", Turkce: "İkinci Katman", Kategori: "Teknoloji", Seviye: "İleri", Slug: "layer-2",
		Aciklama: "Ana blok zincirinin (Layer 1) üzerine inşa edilen, işlemleri daha hızlı ve ucuz hale getirmeyi amaçlayan ölçeklendirme çözümü."},
	{Terim: "Bridge", Turkce: "Köprü", Kategori: "Teknoloji", Seviye: "İleri", Slug: "bridge",
		Aciklama: "Varlıkların veya verilerin iki farklı blok zinciri ağı arasında transferine olanak tanıyan protokol."},
	{Terim: "Wrapped Token", Turkce: "Sarmalanmış Token", Kategori: "Teknoloji", Seviye: "İleri", Slug: "wrapped-token",
		Aciklama: "Bir kripto varlığın (örn. Bitcoin) başka bir blok zincirinde kullanılabilmesi için 1:1 teminatlandırılmış temsili versiyonu (örn. WBTC)."},
	{Terim: "Stablecoin", Turkce: "İstikrarlı Kripto Para", Kategori: "Finans", Seviye: "Başlangıç", Slug: "stablecoin",
		Aciklama: "Değerini genellikle ABD dolarına sabitleyerek fiyat oynaklığını en aza indirmeyi amaçlayan kripto para türü (örn. USDT, USDC)."},
	{Terim: "ICO", Turkce: "İlk Coin Arzı", Kategori: "Finans", Seviye: "Orta", Slug: "ico",
		Aciklama: "Bir projenin, geliştirme için fon toplamak amacıyla yeni token'ını erken yatırımcılara ilk kez satışa sunduğu yöntem."},
	{Terim: "Airdrop", Turkce: "Hava Damlası", Kategori: "Piyasa", Seviye: "Başlangıç", Slug: "airdrop",
		Aciklama: "Bir projenin, farkındalık yaratmak veya erken kullanıcıları ödüllendirmek amacıyla token'larını ücretsiz olarak cüzdanlara dağıtması."},
	{Terim: "Tokenomics", Turkce: "Token Ekonomisi", Kategori: "Finans", Seviye: "İleri", Slug: "tokenomics",
		Aciklama: "Bir kripto varlığın arz, dağıtım, kullanım alanları ve teşvik mekanizmaları gibi ekonomik tasarımının bütünü."},
	{Terim: "Halving", Turkce: "Yarılanma", Kategori: "Piyasa", Seviye: "Orta", Slug: "halving",
		Aciklama: "Bitcoin ağında yaklaşık dört yılda bir, madenci blok ödülünün yarıya indirilerek yeni arzın yavaşlatıldığı önceden programlanmış olay."},
	{Terim: "Web3", Turkce: "", Kategori: "Teknoloji", Seviye: "Orta", Slug: "web3",
		Aciklama: "İnternetin, kullanıcıların verilerinin ve dijital varlıklarının kontrolünü merkeziyetsiz blok zinciri teknolojileriyle geri aldığı üçüncü nesil vizyonu."},
	{Terim: "dApp", Turkce: "Merkeziyetsiz Uygulama", Kategori: "Teknoloji", Seviye: "Orta", Slug: "dapp",
		Aciklama: "Arka planda merkezi bir sunucu yerine akıllı kontratlar ve blok zinciri altyapısı kullanan uygulama."},
	{Terim: "Cold Wallet", Turkce: "Soğuk Cüzdan", Kategori: "Teknoloji", Seviye: "Başlangıç", Slug: "cold-wallet",
		Aciklama: "Özel anahtarları internete bağlı olmayan bir ortamda (donanım cüzdanı, kağıt vb.) saklayan, hackleme riskine karşı daha güvenli cüzdan türü."},
	{Terim: "Hot Wallet", Turkce: "Sıcak Cüzdan", Kategori: "Teknoloji", Seviye: "Başlangıç", Slug: "hot-wallet",
		Aciklama: "İnternete sürekli bağlı olan, kullanımı kolay ama saldırılara karşı cold wallet'a göre daha savunmasız cüzdan türü."},
	{Terim: "KYC", Turkce: "Müşterini Tanı", Kategori: "Finans", Seviye: "Başlangıç", Slug: "kyc",
		Aciklama: "\"Know Your Customer\"; borsaların, kimlik doğrulama yoluyla kullanıcı kimliğini teyit ettiği yasal uyumluluk süreci."},
	{Terim: "Circulating Supply", Turkce: "Dolaşımdaki Arz", Kategori: "Piyasa", Seviye: "Orta", Slug: "circulating-supply",
		Aciklama: "Bir kripto varlığın, kilitli veya rezerve edilmiş miktarlar hariç, o an piyasada aktif olarak dolaşımda olan toplam miktarı."},
	{Terim: "Total Supply", Turkce: "Toplam Arz", Kategori: "Piyasa", Seviye: "Orta", Slug: "total-supply",
		Aciklama: "Henüz üretilmemiş veya kilitli olanlar dahil, bir kripto varlığın var olan veya var olacak toplam birim sayısı."},
	{Terim: "Consensus Mechanism", Turkce: "Konsensüs Mekanizması", Kategori: "Teknoloji", Seviye: "İleri", Slug: "consensus-mechanism",
		Aciklama: "Merkeziyetsiz bir ağdaki tüm katılımcıların, defterin geçerli durumu üzerinde anlaşmaya varmasını sağlayan kural bütünü."},
	{Terim: "Custodial Wallet", Turkce: "Vasi Cüzdan", Kategori: "Teknoloji", Seviye: "Orta", Slug: "custodial-wallet",
		Aciklama: "Özel anahtarların kullanıcı yerine bir üçüncü tarafça (genellikle borsa) saklandığı cüzdan türü."},
	{Terim: "Non-Custodial Wallet", Turkce: "Vasisiz Cüzdan", Kategori: "Teknoloji", Seviye: "Orta", Slug: "non-custodial-wallet",
		Aciklama: "Özel anahtarların yalnızca kullanıcının kontrolünde olduğu, hiçbir üçüncü tarafın erişemediği cüzdan türü."},
}
