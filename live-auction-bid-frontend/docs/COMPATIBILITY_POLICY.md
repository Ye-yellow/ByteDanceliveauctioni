# Compatibility Policy

Status: `PRE_LAUNCH_NO_LEGACY_COMPAT`

This project is not launched yet. During the pre-launch phase, do not preserve old behavior, old payload shapes, old route semantics, or silent compatibility fallbacks. Prefer a clean contract and fail fast when the contract is not satisfied.

## Current Rule

- Do not keep legacy branches, legacy adapters, dual payload parsing, or old UI compatibility paths.
- Do not add fallback values for required backend fields, required frontend state, required environment variables, or required route parameters.
- Missing required data must fail fast with a clear error path.
- Empty states are allowed only for valid empty business data, such as an empty list returned successfully by the API.
- API/schema mismatches should be treated as integration errors, not hidden by UI placeholders.
- New code should follow the current architecture, naming, typing, and component conventions instead of copying old code directly.

## Project-Specific Expectations

### Admin Frontend

- Required dashboard, auction, order, user, and realtime fields should be rendered from explicit contracts.
- Do not display misleading placeholders such as `未设置`, `--`, or static demo data for missing required production fields.
- If a required API response field is absent, surface the integration error instead of silently degrading the page.

### Backend

- Validate required request fields and reject invalid payloads explicitly.
- Startup should fail if required configuration is missing.
- Do not maintain old request/response shapes unless the project status changes to post-launch compatibility mode.

### User H5

- Do not add adapters for old auction state payloads.
- Do not hide missing required auction, bid, payment, or room state behind generic fallback UI.
- Blocking contract failures should be visible during development and testing.

## Launch Transition Marker

Only an explicit product decision can change this policy.

When the product owner says the system is launched and old clients/users must be supported, update this file in all three projects:

- Change status to `POST_LAUNCH_COMPAT_REQUIRED`.
- Document the compatibility window.
- List the old API payloads, routes, UI states, or storage formats that must remain supported.
- Add migration or adapter tests for every supported legacy contract.

Until that update is made, the active policy remains: no legacy compatibility and fail fast on missing contracts.
