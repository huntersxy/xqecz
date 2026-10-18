# 小泉宇宙 (Xiaoquan Universe) - Comprehensive Design Plan
## ACG Fan Creation Platform Redesign

---

## Executive Summary

This document outlines a bold, sophisticated redesign of the 小泉宇宙 (Xiaoquan Universe) platform, an ACG (Anime, Comics, Games) fan art and derivative works sharing community. The redesign balances premium aesthetic sophistication with clean minimalism, creating a distinctive visual identity that elevates the platform beyond generic content sites.

**Implementation Scope (Phase 1):**
- Homepage redesign (navigation, hero, content feed, footer)
- Design system documentation for future phases
- All other pages remain unchanged but planned

**Core Design Principles:**
1. **Sophisticated Premium Feel** - Refined typography, elegant spacing, subtle animations
2. **Clean Minimalism** - Breathing room, intentional whitespace, focused hierarchy
3. **ACG-Centric Identity** - Visual language that resonates with fan creators
4. **Functional Beauty** - Every design choice serves both form and function

---

## I. Design System Foundation

### 1.1 Color Palette

**Primary Palette (Enhanced Creative Theme)**
```
--creative-accent: #c24d6b (Deeper rose, more sophisticated)
--creative-accent-hover: #b03d5a
--creative-paper: #fffefe (Pure white with warm hint)
--creative-canvas: #f8f5f6 (Softer background)
--creative-soft: #fce8ee (Lighter accent background)
--creative-ink: #2d2428 (Richer black)
--creative-muted: #75676f (Refined gray)
--creative-line: #ede4e8 (Softer borders)
```

**Dark Mode Palette**
```
--creative-paper: #221e20
--creative-canvas: #1a1618
--creative-soft: #3a2d31
--creative-ink: #f5ede8
--creative-muted: #c0b4ae
--creative-line: #3d3437
--creative-accent: #f5a3b5
--creative-accent-hover: #ffbdcd
```

**Semantic Colors**
- Success: `#4a9d6f` (Muted green)
- Warning: `#d48b3e` (Warm amber)
- Error: `#d64a5a` (Soft red)
- Info: `#5d7a9e` (Muted blue)

### 1.2 Typography System

**Font Families**
```css
--font-display: 'Playfair Display', 'Noto Serif SC', 'STKaiti', serif
--font-body: 'Inter', -apple-system, 'PingFang SC', sans-serif
--font-accent: 'Kaiti SC', 'LXGW WenKai', 'Noto Serif CJK SC', serif
--font-mono: 'JetBrains Mono', 'Menlo', monospace
```

**Type Scale**
- Display: 56-72px (Hero titles)
- H1: 40-48px (Page titles)
- H2: 28-36px (Section headers)
- H3: 20-24px (Subsection headers)
- Body Large: 16-18px
- Body: 14-15px
- Body Small: 12-13px
- Caption: 10-11px

**Type Styles**
- Line height: 1.5-1.7 for body, 1.1-1.3 for headings
- Letter spacing: 0.01-0.02em for headings, normal for body
- Font weights: 400 (regular), 500 (medium), 600 (semibold), 700 (bold)

### 1.3 Spacing System

**Base Unit: 4px**
```
Space-1: 4px
Space-2: 8px
Space-3: 12px
Space-4: 16px
Space-5: 20px
Space-6: 24px
Space-8: 32px
Space-10: 40px
Space-12: 48px
Space-16: 64px
Space-20: 80px
Space-24: 96px
```

**Layout Containers**
- Max width: 1440px (main content)
- Narrow: 960px (reading content)
- Padding: 28px desktop, 16px mobile

### 1.4 Elevation & Shadows

**Shadow System**
```css
--shadow-sm: 0 2px 8px rgba(83, 57, 35, 0.04)
--shadow-md: 0 4px 16px rgba(83, 57, 35, 0.08)
--shadow-lg: 0 12px 36px rgba(83, 57, 35, 0.12)
--shadow-xl: 0 20px 48px rgba(83, 57, 35, 0.16)
--shadow-accent: 0 6px 20px rgba(194, 77, 107, 0.24)
```

### 1.5 Border Radius

**Radius Scale**
```
Radius-sm: 6px (small elements)
Radius-md: 10px (cards, inputs)
Radius-lg: 16px (large cards)
Radius-xl: 24px (modals, panels)
Radius-full: 999px (pills, buttons)
```

**Organic Variation** (for cards): Slightly varied corners (e.g., `12px 14px 11px 15px`) to add hand-crafted feel

### 1.6 Motion & Animation

**Timing Functions**
```css
--ease-smooth: cubic-bezier(0.4, 0.0, 0.2, 1)
--ease-bounce: cubic-bezier(0.68, -0.55, 0.265, 1.55)
--ease-gentle: cubic-bezier(0.25, 0.46, 0.45, 0.94)
```

**Duration Scale**
- Micro: 150ms (hover states)
- Short: 250ms (simple transitions)
- Medium: 350ms (complex transitions)
- Long: 500ms (page transitions)

**Interaction Patterns**
- Hover: Scale 1.02-1.05, subtle shadow increase
- Active: Scale 0.98
- Focus: 2px outline with 3-4px offset
- Loading: Gentle pulse or rotate

---

## II. Component Design Specifications

### 2.1 Navigation Bar (Top Header)

**Desktop (>1360px)**
- Height: 72px
- Background: Semi-transparent backdrop blur (glassmorphism)
- Sticky with shadow on scroll
- Logo: 36px height, left-aligned with hover effect
- Nav items: Pill-shaped buttons, 12px font, 600 weight
- Search: Expandable right-aligned, 360px max width
- Avatar: 32px circle with dropdown

**Mobile (≤1360px)**
- Height: 64px
- Hamburger drawer navigation
- Search prominent on homepage
- Collapsed logo/brand mark

**Styling**
```css
backdrop-filter: blur(12px) saturate(1.8);
background: rgba(255, 254, 254, 0.85);
border-bottom: 1px solid var(--creative-line);
box-shadow: 0 1px 0 rgba(255, 255, 255, 0.8), 
            0 4px 16px rgba(83, 57, 35, 0.04);
```

### 2.2 Hero Section (Homepage)

**Layout**
- Two-column grid on desktop (60/40 split)
- Stacked on mobile
- Padding: 48-64px vertical

**Left Column: Copy**
- Display title: 56-72px, serif font
- Emphasized word with decorative underline
- Stats ledger: Tabular numbers, muted color
- Lede text: 15px, 1.75 line height, max 40em
- CTA buttons: Primary (filled) + Secondary (ghost)

**Right Column: Live Preview**
- Animated planet visualization (existing UniverseSketch)
- "Recently Lit" live feed with thumbnails
- Subtle grain texture overlay

**Enhancements**
- Grain texture: `radial-gradient(circle, rgba(...) 0.8px, transparent 1px)` at 5px size
- Decorative elements: Sparkle icons (✦, ✧) as accent marks
- Subtle parallax on scroll (optional)

### 2.3 Content Cards (Waterfall Layout)

**Card Structure**
- Organic rounded corners (varied radii)
- 6px padding around entire card
- Hover: Lift with shadow increase + border accent
- Focus: Visible outline for accessibility

**Image Cards**
- Media container: 11px border radius
- Aspect ratio: Natural (no forced crop)
- Lazy loading with blur-up placeholder
- AI badge: Top-left, glassmorphism style

**Text-Only Cards**
- Gradient background with grain texture
- Decorative quote mark
- 5-line excerpt clamp
- "Read more" link bottom-right

**Card Footer**
- Title: 14-15px, 600 weight, 2-line clamp
- Meta row: Avatar (22px) + username + like button
- Tags: Pill-shaped, muted background, 3 max visible
- Spacing: 10-13px between elements

**Interaction States**
```css
/* Hover */
transform: translateY(-2px);
box-shadow: var(--shadow-lg);
border-color: color-mix(in srgb, var(--creative-accent) 45%, var(--creative-line));

/* Active/Click */
transform: translateY(0) scale(0.99);
```

### 2.4 Tag/Filter System

**Tag Chips**
- Organic rounded shape (varied corners)
- Planet icon + label
- Active state: Filled background + dashed inner border
- Hover: Color shift + scale 1.03
- Shuffle button: Circular, icon-only on mobile

**Tag Colors**
- Use consistent accent color for all tags
- Active: Filled soft background
- Inactive: Paper background with border

### 2.5 Footer

**Layout**
- Two-column: Brand info (left) + Legal/Build info (right)
- Centered on mobile
- Border-top: 1px solid line
- Background: Paper color
- Padding: 32-40px vertical

**Brand Column**
- Sparkle icon (40px circle)
- Site name + tagline
- Serif font for name

**Legal Column**
- Copyright + license badge
- ICP filing (conditional)
- Build date + "With Love" message
- Right-aligned on desktop

**Styling**
```css
.footer-badge {
  padding: 4px 10px;
  border: 1px solid var(--creative-line);
  border-radius: 999px;
  background: var(--creative-canvas);
  font-size: 9px;
  letter-spacing: 0.06em;
}
```

### 2.6 Buttons & Interactive Elements

**Primary Button**
- Background: Accent color
- Text: White, 600 weight
- Border-radius: 999px (pill)
- Padding: 12px 20px
- Shadow: Accent shadow on hover
- Icon + text spacing: 8px

**Secondary Button**
- Border: 1px solid line
- Background: Paper
- Text: Accent color
- Same radius and padding

**Ghost Button**
- No border, no background
- Text: Muted → Accent on hover
- Minimal padding

**Icon Buttons**
- Circular (36-44px)
- Muted color → Accent on hover
- Soft background on hover

### 2.7 Forms & Inputs

**Text Input**
- Height: 40-44px
- Border: 1px solid line
- Border-radius: 10px
- Background: Canvas color
- Focus: Accent border + subtle glow
- Placeholder: Muted color

**Search Input**
- Pill-shaped (999px radius)
- Icon inside (left or right)
- Expandable on focus (optional)

**Textarea**
- Same styling as input
- Min-height: 120px
- Resize: vertical only

---

## III. Page-by-Page Design Plan

### 3.1 Homepage (Phase 1 - Implement Now)

**Sections (Top to Bottom)**
1. Navigation Bar (sticky)
2. Recommend Section (if content exists)
3. Hero Section (title, stats, CTA, live preview)
4. Discovery Section (trending tags)
5. Ideas Section (existing component)
6. Content Feed (waterfall masonry)
7. Footer

**Key Changes**
- Enhanced typography hierarchy
- Refined spacing and breathing room
- Improved card hover states
- Better color contrast and accessibility
- Organic, hand-crafted feel with varied corners
- Subtle textures (grain overlay)

### 3.2 Content Detail Page (Phase 2 - Future)

**Layout**
- Full-screen overlay on mobile
- Side-by-side on desktop (image left, info right)
- Close button: Top-right, circular
- Smooth enter/exit animations

**Content Area**
- Large image viewer with zoom
- Title: Display size, serif font
- Description: Comfortable reading width
- Tags: Interactive, clickable
- Actions: Like, share, report
- Comments: Threaded, expandable

**Enhancements**
- Image gallery with thumbnails
- Lightbox zoom on click
- Related content suggestions
- Creator profile card

### 3.3 Quick Upload / AI Studio (Phase 2 - Future)

**Quick Upload Sheet**
- Bottom sheet on mobile
- Modal on desktop
- Drag-drop zone with visual feedback
- Form fields: Grouped logically
- Auto-metadata suggestions
- Progress indicator during upload

**AI Studio**
- Canvas workspace
- Tool palette (left sidebar)
- Settings panel (right sidebar)
- Reference library (bottom drawer)
- Size picker: Visual grid
- Generate button: Prominent, animated

**Styling**
- Darker background for focus
- Tools: Icon + label, active state
- Settings: Grouped with dividers
- Canvas: Centered, max-width constrained

### 3.4 Characters / Ideas / Polls (Phase 2 - Future)

**Characters (Creative Catalog)**
- Grid layout: 3-4 columns desktop, 2 mobile
- Character cards: Portrait + name + tags
- Detail view: Full-screen or drawer
- Filter/sort: Top toolbar

**Ideas Section**
- Horizontal scroll on mobile
- Grid on desktop
- Idea cards: Icon + title + description
- Interaction: Click to expand

**Polls Section**
- Poll cards: Question + options preview
- Battle mode: Side-by-side comparison
- Vote button: Prominent, accent color
- Results: Animated bar chart

### 3.5 Login Page (Phase 2 - Future)

**Layout**
- Centered card (max 400px width)
- Minimal distractions
- Background: Subtle pattern or gradient

**Form**
- Email + password fields
- Remember me checkbox
- Login button: Full width, accent
- Social login: Optional, below
- Register link: Below form

**Styling**
- Card: Elevated, soft shadow
- Fields: Stacked with labels
- Focus states: Prominent
- Error messages: Below field, error color

### 3.6 Admin Panel (Phase 2 - Future)

**Layout**
- Top nav: Same as public site
- Side nav: Admin sections (sticky)
- Main content: Data tables, forms, charts

**Admin Components**
- Data tables: Sortable, filterable, paginated
- Status badges: Color-coded
- Action buttons: Small, secondary style
- Charts: Muted colors, clear labels
- Logs: Monospace font, terminal-like

**Styling**
- Keep Arco Design system for admin
- Match color palette where possible
- Maintain functional clarity over decoration

---

## IV. Responsive Design Strategy

### 4.1 Breakpoints

```css
/* Mobile First Approach */
--breakpoint-xs: 0px      /* Phone portrait */
--breakpoint-sm: 640px    /* Phone landscape */
--breakpoint-md: 768px    /* Tablet portrait */
--breakpoint-lg: 1024px   /* Tablet landscape */
--breakpoint-xl: 1280px   /* Laptop */
--breakpoint-2xl: 1440px  /* Desktop */
--breakpoint-nav: 1360px  /* Nav switches to drawer */
```

### 4.2 Mobile Optimizations

**Navigation**
- Hamburger drawer (left side)
- Search bar: Prominent on homepage
- User avatar: Visible but compact

**Hero Section**
- Single column layout
- Planet: Smaller, above live feed
- CTA buttons: Stacked or single primary

**Content Feed**
- 1-2 columns waterfall (screen width dependent)
- Larger tap targets (min 44px)
- Simplified card info (hide secondary metadata)

**Footer**
- Stacked layout
- Left-aligned text
- Reduced padding

### 4.3 Tablet Considerations

- 2-3 column waterfall
- Keep desktop nav if width allows
- Larger touch targets throughout

### 4.4 Desktop Enhancements

- Multi-column layouts
- Hover states and micro-interactions
- Expanded metadata and information
- Keyboard navigation support

---

## V. Accessibility Standards

### 5.1 Color Contrast

- **Text on background**: Minimum 4.5:1 ratio
- **Large text (≥18px)**: Minimum 3:1 ratio
- **Interactive elements**: 3:1 against adjacent colors
- Test both light and dark modes

### 5.2 Focus States

- Visible focus ring: 2px solid accent, 3-4px offset
- Focus order: Logical, follows visual layout
- Skip-to-content link for keyboard users

### 5.3 ARIA Labels & Semantics

- All interactive elements have labels
- Images have alt text (decorative: alt="")
- Buttons indicate state (aria-pressed, aria-expanded)
- Live regions for dynamic content updates

### 5.4 Keyboard Navigation

- Tab order: Natural and intuitive
- Enter/Space: Activate buttons and links
- Escape: Close modals and drawers
- Arrow keys: Navigate grids and carousels

### 5.5 Screen Reader Support

- Semantic HTML: nav, main, article, aside, footer
- Heading hierarchy: h1 → h2 → h3 (no skips)
- Landmark regions labeled
- Form labels associated with inputs

### 5.6 Motion Preferences

```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
  /* Keep loading indicators animated (slower) */
}
```

---

## VI. Performance Considerations

### 6.1 Image Optimization

- Lazy loading: `loading="lazy"` on images
- Responsive images: `srcset` for different densities
- Modern formats: WebP with fallbacks
- Blur placeholder: Low-res base64 before load
- Dimensions: Specify width/height to prevent layout shift

### 6.2 CSS Optimization

- Critical CSS: Inline above-fold styles
- CSS containment: `contain: layout paint` on cards
- Custom properties: Use for theming (already implemented)
- Minimize specificity: Use classes over nesting

### 6.3 JavaScript Performance

- Code splitting: Route-based chunks
- Lazy components: Import on-demand
- Debounce: Scroll and resize handlers
- RequestIdleCallback: Preload non-critical resources

### 6.4 Waterfall Layout Performance

- Virtual scrolling: Render visible items only (future optimization)
- Position caching: Store calculated positions
- Batch updates: Group DOM writes
- ResizeObserver: Efficient resize detection

---

## VII. Implementation Roadmap

### Phase 1: Homepage Redesign (Current)

**Files to Modify**
1. `packages/frontend/src/App.vue` - Navigation and footer
2. `packages/frontend/src/views/HomeView.vue` - Hero and feed sections
3. `packages/frontend/src/components/WaterfallCard.vue` - Card styling
4. `packages/frontend/src/assets/main.css` - Design tokens and global styles

**Tasks**
- [ ] Update CSS custom properties (color palette)
- [ ] Enhance typography system
- [ ] Refine navigation bar styling
- [ ] Redesign hero section layout and styling
- [ ] Update content card design
- [ ] Improve tag chip styling
- [ ] Redesign footer
- [ ] Test responsive behavior
- [ ] Verify accessibility
- [ ] Check dark mode

**Success Criteria**
- Visual polish significantly improved
- Existing functionality fully intact
- No regressions in responsive layout
- Accessibility maintained or improved
- Performance unchanged or better

### Phase 2: Content Detail & Upload (Future)

- Content detail overlay redesign
- Quick upload sheet styling
- Modal and drawer components
- Form styling updates

### Phase 3: Secondary Pages (Future)

- Characters catalog
- Ideas section expansion
- Polls interface
- Login page redesign

### Phase 4: Admin Panel (Future)

- Admin navigation
- Data tables styling
- Dashboard widgets
- Settings panels

### Phase 5: Polish & Optimization (Future)

- Micro-interactions and animations
- Loading states and skeletons
- Error states and empty states
- Performance audit and optimization
- Accessibility audit

---

## VIII. Design Tokens Reference

### Complete CSS Custom Properties

```css
:root {
  /* Colors - Light Mode */
  --creative-accent: #c24d6b;
  --creative-accent-hover: #b03d5a;
  --creative-paper: #fffefe;
  --creative-canvas: #f8f5f6;
  --creative-soft: #fce8ee;
  --creative-ink: #2d2428;
  --creative-muted: #75676f;
  --creative-line: #ede4e8;
  --creative-warm: #e8c590;
  
  /* Shadows */
  --shadow-sm: 0 2px 8px rgba(83, 57, 35, 0.04);
  --shadow-md: 0 4px 16px rgba(83, 57, 35, 0.08);
  --shadow-lg: 0 12px 36px rgba(83, 57, 35, 0.12);
  --shadow-xl: 0 20px 48px rgba(83, 57, 35, 0.16);
  --shadow-accent: 0 6px 20px rgba(194, 77, 107, 0.24);
  
  /* Typography */
  --font-display: 'Playfair Display', 'Noto Serif SC', 'STKaiti', serif;
  --font-body: 'Inter', -apple-system, 'PingFang SC', sans-serif;
  --font-accent: 'Kaiti SC', 'LXGW WenKai', serif;
  
  /* Spacing (4px base) */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;
  --space-12: 48px;
  --space-16: 64px;
  
  /* Border Radius */
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 16px;
  --radius-xl: 24px;
  --radius-full: 999px;
  
  /* Transitions */
  --ease-smooth: cubic-bezier(0.4, 0.0, 0.2, 1);
  --ease-bounce: cubic-bezier(0.68, -0.55, 0.265, 1.55);
  --transition-fast: 150ms var(--ease-smooth);
  --transition-base: 250ms var(--ease-smooth);
  --transition-slow: 350ms var(--ease-smooth);
}

[arco-theme='dark'] {
  --creative-accent: #f5a3b5;
  --creative-accent-hover: #ffbdcd;
  --creative-paper: #221e20;
  --creative-canvas: #1a1618;
  --creative-soft: #3a2d31;
  --creative-ink: #f5ede8;
  --creative-muted: #c0b4ae;
  --creative-line: #3d3437;
  --creative-warm: #b89260;
  
  --shadow-sm: 0 2px 8px rgba(0, 0, 0, 0.12);
  --shadow-md: 0 4px 16px rgba(0, 0, 0, 0.18);
  --shadow-lg: 0 12px 36px rgba(0, 0, 0, 0.24);
  --shadow-xl: 0 20px 48px rgba(0, 0, 0, 0.32);
}
```

---

## IX. Conclusion

This comprehensive design plan establishes a sophisticated, premium visual identity for 小泉宇宙 that respects its ACG community roots while elevating the platform's aesthetic to match the creativity of its users. The phased approach ensures systematic implementation without disrupting existing functionality.

**Key Differentiators**
- Refined color palette with enhanced sophistication
- Typography hierarchy that guides attention naturally
- Organic shapes and subtle textures adding hand-crafted feel
- Thoughtful spacing creating breathing room
- Smooth interactions that feel responsive and alive

**Next Steps**
1. Review and approve this design plan
2. Begin Phase 1 implementation (homepage)
3. Gather user feedback on redesigned homepage
4. Iterate and refine based on feedback
5. Plan Phase 2 implementation timeline

---

**Document Version**: 1.0  
**Last Updated**: 2026-10-18  
**Author**: Design System Team  
**Status**: Ready for Implementation
