-- 0120: "Görüş" modülü uzman profilleri
--
-- NEDEN VERİ MIGRATION'I: ana sayfanın "Görüş" modülü artık içerik değil
-- PROFİL basıyor (CoinDesk deseni: ad + öne çıkan yazı + 48 px avatar).
-- Profiller olmadan modül boş kalır; her ortamda (demo dahil) aynı
-- sonucu almak için tohum verisi migration'a kondu.
--
-- Kaynak: coindesk.com ana sayfa "Opinion" modülü, 6 Kasım 2026 ölçümü.
-- Avatar ve makale bağlantıları CoinDesk CDN'ine gider (geçici: kendi
-- yazar fotoğraflarımız ve Türkçe makalelerimiz üretilince değişecek).
-- Unvan/bio gerçek yazar profillerinden; uydurma yok.
--
-- password_hash = '!' → bu hesaplarla GİRİŞ YAPILAMAZ (geçerli bir
-- bcrypt/argon2 özeti değil). Profiller yalnız görünüm içindir.

INSERT INTO users (name, email, password_hash, role, status, slug, title, bio, avatar_url, socials)
VALUES
  (
    'Lindsay Fraser',
    'lindsay.fraser@bitcosar.com',
    '!',
    'Yazar',
    'aktif',
    'lindsay-fraser',
    'Blockchain Association Politika Direktörü',
    'Lindsay Fraser, Blockchain Association''da politika stratejisi ve Washington DC savunuculuğunu yöneten Politika Direktörü''dür. Daha önce Uniswap Labs''ta Kıdemli ABD Politika Lideri olarak DeFi sektörünün federal düzeydeki ilişkilerini şekillendirdi; finansal hizmetler sektöründe ve Yönetim ve Bütçe Ofisi''nde çeşitli politika görevlerinde bulundu. Georgetown Üniversitesi''nden yüksek lisans, Vanderbilt Üniversitesi''nden lisans derecesine sahiptir.',
    'https://cdn.sanity.io/images/s3y3vcno/production/063efc1d2f0ffe3e7b8c035fc6fa73dd3f0c2562-5472x3648.jpg?w=96&h=96&fit=crop&crop=focalpoint&auto=format',
    jsonb_build_object(
      'son_makale', 'Seçim Washington''ı değiştiriyor. Kripto çalışmaları sürüyor',
      'son_makale_url', 'https://www.coindesk.com/opinion/2026/11/04/the-election-changes-washington-the-work-on-crypto-continues',
      'son_makale_kaynak', 'CoinDesk'
    )
  ),
  (
    'Hermine Wong',
    'hermine.wong@bitcosar.com',
    '!',
    'Yazar',
    'aktif',
    'hermine-wong',
    'Berkeley Law Öğretim Görevlisi',
    'Hermine Wong, Berkeley Law''ta öğretim görevlisi olarak kuruculara, risk sermayedarlarına, yönetim kurullarına ve yöneticilere yeni teknoloji sektöründe danışmanlık yapıyor. Kendi bürosunu kurmadan önce ABD''nin ilk halka açık kripto borsasında Politika ekibini kurup yönetti. ABD Menkul Kıymetler ve Borsa Komisyonu, Bilgi ve Düzenleme İşleri Ofisi, Yönetim ve Bütçe Ofisi, ABD Dışişleri Bakanlığı ve Pentagon''da on yıl kamu hizmetinde bulundu.',
    'https://cdn.sanity.io/images/s3y3vcno/production/e7d158bc4150f24f3470603a16cb277c19b665df-2009x2009.jpg?w=96&h=96&fit=crop&crop=focalpoint&auto=format',
    jsonb_build_object(
      'son_makale', 'Kripto devleri satın alamayacağı tek şeyi istiyor: oyunuzu',
      'son_makale_url', 'https://www.coindesk.com/opinion/2026/11/02/big-crypto-wants-the-one-thing-it-can-t-buy-your-vote',
      'son_makale_kaynak', 'CoinDesk'
    )
  ),
  (
    'Colin McLaren',
    'colin.mclaren@bitcosar.com',
    '!',
    'Yazar',
    'aktif',
    'colin-mclaren',
    'Solana Policy Institute Hükümet İlişkileri Direktörü',
    'Colin McLaren, Solana Policy Institute''ta (SPI) Hükümet İlişkileri Direktörü olarak politika yapıcıların, adayların ve personelin eğitimini ve Solana ekosisteminin üçüncü taraf ilişkilerini yürütüyor. Aynı zamanda politika yapıcıları Amerikan inovasyon liderliğine teşvik etmeye odaklanan 501c3 statüsündeki American Innovation Project''in kurucu yönetim kurulu üyesidir.',
    'https://cdn.sanity.io/images/s3y3vcno/production/dea752fb643cdef64e5638ce2bf0cda75ad15727-6000x4000.jpg?w=96&h=96&fit=crop&crop=focalpoint&auto=format',
    jsonb_build_object(
      'son_makale', 'Kripto''nun Washington oyun planı işliyor. Şimdi bırakmayın',
      'son_makale_url', 'https://www.coindesk.com/opinion/2026/11/02/crypto-s-washington-playbook-is-working-don-t-abandon-it-now',
      'son_makale_kaynak', 'CoinDesk'
    )
  ),
  (
    'Michael Tanguma',
    'michael.tanguma@bitcosar.com',
    '!',
    'Yazar',
    'aktif',
    'michael-tanguma',
    'Onramp Kurucu Ortağı ve CEO',
    'Michael Tanguma, insanların serveti on yıllar ölçeğinde düşünmesi için tasarlanmış bir finans platformu olan Onramp''in kurucu ortağı ve CEO''sudur. Platform; Onramp destekli ödüllerle nakit hesapları, 50 eyaletin tamamında düşük maliyetli bitcoin aracılığı, bitcoin IRA''ları, doğrudan altın erişimi ve tek bir hesapta %1,5''e varan nakit iadesi sunan ödül kartını bir araya getiriyor.',
    'https://cdn.sanity.io/images/s3y3vcno/production/563cc61c951df6c894f054b6e6bb3d182bb0ccf9-968x598.png?w=96&h=96&fit=crop&crop=focalpoint&auto=format',
    jsonb_build_object(
      'son_makale', 'Bitcoin''in beyaz kağıdı 18 yaşında: küçük işletmeler hâlâ kurtarma paketlerinin bedelini ödüyor',
      'son_makale_url', 'https://www.coindesk.com/opinion/2026/10/31/bitcoin-s-white-paper-turns-18-and-small-businesses-are-still-paying-for-the-bailouts-it-answered',
      'son_makale_kaynak', 'CoinDesk'
    )
  )
ON CONFLICT (email) DO UPDATE SET
  name       = EXCLUDED.name,
  role       = EXCLUDED.role,
  status     = EXCLUDED.status,
  slug       = EXCLUDED.slug,
  title      = EXCLUDED.title,
  bio        = EXCLUDED.bio,
  avatar_url = EXCLUDED.avatar_url,
  socials    = EXCLUDED.socials,
  updated_at = now();
