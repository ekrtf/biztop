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

The app is a single Go binary with no dependencies. The frontend (`internal/gui/`: html, css, ES modules) is embedded in the binary at build time.

Interface
---------

Three tabs, plus an "exercice comptable" selector in the header:

* **Compta / Pilotage**: chiffre d'affaires, charges d'exploitation and résultat d'exploitation for the year, monthly bar chart, and two accordion tables split by plan comptable category (with account numbers). Clicking an amount jumps to Transactions, prefiltered.
* **Compta / Transactions**: every FEC transaction of the exercice with filters: libellé search, catégorie, mois, débit/crédit, montant min/max.
* **Objectifs**: the 5-year plan parsed from `docs/DAVAI_2030.md`, reconciled with the FEC actuals per year. The "Actualiser depuis Attio (codex)" button asks the `codex` CLI (Attio plugin) to estimate upcoming revenue across all open deals, classified by davai type (Projects, Maintenance & Hosting, Prezence, Bodacker). The estimate is cached in `attio_estimate.json` and only refreshed on demand.
* **Management Fees**: expenses that are legally company charges but really manager compensation. Matching rules live in `management_fees.json` (created with defaults on first run): whole accounts with a ratio (e.g. restaurants at 70%, déplacements at 80%) and libellé regexes (e.g. airbnb, quote-part at 100%). Shows the fees per month and the résultat réel ajusté (résultat + fees).

Data
----

* `fecs/*.csv` - FEC exports, one file per year (`;` separated: Date, Libelle, Compte, Libelle du compte, Debit, Credit). Exercices shown in the app are derived from entry dates.
* Charges d'exploitation: accounts `6xxxxx` except `695xxx`, net of credits. Chiffre d'affaires: `706xxx` and `707xxx`, net of debits.
* `management_fees.json` - management fees matching rules (editable).
* `attio_estimate.json` - cached Attio pipeline estimate written by the refresh button.
* `docs/DAVAI_2030.md` - the financial brief holding the 5-year revenue plan.

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
