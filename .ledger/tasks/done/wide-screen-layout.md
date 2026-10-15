# Use available width on large displays

Work directly on `main`. Replace the shared 1,240px page/navigation cap with
Breadboard's centered 90% width. Use 100% at tablet widths and smaller gutters
on narrow screens. Tables, graphs and page headers should expand together.

Completed directly on `main` on 2026-10-15. Shared navigation and page containers
use 90% width without a fixed maximum, 100% at 1,024px and below, and 16px gutters
at 768px and below. Individual form and modal limits remain for readability.

Verification:

- `npm run build` passed; existing Vite bundle-size warning remains.
- Playwright checked all seven main pages with mocked API responses at
  3,840px: page/navigation width 3,456px, table content width 3,408px, and no
  horizontal page overflow. Previously usable content was limited to 1,192px.
- Checked aligned navigation/content and no horizontal page overflow at
  2,560, 1,920, 1,280, 1,152, 1,100, 1,025 and 1,024px.
- The existing [mobile navigation overflow](../../bugs/open/mobile-navigation-horizontal-overflow.md)
  remains at narrow sizes; the main content fits its viewport.
- `make host-rebuild-markovd` completed. Playwright checked the deployed assets
  at `https://markovd.local/runs` with mocked API responses and confirmed the
  same 3,408px table width at 4K.
- `git diff --check` passed.
