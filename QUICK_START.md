# Quick Start Guide - Testing the Redesigned Homepage

## Prerequisites
- Node.js and pnpm installed
- Project dependencies installed

## Start Development Server

```bash
# Navigate to project root
cd D:\CODE\xqecz

# Install dependencies (if not already done)
pnpm install

# Start the development server
pnpm run dev:fe
```

The development server will start at `http://localhost:5173` (or similar port).

## Visual Verification Checklist

### 1. Navigation Bar
- [ ] Glassmorphism backdrop blur visible
- [ ] Logo hover effect (lift + subtle shadow)
- [ ] Nav items have smooth hover states with color change
- [ ] Theme toggle button scales on hover
- [ ] Search input has proper styling
- [ ] User avatar and dropdown work correctly

### 2. Hero Section
- [ ] Animated grain texture visible (subtle movement)
- [ ] Large display title with decorative underline on "造物主"
- [ ] Stats ledger properly formatted with separators
- [ ] Primary CTA button has accent shadow and lift on hover
- [ ] "Recently lit" section with pulsing dot animation
- [ ] Live feed thumbnails scale on hover

### 3. Tag Discovery Section
- [ ] Tag chips have organic rounded corners
- [ ] Active tags show dashed inner border
- [ ] Hover effect lifts tags with color change
- [ ] Planet icons rotate on hover
- [ ] Shuffle button rotates 90° on hover

### 4. Content Cards
- [ ] Cards have organic varied corner radii
- [ ] Hover effect lifts cards with enhanced shadow
- [ ] Title changes color on card hover
- [ ] Like button heart scales on hover
- [ ] Liked hearts have bounce animation
- [ ] Tags have hover effects
- [ ] Text-only cards show "read more" animation

### 5. Footer
- [ ] Sparkle icon rotates on hover
- [ ] Proper spacing and typography
- [ ] License badge styled correctly
- [ ] Build date information visible

### 6. Dark Mode
Switch theme and verify:
- [ ] All colors invert properly
- [ ] Shadows remain visible
- [ ] Contrast is sufficient
- [ ] Accent colors remain visible

### 7. Responsive Behavior

**Mobile (< 768px):**
- [ ] Navigation switches to drawer
- [ ] Hero section stacks vertically
- [ ] FAB (floating action button) appears
- [ ] Cards display in 1-2 columns
- [ ] Touch targets are large enough

**Tablet (768px - 1024px):**
- [ ] 2-3 column card layout
- [ ] Navigation may show drawer or desktop nav
- [ ] Proper spacing maintained

**Desktop (> 1024px):**
- [ ] Full hover effects active
- [ ] Desktop navigation visible
- [ ] Optimal reading width maintained
- [ ] Multi-column card layout

### 8. Interactions & Animations
- [ ] All hover states work smoothly
- [ ] Click feedback (scale down) on buttons
- [ ] Loading spinner visible when loading
- [ ] Smooth transitions throughout
- [ ] No janky animations

### 9. Accessibility
- [ ] Tab key navigates through interactive elements
- [ ] Focus outlines visible (2px accent color)
- [ ] Enter key activates focused elements
- [ ] Escape closes modals/drawers
- [ ] Screen reader landmarks present

### 10. Performance
- [ ] Page loads without layout shift
- [ ] Images lazy load properly
- [ ] Animations run at 60fps
- [ ] No console errors

## Common Issues & Solutions

### Issue: Glassmorphism not visible
**Solution:** Ensure your browser supports `backdrop-filter`. Use Chrome/Edge 76+, Safari 14+, or Firefox 103+.

### Issue: Animations not smooth
**Solution:** Check if "Reduce Motion" is enabled in system preferences. Animations are intentionally minimized when this is enabled.

### Issue: Grain texture not visible
**Solution:** The grain is subtle by design. Look closely at the hero section background. It should have a very light stippled pattern.

### Issue: Colors look different
**Solution:** Verify you're viewing in the correct color mode (light/dark). Toggle with the sun/moon icon in the header.

### Issue: Hover effects not working
**Solution:** Hover effects only work on devices with a mouse pointer. On touch devices, interactions are tap-based.

## Files Changed (Quick Reference)

1. **DESIGN_PLAN.md** - Complete design system documentation
2. **IMPLEMENTATION_SUMMARY.md** - Detailed implementation notes
3. **src/assets/main.css** - Design tokens and global styles
4. **src/App.vue** - Navigation and footer styling
5. **src/views/HomeView.vue** - Hero, discovery, and feed sections
6. **src/components/WaterfallCard.vue** - Card component styling

## Rollback Instructions

If you need to revert the changes:

```bash
# Check git status
git status

# View changes
git diff

# Revert specific file
git checkout HEAD -- <filename>

# Or revert all changes
git reset --hard HEAD
```

## Report Issues

If you encounter any issues:

1. Check browser console for errors
2. Verify file paths are correct
3. Ensure all dependencies are installed
4. Try clearing browser cache
5. Test in different browsers

## Next Steps

After verifying the homepage:

1. **Gather feedback** from team members
2. **Test on real devices** (not just DevTools)
3. **Conduct user testing** with target audience
4. **Plan Phase 2** implementation (content detail pages)
5. **Iterate** based on feedback

---

**Quick Start Date**: 2026-10-18  
**Version**: 1.0
