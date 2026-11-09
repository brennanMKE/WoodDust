# Codex task: Paul Bunyan and Babe illustrations for Dust Almanac

Hand this whole file to Codex. It is self-contained. `site-art.md` in this folder is the design source; this file turns it into a task with exact output paths, sizes and checks.

## Your task

Generate the illustrations below and save each one at the exact path given. The site picks each file up automatically by its filename, so a misspelled path means the image never shows.

**Work in three rounds and stop after each so Brennan can approve.** Character consistency matters more than speed.

1. **Round 1:** the character sheet (A0), then the home hero (A1). Stop and show both.
2. **Round 2:** the emblem and favicon (A2, A3), the share image (A4), and the 404 and no-results mascots (A5, A6). Stop and show them.
3. **Round 3:** the page headers (B1–B11), the spot illustrations (C1–C6), the border (C7) and the poster header (C8).
4. **Round 4:** the shop posters D2 and D3 (Shave, Sunbeam). Stop and show them. D1, D4 and D5 come later. See `docs/posters.md`.

## Rules

- **Build every prompt the same way:** the STYLE block, then the CHARACTERS block (whenever Paul or Babe appears), then the item's Scene, joined with spaces. Do not reword them.
- **Keep the characters consistent:** if your image tool accepts a reference image, attach `images/art/_character-sheet.png` (A0) to every later prompt and add: "Keep Paul and Babe consistent with the attached character sheet."
- **Draw the safety gear correctly.** This is a safety site. Respirator straps go behind the head, the mask is sealed on a clean-shaven chin, and glasses are worn, not perched. Redo any image where Paul does something unsafe, such as sanding without his respirator in a scene that calls for it, or blowing dust with compressed air.
- **No text anywhere:** no letters, numbers, signatures, labels or logos. Redo any image that has some, including fake writing on books and signs.
- **Make it original.** Do not resemble the Bemidji or Brainerd statues, Disney's 1958 *Paul Bunyan*, or any specific vintage pamphlet page. If a result looks familiar, regenerate with "more original character design".
- **Use a real transparent background** where an item says *transparent*: a PNG with an alpha channel, not a painted checkerboard and not a cream fill. The site has a dark mode, so a solid background shows as a box. Check with `sips -g hasAlpha <file>`, which must print `yes`. If your tool cannot produce transparency, stop and say so rather than faking it.
- **Size every file down** to the final size listed, using macOS `sips`:
  - Resize a PNG and keep its alpha: `sips -Z <longest-edge> in.png --out out.png`
  - Crop to 1200×630 JPEG from 1536×1024: `sips -c 806 1536 in.png --out tmp.png && sips -z 630 1200 -s format jpeg -s formatOptions 85 tmp.png --out out.jpg`
- **Only write files** under `images/art/` (including `images/art/posters/`), `images/mascots/`, `images/pages/` and `images/og-default.jpg`. Do not edit code, data or content. Do not commit, push or touch any server.
- **When done, print a table** with path, pixel size, file size, alpha (yes or no), and a pass/redo note.

## STYLE block

> Original illustration in the spirit of early-1900s American lumber-company promotional pamphlets: confident pen-and-ink line art with hand-drawn hatching, a limited two-color print look (warm black ink plus a brick-red second color) on aged cream paper, with occasional flat spot fills of a muted Babe blue (#4a78a8). Folksy, good-humored, tall-tale exaggeration of scale. Clean, readable silhouettes. No photographic detail, no gradients, no modern cartoon gloss. Do not copy or imitate any specific existing illustration, statue, film or character design. No text, letters, numbers, signatures or logos anywhere in the image.

## CHARACTERS block

> Paul Bunyan is a giant, friendly lumberjack in his 40s: broad shoulders, clean-shaven chin and jaw with a short trimmed mustache, thick dark hair under a knit watch cap, red-and-black buffalo-check flannel shirt with rolled sleeves, canvas work pants, suspenders, sturdy boots, calm and capable expression. When he wears safety gear it is drawn accurately and worn correctly: a half-face reusable respirator with two round filter cartridges at the cheeks, sealed on his clean-shaven face with straps behind his head; wraparound safety glasses; earmuffs; work gloves. Babe the Blue Ox is an enormous, gentle blue ox with wide gently curving horns, a pale muzzle, big kind eyes, a curious and slightly comic manner, and a plain leather halter.

---

## A. Characters, brand and mascots

### A0. Character sheet: reference only, never published
- **Save:** `images/art/_character-sheet.png`. The leading `_` keeps it unpublished.
- **Generate:** 1536×1024 on cream. Keep it full size.
- **Scene:** a character reference sheet on plain cream paper. Left: Paul Bunyan shown front view, three-quarter view and side view, full body, standing relaxed. Beside him, a close-up of his face wearing the half-face respirator with two round cartridges and wraparound safety glasses. Right: Babe the Blue Ox shown side view and three-quarter view. Even spacing, no background scenery, no text or labels.

### A1. Home hero
- **Save:** `images/mascots/hero.png`.
- **Generate:** 1536×1024, transparent. **Final:** `sips -Z 1440`.
- **Scene:** Paul Bunyan at a sturdy wooden workbench, sanding a plank with a hand sander. He wears the half-face respirator, wraparound safety glasses and earmuffs. A flexible dust collection hose is attached to the sander and runs to a barrel dust collector beside the bench. Babe the Blue Ox leans in from the right, curious and approving, nose near the collector. A few fine dust motes are being pulled into the hose. Transparent background, PNG, no floor line beyond a short shadow.

### A2. Emblem
- **Save:** `images/art/emblem.png`.
- **Generate:** 1024×1024, transparent. **Final:** `sips -Z 512`.
- **Scene:** a simple circular emblem. Babe the Blue Ox's head, facing forward, wide horns spanning the top of the circle, above an open hardbound almanac with a single wood shaving curling off its pages. Bold, simple linework that stays readable at 32 pixels. Two colors only: warm black and Babe blue. No text, no banner, no lettering. Transparent background, PNG.

### A3. Favicon
- **Save:** `images/art/favicon.png`.
- **Generate:** 1024×1024, transparent. **Final:** `sips -Z 512`.
- **Scene:** only Babe the Blue Ox's head, facing forward, wide horns, extremely simplified with thick lines and a solid Babe blue face, designed to stay recognizable at 16 pixels. Transparent background, PNG, no text.
- **Check:** shrink a copy to 32 px (`sips -Z 32`) and look at it. It must still read as an ox head. Don't keep the copy.

### A4. Default share image, used when a page has no image of its own
- **Save:** `images/og-default.jpg`.
- **Generate:** 1536×1024 on cream. **Final:** crop to 1200×630 JPEG at quality 85, under 300 KB.
- **Scene:** Paul Bunyan sitting on a big log in a timber-framed woodshop, a giant open almanac on his knee. Babe the Blue Ox reads over his shoulder wearing small round spectacles. Clamps, hand planes and a dust collector in the background. Wide composition with the characters centered and empty margin on all sides for cropping to 1.91:1. Cream paper background. The almanac's pages show no readable text.

### A5. Not found (404)
- **Save:** `images/mascots/babe-404.png`.
- **Generate:** 1024×1024, transparent. **Final:** `sips -Z 800`.
- **Scene:** Babe the Blue Ox peeking around the end of a tall stack of sawn lumber, one ear cocked, puzzled and gently comic, as if searching for a board that isn't there. Transparent background, PNG.

### A6. Search with no results
- **Save:** `images/mascots/no-results.png`.
- **Generate:** 1024×1024, transparent. **Final:** `sips -Z 800`.
- **Scene:** Paul Bunyan crouching beside a log round, studying its end grain through an oversized magnifying glass, curious and patient. Transparent background, PNG.

---

## B. Education page headers

Each one is the page's header and its share image.
- **Generate:** 1536×1024 on cream paper, no transparency, with margin on all sides.
- **Final:** crop to **1200×630 JPEG**, quality 85, under 300 KB, saved at the path shown.
- **Append to every scene:** "Wide composition, margin on all sides, cream paper background."

| # | Save to | Scene |
|---|---|---|
| B1 | `images/pages/risks.jpg` | A shaft of window light crossing a woodshop, full of fine floating dust specks. Paul Bunyan, wearing his respirator, holds a large magnifying glass up to the sunbeam to show how much fine dust is in the air. Babe looks at the beam with a wary, wrinkled nose. |
| B2 | `images/pages/protection.jpg` | A tidy workbench. Laid out on it in order from left to right: a dust collector hose, then a half-face respirator, safety glasses and gloves. Paul Bunyan stands behind the bench gesturing at the gear like a proud instructor. Babe nods along. |
| B3 | `images/pages/protection-dust-collection.jpg` | Tall-tale scale. Babe the Blue Ox pulls a wooden wagon carrying an enormous barrel dust collector, while Paul Bunyan connects its flexible hose to the dust port of a table saw. |
| B4 | `images/pages/protection-respirators.jpg` | Paul Bunyan, clean-shaven chin with a trimmed mustache, wearing a half-face respirator with two round cartridges, cupping both palms over the cartridges and breathing in to check the seal, eyes focused. A small hand mirror is propped on the bench beside him. Babe watches closely from behind. |
| B5 | `images/pages/protection-skin-and-eyes.jpg` | Paul Bunyan with sleeves rolled down, wearing wraparound safety glasses, pulling on a pair of work gloves. On a stand beside him are a wash basin, a bar of soap and a towel. Babe sniffs the soap curiously. |
| B6 | `images/pages/protection-cleanup.jpg` | Babe the Blue Ox cheerfully pushing a canister shop vacuum with a wide floor nozzle across a dusty shop floor. Paul Bunyan, wearing his respirator, hangs a push broom and an air-blow nozzle on a wall peg, clearly setting them aside, not using them. |
| B7 | `images/pages/protection-fire-and-explosion.jpg` | Paul Bunyan emptying a dust collector's bin into a lidded metal trash can. A fire extinguisher hangs on the wall nearby. Oily finishing rags are laid flat to dry on a rack. Babe stands well back, calm. |
| B8 | `images/pages/symptoms.jpg` | A simple old-fashioned country doctor's office. Paul Bunyan, giant and slightly sheepish, sits on a reinforced examination table while a calm doctor in a white coat listens to his chest with a stethoscope. Babe peeks in through the doorway, concerned. |
| B9 | `images/pages/limits.jpg` | Paul Bunyan, huge hands held carefully, sets a tiny pinch of fine sawdust on one pan of a delicate brass balance scale, showing how little dust it takes. Babe watches the needle intently. The scale has no numbers. |
| B10 | `images/pages/glossary.jpg` | Babe the Blue Ox wearing small round spectacles, reading a giant dictionary propped open on a sawhorse, one hoof turning a page. The pages show no readable text. |
| B11 | `images/pages/methodology.jpg` (also used for `/sources`) | Paul Bunyan stacking books and bound journals neatly like a lumber pile, with spacer sticks between layers, checking each one off in an open ledger with a carpenter's pencil. Babe holds the ledger steady with his nose. No readable text on spines or pages. |

The `/print/shop-poster` page has no header of its own; it uses C8.

---

## C. Spot illustrations, border and poster

### C1–C6. Spot illustrations
- **Generate:** 1024×1024, transparent. **Final:** `sips -Z 512`.
- **Prompt:** the STYLE block, then "Scene: [scene], isolated, centered, simple bold linework that reads clearly at 64 pixels. Transparent background, PNG." Leave out the CHARACTERS block unless a scene mentions Paul or Babe.

| # | Save to | Scene |
|---|---|---|
| C1 | `images/art/spot-respirator.png` | A single half-face respirator with two round cartridges, three-quarter view |
| C2 | `images/art/spot-dust-collector.png` | A single barrel-style dust collector with a flexible hose |
| C3 | `images/art/spot-end-grain.png` | A single log round showing growth rings and a few checks |
| C4 | `images/art/spot-axe-and-saw.png` | A double-bit axe crossed with a crosscut saw |
| C5 | `images/art/spot-almanac.png` | An open hardbound almanac with a ribbon bookmark and a wood shaving, no readable text |
| C6 | `images/art/spot-babe-hoofprint.png` | A single large ox hoofprint, simple and graphic, for use as a list bullet |

### C7. Footer border, decorative
- **Save:** `images/art/border-forest.png`.
- **Generate:** 1536×1024, transparent. **Final:** crop to a band about 1536×256 around the artwork (`sips -c 256 1536`), keeping the alpha.
- **Scene:** a long, thin horizontal decorative border of a pine forest silhouette with a few stumps, a woodpile and, in one spot, a tiny distant Babe the Blue Ox. Designed to tile seamlessly left to right: the left and right edges must line up. Transparent background, PNG.
- **Check:** place two copies side by side and confirm the seam doesn't show.

### C8. Printable shop poster header
- **Save:** `images/art/poster-header.png`.
- **Generate:** 1536×1024 on cream. Keep it full size; it prints.
- **Scene:** Paul Bunyan standing in full shop gear (half-face respirator, safety glasses, earmuffs, gloves) giving a confident thumbs-up, with Babe the Blue Ox beside him wearing a comically oversized pair of safety goggles pushed up on his forehead. Bold enough to print well in black and red on a home printer. Cream paper background.

## D. Shop posters

Full-page portrait posters for printing (`docs/posters.md`). The layout adds the slogan, the QR code and the object frame around it in vector, so each scene must leave two plain areas: **a slogan area** and **a flat, blank, square surface facing the viewer** about one-sixth of the image width, where the QR code goes.

- **Generate:** 1024×1536 on cream, no transparency. Keep it full size; it is upscaled 2× before printing.
- **Build each prompt** as STYLE, then CHARACTERS, then the poster line below, then the Scene.
- **Poster line:** "Composed as a full-page poster: one bold scene, strong silhouettes, large shapes that read from across a room, woodcut and linocut texture in the hatching, limited palette of warm black ink, brick red and Babe blue on aged cream paper."
- **Check:** the mustache is trimmed and the chin is clean in every view, including reflections. Gear is worn correctly. There are no letters or fake writing on pages, labels, tags or crates. The blank QR surface is present and roughly square to the viewer.
- **Proof:** print a Letter copy before approving. Hatching can fill in on a laser printer.

### D1. Tall Tale (later)
- **Save:** `images/art/posters/tall-tale.png`.
- **Scene:** a tall-tale-scale lumber-camp scene in a pine forest with a low horizon. Paul Bunyan, giant, strides toward the viewer carrying an enormous barrel-style dust collector on one shoulder like a felled log, its flexible hose coiled over his other shoulder like rope. He wears work gloves and his safety glasses down over his eyes, and no respirator, because he is outdoors and not making dust. Babe the Blue Ox, lower right, leans into the traces of a wooden wagon stacked with sawn boards. The front of the wagon carries one plain square wooden crate facing the viewer squarely, with a blank, flat, light-coloured label area about one-sixth of the image width. The upper sixth of the image is open sky left plain for a title. No text anywhere.

### D2. Shave
- **Save:** `images/art/posters/shave.png`.
- **Scene:** a bunkhouse interior with plank walls, a peg rail and morning light. A round shaving mirror hangs on the wall at upper centre and reflects Paul Bunyan's face: his chin and jaw are covered in shaving lather, his short trimmed mustache is carefully kept clear of the lather, and his expression is calm and purposeful. Paul himself is seen from behind at three-quarter view, suspenders hanging down, a towel over one shoulder, a safety razor raised in his right hand. Beside the mirror a half-face respirator with two round filter cartridges hangs on a wooden peg. Babe the Blue Ox stands at the right holding a folded towel in his mouth like an attentive valet. Tucked into the mirror's frame at its right edge is a plain blank square card, flat and facing the viewer, about one-sixth of the image width. The lowest fifth of the image is plain cream wall left empty for a title. No text anywhere.

### D3. Sunbeam
- **Save:** `images/art/posters/sunbeam.png`.
- **Scene:** a dark, quiet timber-framed woodshop drawn mostly in dense cross-hatching, with one small window at the upper left. A single shaft of light crosses the room diagonally to the lower right, thick with floating fine dust motes, and it is the only light in the scene. Paul Bunyan sits on a sawhorse at the lower right wearing his half-face respirator with two round cartridges and wraparound safety glasses, hands on his knees, simply watching the beam. Babe the Blue Ox lies on the floor at the lower left with his nose resting in the light. At the foot of the beam a log round lies on its side so its end grain faces the viewer squarely, brightly lit, plain and uncluttered, about one-sixth of the image width. The upper right of the image is plain dark hatching left empty for a title. No text anywhere.

### D4. Sweep (later)
- **Save:** `images/art/posters/sweep.png`.
- **Scene:** a woodshop floor with a workbench and lumber rack behind. Babe the Blue Ox sits squarely in the centre like a big dog on a heap of push brooms, their wooden handles sticking out from beneath him in all directions, looking straight at the viewer with wide innocent eyes. A flat square wooden tag hangs from his leather halter against his chest, facing the viewer squarely, blank and light-coloured, about one-sixth of the image width. Paul Bunyan stands at the right wearing his half-face respirator with two round cartridges, safety glasses and gloves, one elbow resting on a canister shop vacuum with a wide floor nozzle, one eyebrow raised at Babe. The upper left third of the image is plain cream wall left empty for a title. No text anywhere.

### D5. Almanac (later)
- **Save:** `images/art/posters/almanac.png`.
- **Scene:** night in a clearing with a pine forest silhouette along the top. A giant hardbound almanac with a ribbon bookmark lies open and fills the middle of the image, its two pages plain blank cream with no writing, no lines and no decoration. Below it Paul Bunyan sits on a broad stump holding the book, and Babe the Blue Ox leans in over his shoulder wearing small round spectacles; a lantern on the stump lights both faces warmly. A single wood shaving curls off the book's gutter. The lower right corner is plain cream, left empty for a small seal. No text anywhere, including on the pages.

---

## Checklist: 26 files

```
images/art/_character-sheet.png        images/pages/risks.jpg
images/mascots/hero.png                images/pages/protection.jpg
images/art/emblem.png                  images/pages/protection-dust-collection.jpg
images/art/favicon.png                 images/pages/protection-respirators.jpg
images/og-default.jpg                  images/pages/protection-skin-and-eyes.jpg
images/mascots/babe-404.png            images/pages/protection-cleanup.jpg
images/mascots/no-results.png          images/pages/protection-fire-and-explosion.jpg
images/art/spot-respirator.png         images/pages/symptoms.jpg
images/art/spot-dust-collector.png     images/pages/limits.jpg
images/art/spot-end-grain.png          images/pages/glossary.jpg
images/art/spot-axe-and-saw.png        images/pages/methodology.jpg
images/art/spot-almanac.png            images/art/border-forest.png
images/art/spot-babe-hoofprint.png     images/art/poster-header.png
```

Posters (§D), round 4: `images/art/posters/shave.png`, `images/art/posters/sunbeam.png`.
