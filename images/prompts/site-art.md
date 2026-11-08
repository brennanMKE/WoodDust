# Dust Almanac — illustration prompts

Prompts for ChatGPT image generation (or a local model) to create the illustrations that give the site its personality. Species photos are separate: they stay photoreal and their prompts are in each `data/species/*.json` (`image_prompt`).

## How to use these

1. **Make the character sheet first (prompt 0).** Upload that image with every later prompt and say "keep Paul and Babe consistent with the attached character sheet". This is the single biggest factor in keeping the characters looking the same across images.
2. Paste the **Style block** followed by the prompt's **Scene**. The style block is already written into every full prompt below, so each one can be copied as-is.
3. ChatGPT sizes: square 1024×1024, landscape 1536×1024, portrait 1024×1536. The target crop is listed for each image; leave margin when cropping.
4. Ask for a **transparent background** where noted. ChatGPT supports it; say "transparent background, PNG".
5. Save to the listed path. The site picks images up automatically by filename.
6. Reject any result that looks like a known depiction: the Bemidji or Brainerd statues, Disney's 1958 short, or any specific old pamphlet page. If it does, regenerate with "more original character design".

## Style block (shared)

> Original illustration in the spirit of early-1900s American lumber-company promotional pamphlets: confident pen-and-ink line art with hand-drawn hatching, a limited two-color print look (warm black ink plus a brick-red second color) on aged cream paper, with occasional flat spot fills of a muted Babe blue (#4a78a8). Folksy, good-humored, tall-tale exaggeration of scale. Clean, readable silhouettes. No photographic detail, no gradients, no modern cartoon gloss. Do not copy or imitate any specific existing illustration, statue, film or character design. No text, letters, numbers, signatures or logos anywhere in the image.

## Character design (ours, for consistency)

- **Paul Bunyan:** giant, friendly lumberjack in his 40s. Broad shoulders, **clean-shaven chin and jaw with a short trimmed mustache** (a beard would break a respirator seal, and the site says so). Thick dark hair under a knit watch cap. Red-and-black buffalo-check flannel shirt with rolled sleeves, canvas work pants, suspenders, sturdy boots. Calm, capable expression.
- **Babe the Blue Ox:** enormous, gentle blue ox with wide, gently curving horns, a pale muzzle, big kind eyes and a curious, slightly comic manner. Plain leather halter.
- **Safety gear, drawn accurately:** a half-face reusable respirator with two round filter cartridges at the cheeks, wraparound safety glasses, earmuffs, nitrile or leather gloves. This is a safety site, so the gear must look correct and be worn correctly (straps on, mask sealed on a clean-shaven face).

---

## 0. Character sheet (reference only, not published)

**File:** `images/art/_character-sheet.png` · 1536×1024

> [Style block] Scene: a character reference sheet on plain cream paper. Left: Paul Bunyan shown front view, three-quarter view and side view, full body, standing relaxed: giant friendly lumberjack, clean-shaven chin and jaw with a short trimmed mustache, thick dark hair under a knit watch cap, red-and-black buffalo-check flannel shirt with rolled sleeves, canvas work pants, suspenders, boots. Beside him a close-up of his face wearing a half-face respirator with two round cartridges and wraparound safety glasses. Right: Babe the Blue Ox shown side view and three-quarter view: enormous gentle blue ox, wide gently curving horns, pale muzzle, big kind eyes, plain leather halter. Even spacing, no background scenery, no text or labels.

## 1. Home hero

**File:** `images/mascots/hero.png` · generate 1536×1024, transparent background · used about 720px wide
**Alt:** Paul Bunyan, wearing a respirator and safety glasses, sands a board at a workbench while Babe the Blue Ox watches a dust collector hose pull the dust away.

> [Style block] Scene: Paul Bunyan at a sturdy wooden workbench, sanding a plank with a hand sander. He wears a half-face respirator with two round cartridges, wraparound safety glasses and earmuffs. A flexible dust collection hose is attached to the sander and runs to a barrel dust collector beside the bench. Babe the Blue Ox leans in from the right, curious and approving, nose near the collector. A few fine dust motes are being pulled into the hose. Transparent background, PNG, no floor line beyond a short shadow.

## 2. Site emblem / logo mark

**File:** `images/art/emblem.png` · 1024×1024, transparent · also scaled down for the favicon
**Alt:** Dust Almanac emblem: Babe the Blue Ox's head above an open book.

> [Style block] Scene: a simple circular emblem. Babe the Blue Ox's head, facing forward, wide horns spanning the top of the circle, above an open hardbound almanac with a single wood shaving curling off its pages. Bold, simple linework that stays readable at 32 pixels. Two colors only: warm black and Babe blue on transparent. No text, no banner, no lettering. Transparent background, PNG.

**Favicon variant:** `images/art/favicon.png` · 1024×1024, transparent

> [Style block] Scene: only Babe the Blue Ox's head, facing forward, wide horns, extremely simplified with thick lines and a solid Babe blue face, designed to stay recognizable at 16 pixels. Transparent background, PNG, no text.

## 3. Default share image (Open Graph fallback)

**File:** `images/og-default.jpg` · generate 1536×1024, crop to 1200×630
**Alt:** Paul Bunyan and Babe the Blue Ox reading a giant open almanac in a woodshop.

> [Style block] Scene: Paul Bunyan sitting on a big log in a timber-framed woodshop, a giant open almanac on his knee. Babe the Blue Ox reads over his shoulder wearing small round spectacles. Clamps, hand planes and a dust collector in the background. Wide composition with the characters centered and empty margin on all sides for cropping to 1.91:1. Cream paper background, no text.

## 4. Not found (404)

**File:** `images/mascots/babe-404.png` · 1024×1024, transparent
**Alt:** Babe the Blue Ox peeks around a tall stack of lumber, looking puzzled.

> [Style block] Scene: Babe the Blue Ox peeking around the end of a tall stack of sawn lumber, one ear cocked, puzzled and gently comic, as if searching for a board that isn't there. Transparent background, PNG.

## 5. Search: no results

**File:** `images/mascots/no-results.png` · 1024×1024, transparent
**Alt:** Paul Bunyan studies the end grain of a log through a large magnifying glass.

> [Style block] Scene: Paul Bunyan crouching beside a log round, studying its end grain through an oversized magnifying glass, curious and patient. Transparent background, PNG.

## 6. Education page headers

Each is used as the page's share image (`images/pages/<file>.jpg`, cropped to 1200×630) and as a header illustration on the page. Generate at 1536×1024 on cream paper (no transparency) with margin for the crop.

### `/risks` — `images/pages/risks.jpg`
**Alt:** Paul Bunyan holds up a magnifying glass to a sunbeam full of fine floating dust in his shop.

> [Style block] Scene: a shaft of window light crossing a woodshop, full of fine floating dust specks. Paul Bunyan, wearing his respirator, holds a large magnifying glass up to the sunbeam to show how much fine dust is in the air. Babe looks at the beam with a wary, wrinkled nose. Wide composition, margin on all sides, cream paper background.

### `/protection` — `images/pages/protection.jpg`
**Alt:** Paul Bunyan and Babe stand at a workbench with a dust collector hose, a respirator, safety glasses and gloves laid out.

> [Style block] Scene: a tidy workbench. Laid out on it in order from left to right: a dust collector hose, then a half-face respirator, safety glasses and gloves. Paul Bunyan stands behind the bench gesturing at the gear like a proud instructor. Babe nods along. Wide composition, margin on all sides, cream paper background.

### `/protection/dust-collection` — `images/pages/protection-dust-collection.jpg`
**Alt:** Babe the Blue Ox pulls a wagon carrying a giant dust collector while Paul connects a hose to a table saw.

> [Style block] Scene: tall-tale scale. Babe the Blue Ox pulls a wooden wagon carrying an enormous barrel dust collector, while Paul Bunyan connects its flexible hose to the dust port of a table saw. Wide composition, margin on all sides, cream paper background.

### `/protection/respirators` — `images/pages/protection-respirators.jpg`
**Alt:** Paul Bunyan checks the seal of his half-face respirator, cupping the cartridges with his hands.

> [Style block] Scene: Paul Bunyan, clean-shaven chin with a trimmed mustache, wearing a half-face respirator with two round cartridges, cupping both palms over the cartridges and breathing in to check the seal, eyes focused. A small hand mirror is propped on the bench beside him. Babe watches closely from behind. Wide composition, margin on all sides, cream paper background.

### `/protection/skin-and-eyes` — `images/pages/protection-skin-and-eyes.jpg`
**Alt:** Paul Bunyan pulls on work gloves, wearing long sleeves and safety glasses, beside a basin for washing up.

> [Style block] Scene: Paul Bunyan with sleeves rolled down, wearing wraparound safety glasses, pulling on a pair of work gloves. On a stand beside him are a wash basin, a bar of soap and a towel. Babe sniffs the soap curiously. Wide composition, margin on all sides, cream paper background.

### `/protection/cleanup` — `images/pages/protection-cleanup.jpg`
**Alt:** Babe the Blue Ox pushes a shop vacuum while Paul hangs the broom and air nozzle on the wall, unused.

> [Style block] Scene: Babe the Blue Ox cheerfully pushing a canister shop vacuum with a wide floor nozzle across a dusty shop floor. Paul Bunyan, wearing his respirator, hangs a push broom and an air-blow nozzle on a wall peg, clearly setting them aside. Wide composition, margin on all sides, cream paper background.

### `/protection/fire-and-explosion` — `images/pages/protection-fire-and-explosion.jpg`
**Alt:** Paul Bunyan empties a dust collector bin into a lidded metal can, with a fire extinguisher on the wall behind him.

> [Style block] Scene: Paul Bunyan emptying a dust collector's bin into a lidded metal trash can. A fire extinguisher hangs on the wall nearby. Oily finishing rags are laid flat to dry on a rack. Babe stands well back, calm. Wide composition, margin on all sides, cream paper background.

### `/symptoms` — `images/pages/symptoms.jpg`
**Alt:** Paul Bunyan sits on an examination table while a country doctor listens to his chest with a stethoscope.

> [Style block] Scene: a simple old-fashioned country doctor's office. Paul Bunyan, giant and slightly sheepish, sits on a reinforced examination table while a calm doctor in a white coat listens to his chest with a stethoscope. Babe peeks in through the doorway, concerned. Wide composition, margin on all sides, cream paper background.

### `/limits` — `images/pages/limits.jpg`
**Alt:** Paul Bunyan weighs a tiny pinch of sawdust on a delicate brass balance scale.

> [Style block] Scene: Paul Bunyan, huge hands held carefully, sets a tiny pinch of fine sawdust on one pan of a delicate brass balance scale, showing how little dust it takes. Babe watches the needle intently. Wide composition, margin on all sides, cream paper background.

### `/glossary` — `images/pages/glossary.jpg`
**Alt:** Babe the Blue Ox, wearing spectacles, reads a giant dictionary propped on a sawhorse.

> [Style block] Scene: Babe the Blue Ox wearing small round spectacles, reading a giant dictionary propped open on a sawhorse, one hoof turning a page. Wide composition, margin on all sides, cream paper background.

### `/methodology` and `/sources` — `images/pages/methodology.jpg` (also used for `/sources`)
**Alt:** Paul Bunyan stacks books and journals like lumber, checking each one against a ledger.

> [Style block] Scene: Paul Bunyan stacking books and bound journals neatly like a lumber pile, with spacer sticks between layers, checking each one off in an open ledger with a carpenter's pencil. Babe holds the ledger steady with his nose. Wide composition, margin on all sides, cream paper background.

## 7. Spot illustrations and ornaments

All saved as transparent PNGs at 1024×1024 and displayed small (48–160px).

| File | Alt | Scene |
|---|---|---|
| `images/art/spot-respirator.png` | A half-face respirator. | A single half-face respirator with two round cartridges, three-quarter view. |
| `images/art/spot-dust-collector.png` | A barrel dust collector. | A single barrel-style dust collector with a flexible hose. |
| `images/art/spot-end-grain.png` | The end grain of a log. | A single log round showing growth rings and a few checks. |
| `images/art/spot-axe-and-saw.png` | A crossed axe and saw. | A double-bit axe crossed with a crosscut saw. |
| `images/art/spot-almanac.png` | An open almanac. | An open hardbound almanac with a ribbon bookmark and a wood shaving. |
| `images/art/spot-babe-hoofprint.png` | Babe's hoofprint. | A single large ox hoofprint, simple and graphic, for use as a list bullet. |

Prompt for each spot (fill in the Scene):

> [Style block] Scene: [scene from table], isolated, centered, simple bold linework that reads clearly at 64 pixels. Transparent background, PNG.

**Footer border:** `images/art/border-forest.png` · 1536×1024, transparent, cropped to a thin horizontal band
**Alt:** (decorative; empty alt)

> [Style block] Scene: a long, thin horizontal decorative border of a pine forest silhouette with a few stumps, a woodpile and, in one spot, a tiny distant Babe the Blue Ox. Designed to tile seamlessly left to right. Transparent background, PNG.

## 8. Printable shop poster header

**File:** `images/art/poster-header.png` · 1536×1024 on cream (prints in black and red)
**Alt:** Paul Bunyan in full shop safety gear gives a thumbs-up beside Babe the Blue Ox.

> [Style block] Scene: Paul Bunyan standing in full shop gear (half-face respirator, safety glasses, earmuffs, gloves) giving a confident thumbs-up, with Babe the Blue Ox beside him wearing a comically oversized pair of safety goggles pushed up on his forehead. Bold enough to print well in black and red on a home printer. Cream paper background.
