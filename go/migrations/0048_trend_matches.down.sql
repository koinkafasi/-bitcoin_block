DROP TABLE IF EXISTS trend_alerts;
DROP TABLE IF EXISTS trend_briefs;
DROP TABLE IF EXISTS trend_matches;
DELETE FROM settings WHERE section = 'trend' AND key IN
    ('gsc_property','gsc_last_sync','ga4_property','odeme_aktif','bildirim_aktif');
