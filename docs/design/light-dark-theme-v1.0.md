# Light, Dark, and System Theme v1.0

## Required modes

Every suite application supports native Light Mode, Dark Mode, and System preference. The quick theme control appears in the upper-right application area, is keyboard accessible, exposes its current selection, and can be operated without opening administration settings.

User preference should persist to the user profile when signed in and to safe local storage for bootstrap/anonymous contexts. System mode follows operating-system changes. Server rendering or initial paint should avoid a disruptive theme flash. A product may use an organizational default only until the user selects a permitted preference.

## Intentional palettes

Dark Mode is designed independently, not produced by inversion. Surface hierarchy, borders, elevation, muted text, charts, focus indicators, disabled controls, illustrations, and status colors each receive tested dark-theme values. Pure black/white extremes are used sparingly to reduce glare while retaining contrast.

Semantic roles—not raw color names—drive components: `surface`, `surface-raised`, `text-primary`, `text-muted`, `border`, `focus`, `status-success`, `status-warning`, `status-critical`, `status-info`, and `status-unknown`. Product accent tokens cannot alter status meaning.

## Validation

Each component is tested in all three modes for WCAG 2.2 AA contrast, focus visibility, hover/active/disabled states, forced colors where supported, high-density data, charts, loading/empty/error states, and 1920x1080 dashboard/kiosk use. Status always has a textual or symbolic equivalent. Visual-regression baselines cover both explicit themes; System mode validates preference switching.
