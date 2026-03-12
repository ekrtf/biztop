#!/usr/bin/env python3
"""
BizTop - Business monitoring dashboard
French accounting / trésorerie / BP vs Réel
"""

import csv, io, json, os, re
from collections import defaultdict
from datetime import datetime, date
from flask import Flask, jsonify, render_template_string, request

app = Flask(__name__)
DATA_DIR = os.path.dirname(os.path.abspath(__file__))
FECS_DIR = os.path.join(DATA_DIR, "fecs")
BP_FILE = os.path.join(DATA_DIR, "bp/BP-2026.xlsx")
CONFIG_FILE = os.path.join(DATA_DIR, "management_services.json")

# ─────────────────────────────────────────────
# VENDOR NORMALIZATION
# ─────────────────────────────────────────────
VENDOR_RULES = [
    # Internal / skip
    (r"Encaissement|Facturation|Facture d'acompte|Account de|2e partie|3e et derni", "__skip__"),
    (r"Virement interne|From Reserve|To Reserve|PRE VOLUM.*GERANCE", "__skip__"),
    # Team / payroll
    (r"To Sina Ganji|Sina Ganji", "Sina Ganji (équipe)"),
    (r"To Thai Luong|MINH THANG NGUYEN|Thai Luong", "Thai Luong (équipe)"),
    (r"Maxime Soydas|Virement vers Maxime", "Maxime Soydas (équipe)"),
    (r"To Emile Kratiroff|Emile Kratiroff", "__skip__"),
    (r"PAJANIRADJA|ENTRE NOUS|Agilane", "Pajaniradja / Entre Nous"),
    (r"To Varkinia|Dominik Arabasz", "Varkinia / Arabasz"),
    # Suppliers via Wise/VIKOISOFT
    (r"VIKOISOFT|vikoisoft", "Vikoisoft (sous-traitance)"),
    (r"Virement vers Wise|wise payment|Wise Europe", "Wise (virement)"),
    # Infra & cloud
    (r"TOTALENERGIES|TOTAL ENERGY|TotalEnergies", "TotalEnergies"),
    (r"GOOGLE|GSUITE|Google Workspace", "Google"),
    (r"MICROSOFT|Microsoft", "Microsoft"),
    (r"APPLE\.COM/BILL", "Apple (abo)"),
    (r"APPLE STORE|LDLC|SQ \*SQUARE SHOP", "Apple / Hardware"),
    (r"AMAZON", "Amazon"),
    (r"CLAUDE\.AI|ANTHROPIC", "Anthropic / Claude"),
    (r"OPENAI|CHATGPT", "OpenAI"),
    (r"SHADOW", "Shadow PC"),
    (r"RENDER\.COM|RENDER", "Render"),
    (r"SCALEWAY", "Scaleway"),
    (r"CLOUDFLARE", "Cloudflare"),
    (r"Vercel", "Vercel"),
    (r"OVH", "OVH"),
    (r"GRAFANA", "Grafana Labs"),
    (r"REPLIT", "Replit"),
    (r"Cursor", "Cursor AI"),
    # SaaS tools
    (r"FOLK\.APP|WWW\.FOLK", "Folk CRM"),
    (r"SMALLPDF", "SmallPDF"),
    (r"1PASSWORD", "1Password"),
    (r"SQUARE|SQ \*SQUARE FR SUBS", "Square"),
    # Travel
    (r"AIRBNB", "Airbnb"),
    (r"AIR FRANCE|Aeromexico|Tap Air Portugal|Billet Avion", "Billets d'avion"),
    (r"SNCF|TGV|TRANSILIEN|RATP", "Transport en commun"),
    (r"UBER", "Uber"),
    (r"YESPARK|PARKING.*YESPARK", "Yespark (parking)"),
    (r"Quinta Da|Casa Yuma|ALICE CAP", "Hébergement voyage"),
    # Food
    (r"DELIVEROO", "Deliveroo"),
    (r"MAITRE DATTIER|LE VAVIN|AO IZAKAYA|SAVEURS DE MARJO", "Restaurants"),
    # Finance / admin
    (r"AMV|A M V|1225132", "AMV Assurances"),
    (r"SWAN.*foreign|Card transaction.*SWAN", "Swan (frais FX)"),
    (r"GOCARDLESS", "GoCardless"),
    (r"CANAL\+|CANAL PLUS", "Canal+"),
    # Key clients/vendors
    (r"VOLUM|SAS Volum", "Volum"),
    (r"FRASS|DAVAI FRASS", "Frass"),
    (r"CHAUVIN|CHAUVIN PARIS", "Chauvin"),
    (r"MYWAY|MyWay", "MyWay Technology"),
    (r"THUNDER INVEST", "Thunder Invest"),
    (r"ACTENA", "Actena Automobil"),
    (r"HERMEL|Quote Part Hermel", "Hermel (loyer)"),
    (r"EB \*ENTREPRENEUR", "Eventbrite"),
    (r"AME TEAM|AXXE", "AME Team Axxe"),
]

def normalize_vendor(libelle: str) -> str:
    for pattern, name in VENDOR_RULES:
        if re.search(pattern, libelle, re.IGNORECASE):
            return name
    # Truncate long invoice refs
    libelle = re.sub(r"/INV/[A-Z0-9]+/[0-9]+ //[0-9]+-\w+ - ", "", libelle)
    libelle = re.sub(r"[0-9]{6,}-\d+ - ", "", libelle)
    return libelle[:50]

# ─────────────────────────────────────────────
# ACCOUNT CATEGORY MAPPING
# ─────────────────────────────────────────────
CATEGORY_MAP = {
    "Abonnements logiciels": "SaaS & Logiciels",
    "Location de materiel": "Infrastructure & Cloud",
    "Eau, gaz, electricite": "Infrastructure & Cloud",
    "Internet, telephone et frais postaux": "Télécom & Internet",
    "Prestation de service": "Sous-traitance",
    "Redevances pour concessions, brevets, licences, marques, procedes, logiciels": "Licences",
    "Loyers et charges locatives": "Loyers & Bureaux",
    "Entretien et reparations": "Entretien",
    "Frais de deplacements": "Déplacements",
    "Vehicule et carburant": "Véhicule & Carburant",
    "Restaurants et repas d'affaires": "Restaurants & Repas",
    "Reception, representation, congres": "Réception & Représentation",
    "Publicite, publications, relations publiques": "Marketing & Pub",
    "Services bancaires": "Frais Bancaires",
    "Assurance professionnelle": "Assurances",
    "Autres impôts et taxes": "Impôts & Taxes",
    "Penalites, amendes fiscales et penales": "Amendes",
    "Materiel et outillage": "Matériel",
    "Marchandise pour la revente": "Achats / Sous-traitance",
    "Immobilisations corporelles": "Immobilisations",
    "Impôts sur les benefices": "IS",
    "Amortissements des immobilisations corporelles": "Amortissements",
    "Taxe sur les vehicules de societes": "TVS",
    "Frais d'actes et de contentieux": "Frais Juridiques",
    "Vente de service": "CA - Prestations",
    "Vente de marchandise": "CA - Ventes",
    "Vente de service en UE": "CA - Prestations UE",
    "Vente de service hors UE": "CA - Prestations Export",
}

# Categories that are "services de direction" by default (mixed personal/business)
DEFAULT_MGMT_SERVICES_CATEGORIES = [
    "Restaurants & Repas",
    "Réception & Représentation",
    "Déplacements",
    "Véhicule & Carburant",
    "Loyers & Bureaux",
]

DEFAULT_MGMT_SERVICES_VENDORS = ["Canal+", "Airbnb", "Apple", "Shadow PC"]

def load_mgmt_config():
    if os.path.exists(CONFIG_FILE):
        with open(CONFIG_FILE) as f:
            return json.load(f)
    return {
        "categories": DEFAULT_MGMT_SERVICES_CATEGORIES,
        "vendors": DEFAULT_MGMT_SERVICES_VENDORS,
    }

def save_mgmt_config(config):
    with open(CONFIG_FILE, "w") as f:
        json.dump(config, f, indent=2, ensure_ascii=False)

# ─────────────────────────────────────────────
# CSV PARSING
# ─────────────────────────────────────────────
def parse_amount(s):
    return float(s.replace(",", ".")) if s and s.strip() else 0.0

def load_fecs():
    rows = []
    for filename in sorted(os.listdir(FECS_DIR)):
        if not filename.endswith(".csv"):
            continue
        year = filename.replace(".csv", "")
        filepath = os.path.join(FECS_DIR, filename)
        with open(filepath, encoding="utf-8-sig") as f:
            reader = csv.DictReader(f, delimiter=";")
            for row in reader:
                try:
                    d = datetime.strptime(row["Date"].strip(), "%d/%m/%Y")
                except (ValueError, KeyError):
                    continue
                debit = parse_amount(row.get("Debit", "0"))
                credit = parse_amount(row.get("Credit", "0"))
                compte = row.get("Compte", "").strip()
                libelle_compte = row.get("Libelle du compte", "").strip()
                libelle = row.get("Libelle", "").strip()
                vendor = normalize_vendor(libelle)
                rows.append({
                    "date": d,
                    "month": d.strftime("%Y-%m"),
                    "year": d.year,
                    "libelle": libelle,
                    "vendor": vendor,
                    "compte": compte,
                    "libelle_compte": libelle_compte,
                    "categorie": CATEGORY_MAP.get(libelle_compte, libelle_compte),
                    "debit": debit,
                    "credit": credit,
                })
    return rows

def is_expense(row):
    return row["compte"].startswith("6") and not row["compte"].startswith("695")

def is_income(row):
    return row["compte"].startswith("706") or row["compte"].startswith("707")

def is_bank(row):
    return row["compte"].startswith("512")

def is_is(row):  # Impôt sur les sociétés
    return row["compte"].startswith("695")

# ─────────────────────────────────────────────
# BP PARSING
# ─────────────────────────────────────────────
MONTHS = ["January","February","March","April","May","June",
          "July","August","September","October","November","December"]
MONTH_FR = ["Jan","Fév","Mar","Avr","Mai","Juin","Juil","Aoû","Sep","Oct","Nov","Déc"]

def parse_bp():
    try:
        import openpyxl
        wb = openpyxl.load_workbook(BP_FILE, read_only=True)
        ws = wb["Sheet1"]
        rows = list(ws.iter_rows(values_only=True))
    except Exception:
        return {"revenue": {}, "expenses": {}}

    def safe_num(v):
        if isinstance(v, (int, float)):
            return float(v)
        return 0.0

    # Revenue rows 7-31 (0-indexed: 6-30), months cols C-N (idx 2-13)
    bp_revenue = defaultdict(lambda: [0.0]*12)
    bp_expenses = defaultdict(lambda: [0.0]*12)

    # Section boundaries (0-indexed row numbers)
    rev_start, rev_end = 6, 31
    exp_start, exp_end = 43, 62

    for ri in range(rev_start, rev_end):
        row = rows[ri] if ri < len(rows) else None
        if not row:
            continue
        name = row[1]
        if not name or name in ("Projects","Products","Services","MONTHLY","REAL","TOTAL"):
            continue
        for mi in range(12):
            v = safe_num(row[2 + mi])
            if v:
                bp_revenue[str(name)][mi] += v

    for ri in range(exp_start, exp_end):
        row = rows[ri] if ri < len(rows) else None
        if not row:
            continue
        name = row[1]
        if not name or name in ("Team","Office","Misc","MONTHLY","REAL","TOTAL"):
            continue
        for mi in range(12):
            v = safe_num(row[2 + mi])
            if v:
                bp_expenses[str(name)][mi] += v

    return {
        "revenue": {k: list(v) for k, v in bp_revenue.items()},
        "expenses": {k: list(v) for k, v in bp_expenses.items()},
    }

# ─────────────────────────────────────────────
# DATA AGGREGATION
# ─────────────────────────────────────────────
def aggregate(rows, mgmt_cfg):
    mgmt_cats = set(mgmt_cfg.get("categories", []))
    mgmt_vendors = set(mgmt_cfg.get("vendors", []))

    # Monthly CA, expenses, tréso
    monthly = defaultdict(lambda: {
        "ca": 0.0, "charges": 0.0, "charges_netto": 0.0,
        "encaissements": 0.0, "decaissements": 0.0,
        "is": 0.0, "mgmt": 0.0,
    })
    by_vendor = defaultdict(lambda: {"total": 0.0, "categorie": "", "months": defaultdict(float), "is_mgmt": False})
    by_category = defaultdict(lambda: {"total": 0.0, "months": defaultdict(float), "is_mgmt": False})
    bank_balance = {}  # month → end balance

    bank_running = 0.0
    months_seen = set()

    for r in rows:
        m = r["month"]
        months_seen.add(m)

        if is_income(r):
            monthly[m]["ca"] += r["credit"]  # revenue = credit on compte 706/707

        if is_expense(r):
            monthly[m]["charges"] += r["debit"]
            is_mgmt = (r["categorie"] in mgmt_cats) or (r["vendor"] in mgmt_vendors)
            if is_mgmt:
                monthly[m]["mgmt"] += r["debit"]
            else:
                monthly[m]["charges_netto"] += r["debit"]

            v = r["vendor"]
            if v == "__skip__":
                continue
            by_vendor[v]["total"] += r["debit"]
            by_vendor[v]["categorie"] = r["categorie"]
            by_vendor[v]["months"][m] += r["debit"]
            by_vendor[v]["is_mgmt"] = is_mgmt

            c = r["categorie"]
            by_category[c]["total"] += r["debit"]
            by_category[c]["months"][m] += r["debit"]
            by_category[c]["is_mgmt"] = c in mgmt_cats

        if is_is(r):
            monthly[m]["is"] += r["debit"]

        if is_bank(r):
            monthly[m]["encaissements"] += r["debit"]
            monthly[m]["decaissements"] += r["credit"]

    # Compute bank balance cumulatively (sorted months)
    balance = 0.0
    for m in sorted(months_seen):
        balance += monthly[m]["encaissements"] - monthly[m]["decaissements"]
        bank_balance[m] = round(balance, 2)

    return {
        "monthly": {k: v for k, v in sorted(monthly.items())},
        "bank_balance": bank_balance,
        "by_vendor": dict(sorted(by_vendor.items(), key=lambda x: -x[1]["total"])),
        "by_category": dict(sorted(by_category.items(), key=lambda x: -x[1]["total"])),
    }

# ─────────────────────────────────────────────
# FLASK ROUTES
# ─────────────────────────────────────────────
@app.route("/")
def index():
    return render_template_string(HTML_TEMPLATE)

@app.route("/api/data")
def api_data():
    rows = load_fecs()
    mgmt_cfg = load_mgmt_config()
    agg = aggregate(rows, mgmt_cfg)
    bp = parse_bp()

    months = sorted(agg["monthly"].keys())
    months_fr = []
    for m in months:
        y, mo = m.split("-")
        months_fr.append(f"{MONTH_FR[int(mo)-1]} {y}")

    return jsonify({
        "months": months,
        "months_fr": months_fr,
        "monthly": agg["monthly"],
        "bank_balance": agg["bank_balance"],
        "by_vendor": {
            k: {
                "total": round(v["total"], 2),
                "categorie": v["categorie"],
                "is_mgmt": v["is_mgmt"],
                "months": {mk: round(mv, 2) for mk, mv in v["months"].items()},
            }
            for k, v in list(agg["by_vendor"].items())[:80]
        },
        "by_category": {
            k: {
                "total": round(v["total"], 2),
                "is_mgmt": v["is_mgmt"],
                "months": {mk: round(mv, 2) for mk, mv in v["months"].items()},
            }
            for k, v in agg["by_category"].items()
        },
        "bp": bp,
        "mgmt_config": mgmt_cfg,
    })

@app.route("/api/mgmt_config", methods=["GET","POST"])
def api_mgmt_config():
    if request.method == "POST":
        save_mgmt_config(request.json)
        return jsonify({"ok": True})
    return jsonify(load_mgmt_config())

# ─────────────────────────────────────────────
# HTML TEMPLATE
# ─────────────────────────────────────────────
HTML_TEMPLATE = r"""<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>BizTop — Tableau de bord</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
<style>
  :root {
    --bg: #0e1117;
    --bg2: #161b22;
    --bg3: #1c2128;
    --border: #30363d;
    --text: #e6edf3;
    --text2: #8b949e;
    --green: #3fb950;
    --red: #f85149;
    --blue: #58a6ff;
    --yellow: #d29922;
    --purple: #bc8cff;
    --orange: #ffa657;
    --cyan: #76e3ea;
    --accent: #238636;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { background: var(--bg); color: var(--text); font-family: 'SF Mono', 'Consolas', monospace; font-size: 13px; }

  /* HEADER */
  header { background: var(--bg2); border-bottom: 1px solid var(--border); padding: 12px 20px; display: flex; align-items: center; gap: 16px; }
  header h1 { font-size: 18px; color: var(--green); letter-spacing: 1px; }
  header span { color: var(--text2); font-size: 11px; }
  .tabs { display: flex; gap: 4px; margin-left: auto; }
  .tab { padding: 6px 14px; border-radius: 6px; cursor: pointer; border: 1px solid var(--border); background: transparent; color: var(--text2); font-size: 12px; font-family: inherit; transition: all 0.15s; }
  .tab:hover { color: var(--text); border-color: var(--blue); }
  .tab.active { background: var(--blue); color: #000; border-color: var(--blue); font-weight: bold; }

  /* LAYOUT */
  main { padding: 16px 20px; }
  .page { display: none; }
  .page.active { display: block; }

  /* KPI CARDS */
  .kpi-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-bottom: 20px; }
  .kpi { background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; padding: 14px 16px; }
  .kpi .label { color: var(--text2); font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; }
  .kpi .value { font-size: 22px; font-weight: bold; margin-top: 4px; }
  .kpi .sub { font-size: 11px; color: var(--text2); margin-top: 3px; }
  .kpi.green .value { color: var(--green); }
  .kpi.red .value { color: var(--red); }
  .kpi.blue .value { color: var(--blue); }
  .kpi.yellow .value { color: var(--yellow); }
  .kpi.purple .value { color: var(--purple); }
  .kpi.orange .value { color: var(--orange); }

  /* CHARTS */
  .chart-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
  .chart-row.full { grid-template-columns: 1fr; }
  .chart-box { background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; padding: 16px; }
  .chart-box h3 { font-size: 12px; color: var(--text2); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 12px; }
  .chart-box canvas { max-height: 260px; }

  /* TABLES */
  .table-box { background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; padding: 16px; margin-bottom: 16px; }
  .table-box h3 { font-size: 12px; color: var(--text2); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 12px; display: flex; align-items: center; gap: 8px; }
  .search-bar { margin-left: auto; background: var(--bg3); border: 1px solid var(--border); border-radius: 6px; padding: 5px 10px; color: var(--text); font-family: inherit; font-size: 12px; width: 200px; }
  .search-bar::placeholder { color: var(--text2); }
  table { width: 100%; border-collapse: collapse; }
  th { color: var(--text2); font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; padding: 6px 10px; text-align: left; border-bottom: 1px solid var(--border); }
  td { padding: 7px 10px; border-bottom: 1px solid #21262d; font-size: 12px; }
  tr:hover td { background: var(--bg3); }
  tr:last-child td { border-bottom: none; }
  .badge { display: inline-block; padding: 2px 7px; border-radius: 10px; font-size: 10px; }
  .badge.mgmt { background: rgba(188, 140, 255, 0.15); color: var(--purple); border: 1px solid rgba(188,140,255,0.3); }
  .badge.normal { background: rgba(63, 185, 80, 0.1); color: var(--green); border: 1px solid rgba(63,185,80,0.25); }
  .num { text-align: right; font-variant-numeric: tabular-nums; }
  .red-num { color: var(--red); }
  .green-num { color: var(--green); }

  /* TOGGLE */
  .toggle-group { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px; }
  .toggle-item { padding: 4px 10px; border-radius: 20px; cursor: pointer; border: 1px solid var(--border); font-size: 11px; color: var(--text2); user-select: none; }
  .toggle-item:hover { border-color: var(--purple); color: var(--text); }
  .toggle-item.active { background: rgba(188,140,255,0.2); color: var(--purple); border-color: var(--purple); }

  /* TRESO */
  .treso-balance { font-size: 28px; font-weight: bold; }

  /* BP */
  .bp-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }

  /* YEAR FILTER */
  .filter-row { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; }
  .filter-row select, .filter-row button { background: var(--bg3); border: 1px solid var(--border); border-radius: 6px; padding: 6px 12px; color: var(--text); font-family: inherit; font-size: 12px; cursor: pointer; }
  .filter-row button:hover { border-color: var(--blue); color: var(--blue); }
  .filter-row .active-btn { border-color: var(--blue); color: var(--blue); background: rgba(88,166,255,0.1); }

  /* SCROLLABLE TABLE */
  .scroll-table { max-height: 400px; overflow-y: auto; }

  /* SECTION TITLE */
  .section-title { color: var(--text2); font-size: 11px; text-transform: uppercase; letter-spacing: 1px; padding: 6px 0; margin-bottom: 8px; border-bottom: 1px solid var(--border); }

  /* SPARKBAR */
  .sparkbar { display: flex; height: 6px; border-radius: 3px; overflow: hidden; }
  .sparkbar-fill { background: var(--blue); height: 100%; }
  .sparkbar-mgmt { background: var(--purple); height: 100%; }
</style>
</head>
<body>

<header>
  <h1>▶ BizTop</h1>
  <span id="last-updated">Chargement...</span>
  <div class="tabs">
    <button class="tab active" onclick="showPage('dashboard')">Dashboard</button>
    <button class="tab" onclick="showPage('vendors')">Fournisseurs</button>
    <button class="tab" onclick="showPage('categories')">Catégories</button>
    <button class="tab" onclick="showPage('treso')">Trésorerie</button>
    <button class="tab" onclick="showPage('bp')">BP vs Réel</button>
    <button class="tab" onclick="showPage('mgmt')">Services Direction</button>
  </div>
</header>

<main>
<!-- DASHBOARD -->
<div class="page active" id="page-dashboard">
  <div class="kpi-row" id="kpi-row"></div>
  <div class="chart-row">
    <div class="chart-box">
      <h3>CA mensuel vs Charges</h3>
      <canvas id="chart-ca-charges"></canvas>
    </div>
    <div class="chart-box">
      <h3>Résultat mensuel (CA - Charges)</h3>
      <canvas id="chart-resultat"></canvas>
    </div>
  </div>
  <div class="chart-row full">
    <div class="chart-box">
      <h3>Répartition des charges par catégorie (cumul)</h3>
      <canvas id="chart-cat-donut" style="max-height:300px"></canvas>
    </div>
  </div>
</div>

<!-- VENDORS -->
<div class="page" id="page-vendors">
  <div class="filter-row">
    <span style="color:var(--text2)">Année :</span>
    <button id="btn-all" class="active-btn" onclick="setYearFilter('all')">Tout</button>
    <button id="btn-2025" onclick="setYearFilter('2025')">2025</button>
    <button id="btn-2026" onclick="setYearFilter('2026')">2026</button>
    <span style="color:var(--text2);margin-left:12px">Vue :</span>
    <button id="btn-all-vendors" class="active-btn" onclick="setVendorFilter('all')">Tous</button>
    <button id="btn-mgmt-vendors" onclick="setVendorFilter('mgmt')">Services Direction</button>
    <button id="btn-pure-vendors" onclick="setVendorFilter('pure')">Hors Direction</button>
  </div>
  <div class="table-box">
    <h3>Dépenses par fournisseur
      <input class="search-bar" id="vendor-search" type="text" placeholder="Rechercher..." oninput="renderVendors()">
    </h3>
    <div class="scroll-table">
      <table>
        <thead><tr>
          <th>Fournisseur</th><th>Catégorie</th><th>Type</th>
          <th class="num">Total HT</th><th style="width:100px">Répartition</th>
        </tr></thead>
        <tbody id="vendor-tbody"></tbody>
      </table>
    </div>
  </div>
  <div class="chart-row">
    <div class="chart-box">
      <h3>Top 15 fournisseurs</h3>
      <canvas id="chart-vendors-bar"></canvas>
    </div>
    <div class="chart-box">
      <h3>Évolution mensuelle (top 5)</h3>
      <canvas id="chart-vendors-line"></canvas>
    </div>
  </div>
</div>

<!-- CATEGORIES -->
<div class="page" id="page-categories">
  <div class="filter-row">
    <button id="btn-cat-all" class="active-btn" onclick="setYearFilter('all','cat')">Tout</button>
    <button id="btn-cat-2025" onclick="setYearFilter('2025','cat')">2025</button>
    <button id="btn-cat-2026" onclick="setYearFilter('2026','cat')">2026</button>
  </div>
  <div class="chart-row">
    <div class="chart-box">
      <h3>Charges par catégorie</h3>
      <canvas id="chart-categories-bar"></canvas>
    </div>
    <div class="chart-box">
      <h3>Évolution mensuelle des charges</h3>
      <canvas id="chart-categories-stack"></canvas>
    </div>
  </div>
  <div class="table-box">
    <h3>Détail par catégorie</h3>
    <table>
      <thead><tr>
        <th>Catégorie</th><th>Type</th><th class="num">Total HT</th><th class="num">% du total</th>
      </tr></thead>
      <tbody id="cat-tbody"></tbody>
    </table>
  </div>
</div>

<!-- TRÉSORERIE -->
<div class="page" id="page-treso">
  <div class="kpi-row" id="treso-kpi-row"></div>
  <div class="chart-row full">
    <div class="chart-box">
      <h3>Solde bancaire cumulatif</h3>
      <canvas id="chart-balance" style="max-height:280px"></canvas>
    </div>
  </div>
  <div class="chart-row">
    <div class="chart-box">
      <h3>Encaissements vs Décaissements</h3>
      <canvas id="chart-cashflow"></canvas>
    </div>
    <div class="chart-box">
      <h3>Flux net mensuel</h3>
      <canvas id="chart-net-flux"></canvas>
    </div>
  </div>
  <div class="table-box">
    <h3>Plan de trésorerie mensuel</h3>
    <div class="scroll-table">
      <table>
        <thead><tr>
          <th>Mois</th><th class="num">Encaissements</th><th class="num">Décaissements</th>
          <th class="num">Flux Net</th><th class="num">Solde Cumulatif</th>
        </tr></thead>
        <tbody id="treso-tbody"></tbody>
      </table>
    </div>
  </div>
</div>

<!-- BP vs RÉEL -->
<div class="page" id="page-bp">
  <div class="chart-row">
    <div class="chart-box">
      <h3>CA Prévisionnel 2026 par projet</h3>
      <canvas id="chart-bp-rev"></canvas>
    </div>
    <div class="chart-box">
      <h3>Charges Prévisionnelles 2026 par poste</h3>
      <canvas id="chart-bp-exp"></canvas>
    </div>
  </div>
  <div class="chart-row full">
    <div class="chart-box">
      <h3>BP Mensuel 2026 — Prévisionnel vs Réel</h3>
      <canvas id="chart-bp-monthly" style="max-height:300px"></canvas>
    </div>
  </div>
  <div class="table-box">
    <h3>Projets de CA 2026 (BP)</h3>
    <table>
      <thead><tr>
        <th>Projet</th>
        <th class="num">Jan</th><th class="num">Fév</th><th class="num">Mar</th>
        <th class="num">Avr</th><th class="num">Mai</th><th class="num">Juin</th>
        <th class="num">Juil</th><th class="num">Aoû</th><th class="num">Sep</th>
        <th class="num">Oct</th><th class="num">Nov</th><th class="num">Déc</th>
        <th class="num">Total</th>
      </tr></thead>
      <tbody id="bp-rev-tbody"></tbody>
    </table>
  </div>
</div>

<!-- SERVICES DE DIRECTION -->
<div class="page" id="page-mgmt">
  <div class="table-box">
    <h3>Configuration — Services de Direction (charges mixtes personnel/business)</h3>
    <p style="color:var(--text2);font-size:11px;margin-bottom:12px;line-height:1.6">
      Les charges ci-dessous sont légales mais représentent des avantages mixtes.
      Activez/désactivez pour isoler les charges purement business des charges de direction.
      Ces catégories apparaissent en <span style="color:var(--purple)">violet</span> dans les autres vues.
    </p>
    <div class="section-title">Catégories</div>
    <div class="toggle-group" id="mgmt-cat-toggles"></div>
    <div class="section-title" style="margin-top:16px">Fournisseurs spécifiques</div>
    <div class="toggle-group" id="mgmt-vendor-toggles"></div>
    <button onclick="saveMgmtConfig()" style="margin-top:16px;background:var(--accent);border:none;border-radius:6px;padding:8px 16px;color:#fff;font-family:inherit;font-size:12px;cursor:pointer;">
      ✓ Enregistrer la configuration
    </button>
  </div>
  <div class="kpi-row" id="mgmt-kpi-row"></div>
  <div class="chart-row">
    <div class="chart-box">
      <h3>Impact des services de direction sur les charges</h3>
      <canvas id="chart-mgmt-split"></canvas>
    </div>
    <div class="chart-box">
      <h3>Charges direction vs Business pur (mensuel)</h3>
      <canvas id="chart-mgmt-monthly"></canvas>
    </div>
  </div>
</div>

</main>

<script>
let DATA = null;
let vendorYearFilter = 'all';
let vendorTypeFilter = 'all';
let catYearFilter = 'all';
let chartInstances = {};
let pendingMgmtCats = new Set();
let pendingMgmtVendors = new Set();

function fmt(n) {
  if (n === undefined || n === null) return '—';
  return new Intl.NumberFormat('fr-FR', {style:'currency',currency:'EUR',maximumFractionDigits:0}).format(n);
}
function fmtK(n) {
  if (Math.abs(n) >= 1000) return (n/1000).toFixed(1) + 'k €';
  return n.toFixed(0) + ' €';
}

function showPage(name) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.getElementById('page-' + name).classList.add('active');
  event.target.classList.add('active');
  if (DATA) renderPage(name);
}

function destroyChart(id) {
  if (chartInstances[id]) { chartInstances[id].destroy(); delete chartInstances[id]; }
}

const COLORS = [
  '#58a6ff','#3fb950','#ffa657','#f85149','#bc8cff',
  '#76e3ea','#d29922','#ff7b72','#56d364','#a5d6ff',
  '#f0883e','#8b949e','#e3b341','#b392f0','#79c0ff',
];

function mkChart(id, type, data, options={}) {
  destroyChart(id);
  const ctx = document.getElementById(id).getContext('2d');
  const defaults = {
    responsive: true,
    maintainAspectRatio: true,
    plugins: { legend: { labels: { color: '#8b949e', font: { size: 11 } } } },
    scales: type !== 'pie' && type !== 'doughnut' ? {
      x: { ticks: { color: '#8b949e', font: { size: 10 } }, grid: { color: '#21262d' } },
      y: { ticks: { color: '#8b949e', font: { size: 10 }, callback: v => fmtK(v) }, grid: { color: '#21262d' } },
    } : {},
  };
  chartInstances[id] = new Chart(ctx, {
    type,
    data,
    options: Object.assign({}, defaults, options),
  });
}

async function loadData() {
  const r = await fetch('/api/data');
  DATA = await r.json();
  document.getElementById('last-updated').textContent =
    'Données : ' + DATA.months_fr[0] + ' → ' + DATA.months_fr[DATA.months_fr.length-1];
  renderPage('dashboard');
  renderPage('treso');
  renderPage('vendors');
  renderPage('categories');
  renderPage('bp');
  renderPage('mgmt');
}

function filterMonths(yearStr) {
  if (!yearStr || yearStr === 'all') return { months: DATA.months, labels: DATA.months_fr };
  const filtered = DATA.months.filter(m => m.startsWith(yearStr));
  const labels = filtered.map((m, i) => {
    const idx = DATA.months.indexOf(m);
    return DATA.months_fr[idx];
  });
  return { months: filtered, labels };
}

function renderPage(name) {
  if (!DATA) return;
  if (name === 'dashboard') renderDashboard();
  if (name === 'vendors') renderVendors();
  if (name === 'categories') renderCategories();
  if (name === 'treso') renderTreso();
  if (name === 'bp') renderBP();
  if (name === 'mgmt') renderMgmt();
}

// ── DASHBOARD ──────────────────────────────────────────────
function renderDashboard() {
  const all_months = DATA.months;
  let totalCA = 0, totalCharges = 0, totalMgmt = 0;
  all_months.forEach(m => {
    const d = DATA.monthly[m] || {};
    totalCA += d.ca || 0;
    totalCharges += d.charges || 0;
    totalMgmt += d.mgmt || 0;
  });
  const lastBalance = Object.values(DATA.bank_balance).pop() || 0;
  const resultat = totalCA - totalCharges;

  const kpiRow = document.getElementById('kpi-row');
  kpiRow.innerHTML = `
    <div class="kpi green"><div class="label">CA Total</div><div class="value">${fmt(totalCA)}</div><div class="sub">Toutes périodes</div></div>
    <div class="kpi red"><div class="label">Charges Totales</div><div class="value">${fmt(totalCharges)}</div><div class="sub">Dont ${fmt(totalMgmt)} direction</div></div>
    <div class="kpi ${resultat >= 0 ? 'blue' : 'red'}"><div class="label">Résultat Brut</div><div class="value">${fmt(resultat)}</div><div class="sub">CA – Charges</div></div>
    <div class="kpi yellow"><div class="label">Charges Direction</div><div class="value">${fmt(totalMgmt)}</div><div class="sub">${totalCharges ? (totalMgmt/totalCharges*100).toFixed(1) : 0}% des charges</div></div>
    <div class="kpi ${lastBalance >= 0 ? 'cyan' : 'red'}"><div class="label">Solde Banque</div><div class="value" style="color:var(--cyan)">${fmt(lastBalance)}</div><div class="sub">Cumulatif</div></div>
  `;

  const labels = DATA.months_fr;
  const caData = DATA.months.map(m => (DATA.monthly[m]||{}).ca||0);
  const chargesData = DATA.months.map(m => (DATA.monthly[m]||{}).charges||0);
  const mgmtData = DATA.months.map(m => (DATA.monthly[m]||{}).mgmt||0);
  const resultatData = DATA.months.map(m => ((DATA.monthly[m]||{}).ca||0) - ((DATA.monthly[m]||{}).charges||0));

  mkChart('chart-ca-charges', 'bar', {
    labels,
    datasets: [
      { label: 'CA', data: caData, backgroundColor: 'rgba(63,185,80,0.7)', borderRadius: 3 },
      { label: 'Charges', data: chargesData, backgroundColor: 'rgba(248,81,73,0.6)', borderRadius: 3 },
      { label: 'Direction', data: mgmtData, backgroundColor: 'rgba(188,140,255,0.5)', borderRadius: 3 },
    ]
  });

  mkChart('chart-resultat', 'bar', {
    labels,
    datasets: [{
      label: 'Résultat',
      data: resultatData,
      backgroundColor: resultatData.map(v => v >= 0 ? 'rgba(88,166,255,0.7)' : 'rgba(248,81,73,0.7)'),
      borderRadius: 3,
    }]
  });

  // Donut by category
  const catEntries = Object.entries(DATA.by_category)
    .filter(([k,v]) => !['CA - Prestations','CA - Ventes','CA - Prestations UE','CA - Prestations Export'].includes(k))
    .sort((a,b) => b[1].total - a[1].total).slice(0, 12);
  mkChart('chart-cat-donut', 'doughnut', {
    labels: catEntries.map(([k]) => k),
    datasets: [{
      data: catEntries.map(([,v]) => v.total),
      backgroundColor: COLORS,
      borderColor: '#0e1117',
      borderWidth: 2,
    }]
  }, { plugins: { legend: { position: 'right', labels: { color: '#8b949e', font: { size: 11 }, boxWidth: 12 } } } });
}

// ── VENDORS ────────────────────────────────────────────────
function setYearFilter(yr, ctx='vendor') {
  if (ctx === 'vendor') {
    vendorYearFilter = yr;
    ['all','2025','2026'].forEach(y => {
      const el = document.getElementById('btn-' + y);
      if (el) el.className = (y === yr) ? 'active-btn' : '';
    });
  } else {
    catYearFilter = yr;
    ['all','2025','2026'].forEach(y => {
      const el = document.getElementById('btn-cat-' + y);
      if (el) el.className = (y === yr) ? 'active-btn' : '';
    });
    renderCategories();
    return;
  }
  renderVendors();
}

function setVendorFilter(t) {
  vendorTypeFilter = t;
  ['all','mgmt','pure'].forEach(x => {
    const el = document.getElementById('btn-' + x + '-vendors');
    if (el) el.className = (x === t) ? 'active-btn' : '';
  });
  renderVendors();
}

function getVendorTotal(v, year) {
  if (year === 'all') return v.total;
  return Object.entries(v.months).filter(([m]) => m.startsWith(year)).reduce((s,[,x]) => s+x, 0);
}

function renderVendors() {
  if (!DATA) return;
  const q = (document.getElementById('vendor-search')?.value || '').toLowerCase();
  let entries = Object.entries(DATA.by_vendor)
    .filter(([k]) => k !== '__skip__')
    .map(([k,v]) => [k, { ...v, filteredTotal: getVendorTotal(v, vendorYearFilter) }])
    .filter(([,v]) => v.filteredTotal > 0.5)
    .filter(([k,v]) => !q || k.toLowerCase().includes(q) || v.categorie.toLowerCase().includes(q))
    .filter(([,v]) => vendorTypeFilter === 'all' || (vendorTypeFilter === 'mgmt' ? v.is_mgmt : !v.is_mgmt))
    .sort((a,b) => b[1].filteredTotal - a[1].filteredTotal);

  const maxTotal = entries[0]?.[1]?.filteredTotal || 1;
  const tbody = document.getElementById('vendor-tbody');
  tbody.innerHTML = entries.slice(0, 100).map(([k,v]) => {
    const pct = v.filteredTotal / maxTotal;
    return `<tr>
      <td>${k}</td>
      <td style="color:var(--text2)">${v.categorie}</td>
      <td><span class="badge ${v.is_mgmt ? 'mgmt' : 'normal'}">${v.is_mgmt ? 'Direction' : 'Business'}</span></td>
      <td class="num ${v.is_mgmt ? '' : 'green-num'}">${fmt(v.filteredTotal)}</td>
      <td><div class="sparkbar"><div class="${v.is_mgmt ? 'sparkbar-mgmt' : 'sparkbar-fill'}" style="width:${(pct*100).toFixed(1)}%"></div></div></td>
    </tr>`;
  }).join('');

  // Bar chart top 15
  const top15 = entries.slice(0, 15);
  mkChart('chart-vendors-bar', 'bar', {
    labels: top15.map(([k]) => k.length > 20 ? k.slice(0,18)+'…' : k),
    datasets: [{
      label: 'Total HT',
      data: top15.map(([,v]) => v.filteredTotal),
      backgroundColor: top15.map(([,v]) => v.is_mgmt ? 'rgba(188,140,255,0.7)' : 'rgba(88,166,255,0.7)'),
      borderRadius: 3,
    }]
  }, { indexAxis: 'y', plugins: { legend: { display: false } } });

  // Line chart top 5 evolution
  const top5 = entries.slice(0, 5);
  const { months, labels } = filterMonths(vendorYearFilter === 'all' ? null : vendorYearFilter);
  mkChart('chart-vendors-line', 'line', {
    labels,
    datasets: top5.map(([k,v], i) => ({
      label: k.length > 25 ? k.slice(0,23)+'…' : k,
      data: months.map(m => v.months[m] || 0),
      borderColor: COLORS[i],
      backgroundColor: COLORS[i] + '22',
      tension: 0.3,
      fill: false,
      pointRadius: 3,
    }))
  });
}

// ── CATEGORIES ──────────────────────────────────────────────
function renderCategories() {
  if (!DATA) return;
  const yr = catYearFilter;
  const { months, labels } = filterMonths(yr === 'all' ? null : yr);
  const INCOME_CATS = new Set(['CA - Prestations','CA - Ventes','CA - Prestations UE','CA - Prestations Export']);

  const catEntries = Object.entries(DATA.by_category)
    .filter(([k]) => !INCOME_CATS.has(k))
    .map(([k,v]) => {
      const total = months.reduce((s,m) => s + (v.months[m]||0), 0);
      return [k, { ...v, filteredTotal: total }];
    })
    .filter(([,v]) => v.filteredTotal > 1)
    .sort((a,b) => b[1].filteredTotal - a[1].filteredTotal);

  const grandTotal = catEntries.reduce((s,[,v]) => s + v.filteredTotal, 0);

  // Bar chart
  mkChart('chart-categories-bar', 'bar', {
    labels: catEntries.map(([k]) => k),
    datasets: [{
      label: 'Total HT',
      data: catEntries.map(([,v]) => v.filteredTotal),
      backgroundColor: catEntries.map(([,v]) => v.is_mgmt ? 'rgba(188,140,255,0.7)' : 'rgba(88,166,255,0.7)'),
      borderRadius: 3,
    }]
  }, { indexAxis: 'y', plugins: { legend: { display: false } } });

  // Stacked area chart by category over months
  const top8 = catEntries.slice(0, 8);
  mkChart('chart-categories-stack', 'bar', {
    labels,
    datasets: top8.map(([k,v], i) => ({
      label: k,
      data: months.map(m => v.months[m] || 0),
      backgroundColor: COLORS[i] + 'bb',
      borderRadius: 2,
    }))
  }, { scales: {
    x: { stacked: true, ticks: { color: '#8b949e', font: { size: 10 } }, grid: { color: '#21262d' } },
    y: { stacked: true, ticks: { color: '#8b949e', font: { size: 10 }, callback: v => fmtK(v) }, grid: { color: '#21262d' } },
  }});

  const tbody = document.getElementById('cat-tbody');
  tbody.innerHTML = catEntries.map(([k,v]) => `
    <tr>
      <td>${k}</td>
      <td><span class="badge ${v.is_mgmt ? 'mgmt' : 'normal'}">${v.is_mgmt ? 'Direction' : 'Business'}</span></td>
      <td class="num">${fmt(v.filteredTotal)}</td>
      <td class="num" style="color:var(--text2)">${grandTotal ? (v.filteredTotal/grandTotal*100).toFixed(1) : 0}%</td>
    </tr>`).join('');
}

// ── TRÉSORERIE ──────────────────────────────────────────────
function renderTreso() {
  if (!DATA) return;
  const months = DATA.months;
  const labels = DATA.months_fr;
  const enc = months.map(m => (DATA.monthly[m]||{}).encaissements||0);
  const dec = months.map(m => (DATA.monthly[m]||{}).decaissements||0);
  const net = months.map((m,i) => enc[i] - dec[i]);
  const balances = months.map(m => DATA.bank_balance[m] || 0);

  const lastBal = balances[balances.length-1] || 0;
  const totalEnc = enc.reduce((a,b) => a+b, 0);
  const totalDec = dec.reduce((a,b) => a+b, 0);
  const avgMonthly = net.reduce((a,b) => a+b, 0) / (net.length || 1);

  document.getElementById('treso-kpi-row').innerHTML = `
    <div class="kpi blue"><div class="label">Solde Actuel</div><div class="value">${fmt(lastBal)}</div><div class="sub">Cumulatif toutes banques</div></div>
    <div class="kpi green"><div class="label">Encaissements</div><div class="value">${fmt(totalEnc)}</div><div class="sub">Total reçu</div></div>
    <div class="kpi red"><div class="label">Décaissements</div><div class="value">${fmt(totalDec)}</div><div class="sub">Total payé</div></div>
    <div class="kpi ${avgMonthly >= 0 ? 'orange' : 'red'}"><div class="label">Flux Net Moyen</div><div class="value">${fmt(avgMonthly)}</div><div class="sub">Par mois</div></div>
  `;

  mkChart('chart-balance', 'line', {
    labels,
    datasets: [{
      label: 'Solde bancaire',
      data: balances,
      borderColor: '#58a6ff',
      backgroundColor: 'rgba(88,166,255,0.1)',
      tension: 0.3,
      fill: true,
      pointRadius: 3,
    }]
  });

  mkChart('chart-cashflow', 'bar', {
    labels,
    datasets: [
      { label: 'Encaissements', data: enc, backgroundColor: 'rgba(63,185,80,0.7)', borderRadius: 3 },
      { label: 'Décaissements', data: dec, backgroundColor: 'rgba(248,81,73,0.6)', borderRadius: 3 },
    ]
  });

  mkChart('chart-net-flux', 'bar', {
    labels,
    datasets: [{
      label: 'Flux Net',
      data: net,
      backgroundColor: net.map(v => v >= 0 ? 'rgba(88,166,255,0.7)' : 'rgba(248,81,73,0.7)'),
      borderRadius: 3,
    }]
  });

  const tbody = document.getElementById('treso-tbody');
  let cumul = 0;
  tbody.innerHTML = months.map((m, i) => {
    const flux = net[i];
    cumul += flux;
    return `<tr>
      <td>${labels[i]}</td>
      <td class="num green-num">${fmt(enc[i])}</td>
      <td class="num red-num">${fmt(dec[i])}</td>
      <td class="num ${flux >= 0 ? 'green-num' : 'red-num'}">${fmt(flux)}</td>
      <td class="num ${cumul >= 0 ? '' : 'red-num'}">${fmt(DATA.bank_balance[m] || 0)}</td>
    </tr>`;
  }).join('');
}

// ── BP ──────────────────────────────────────────────────────
function renderBP() {
  if (!DATA || !DATA.bp) return;
  const bp = DATA.bp;
  const MONTH_LABELS = ['Jan','Fév','Mar','Avr','Mai','Juin','Juil','Aoû','Sep','Oct','Nov','Déc'];

  // Revenue stacked bar by project
  const revEntries = Object.entries(bp.revenue).sort((a,b) =>
    b[1].reduce((s,x)=>s+x,0) - a[1].reduce((s,x)=>s+x,0));
  mkChart('chart-bp-rev', 'bar', {
    labels: MONTH_LABELS,
    datasets: revEntries.slice(0,10).map(([k,vals], i) => ({
      label: k,
      data: vals,
      backgroundColor: COLORS[i % COLORS.length] + 'cc',
      borderRadius: 2,
    }))
  }, { scales: {
    x: { stacked: true, ticks: { color: '#8b949e', font: { size: 10 } }, grid: { color: '#21262d' } },
    y: { stacked: true, ticks: { color: '#8b949e', font: { size: 10 }, callback: v => fmtK(v) }, grid: { color: '#21262d' } },
  }});

  // Expenses stacked bar
  const expEntries = Object.entries(bp.expenses).sort((a,b) =>
    b[1].reduce((s,x)=>s+x,0) - a[1].reduce((s,x)=>s+x,0));
  mkChart('chart-bp-exp', 'bar', {
    labels: MONTH_LABELS,
    datasets: expEntries.slice(0,10).map(([k,vals], i) => ({
      label: k,
      data: vals,
      backgroundColor: COLORS[i % COLORS.length] + 'cc',
      borderRadius: 2,
    }))
  }, { scales: {
    x: { stacked: true, ticks: { color: '#8b949e', font: { size: 10 } }, grid: { color: '#21262d' } },
    y: { stacked: true, ticks: { color: '#8b949e', font: { size: 10 }, callback: v => fmtK(v) }, grid: { color: '#21262d' } },
  }});

  // BP monthly total vs Réel
  const bpMonthlyRev = MONTH_LABELS.map((_, i) =>
    Object.values(bp.revenue).reduce((s, vals) => s + (vals[i]||0), 0));
  const bpMonthlyExp = MONTH_LABELS.map((_, i) =>
    Object.values(bp.expenses).reduce((s, vals) => s + (vals[i]||0), 0));
  const bpMonthlyProfit = bpMonthlyRev.map((r, i) => r - bpMonthlyExp[i]);

  // Réel 2026 months
  const real2026months = DATA.months.filter(m => m.startsWith('2026'));
  const realCA2026 = Array(12).fill(0);
  const realCharges2026 = Array(12).fill(0);
  real2026months.forEach(m => {
    const mi = parseInt(m.split('-')[1]) - 1;
    realCA2026[mi] = (DATA.monthly[m]||{}).ca || 0;
    realCharges2026[mi] = (DATA.monthly[m]||{}).charges || 0;
  });
  const realProfit2026 = realCA2026.map((r,i) => r - realCharges2026[i]);

  mkChart('chart-bp-monthly', 'line', {
    labels: MONTH_LABELS,
    datasets: [
      { label: 'CA Prévisionnel', data: bpMonthlyRev, borderColor: '#3fb950', backgroundColor: 'rgba(63,185,80,0.1)', borderDash: [5,3], tension: 0.3, fill: false },
      { label: 'CA Réel 2026', data: realCA2026, borderColor: '#3fb950', backgroundColor: 'rgba(63,185,80,0.3)', tension: 0.3, fill: false, borderWidth: 2 },
      { label: 'Charges Prév.', data: bpMonthlyExp, borderColor: '#f85149', backgroundColor: 'rgba(248,81,73,0.1)', borderDash: [5,3], tension: 0.3, fill: false },
      { label: 'Charges Réel 2026', data: realCharges2026, borderColor: '#f85149', tension: 0.3, fill: false, borderWidth: 2 },
      { label: 'Résultat Prév.', data: bpMonthlyProfit, borderColor: '#58a6ff', borderDash: [5,3], tension: 0.3, fill: false },
      { label: 'Résultat Réel 2026', data: realProfit2026, borderColor: '#58a6ff', tension: 0.3, fill: false, borderWidth: 2 },
    ]
  });

  // Revenue table
  const tbody = document.getElementById('bp-rev-tbody');
  tbody.innerHTML = Object.entries(bp.revenue).map(([k,vals]) => {
    const total = vals.reduce((s,x) => s+x, 0);
    return `<tr>
      <td>${k}</td>
      ${vals.map(v => `<td class="num" style="color:${v ? 'var(--blue)' : 'var(--text2)'}">${v ? fmt(v) : '—'}</td>`).join('')}
      <td class="num green-num"><strong>${fmt(total)}</strong></td>
    </tr>`;
  }).join('') + `<tr style="border-top:1px solid var(--border)">
    <td><strong>TOTAL</strong></td>
    ${Array(12).fill(0).map((_,i) => {
      const v = Object.values(bp.revenue).reduce((s,vals) => s+(vals[i]||0), 0);
      return `<td class="num green-num"><strong>${v ? fmt(v) : '—'}</strong></td>`;
    }).join('')}
    <td class="num green-num"><strong>${fmt(Object.values(bp.revenue).flat().reduce((a,b)=>a+b,0))}</strong></td>
  </tr>`;
}

// ── MGMT SERVICES ───────────────────────────────────────────
function renderMgmt() {
  if (!DATA) return;
  const cfg = DATA.mgmt_config;
  pendingMgmtCats = new Set(cfg.categories || []);
  pendingMgmtVendors = new Set(cfg.vendors || []);

  // All available categories (expense only)
  const INCOME_CATS = new Set(['CA - Prestations','CA - Ventes','CA - Prestations UE','CA - Prestations Export']);
  const allCats = Object.keys(DATA.by_category).filter(k => !INCOME_CATS.has(k));
  const allVendors = Object.keys(DATA.by_vendor).filter(k => k !== '__skip__').slice(0, 40);

  const catToggleEl = document.getElementById('mgmt-cat-toggles');
  catToggleEl.innerHTML = allCats.map(c => `
    <div class="toggle-item ${pendingMgmtCats.has(c) ? 'active' : ''}"
         onclick="toggleMgmt('cat','${c.replace(/'/g,"\\'")}',this)">${c}</div>
  `).join('');

  const vendorToggleEl = document.getElementById('mgmt-vendor-toggles');
  vendorToggleEl.innerHTML = allVendors.map(v => `
    <div class="toggle-item ${pendingMgmtVendors.has(v) ? 'active' : ''}"
         onclick="toggleMgmt('vendor','${v.replace(/'/g,"\\'")}',this)">${v}</div>
  `).join('');

  // KPIs
  const months = DATA.months;
  let totalCharges = 0, totalMgmt = 0;
  months.forEach(m => {
    totalCharges += (DATA.monthly[m]||{}).charges || 0;
    totalMgmt += (DATA.monthly[m]||{}).mgmt || 0;
  });
  const pureBiz = totalCharges - totalMgmt;

  document.getElementById('mgmt-kpi-row').innerHTML = `
    <div class="kpi red"><div class="label">Charges Totales</div><div class="value">${fmt(totalCharges)}</div></div>
    <div class="kpi purple"><div class="label">Services Direction</div><div class="value">${fmt(totalMgmt)}</div><div class="sub">${totalCharges ? (totalMgmt/totalCharges*100).toFixed(1) : 0}% des charges</div></div>
    <div class="kpi blue"><div class="label">Charges Business Pur</div><div class="value">${fmt(pureBiz)}</div><div class="sub">${totalCharges ? (pureBiz/totalCharges*100).toFixed(1) : 0}% des charges</div></div>
  `;

  // Split donut
  const INCOME_CATS2 = new Set(['CA - Prestations','CA - Ventes','CA - Prestations UE','CA - Prestations Export']);
  const mgmtCatTotals = {};
  const pureCatTotals = {};
  Object.entries(DATA.by_category).forEach(([k,v]) => {
    if (INCOME_CATS2.has(k)) return;
    if (v.is_mgmt) mgmtCatTotals[k] = v.total;
    else pureCatTotals[k] = v.total;
  });

  const mgmtEntries = Object.entries(mgmtCatTotals).sort((a,b) => b[1]-a[1]);
  const pureEntries = Object.entries(pureCatTotals).sort((a,b) => b[1]-a[1]);
  mkChart('chart-mgmt-split', 'doughnut', {
    labels: [...mgmtEntries.map(([k]) => `[DIR] ${k}`), ...pureEntries.slice(0,6).map(([k]) => k)],
    datasets: [{
      data: [...mgmtEntries.map(([,v]) => v), ...pureEntries.slice(0,6).map(([,v]) => v)],
      backgroundColor: [
        ...mgmtEntries.map((_, i) => `rgba(188,140,255,${0.9-i*0.1})`),
        ...pureEntries.slice(0,6).map((_, i) => COLORS[i % COLORS.length] + 'bb'),
      ],
      borderColor: '#0e1117',
      borderWidth: 2,
    }]
  }, { plugins: { legend: { position: 'right', labels: { color: '#8b949e', font: { size: 10 }, boxWidth: 10 } } } });

  // Monthly mgmt vs pure
  const labels = DATA.months_fr;
  const mgmtMonthly = DATA.months.map(m => (DATA.monthly[m]||{}).mgmt || 0);
  const pureMonthly = DATA.months.map(m => ((DATA.monthly[m]||{}).charges||0) - ((DATA.monthly[m]||{}).mgmt||0));
  mkChart('chart-mgmt-monthly', 'bar', {
    labels,
    datasets: [
      { label: 'Business pur', data: pureMonthly, backgroundColor: 'rgba(88,166,255,0.7)', borderRadius: 3 },
      { label: 'Services Direction', data: mgmtMonthly, backgroundColor: 'rgba(188,140,255,0.7)', borderRadius: 3 },
    ]
  }, { scales: {
    x: { stacked: true, ticks: { color: '#8b949e', font: { size: 10 } }, grid: { color: '#21262d' } },
    y: { stacked: true, ticks: { color: '#8b949e', font: { size: 10 }, callback: v => fmtK(v) }, grid: { color: '#21262d' } },
  }});
}

function toggleMgmt(type, name, el) {
  const set = type === 'cat' ? pendingMgmtCats : pendingMgmtVendors;
  if (set.has(name)) { set.delete(name); el.classList.remove('active'); }
  else { set.add(name); el.classList.add('active'); }
}

async function saveMgmtConfig() {
  const cfg = { categories: [...pendingMgmtCats], vendors: [...pendingMgmtVendors] };
  await fetch('/api/mgmt_config', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify(cfg)
  });
  await loadData();
}

loadData();
</script>
</body>
</html>
"""

if __name__ == "__main__":
    print("BizTop démarré → http://localhost:5055")
    app.run(host="0.0.0.0", port=5055, debug=True)
