DROP TABLE IF EXISTS seed_runs;
DROP TABLE IF EXISTS regulatory;
DROP TABLE IF EXISTS carcinogen_classes;
DROP TABLE IF EXISTS exposure_limits;
DROP TABLE IF EXISTS hazard_sources;
DROP TABLE IF EXISTS hazards;
DROP TABLE IF EXISTS species_names;
DROP TABLE IF EXISTS species;
DROP TABLE IF EXISTS sources;
DROP FUNCTION IF EXISTS wd_normalize(text);
-- Extensions are left installed; other objects may depend on them.
