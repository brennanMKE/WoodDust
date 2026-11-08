# Codex art review 1: fixes (2026-11-08)

Review of the 26 images from `images/prompts/codex-art.md`. Most are good. Fix only the items below and leave everything listed under "Approved" untouched. The rules in `AGENTS.md` and `codex-art.md` still apply: same style, paths, sizes and real alpha, no text, and no commits.

For every edit, attach the current image plus `images/art/_character-sheet.png`. Start the prompt with "Edit the attached illustration. Keep everything not mentioned exactly as it is."

## Approved: do not change

`emblem.png`, `og-default.jpg`, `babe-404.png`, `no-results.png` (the fixed version is great), `spot-almanac.png`, `spot-axe-and-saw.png`, `spot-dust-collector.png`, `pages/risks.jpg`, `pages/protection-respirators.jpg`, `pages/protection-skin-and-eyes.jpg`, `pages/glossary.jpg`, `pages/methodology.jpg`.

Approved, but they need re-encoding only (see §3): `pages/protection-dust-collection.jpg`, `pages/symptoms.jpg`.

## 1. Paul's beard: highest priority

Paul must have a **clean-shaven chin and jaw with only a short trimmed mustache**. The site teaches that facial hair breaks a respirator's seal, so a beard or stubble under his mask contradicts the page it sits on. It is visible in:

| File | What to fix |
|---|---|
| `art/_character-sheet.png` | **Fix this first, because every later image copies it.** The masked close-up shows dark stubble along the jaw and chin. Make the skin around the mask clean-shaven, and show the mustache above the mask's nose bridge. |
| `pages/limits.jpg` | A full beard shows around the respirator. Remove it: smooth, clean-shaven skin where the mask meets the face. |
| `mascots/hero.png` | Stubble along the jaw around the mask. Same fix. |
| `art/poster-header.png` | Check the jaw around the mask. If there's any stubble shadow, remove it. |

**Prompt fragment:** "Paul's chin, jaw and cheeks are completely clean-shaven, smooth skin with no stubble or beard shadow, so the respirator seals against bare skin. Only a short trimmed mustache."

## 2. Per-image fixes

### `mascots/hero.png`: Paul is still inside the bench
The bench itself is fixed: four square legs, aprons and stretchers. But Paul's body is impossible. His hips and torso are behind the bench top, while his thighs, knees and boots are in front of the front legs, so his legs pass through the bench.

Since telling him to stand in front of the bench didn't work, **change the arrangement:**

> Paul stands at the LEFT END of the workbench, facing right, both boots on the floor beyond the bench's left end, leaning over the top to sand the plank. No part of his body overlaps the bench legs or rails. The bench, sander, hose, dust collector and Babe stay where they are.

Also:
- remove the stubble (§1);
- remove the colored fringe and halo along the transparent edges, visible on a dark background;
- reduce the over-sharpened, crunchy texture so it matches `babe-404.png`.

### `art/spot-babe-hoofprint.png`: reads as a pair of lungs
Two rounded, blue, hatched lobes look exactly like lungs, which is a real problem on a respiratory-health site. Redraw:

> A single cloven ox hoofprint pressed into soil, seen from directly above: two narrow, slightly curved toe impressions that come to points at the front and nearly touch, a small gap between them, and two tiny dewclaw dots behind. Warm black ink with a light brick-red soil texture inside the print, no blue. Simple bold shapes that read as a footprint at 24 pixels. Transparent background, PNG.

### `art/spot-end-grain.png`: rings are a spiral
Growth rings are concentric circles, never a spiral. Redraw with **concentric, slightly irregular rings**, the pith (center) a little off-center, wider early rings and tighter outer rings, and the same checks and bark.

### `art/spot-respirator.png`: one strap
Real half-face respirators have **two** elastic straps, a crown strap over the top of the head and a neck strap below, each with a buckle. Add the lower neck strap; keep everything else.

### `pages/protection.jpg`: two fixes
- **The respirator on the table** has a rigid over-the-head band like earmuffs. Replace it with a respirator lying face-up with two loose elastic straps, matching `spot-respirator.png` after its fix.
- **The gloves** sit against Babe's chest and read as Babe's hands in gloves. Move them left, between the safety glasses and the respirator, with clear table between them and Babe.

### `pages/protection-cleanup.jpg`: vacuum kicking up dust
The dust clouds in front of the nozzle suggest the vacuum is stirring dust up, which is the opposite of the page's message. Remove the clouds. Show loose sawdust ahead of the nozzle and a **clean, dust-free strip** behind it. Babe pushing on his hind legs is fine: it's a tall tale.

### `pages/protection-fire-and-explosion.jpg`: dust plume
Paul dumps the bin from high up and a big cloud billows out of the can. A cloud of fine dust is exactly the fire and explosion hazard the page warns about. Change it to:

> Paul crouches and lowers the collector bin INTO the metal can, tipping it gently so the dust slides out with only a few specks in the air; the can's lid leans against the can within reach.

Keep the fire extinguisher, the drying rags and Babe.

### `art/favicon.png`: too detailed for a tab icon
At 16–32 px the fur hatching, halter and shading turn to mud. Make a **flat icon version**:

> Babe's head facing forward: solid flat Babe-blue face, two cream horns, cream muzzle, two simple dark eyes, thick warm-black outline. No hatching, no fur texture, no halter, no shading. Transparent background.

Check it by shrinking copies to 16 px and 32 px. It must still read as an ox head. Don't save the copies.

### `art/border-forest.png`: prove it tiles
Place two copies side by side (`sips` can't do this; use any tool, or check that the leftmost and rightmost columns of pixels match) and confirm the join is invisible. If it isn't, fix the edges so the ground line and tree spacing continue across the seam.

## 3. File sizes

| File | Now | Target | Fix |
|---|---|---|---|
| `pages/protection-dust-collection.jpg` | 320 KB | < 300 KB | Re-encode at quality 80: `sips -s format jpeg -s formatOptions 80 in --out out` |
| `pages/symptoms.jpg` | 320 KB | < 300 KB | Same |
| `mascots/hero.png` | 2.3 MB | < 600 KB | Lossy PNG quantization with alpha, e.g. `pngquant --quality=70-90 --skip-if-larger --strip`. If `pngquant` isn't installed, **ask Brennan before installing anything**. |
| `mascots/babe-404.png`, `no-results.png` | 1.2 MB, 0.96 MB | < 400 KB each | Same |
| `art/poster-header.png` | 3.5 MB | < 1 MB | Same; it prints, so keep 1536×1024 |
| `art/spot-*.png`, `emblem.png`, `favicon.png` | 160–460 KB | < 120 KB each | Same |

Compression must not change dimensions or drop the alpha channel: `sips -g hasAlpha` must still print `yes` where it did before. Look at each compressed file against a dark background for banding or halos.

## 4. Optional, lower priority

Paul wears his full respirator and earmuffs even in scenes with no dust, such as reading books (`methodology.jpg`) and washing up (`protection-skin-and-eyes.jpg`). Both are approved as they are. In any **new** scene, he should wear the respirator only when dust is in the air.

## 5. When done

Print a table with the file, what changed, pixel size, file size, alpha, and pass or redo. Then stop for Brennan's review. Don't commit.
