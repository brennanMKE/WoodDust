// The plaque a visitor is building, kept in memory only (never stored), so
// the print route can show the same sheet. The shop name stays in the
// browser unless the visitor opts in to sharing it (docs/posters.md).
import type { PosterLink } from './types';

export interface PosterDraft {
  plaque: string | null;
  link: PosterLink;
  /** Open the print dialog once the print view has drawn. */
  autoPrint: boolean;
}

export const drafts: Record<string, PosterDraft | undefined> = $state({});
