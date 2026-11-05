# Kripto Para İkonları

Bu paket, kripto para simgelerini üçüncü taraf bir CDN veya API'den çekmeden
kendi altyapımızdan sunar. Böylece dış servis kesintisi, hotlink engeli veya
lisans belirsizliği oluşmaz.

## Kaynak

- Depo: https://github.com/spothq/cryptocurrency-icons
- Sürüm: `0.18.1`
- Commit: `1a63530be6e374711a8554f31b17e4cb92c25fa5` (2022-08-22)
- Alınan dizin: yalnız `svg/color/*.svg` (483 ikon)
- Lisans: **CC0 1.0 Universal** — ayrıntı için `LICENSE.md`

CC0, atıf zorunluluğu getirmez; yine de kaynak ve lisans bilgisi burada
tutulur. İkonların üzerinde değişiklik yapılmamıştır.

## Kullanım

```go
url := cryptoicon.URL("BTC") // /media/crypto/btc.svg?v=1a63530
data, ok := cryptoicon.Bytes("BTC")
```

`URL`, ikon yoksa boş dize döndürür. Şablonlar bu durumda mevcut harf
avatarına (`.krp-ico-yok`) düşer; dış URL üretilmez.

## Yeni ikon ekleme

1. SVG dosyasını `icons/<sembol-küçük-harf>.svg` olarak ekleyin.
2. `go test ./internal/cryptoicon` çalıştırın.
3. Gerekirse `SourceVersion` değerini güncelleyin (tarayıcı önbelleği için).

Sembol adı ile dosya adı farklıysa `cryptoicon.go` içindeki `aliases`
haritasına **yalnız anlamı doğru** eşlemeyi ekleyin. Örneğin `lun.svg`
"Lunyr" ikonudur; Terra Luna için kullanılmamalıdır.

## Güvenlik

- Gömülü dosyalar yalnız `icons/*.svg` ile sınırlıdır.
- `normalize`, `a-z`, `0-9` ve `$` dışındaki karakterleri atar; `../`
  gibi yol parçaları aramaya giremez.
- Sunucu tarafında yalnız bilinen semboller 200 döner; diğerleri 404'tür.
