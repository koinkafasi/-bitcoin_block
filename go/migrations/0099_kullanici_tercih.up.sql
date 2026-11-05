-- Kullanıcı arayüz tercihleri (Faz 173, A-3).
--
-- NEDEN: içerik formundaki katlanabilir grupların açık/kapalı durumu ve
-- daraltılmış yan menü yalnız tarayıcının localStorage'ında tutuluyordu.
-- Aynı editör başka bir cihazda (ya da gizli sekmede) her şeyi baştan
-- açmak zorunda kalıyordu; "Künye'yi hep açan editör" bilgisi taşınmıyordu.
--
-- NEDEN ayrı tablo: settings (section, key) sütunları SİTE geneli ayarlar
-- için. Kullanıcı başına tercih oraya konsaydı anahtar
-- "kullanici.<id>...." gibi bir dizgeye dönüşür, kullanıcı silindiğinde
-- kayıtlar yetim kalırdı. FK + ON DELETE CASCADE bunu çözüyor.
--
-- NEDEN jsonb: tercih anahtarları istemci tarafında üretiliyor
-- (tb_form_grup:Künye gibi). Her yeni anahtar için migration yazmak
-- istemiyoruz; şema değişmeden yeni anahtar eklenebilsin.
CREATE TABLE IF NOT EXISTS user_prefs (
    user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    prefs      JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
