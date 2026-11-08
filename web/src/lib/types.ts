// Shapes from docs/contract.md §3. Keep in sync with the Go API.

export type HazardType =
  | 'respiratory_sensitizer'
  | 'respiratory_irritant'
  | 'skin_sensitizer'
  | 'skin_irritant'
  | 'eye_irritant'
  | 'hypersensitivity_pneumonitis'
  | 'systemic_toxic'
  | 'cardiac'
  | 'carcinogen'
  | 'splinter_infection'
  | 'photosensitization';

export type EvidenceLevel = 0 | 1 | 2 | 3;
export type WoodType = 'hardwood' | 'softwood';

export interface Image {
  url: string;
  alt: string;
  width?: number;
  height?: number;
}

export interface SpeciesSummary {
  slug: string;
  name: string;
  common_names: string[];
  botanical_name: string;
  wood_type: WoodType;
  region: string;
  featured_rank?: number | null;
  hazard_types: HazardType[];
  max_evidence_level: 1 | 2 | 3 | null;
  image: Image;
  url: string;
}

export interface SpeciesList {
  species: SpeciesSummary[];
}

export interface Source {
  key: string;
  tier: 'A' | 'B' | 'C' | 'D';
  citation: string;
  title?: string;
  doi?: string;
  pmid?: string;
  pmcid?: string;
  url?: string;
  accessed?: string;
  license?: string;
  verified: boolean;
  verification_note?: string;
}

export type SourceMap = Record<string, Source>;

export interface Hazard {
  type: HazardType;
  effects: string[];
  route: string[];
  source_part?: string[];
  summary: string;
  evidence_level: EvidenceLevel;
  sources: string[];
}

export interface ExposureLimit {
  body: string;
  value_mg_m3: number;
  fraction?: 'inhalable' | 'total' | 'respirable';
  note?: string;
  source: string;
}

export interface CarcinogenClass {
  body: string;
  class: string;
  source: string;
}

export interface KnownCompound {
  name: string;
  role?: string;
  source: string;
}

export interface RegulatorySummary {
  eu_hardwood_mg_m3?: number;
  uk_hardwood_mg_m3?: number;
  uk_softwood_mg_m3?: number;
  niosh_rel_mg_m3?: number;
}

export interface SpeciesDetail {
  id: string;
  slug: string;
  name: string;
  url: string;
  botanical_name: string;
  botanical_name_verified?: boolean;
  botanical_name_source?: string;
  synonyms?: string[];
  family: string;
  common_names: string[];
  wood_type: WoodType;
  region: string;
  description?: string;
  hazards: Hazard[];
  exposure_limits?: ExposureLimit[];
  carcinogen_class?: CarcinogenClass[];
  known_compounds?: KnownCompound[];
  related_species_note?: string;
  image_alt?: string;
  image: Image;
  featured_rank?: number | null;
  last_reviewed: string;
  sources: SourceMap;
  regulatory?: RegulatorySummary;
}

export interface SearchResult {
  slug: string;
  name: string;
  botanical_name: string;
  matched_name: string;
  match_kind: 'common' | 'trade' | 'botanical' | 'synonym';
  score: number;
  url: string;
}

export interface SearchResponse {
  query: string;
  results: SearchResult[];
}

export interface TocEntry {
  id: string;
  text: string;
  level: number;
}

export interface ContentPage {
  slug: string;
  title: string;
  description?: string;
  last_reviewed?: string;
  html: string;
  toc?: TocEntry[];
  sources?: SourceMap;
  image?: Image;
}

export interface SourcesResponse {
  sources: SourceMap;
}

export interface SiteInfo {
  name: string;
  base_url: string;
}

export interface InitialPayload {
  route: string;
  data: unknown;
  site?: Partial<SiteInfo>;
}
