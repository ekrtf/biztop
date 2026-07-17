BizTop is a personal CEO cockpit for running Davai toward the plan in
`docs/DAVAI_2030.md`.

The app should translate financial, sales, delivery, automation, and operations
signals into clear business decisions:

* what still needs to be sold
* what has been sold but still needs to be delivered
* what is ready to invoice
* what has been invoiced but not collected
* which projects put revenue, margin, or cash at risk
* how much capacity Hatch creates without hiring
* which maintenance projects Sentinel says are fragile
* whether the business is staying inside the 10-person, high-margin,
  AI-leveraged model

Data sources:

* FEC files are the accounting source of truth for invoiced revenue, expenses,
  profit, and cash collection.
* Attio is the source for sales opportunities that are not yet booked.
* Compass is the source for booked customer work, delivery state, blockers, and
  invoiceability signals.
* Hatch is the source for automated delivery work, agent reliability, PR/CI
  throughput, and estimated labor avoided.
* Sentinel is the source for production and maintenance health, incident risk,
  and recurring revenue that may be at risk.

BizTop must not become a mirror of those tools. It should show the business
consequence of their signals: money, risk, capacity, margin, and the next CEO
actions.

The detailed implementation plan is in
`docs/OPERATING_COCKPIT_IMPLEMENTATION_PLAN.md`.
