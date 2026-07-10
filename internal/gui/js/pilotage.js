'use strict';

import { MONTHS, fmtNum, fmtEur, fetchJSON, mkChart, AXIS_TICKS, GRID } from './util.js';

const dataCache = new Map();

async function getData(year) {
  if (!dataCache.has(year)) {
    dataCache.set(year, await fetchJSON(`/api/data?year=${year}`));
  }
  return dataCache.get(year);
}

export async function showPilotage(year) {
  const data = await getData(year);
  renderKpis(data.totals);
  renderChart(data.monthly);
  renderSection('section-revenue', "Chiffre d'affaires", 'ca', data.revenue, data.monthly.ca, data.totals.ca, data.year);
  renderSection('section-charges', "Charges d'exploitation", 'charges', data.charges, data.monthly.charges, data.totals.charges, data.year);
}

function renderKpis(totals) {
  document.getElementById('kpi-ca').textContent = fmtEur(totals.ca);
  document.getElementById('kpi-charges').textContent = fmtEur(totals.charges);
  document.getElementById('kpi-resultat').textContent = fmtEur(totals.resultat);
}

function renderChart(monthly) {
  mkChart('chart-monthly', {
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
      layout: { padding: 0, autoPadding: false },
      scales: {
        x: { ticks: AXIS_TICKS, grid: GRID },
        y: {
          // On the right and exactly one column wide (1/13 of the chart),
          // mirroring the tables: 12 month columns + the Total column.
          position: 'right',
          afterFit: scale => { scale.width = scale.chart.width / 13; },
          ticks: { ...AXIS_TICKS, callback: v => fmtNum(v) },
          grid: GRID,
        },
      },
    },
  });
}

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
    label.title = `${row.libelle} (${row.compte})`;
    label.innerHTML = `<span class="cat-label">${row.libelle}</span><span class="cat-compte">${row.compte}</span>`;
    tr.appendChild(label);
    row.months.forEach((v, i) => tr.appendChild(amountCell(v, row.compte, i + 1)));
    tr.appendChild(amountCell(row.total, row.compte, 0, 'total-col'));
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);

  section.replaceChildren(table);
}

// Clicking an amount routes to the Transactions subtab, prefiltered.
document.getElementById('view-pilotage').addEventListener('click', e => {
  const btn = e.target.closest('button.amount');
  if (!btn) return;
  const year = btn.closest('.accordion').dataset.year;
  location.hash = `#/compta/transactions/${year}/${btn.dataset.compte}/${btn.dataset.month}`;
});
