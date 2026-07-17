# BizTop Operating Cockpit Implementation Plan

This plan turns BizTop from a financial dashboard into a CEO cockpit for running
Davai toward `docs/DAVAI_2030.md`.

The current app answers:

- How much revenue was invoiced?
- How much cash was collected?
- What does the Attio pipeline imply?
- What is the current accounting and owner-adjusted profit?

The target app must also answer:

- What work is sold, not delivered, not invoiced, or not collected?
- Which customer projects put revenue, margin, or cash timing at risk?
- How much delivery capacity does Hatch create without hiring?
- Which maintenance revenue is protected or threatened by Sentinel?
- Is the 10-person, AI-leveraged business model actually working?

## Product Principle

Do not make BizTop a mirror of Compass, Sentinel, Hatch, or Attio.

BizTop should ingest their state and translate it into business consequences:
money, risk, capacity, margin, and next CEO actions.

Raw tickets stay in Compass. Raw incidents stay in Sentinel. Raw agent events
stay in Hatch. BizTop shows what they mean for the plan.

## Current Gaps To Fix

1. `GOAL.md` is too thin. It names reconciliation, but not the operating
   questions, cadence, thresholds, or decisions.
2. `rules.yml` captures only CA, margin, tax, payout, management-fee rules, and
   Attio revenue types. It does not capture customer economics, project
   contracts, team capacity, reserves, recurring revenue, or automation costs.
3. Mission Control skips the execution layer between Attio and accounting:
   booked work, delivery progress, invoiceable work, and delivery risk.
4. Attio MRR is not modeled as a real recurring revenue stream across future
   years.
5. Cash reconciliation provides a number but not an invoice collection queue or
   confidence level.
6. Client concentration is historical and revenue-only. It does not expose
   current-year concentration, margin, recurrence, or risk.
7. Management-fee add-back is useful but mixes business performance, owner
   extraction, and tax/legal assumptions.
8. Data freshness is not visible enough.
9. Coverage is below the project target: current total coverage is 84.3%.

## Target Mission Control

The main funnel should become:

```text
Objectif 2030 plan
  -> Attio weighted opportunity
  -> Compass booked backlog
  -> Compass invoiceable now
  -> FEC CA facturé
  -> FEC cash encaissé
```

The main page should show four operational blocks:

1. Revenue gap
   - left to sell
   - left to book
   - left to deliver
   - invoiceable now
   - left to collect

2. Delivery risk
   - open Compass work by status
   - blocked or overdue work attached to revenue
   - projects in review, validation, and to deploy that are near invoiceability
   - projects with high booked value and low recent movement

3. Automation leverage
   - Hatch tickets picked, completed, failed, and PRs opened
   - estimated human hours avoided
   - estimated labor cost avoided
   - agent failure rate and redo risk
   - delivery capacity created without hiring

4. Maintenance health
   - Sentinel-monitored MRR/ARR
   - maintenance ARR at risk
   - active production alerts
   - recurring projects with incidents, repeated occurrences, or stale
     acknowledgement
   - Sentinel issues resolved or ticketed without human triage

## Data Sources

### FEC

Current source:

- `fecs/*.csv`

Keep this as the source of truth for accounting actuals:

- invoiced revenue
- expenses
- accounting profit
- cash collected by current reconciliation
- management fees

Add:

- source freshness: latest entry date per FEC file
- invalid row count instead of silently skipping dates
- unmatched invoice and settlement lists from cash reconciliation
- receivable age buckets

### Attio

Current source:

- `codex exec` through `internal/repository/codex.go`
- cached in `attio_estimate.json`

Keep Attio as unsold or not-yet-committed opportunity.

Change the estimate schema to include:

- `deal_id`
- `name`
- `customer`
- `type`
- `billing`
- `amount_eur`
- `mrr_eur`
- `expected_start_month`
- `expected_invoice_month`
- `probability`
- `stage`
- `last_activity_at`
- `next_step`
- `source_confidence`
- `notes`

Change MRR handling:

- annualize MRR from expected start month through the end of the current year
- carry MRR into future plan years until an explicit end date or churn
  assumption
- show one-shot and recurring pipeline separately

### Compass

Current inspected interface:

- `compass tickets --all --open --format json`
- `compass tickets --all --format json --updated-since YYYY-MM-DD`
- `compass projects`

Available ticket fields include:

- `id`
- `uniqueNumber`
- `projectId`
- `projectName`
- `status`
- `type`
- `dueDate`
- `title`
- `assignedTo`
- `isBlocked`
- `labels`
- `linkedTicketIds`
- `createdAt`
- `updatedAt`

Available project fields include:

- `id`
- `name`
- `mode`
- `customer`
- `startDate`
- `endDate`
- `description`
- `githubRepoUrl`
- `devEnvUrl`

Compass does not currently expose contract value, milestone value, estimate,
budget consumed, or invoice status. BizTop must not pretend those exist.

Add a BizTop rules section:

```yaml
project_economics:
  default_loaded_hourly_cost: 85
  default_billable_hourly_rate: 120
  status_weights:
    inbox: 0.0
    discussion: 0.05
    planned: 0.1
    triage: 0.05
    needs_doing: 0.15
    debt: 0.0
    in_progress: 0.45
    in_review: 0.75
    validation: 0.85
    to_validate: 0.9
    to_deploy: 0.95
    done: 1.0
  projects:
    - compass_project: VIGI 5
      customer_alias: VIGI 5
      revenue_model: fixed_project
      booked_value: 70000
      invoice_policy: on_validation
    - compass_project: Prezence
      customer_alias: Davai
      revenue_model: product
      monthly_recurring_revenue: 12000
```

Use Compass for:

- booked backlog: configured booked value less invoiced CA for mapped customer
- delivery progress: ticket status weighted by `status_weights`
- invoiceable now: mapped project value in `to_validate`, `to_deploy`, or done
  but not matched to FEC invoice
- delivery risk: blocked, overdue, stale, or high-value projects with no recent
  updates
- throughput: recent tickets moved to done/archive

Do not use raw ticket count as value. Use ticket count only as a risk and
throughput proxy unless project economics provides a value model.

### Hatch

Current inspected interfaces:

- `hatch dashboard --listen 127.0.0.1:3210`
- `GET /api/dashboard/fleet`
- Hatch shared Postgres tables if the dashboard is not running

Useful Hatch dashboard fields:

- summary:
  - `projects`
  - `agents`
  - `activeAgents`
  - `staleAgents`
  - `missingRuntimeAgents`
  - `failingAgents`
  - `openPrs`
  - `ciUnhealthy`
- throughput:
  - `ticketsPicked`
  - `ticketsCompleted`
  - `prOpened`
  - `ciPassed`
  - `ciFailed`
  - `failures`
  - `runsCompleted`
  - `runsFailed`
  - `averageCycleMillis`
  - `ciRetryCount`
- project rows:
  - `activeTasks`
  - `failingAgents`
  - `staleAgents`
  - `recentFailures`
  - `currentWork`
- task runs:
  - ticket number, status, outcome, PR URL, CI status, failure classification,
    duration

Hatch means near-zero labor cost, but not zero business cost.

Add a BizTop rules section:

```yaml
automation:
  hatch:
    dashboard_url: http://127.0.0.1:3210/api/dashboard/fleet
    human_hours_saved_by_ticket_type:
      performer: 6
      support: 2
    default_human_hours_saved: 4
    loaded_hourly_cost: 85
    agent_monthly_infra_cost: 25
    review_overhead_hours_per_pr: 0.5
```

Use Hatch for:

- estimated human hours avoided
- estimated labor cost avoided
- delivery capacity created
- agent failure and redo risk
- CI health and blocked automation
- projects where automation is actively moving Compass tickets

Formula:

```text
hatch_hours_saved =
  completed_hatch_tickets * configured_hours_by_ticket_type
  - human_review_overhead
  - redo_hours_for_failed_runs

hatch_cost_saved =
  hatch_hours_saved * loaded_hourly_cost
  - agent_infra_cost
```

Show this as an estimate, not accounting truth.

### Sentinel

Current inspected source:

- Sentinel stores business state in PostgreSQL.
- Alerts are in `alerts`.
- Projects are in `projects`.
- Ticket creation records are in `tickets`.
- Alert status history is in `alert_status_events`.
- Analysis decisions are in `analysis_runs`.
- Runtime state and heartbeat tables exist in later migrations.

Useful fields:

- projects:
  - `id`
  - `slug`
  - `name`
- alerts:
  - `project_id`
  - `provider`
  - `status`
  - `occurrence_count`
  - `normalized`
  - `first_seen_at`
  - `last_seen_at`
  - `acknowledged_at`
  - `snoozed_until`
- analysis runs:
  - `status`
  - `decision`
  - `result`
  - `error`
- tickets:
  - `external_id`
  - `title`
  - `url`
  - `status`

Add a BizTop rules section:

```yaml
operations:
  sentinel:
    database_url_env: SENTINEL_DATABASE_URL
    stale_alert_hours: 24
    repeated_occurrence_threshold: 3
    maintenance_projects:
      - sentinel_project: prezence
        compass_project: Prezence
        customer_alias: Davai
        monthly_recurring_revenue: 12000
      - sentinel_project: ikvp
        compass_project: Appli IKVP
        customer_alias: INGRID KERMOAL VENTE PRIVEE
        monthly_recurring_revenue: 1500
```

Use Sentinel for:

- active production alerts
- repeated occurrences by project
- unacknowledged or stale alerts
- alerts that created Compass tickets
- maintenance MRR/ARR at risk
- support labor avoided through automatic triage/ticketing

Risk scoring:

```text
maintenance_risk_score =
  severity_weight
  + repeated_occurrence_weight
  + stale_unacknowledged_weight
  + open_compass_ticket_weight
```

ARR at risk:

```text
arr_at_risk = monthly_recurring_revenue * 12
```

Only count ARR at risk for mapped maintenance/product projects.

## New BizTop Domain Model

Add new domain structs in `internal/domain`:

```go
type OperatingCockpit struct {
    GeneratedAt time.Time
    DataFreshness DataFreshness
    Revenue RevenueOperatingSummary
    Delivery DeliverySummary
    Automation AutomationSummary
    Operations OperationsSummary
    CEOActions []CEOAction
}

type CEOAction struct {
    Severity string // red, yellow, green
    Area string // sales, delivery, cash, automation, operations
    Title string
    Detail string
    AmountEur float64
    Source string // attio, compass, fec, hatch, sentinel
    SourceURL string
}
```

The important design choice: each imported system gets an adapter-specific raw
model and a BizTop business summary model. Do not pass raw Compass, Hatch, or
Sentinel data directly into the UI.

## Repository Layer

Add repositories:

- `internal/repository/compass.go`
- `internal/repository/hatch.go`
- `internal/repository/sentinel.go`
- `internal/repository/operating_cache.go`

Compass adapter:

- executes `compass tickets --all --open --format json --limit 1000`
- executes `compass tickets --all --format json --updated-since <date> --limit
  1000`
- executes `compass projects`
- caches normalized output to `compass_snapshot.json`
- returns stale cache with a warning if Compass is unreachable

Hatch adapter:

- first tries configured dashboard URL
- if unavailable, optionally reads Hatch Postgres later
- caches to `hatch_snapshot.json`
- returns stale cache with a warning if unavailable

Sentinel adapter:

- reads PostgreSQL through `database/sql`
- uses `SENTINEL_DATABASE_URL` or configured env var
- does not require Sentinel HTTP changes in the first pass
- caches to `sentinel_snapshot.json`
- returns empty with a warning when no database URL is configured

## Service Layer

Add services:

- `internal/service/operating.go`
- `internal/service/delivery.go`
- `internal/service/automation.go`
- `internal/service/operations.go`
- `internal/service/data_quality.go`

Responsibilities:

- `delivery.go`: translate Compass projects/tickets and project economics into
  backlog, invoiceability, and delivery risk.
- `automation.go`: translate Hatch throughput into hours saved, cost saved,
  and automation reliability.
- `operations.go`: translate Sentinel alerts into maintenance risk and ARR at
  risk.
- `data_quality.go`: report freshness and stale caches.
- `operating.go`: combine FEC, Attio, Compass, Hatch, and Sentinel into the
  final CEO cockpit.

## HTTP API

Add routes:

- `GET /api/operating`
- `POST /api/operating/refresh`
- `GET /api/delivery`
- `POST /api/delivery/refresh`
- `GET /api/automation`
- `GET /api/operations`

Keep refresh explicit at first. Do not make page loads block on Compass,
Sentinel, Hatch, or Attio.

## UI Plan

### Phase 1 UI

Modify Mission Control:

- add a new funnel level: `Backlog Compass`
- add `Invoiceable now`
- add `Delivery risk` cards
- add `Automation leverage` cards
- add `Maintenance health` cards
- add a compact `CEO actions` table at the top

Add a new tab only if Mission Control becomes too dense:

- `Operations`

The first version should prefer one good cockpit over many tabs.

### CEO Actions Table

Show no more than 10 rows:

- severity
- area
- action
- amount or value at risk
- source

Examples:

- `Cash`: collect 99.7k billed but not received
- `Sales`: 206.8k still needs to be sold for 2026
- `Delivery`: 20 tickets are to deploy and may be invoiceable
- `Automation`: Hatch has 3 failing agents, capacity estimate unreliable
- `Ops`: Prezence has repeated production alerts, ARR at risk

## Implementation Phases

### Phase 0: Goal and Rules

Deliverables:

- rewrite `GOAL.md` as an operating spec
- extend `rules.yml` with:
  - project economics
  - Compass status weights
  - Hatch automation assumptions
  - Sentinel maintenance mappings
  - reserve policy and owner extraction assumptions
- update README Data section for new sources

Tests:

- rules loader rejects unknown keys and invalid ratios
- project mappings validate unique names and non-negative values

### Phase 1: Compass Delivery Layer

Deliverables:

- Compass repository adapter with cache
- delivery service
- `/api/delivery`
- Mission Control additions:
  - booked backlog
  - invoiceable now
  - blocked/overdue/stale project list

Minimal value calculation:

- match Compass project to configured economics
- match accounting CA by configured `customer_alias`
- estimate remaining booked value as `booked_value - invoiced_ca_for_customer`
- estimate invoiceable now from status-weighted progress and invoice policy

Tests:

- Compass JSON parsing
- cache fallback
- status weighting
- blocked and overdue risk scoring
- invoiceable calculation

### Phase 2: Hatch Automation Layer

Deliverables:

- Hatch dashboard adapter
- automation service
- `/api/automation`
- Mission Control cards:
  - hours saved
  - labor cost avoided
  - active agents
  - failing/stale agents
  - PRs opened and CI unhealthy

Tests:

- Hatch dashboard JSON parsing
- hours-saved calculation
- failure-rate calculation
- stale dashboard handling

### Phase 3: Sentinel Maintenance Layer

Deliverables:

- Sentinel Postgres adapter
- operations service
- `/api/operations`
- Mission Control cards:
  - active production alerts
  - ARR at risk
  - repeated incidents
  - generated Compass tickets
  - stale unacknowledged alerts

Tests:

- Sentinel SQL queries with test database
- alert severity extraction from normalized JSON
- ARR-at-risk mapping
- stale alert risk scoring

### Phase 4: Unified Operating Cockpit

Deliverables:

- `/api/operating`
- `/api/operating/refresh`
- CEO actions generated from all sources
- data freshness panel
- Mission Control uses operating summary instead of separate fetches

Tests:

- cross-source merge
- action ordering by severity and amount
- stale/missing source warnings
- no raw external records leaked into UI response

### Phase 5: Cash and Receivables Upgrade

Deliverables:

- expose unmatched invoices and settlements
- receivable aging
- collection action list
- confidence on cash reconciliation

Tests:

- unmatched invoice cases
- partial payment cases
- aging buckets
- cross-year settlement cases

### Phase 6: Capital Allocation and 2030 Plan

Deliverables:

- reserve policy
- owner salary/dividend/extraction model
- acquisition and real-estate cash capacity
- headcount cap tracking
- recurring revenue target tracking

Tests:

- reserve calculations
- dividend payout constraints
- scenario calculations

## Suggested File Changes

```text
GOAL.md
README.md
rules.yml
internal/domain/operating.go
internal/repository/compass.go
internal/repository/hatch.go
internal/repository/sentinel.go
internal/repository/operating_cache.go
internal/service/delivery.go
internal/service/automation.go
internal/service/operations.go
internal/service/operating.go
internal/service/data_quality.go
internal/handler/handler.go
internal/gui/index.html
internal/gui/js/mission.js
internal/gui/js/operating.js
internal/gui/style.css
```

## Data Freshness Rules

Show freshness on Mission Control:

- FEC latest entry date
- Attio fetched at
- Compass fetched at
- Hatch fetched at
- Sentinel fetched at

Red flags:

- FEC has no entries in the last 14 days during an active year
- Attio cache older than 7 days
- Compass cache older than 24 hours
- Hatch dashboard unavailable or older than 6 hours
- Sentinel database unavailable for mapped maintenance projects

## Testing Standard

The repo asks to aim for 95% coverage. Every implementation phase should raise
coverage instead of adding untested adapters.

Minimum gates:

```sh
go test ./... -cover
go test ./... -coverprofile=/tmp/biztop.cover
go tool cover -func=/tmp/biztop.cover
```

New code should target:

- repository adapters: mocked command/HTTP/SQL dependencies
- service calculations: table tests
- handlers: success, stale cache, missing source, invalid refresh method
- frontend: keep logic small enough to test at service/API level first

Do not ship external source ingestion without cache fallback tests.

## Risks and Constraints

1. Compass has no monetary fields. BizTop needs explicit project economics in
   `rules.yml` before it can show money from Compass.
2. Hatch activity does not equal saved labor unless assumptions are documented.
   Show estimates and assumptions.
3. Sentinel alert severity depends on provider-normalized JSON. The first pass
   should handle missing severity conservatively.
4. Cross-source customer/project naming will be messy. Use explicit mappings,
   not fuzzy matching, for money.
5. Do not make automatic refreshes run on every page load. The app should remain
   fast and deterministic.

## First Pull Request Scope

The first PR should be small enough to ship:

1. Rewrite `GOAL.md`.
2. Add `project_economics`, `automation`, and `operations` config structures.
3. Add Compass adapter and delivery service.
4. Add `/api/delivery`.
5. Add a Mission Control section with:
   - open Compass tickets by status
   - blocked ticket count
   - projects with the most open work
   - configured booked backlog
   - invoiceable-now estimate
6. Add tests for the new config, adapter parsing, and delivery calculations.
7. Update README.

Do Hatch and Sentinel after Compass because Compass is the common project spine
that both systems connect back to.
