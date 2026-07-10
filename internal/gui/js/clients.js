'use strict';

import { MONTHS, fmtNum, fmtEur, escapeHtml, fetchJSON, mkChart, AXIS_TICKS, GRID } from './util.js';

// Categorical palette validated for CVD separation and contrast on --bg.
const PALETTE = ['#3987e5', '#199e70', '#c98500', '#008300',
                 '#9085e9', '#e66767', '#d55181', '#d95926'];
const AUTRES = '#8b949e';

function monthLabel(ym) {
  const [y, m] = ym.split('-');
  return `${MONTHS[parseInt(m, 10) - 1]} ${y}`;
}

export async function showClients() {
  const data = await fetchJSON('/api/customers');

  mkChart('chart-clients', {
    type: 'bar',
    data: {
      labels: data.months.map(monthLabel),
      datasets: data.series.map((s, i) => ({
        label: s.name,
        data: s.data,
        backgroundColor: s.name === 'Autres' ? AUTRES : PALETTE[i % PALETTE.length],
        borderColor: '#0e1117',
        borderWidth: 1,
        borderRadius: 2,
      })),
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          position: 'bottom',
          labels: { color: '#8b949e', font: { size: 10 }, boxWidth: 10, boxHeight: 10 },
        },
      },
      scales: {
        x: { stacked: true, ticks: AXIS_TICKS, grid: GRID },
        y: { stacked: true, ticks: { ...AXIS_TICKS, callback: v => fmtNum(v) }, grid: GRID },
      },
    },
  });

  document.getElementById('clients-thead').innerHTML = `
    <tr>
      <th>Client</th>
      <th class="num">Factures</th>
      ${data.years.map(y => `<th class="num">${y}</th>`).join('')}
      <th class="num">Total</th>
      <th class="num">Part du CA</th>
    </tr>`;

  document.getElementById('clients-tbody').innerHTML = data.customers.map(c => `
    <tr>
      <td>${escapeHtml(c.name)}</td>
      <td class="num">${c.invoices}</td>
      ${c.by_year.map(v => `<td class="num${v ? '' : ' zero'}">${fmtNum(v)}</td>`).join('')}
      <td class="num">${fmtNum(c.total)}</td>
      <td class="num">${data.total ? (c.total / data.total * 100).toFixed(1) : '0.0'}%</td>
    </tr>`).join('');

  const yearTotals = data.years.map((_, i) =>
    data.customers.reduce((sum, c) => sum + c.by_year[i], 0));
  document.getElementById('clients-tfoot').innerHTML = `
    <tr>
      <td>Total (${data.customers.length} clients)</td>
      <td></td>
      ${yearTotals.map(v => `<td class="num">${fmtNum(v)}</td>`).join('')}
      <td class="num">${fmtNum(data.total)}</td>
      <td></td>
    </tr>`;
}
