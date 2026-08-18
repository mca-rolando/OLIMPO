# Future GitHub Workflows

No active CI/CD workflow is configured during the documentation bootstrap. Future workflows should be introduced through reviewed, least-privilege changes and should initially cover:

- documentation and Markdown linting plus relative-link validation;
- unit, component, API-contract, event-contract, integration, accessibility, visual-regression, resilience, security, and end-to-end tests;
- dependency and supply-chain review, secret scanning, and static analysis;
- reproducible application and package builds;
- controlled package publication with provenance; and
- tagged releases with changelog and compatibility checks.

Publication and deployment jobs must use protected environments, scoped short-lived credentials, explicit approvals, pinned actions, and artifact integrity controls. They must not be enabled until implementation and release governance are approved.
