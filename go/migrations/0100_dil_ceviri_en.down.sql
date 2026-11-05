-- Faz 189 geri alma: bu migration'ın eklediği İngilizce anahtarlar silinir.
-- DİKKAT: yalnız bu dosyanın eklediği anahtarlar; panelden elle eklenmiş
-- diğer çevirilere dokunulmaz.
DELETE FROM translations
WHERE locale = 'en'
  AND key IN (
    'site.name','site.suffix','home','breaking','recent','trending','related',
    'comments','login','logout','language','all_rights','categories','about',
    'contact','search','archive','weather','markets','sports','photo_gallery',
    'video_gallery','most_read','newsletter','subscription','no_content'
  );
-- DİKKAT: kategori anahtarları (category.*) SİLİNMEZ: panelden elle
-- düzenlenmiş olabilir; geri alma yalnız sabit anahtarları kaldırır.
