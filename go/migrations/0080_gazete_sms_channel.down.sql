-- 0080 geri alınırsa 'sms' aboneleri kısıtı ihlal eder; önce whatsapp'a
-- çevrilir (telefonları zaten var), sonra eski kısıt geri kurulur.
UPDATE gazete_subscriptions SET channel = 'whatsapp' WHERE channel = 'sms';
ALTER TABLE gazete_subscriptions DROP CONSTRAINT IF EXISTS gazete_subscriptions_channel_check;
ALTER TABLE gazete_subscriptions ADD CONSTRAINT gazete_subscriptions_channel_check
    CHECK (channel IN ('email','whatsapp'));