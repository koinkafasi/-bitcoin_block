-- 0057: Dünya'yı üst menüye al (GZT ana menüsü gibi)
UPDATE categories SET in_header = TRUE, in_menu = TRUE WHERE slug = 'dunya';
