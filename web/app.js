'use strict';

const MONTHS = ['janv.', 'févr.', 'mars', 'avr.', 'mai', 'juin',
                'juil.', 'août', 'sept.', 'oct.', 'nov.', 'déc.'];

const numFmt = new Intl.NumberFormat('fr-FR', { maximumFractionDigits: 0 });
const eurFmt = new Intl.NumberFormat('fr-FR', {
  style: 'currency', currency: 'EUR', maximumFractionDigits: 0,
});

const dataCache = new Map();
let chart = null;

function fmtNum(v) { return numFmt.format(v); }
function fmtEur(v) { return eurFmt.format(v); }

// ── ROUTING ─────────────────────────────────────────────────
// #/<year>            dashboard for that exercice
// #/<year>/<compte>/<month>   transactions (month 0 = whole year)

function parseRoute() {
  const parts = location.hash.replace(/^#\/?/, '').split('/').filter(Boolean);
  return {
    year: parseInt(parts[0], 10) || null,
    compte: parts[1] || null,
    month: parseInt(parts[2], 10) || 0,
  };
}

async function route() {
  const r = parseRoute();
  if (r.compte && r.year) {
    await showTransactions(r.year, r.compte, r.month);
  } else {
    await showDashboard(r.year);
  }
}

window.addEventListener('hashchange', route);

// ── DATA ────────────────────────────────────────────────────

async function getData(year) {
  const key = year || 'latest';
  if (!dataCache.has(key)) {
    const url = year ? `/api/data?year=${year}` : '/api/data';
    const res = await fetch(url);
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    dataCache.set(key, data);
    dataCache.set(data.year, data);
  }
  return dataCache.get(key);
}

// ── DASHBOARD ───────────────────────────────────────────────

async function showDashboard(year) {
  const data = await getData(year);
  document.getElementById('view-transactions').hidden = true;
  document.getElementById('view-dashboard').hidden = false;

  renderYearNav(data.years, data.year);
  renderKpis(data.totals);
  renderChart(data.monthly);
  renderSection('section-revenue', "Chiffre d'affaires", 'ca', data.revenue, data.monthly.ca, data.totals.ca, data.year);
  renderSection('section-charges', "Charges d'exploitation", 'charges', data.charges, data.monthly.charges, data.totals.charges, data.year);
}

function renderYearNav(years, active) {
  const nav = document.getElementById('year-nav');
  nav.replaceChildren(...years.map(y => {
    const btn = document.createElement('button');
    btn.className = 'year-btn' + (y === active ? ' active' : '');
    btn.textContent = y;
    btn.addEventListener('click', () => { location.hash = `#/${y}`; });
    return btn;
  }));
}

function renderKpis(totals) {
  document.getElementById('kpi-ca').textContent = fmtEur(totals.ca);
  document.getElementById('kpi-charges').textContent = fmtEur(totals.charges);
  document.getElementById('kpi-resultat').textContent = fmtEur(totals.resultat);
}

function renderChart(monthly) {
  if (chart) chart.destroy();
  const ctx = document.getElementById('chart-monthly').getContext('2d');
  chart = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: MONTHS,
      datasets: [
        {
          label: "Chiffre d'affaires",
          data: monthly.ca,
          backgroundColor: 'rgba(63,185,80,0.65)',
          borderRadius: 3,
        },
        {
          label: "Charges d'exploitation",
          data: monthly.charges,
          backgroundColor: 'rgba(248,81,73,0.6)',
          borderRadius: 3,
        },
        {
          label: "Résultat d'exploitation",
          data: monthly.resultat,
          type: 'line',
          borderColor: '#58a6ff',
          backgroundColor: '#0e1117',
          pointBackgroundColor: '#e6edf3',
          pointRadius: 4,
          tension: 0.1,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { ticks: { color: '#8b949e', font: { size: 10 } }, grid: { color: '#21262d' } },
        y: {
          ticks: {
            color: '#8b949e',
            font: { size: 10 },
            callback: v => fmtNum(v),
          },
          grid: { color: '#21262d' },
        },
      },
    },
  });
}

// ── ACCORDION SECTIONS ──────────────────────────────────────

function amountCell(value, compte, month, extraClass = '') {
  const td = document.createElement('td');
  td.className = ('num ' + extraClass).trim();
  if (!value) {
    td.classList.add('zero');
    td.textContent = '0';
    return td;
  }
  const btn = document.createElement('button');
  btn.className = 'amount';
  btn.dataset.compte = compte;
  btn.dataset.month = month;
  btn.textContent = fmtNum(value);
  td.appendChild(btn);
  return td;
}

function renderSection(id, title, kind, rows, monthlyTotals, total, year) {
  const section = document.getElementById(id);
  section.dataset.year = year;

  const table = document.createElement('table');
  const thead = document.createElement('thead');

  const monthsRow = document.createElement('tr');
  monthsRow.appendChild(document.createElement('th'));
  for (const m of MONTHS) {
    const th = document.createElement('th');
    th.className = 'num';
    th.textContent = m;
    monthsRow.appendChild(th);
  }
  const thTotal = document.createElement('th');
  thTotal.className = 'num';
  thTotal.textContent = 'Total';
  monthsRow.appendChild(thTotal);
  thead.appendChild(monthsRow);

  const headRow = document.createElement('tr');
  headRow.className = 'acc-head';
  const titleCell = document.createElement('td');
  titleCell.innerHTML = `<span class="chevron">&#9662;</span><span class="acc-title-${kind}">${title}</span>`;
  headRow.appendChild(titleCell);
  for (const v of monthlyTotals) {
    const td = document.createElement('td');
    td.className = 'num' + (v ? '' : ' zero');
    td.textContent = fmtNum(v);
    headRow.appendChild(td);
  }
  const totalCell = document.createElement('td');
  totalCell.className = 'num total-col';
  totalCell.textContent = fmtNum(total);
  headRow.appendChild(totalCell);
  headRow.addEventListener('click', () => section.classList.toggle('collapsed'));
  thead.appendChild(headRow);
  table.appendChild(thead);

  const tbody = document.createElement('tbody');
  for (const row of rows) {
    const tr = document.createElement('tr');
    const label = document.createElement('td');
    label.innerHTML = `<span class="cat-label">${row.libelle}</span><span class="cat-compte">${row.compte}</span>`;
    tr.appendChild(label);
    row.months.forEach((v, i) => tr.appendChild(amountCell(v, row.compte, i + 1)));
    tr.appendChild(amountCell(row.total, row.compte, 0, 'total-col'));
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);

  section.replaceChildren(table);
}

document.getElementById('view-dashboard').addEventListener('click', e => {
  const btn = e.target.closest('button.amount');
  if (!btn) return;
  const year = btn.closest('.accordion').dataset.year;
  location.hash = `#/${year}/${btn.dataset.compte}/${btn.dataset.month}`;
});

// ── TRANSACTIONS ────────────────────────────────────────────

async function showTransactions(year, compte, month) {
  const res = await fetch(`/api/transactions?year=${year}&compte=${compte}&month=${month}`);
  if (!res.ok) throw new Error(await res.text());
  const data = await res.json();

  document.getElementById('view-dashboard').hidden = true;
  document.getElementById('view-transactions').hidden = false;
  document.getElementById('tx-back').href = `#/${year}`;

  const period = month ? `${MONTHS[month - 1]} ${year}` : `exercice ${year}`;
  document.getElementById('tx-title').textContent =
    `${data.libelle || compte} (${compte}) · ${period} · ${data.transactions.length} transactions`;

  const tbody = document.getElementById('tx-tbody');
  tbody.replaceChildren(...data.transactions.map(tx => {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td>${tx.date}</td>
      <td>${escapeHtml(tx.libelle)}</td>
      <td class="num debit">${tx.debit ? fmtNum(tx.debit) : ''}</td>
      <td class="num credit">${tx.credit ? fmtNum(tx.credit) : ''}</td>`;
    return tr;
  }));

  const tfoot = document.getElementById('tx-tfoot');
  tfoot.innerHTML = `
    <tr>
      <td colspan="2">Total</td>
      <td class="num debit">${fmtNum(data.total_debit)}</td>
      <td class="num credit">${fmtNum(data.total_credit)}</td>
    </tr>`;
}

function escapeHtml(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

route();
