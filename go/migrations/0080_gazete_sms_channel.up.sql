-- 0080: SMS dağıtım kanalı — abonelik channel kontrolüne 'sms' eklendi.
-- (E-posta ve WhatsApp vardı; SMS aboneliği yeni kanal.)
ALTER TABLE gazete_subscriptions DROP CONSTRAINT IF EXISTS gazete_subscriptions_channel_check;
ALTER TABLE gazete_subscriptions ADD CONSTRAINT gazete_subscriptions_channel_check
    CHECK (channel IN ('email','whatsapp','sms'));