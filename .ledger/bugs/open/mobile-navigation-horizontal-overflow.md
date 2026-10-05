# Bug: Mobile Navigation Causes Horizontal Page Overflow

## Summary

The global application navigation retains its desktop width on a 390px
viewport, making the page horizontally scrollable.

## Reproduction

1. Open any authenticated page at a 390px-wide viewport.
2. Inspect the document scroll width or scroll horizontally.
3. Observe navigation controls extending beyond the viewport.

## Expected

Navigation adapts to the mobile viewport without creating horizontal page
overflow or displacing page content.

## Actual

The `.nav-right` element extends to approximately 638px on a 390px viewport.
Theme and logout controls are outside the initial viewport, and the document is
horizontally scrollable.

## Impact

Medium. Mobile users cannot access the full navigation without horizontal
scrolling, and screenshots taken after fullscreen transitions can appear
misaligned because horizontal scroll state is retained.

## Evidence

Observed with Playwright on 2026-07-12 while verifying workflow definition
diagrams:

- Viewport: `390x844`
- `.nav-right` right edge: approximately `638px`
- Workflow graph bounds: `x=24`, `width=342`, with no graph-local overflow
- Diagram controls did not overlap; the overflow source was the global nav

## Files Likely Involved

- `ui/src/components/Layout.tsx`
- `ui/src/index.css`
