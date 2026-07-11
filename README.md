Biztop
======

> `btop` for business.

This project is Davai's financial tool. My attempts at *Business Engineering*.

The idea is to get a claude code session with accounting markdown theory, feed it the FEC files for historical data and add the Business Plan. It should output a good dashboard.

Run
---

```
go run ./cmd/biztop
```

Then open http://localhost:5055. Run it from the project root: data paths (fecs/, docs/, config files) are resolved from the working directory.

The app is a single Go binary (only dependency: yaml). The frontend (`internal/gui/`: html, css, ES modules) is embedded in the binary at build time.

Interface
---------

Six tabs, plus an "exercice comptable" selector in the header (Compta and Management Fees only). The layout is responsive with four tiers: mobile (under 640px, single column, swipeable tabs), tablet (under 1024px, stacked overview and funnel), laptop (the base layout) and large desktop (1440px and up, content centered at a max width). Wide tables scroll horizontally inside their box on small screens.

* **Mission Control**: the landing view, one funnel from the most abstract to the most real: objectif (the ambition in `rules.yml`) > projection Attio (facturé + weighted pipeline) > CA facturé (signed and invoiced, no money yet) > encaissé (actually on the bank account). Each level shows its "reste à faire" toward the one above (reste à vendre, à facturer, à encaisser), plus a rentabilité panel (target vs actual net margin, gross profit, estimated IS per the `rules.yml` scale — see `docs/IS_CHEAT_SHEET.md` —, net profit with management fees added back, and dividends vs objective at the configured payout), a cumulative trajectory chart and a per-year table over the 5-year plan. Pipeline MRR deals are projected monthly until the end of their year; the encaissé level matches invoices (411 débit) to their settlements (411 crédit + 512 banque) by invoice reference then exact amount, since the simplified FEC has no lettrage.
* **Compta / Pilotage**: chiffre d'affaires, charges d'exploitation and résultat d'exploitation for the year, monthly bar chart, and two accordion tables split by plan comptable category (with account numbers). Clicking an amount jumps to Transactions, prefiltered.
* **Compta / Transactions**: every FEC transaction of the exercice with filters: libellé search, catégorie, mois, débit/crédit, montant min/max.
* **Clients**: revenue per client over the whole FEC history: a monthly stacked bar chart (top 8 clients, the rest folded into "Autres") and a table with per-exercice totals and each client's share of the CA. The client name is parsed from the revenue entry libellé, after the last " - " ("Facturation 202604-160 - CHAUVIN PARIS").
* **Objectifs**: the 5-year plan objectives from `rules.yml` (CA and margin lower bounds, the profit target is derived), reconciled with the FEC actuals per year. The "Actualiser depuis Attio (codex)" button asks the `codex` CLI (Attio plugin) to estimate upcoming revenue across all open deals, classified by the davai types defined in `rules.yml` (one-shot types use the deal value, MRR types use the Attio MRR field). The estimate is cached in `attio_estimate.json` and only refreshed on demand.
* **Management Fees**: expenses that are legally company charges but really manager compensation. Matching rules live in `rules.yml`: whole accounts with a ratio (e.g. restaurants at 70%, déplacements at 80%), libellé regexes (e.g. airbnb, quote-part at 100%) and exclude patterns that veto a match. Shows the fees per month and the résultat réel ajusté (résultat + fees).

Data
----

* `fecs/*.csv` - FEC exports, one file per year (`;` separated: Date, Libelle, Compte, Libelle du compte, Debit, Credit). Exercices shown in the app are derived from entry dates.
* Charges d'exploitation: accounts `6xxxxx` except `695xxx`, net of credits. Chiffre d'affaires: `706xxx` and `707xxx`, net of debits.
* `rules.yml` - single source of truth for business rules: management fees matching (with ratios and excludes), the 5-year plan objectives (CA and margin lower bounds per year), the IS scale and dividend payout, and the Attio davai types (one-shot vs MRR billing). Read on every request, strict keys (a typo is an error).
* `attio_estimate.json` - cached Attio pipeline estimate written by the refresh button.
* `docs/DAVAI_2030.md` - the financial brief the plan objectives come from.

Structure
---------

```
biztop/
├── cmd/biztop/          # main: config + wiring + start
├── internal/
│   ├── domain/          # entities + accounting rules
│   ├── handler/         # HTTP handlers, serves the embedded gui
│   ├── service/         # business logic (compta, fees, objectives)
│   ├── repository/      # data access: FEC csv, config files, codex CLI
│   └── gui/             # frontend assets, embedded via go:embed
```
