-- Faz 195 geri alma: yalnız bu dosyanın eklediği anahtarlar silinir.
-- Panelden elle eklenmiş/düzenlenmiş diğer çevirilere dokunulmaz.
DELETE FROM translations
WHERE key IN ('font_size', 'staff_panel', 'admin_group', 'menu_close');
