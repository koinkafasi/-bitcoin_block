-- theblock — P5-F Hesaplayıcı içerik türü (0016)
INSERT INTO content_types (key, label, sort_order) VALUES ('hesaplayici', 'Hesaplayıcı', 30)
ON CONFLICT (key) DO NOTHING;
