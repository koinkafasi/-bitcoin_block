-- 0132 geri alma: GDPR sayfası kaldırılır. KVKK, gizlilik ve çerez metinlerinin
-- önceki hâli 0091'dedir; gerekirse o göçteki INSERT'ler yeniden uygulanır.
DELETE FROM pages WHERE slug = 'gdpr';
