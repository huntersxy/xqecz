# XQECZ creative homepage design

## Discovery

- Artifact: creative community homepage and content browser.
- Audience: people browsing and publishing anime fan creations.
- Primary action: discover a work or light a new creative planet.
- Positioning: warm, creator-led, exploratory, and hand-drawn rather than a generic AI gallery.
- Adjectives: warm, exploratory, handcrafted, restrained.
- Aesthetic essence: warm editorial orbit.
- Single-minded proposition: every upload can become a small world worth visiting.
- Archetype: Creator + Explorer.
- Mode: both light and dark; the masthead is dense and compact so the content feed owns the first viewport.

## Aesthetic

- Direction: warm paper, hand-drawn orbit lines, and a quiet moving constellation.
- Defining trait: horizontal recommendation strip leads into a compact masthead — title, live ledger, latest-work index and a small interactive artwork planet — then the coordinate filter and the feed.
- Signature move: the artwork planet is reduced to a seal beside a three-row live index, so the waterfall feed starts inside the first viewport instead of below a decorative stage.

## Typography

- Display: existing `--creative-title-font` Kaiti/CJK serif stack.
- Body: existing site sans stack from `main.css`.
- Scale: one-line masthead title at `clamp(40px, 4.6vw, 62px)`; intro copy 13px, live index 12px with 10.5px metadata. No display step above 6rem.
- Measure: intro copy stays below 40em; metadata truncates with ellipsis rather than wrapping.

## Color

- Strategy: retain the established warm paper and rose accent so the homepage remains continuous with Studio, upload, and catalog surfaces.
- Distribution: roughly 60% canvas/paper neutrals, 30% ink/muted text and rules, 10% rose/warm accents.
- Palette source: `packages/frontend/src/assets/main.css` `--creative-*` tokens; Hero does not add a second palette.

## Spacing, radius, shadow

- Spacing base: 4px, with 8/12/16/24/32/48px rhythm.
- Radius: small 10–12px controls and pill only where it denotes a control; no rounded Hero container.
- Shadow approach: defined edges and subtle texture; no diffuse shadow stacked with borders.

## Layout and composition

- Grid: a single dense band. Copy and live ledger on the left; planet, star-map entry and three-row latest index on the right, separated by a 1px column rule (no card, no filled panel).
- Signature layout move: title and ledger share one baseline row (`space-between`), while the live index aligns to the planet's full height.
- Responsive: below 900px the band stacks with a hairline above the live block; below 768px the non-interactive planet is dropped and the index runs full width. Desktop band target height is 210–270px.

## Components and states

- Primary button: filled rose action for creation.
- Secondary action: text button that scrolls to the feed, with a moving down-arrow on hover.
- Live ledger: works total, tag total and today's additions, separated by middots; numerals use tabular figures and ink weight, not a metric card.
- Live index: three newest titled works (UUID-only titles filtered out) with thumbnail, author and date; a container-height placeholder prevents layout shift while loading.
- Planet: container-sized interactive globe with a visible “进入星图” text entry and a small pause control; it remains the only entry to the star map.
- Focus: visible accent focus ring; all controls remain keyboard reachable.
- Data states: totals degrade to an em dash while content is loading.

## Motion

- Motion is carried by the planet's slow rotation; interaction feedback stays under 300ms and no decorative canvas loop runs in the masthead.
- Rotation pauses with `prefers-reduced-motion`, when hidden, off-screen, or via the visible pause control.

## Imagery and illustration

- Use the existing `UniverseSketch` and live artwork pool as the compact masthead visual; the planet's render buffer follows its container so the larger seal stays sharp.
- Keep one pencil-thin orbit ellipse, light grain, and the coordinate vocabulary; avoid telemetry labels, fake affordances, stock imagery, dark sci-fi panels, or decorative card grids.
- `ArtworkPlanet` owns its animation frame, container resize, visibility pause, reduced-motion fallback and teardown; the masthead adds no second animated canvas.

## Accessibility

- Keep `aria-labelledby="home-title"`, visible focus, semantic buttons, and readable text contrast.
- Decorative orbit strokes remain `aria-hidden`; visible labels are real links or controls.
- Respect `prefers-reduced-motion` for rotation and hover transitions.

## Slop audit

- Result: pass after the dense masthead redesign.
- Checks: no kicker or eyebrow above the title, no section numbers, no monospace telemetry, no fake drag/scroll affordances, no metric-card triad, no nested cards, no gradient text; one primary action and one real secondary path.

## Changelog

- 2026-10-18: replaced the centered Hero treatment with an asymmetric editorial orbit composition while preserving existing creative tokens and content contracts.
- 2026-10-18: tuned the orbital stage back to the site's warm paper system and kept the Canvas particle field as a low-contrast ambient layer.
- 2026-10-18: replaced the split Hero card with a dense masthead (title, live ledger, three-row latest index, compact artwork planet) so the feed begins in the first viewport; removed the orbit Canvas and all telemetry labels.
