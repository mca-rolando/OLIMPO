# Kiosk and NOC Layout v1.0

## Purpose

The reusable kiosk layout prioritizes persistent operational awareness, especially for ARGUS NOC dashboards. Its primary target is 1920x1080 full-screen browser operation, while remaining usable at documented alternative resolutions.

## Layout and behavior

- Hide or collapse the sidebar and omit administration/edit controls.
- Maximize dashboard area; essential KPIs, current critical conditions, site health, freshness, and time fit comfortably in one 1080p viewport without routine scrolling.
- Use high-legibility typography, restrained density, redundant status indicators, and deliberate Dark Mode suitable for prolonged viewing.
- Refresh automatically with visible last-update time, countdown/state, backoff, and a conspicuous stale/offline indicator. Refresh must not steal focus or reset operator context.
- Avoid unnecessary dialogs, transient-only information, hover-only actions, and animations. Honor reduced motion.
- Use a read-only Kiosk role with narrowly scoped entity access. Full-screen presentation does not weaken authentication, authorization, session, or data-minimization requirements.

```text
+------------------------------------------------------------------+
| ARGUS NOC      42 Sites Healthy   2 Warning   1 Critical  14:30  |
+-------------------------------+----------------------------------+
| Critical conditions           | Fleet / site health              |
| MIA-04 Connectivity outage    | [accessible dashboard chart]     |
| Evidence • duration • owner   |                                  |
+-------------------------------+----------------------------------+
| Recent significant events     | Integration / data freshness     |
+-------------------------------+----------------------------------+
```

## Acceptance criteria

Validate at exactly 1920x1080 in Light and Dark modes, including browser chrome assumptions and full screen. Confirm WCAG 2.2 AA contrast, no color-only state, keyboard reachability for available controls, readable critical details at viewing distance, no clipped content, bounded data density, stable auto-refresh, session-expiry behavior, and visible response to OLIMPO/product/network outages.
