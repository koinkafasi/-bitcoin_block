-- 0088: transkript için dalga formu (waveform) verisi.
-- Ses panelinde SoundCloud benzeri dalga formu çizmek ve tıklayarak sarmak
-- için normalize edilmiş sesten çıkarılan tepe noktaları (0..1 arası float
-- dizisi, ~400 kova) saklanır. Yoksa oynatıcı düz ilerleme çubuğuna düşer.
ALTER TABLE audio_transcripts
    ADD COLUMN IF NOT EXISTS waveform JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN audio_transcripts.waveform IS
    'Dalga formu tepe noktaları: 0..1 arası float dizisi (yaklaşık 400 kova).';
