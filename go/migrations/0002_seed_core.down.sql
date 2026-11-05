-- theblock P0 çekirdek SEED — geri alma (0002 down).
DELETE FROM settings WHERE section IN ('genel','gorsel','moduller','yapay_zeka');
DELETE FROM content_types;
