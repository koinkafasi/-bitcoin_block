-- 0120 geri alımı: yalnız bu migration'ın eklediği profiller silinir.
DELETE FROM users
WHERE email IN (
  'lindsay.fraser@bitcosar.com',
  'hermine.wong@bitcosar.com',
  'colin.mclaren@bitcosar.com',
  'michael.tanguma@bitcosar.com'
);
