-- 0081 geri alınırsa webhook olay kayıtları ve sütunlar kaldırılır.
DROP TABLE IF EXISTS stripe_events;
ALTER TABLE users DROP COLUMN IF EXISTS membership_until;
ALTER TABLE users DROP COLUMN IF EXISTS stripe_subscription_id;
ALTER TABLE users DROP COLUMN IF EXISTS stripe_customer_id;