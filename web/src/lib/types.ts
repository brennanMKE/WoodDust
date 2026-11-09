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
  /** Content hashes of images the SPA uses by fixed path (path under /images/ -> hash). */
  assets?: Record<string, string>;
}

export interface InitialPayload {
  route: string;
  data: unknown;
  site?: Partial<SiteInfo>;
}

// --- Submissions (contract §6) and admin (§7) -------------------------------

/** Initial data for `/submit` and `/submit/{slug}`. */
export interface SubmitData {
  form_token: string;
  kinds?: unknown[];
  target?: { slug: string; name: string; botanical_name?: string } | null;
  prefill_name?: string;
}

export interface SubmissionNote {
  at: string;
  text: string;
}

export interface SubmissionStatusData {
  reference: string;
  kind: string;
  status: string;
  created_at: string;
  updated_at: string;
  notes?: SubmissionNote[];
}

export interface AdminUser {
  display_name: string;
  is_admin: boolean;
}

export interface MeResponse {
  user: AdminUser;
}

// --- Shop posters (contract §2, §3) -------------------------------------------

export interface PosterLink {
  short_url: string;
  /** SVG markup of the QR code (pure black on white, quiet zone included). */
  qr_svg: string;
  /** true: the stock link, because a separate one couldn't be made. */
  fallback?: boolean;
}

export interface PosterDownload {
  label: string;
  format: 'pdf' | 'png' | 'svg';
  url: string;
}

export interface PosterView {
  id: string;
  title: string;
  slogan: string;
  destination: string;
  art: { url: string; available: boolean; width: number; height: number };
  /** posters/src/<id>.svg with comments stripped; slots marked data-slot. */
  template: string;
  stock: PosterLink;
  downloads: PosterDownload[];
}

/** `/posters` initial data and GET /api/v1/posters. */
export interface PostersData {
  posters: PosterView[];
  form_token: string;
  /** The server can mint a plaque's own link (POST /api/v1/posters/links). */
  custom: boolean;
}

/** `/posters/{id}/print` initial data and GET /api/v1/posters/{id}. */
export interface PosterData {
  poster: PosterView;
}
