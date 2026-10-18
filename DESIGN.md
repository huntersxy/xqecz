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
- Mode: both light and dark; density is airy at the top and balanced in the feed.

## Aesthetic

- Direction: warm paper, hand-drawn orbit lines, and a quiet moving constellation.
- Defining trait: horizontal recommendation strip leads into an asymmetric two-column Hero, then a compact coordinate filter and the content field.
- Signature move: the recommendation strip acts as a visual launch rail into a paper-like orbital Hero stage instead of a separate card section.

## Typography

- Display: existing `--creative-title-font` Kaiti/CJK serif stack.
- Body: existing site sans stack from `main.css`.
- Scale: fluid Hero title with `clamp()`; body remains at readable 13–14px with 1.7–1.85 line height.
- Measure: Hero copy stays below 55ch; long content remains left aligned.

## Color

- Strategy: retain the established warm paper and rose accent so the homepage remains continuous with Studio, upload, and catalog surfaces.
- Distribution: roughly 60% canvas/paper neutrals, 30% ink/muted text and rules, 10% rose/warm accents.
- Palette source: `packages/frontend/src/assets/main.css` `--creative-*` tokens; Hero does not add a second palette.

## Spacing, radius, shadow

- Spacing base: 4px, with 8/12/16/24/32/48px rhythm.
- Radius: small 10–12px controls and pill only where it denotes a control; no rounded Hero container.
- Shadow approach: defined edges and subtle texture; no diffuse shadow stacked with borders.

## Layout and composition

- Grid: asymmetric two-column desktop composition: copy and orbital stage; one column on mobile.
- Signature layout move: the Hero stage shares the paper surface with the copy and uses a fine divider to keep the reading edge clear.
- Responsive: desktop-first for the wide homepage, with a mobile single-column restack and the visual stage below the CTA.

## Components and states

- Primary button: filled rose action for creation.
- Secondary actions: text or outlined controls; one primary action per Hero.
- Focus: visible accent focus ring; all controls remain keyboard reachable.
- Data states: Hero metrics use live content totals but degrade to an em dash while content is loading.

## Motion

- Animate only opacity and transform; interaction feedback stays under 300ms while ambient motion loops slowly.
- Preserve the orbital illustration's existing reduced-motion behavior and pause the Canvas when hidden.

## Imagery and illustration

- Use the existing `UniverseSketch` and live artwork pool as the Hero visual, backed by `HeroOrbitCanvas.vue` for a deterministic looping particle field.
- Keep the hand-drawn orbit lines, light grain, and coordinate language; avoid generic gradients, stock images, dark sci-fi panels, or decorative card grids.
- Canvas owns its animation frame, ResizeObserver, visibility pause, reduced-motion fallback, and teardown; CSS owns two restrained slow orbit loops.

## Accessibility

- Keep `aria-labelledby="home-title"`, visible focus, semantic buttons, and readable text contrast.
- Decorative rail and orbit labels remain `aria-hidden` where they do not add content.
- Respect `prefers-reduced-motion` through the existing illustration styles.

## Slop audit

- Result: pass after Hero redesign.
- Checks: no new generic card grid, no purple gradient, no extra competing CTA, no nested Hero cards, and the signature launch-rail composition remains specific to this site.

## Changelog

- 2026-10-18: replaced the centered Hero treatment with an asymmetric editorial orbit composition while preserving existing creative tokens and content contracts.
- 2026-10-18: tuned the orbital stage back to the site's warm paper system and kept the Canvas particle field as a low-contrast ambient layer.
