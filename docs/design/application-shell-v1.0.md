# Application Shell v1.0

## Anatomy

The shared shell gives every suite product a familiar frame: a left navigation sidebar, top application bar, main content region, application switcher, global/federated search entry, notifications, help/context, user menu, and an upper-right quick theme selector.

```text
+------------------------------------------------------------------+
| Product / breadcrumb          Search        Alerts  Theme  User  |
+------------------+-----------------------------------------------+
| Applications     |                                               |
| OLIMPO            |                Main content                   |
| HERMES            |                                               |
| ARGUS             |                                               |
| METIS             |                                               |
|-------------------|                                               |
| Product nav       |                                               |
+------------------+-----------------------------------------------+
```

The application switcher lists only authorized products and uses registered base/deep-link contracts. With SSO, navigation normally does not prompt again, but each destination still validates its own token and authorization. Product navigation changes by domain; shell placement, keyboard behavior, responsive semantics, and accessibility remain shared.

## Interaction and resilience

The active product, page, organization/site context, theme, connection/freshness state, and pending critical notifications are clear. Deep links use stable identifiers and resolve at the destination. Cross-product search labels source and freshness. If OLIMPO is unavailable, a product can render a locally packaged shell subset, retain product navigation and theme, and mark switcher/search/central notifications as degraded rather than blocking domain work.

On narrower screens the sidebar collapses into an accessible disclosure; content reflows rather than simply clipping. Touch targets, skip links, focus restoration, landmarks, headings, and browser history are specified and tested. Administration controls require authorization and are never the sole enforcement point.
