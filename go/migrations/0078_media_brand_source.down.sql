ALTER TABLE media_files DROP CONSTRAINT IF EXISTS media_files_source_check;
ALTER TABLE media_files ADD CONSTRAINT media_files_source_check
    CHECK (source IN ('upload','library','internet','render','generate','pexels','ai'));