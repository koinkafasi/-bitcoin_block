DROP TABLE IF EXISTS logs;
DELETE FROM settings WHERE section IN ('gorsel') AND key IN ('site_font','baslik_font');
DELETE FROM settings WHERE section = 'moduller' AND key = 'izinler';
DELETE FROM settings WHERE section = 'baglantilar';
DELETE FROM settings WHERE section = 'yapay_zeka' AND key IN ('model','base_url');
