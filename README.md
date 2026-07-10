Biztop
======

> `btop` for business.

This project is Davai's financial tool. My attempts at *Business Engineering*.

The idea is to get a claude code session with accounting markdown theory, feed it the FEC files for historical data and add the Business Plan. It should output a good dashboard.

Run
---

```
go run .
```

Then open http://localhost:5055.

The app is a single Go binary with no dependencies. The frontend lives in `web/` (html, css, js) and is embedded in the binary at build time.

Interface
---------

The dashboard shows one "exercice comptable" at a time, selectable in the header:

* Top: chiffre d'affaires, charges d'exploitation and résultat d'exploitation for the year, with a monthly bar chart (CA vs charges) and the résultat as a line.
* Two accordion tables, month by month: chiffre d'affaires and charges d'exploitation, split by plan comptable category (with the account number).
* Clicking an amount opens the list of transactions behind it (category + month, or the whole year via the Total column).

Data
----

* `fecs/*.csv` - FEC exports, one file per year (`;` separated: Date, Libelle, Compte, Libelle du compte, Debit, Credit). Years shown in the app are derived from the entry dates, not the file names.
* Charges d'exploitation: accounts `6xxxxx` except `695xxx` (impôt sur les bénéfices), net of credits.
* Chiffre d'affaires: accounts `706xxx` and `707xxx`, net of debits.
* `bp/` and `docs/` hold the business plan and accounting theory; they are not read by the app for now.
