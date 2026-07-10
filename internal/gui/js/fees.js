'use strict';

import { MONTHS, fmtNum, fmtEur, escapeHtml, fetchJSON, mkChart, AXIS_TICKS, GRID } from './util.js';

export async function showFees(year) {
  const data = await fetchJSON(`/api/fees?year=${year}`);

  document.getElementById('fees-total').textContent = fmtEur(data.total);
  document.getElementById('fees-share').textContent =
    data.resultat + data.total !== 0
      ? `${(data.total / (data.resultat + data.total) * 100).toFixed(1)}% du résultat réel`
      : '';
  document.getElementById('fees-resultat').textContent = fmtEur(data.resultat);
  document.getElementById('fees-ajuste').textContent = fmtEur(data.resultat_ajuste);

  const cfg = data.config || {};
  const rules = [
    ...(cfg.comptes || []).map(c => `${c.compte} à ${Math.round(c.ratio * 100)}%`),
    ...(cfg.libelle_patterns || []).map(p => `"${p.pattern}" à ${Math.round(p.ratio * 100)}%`),
    ...(cfg.exclude_patterns || []).map(p => `sauf "${p}"`),
  ];
  document.getElementById('fees-rules').textContent =
    `règles (rules.yml) : ${rules.join(', ')}`;

  mkChart('chart-fees', {
    type: 'bar',
    data: {
      labels: MONTHS,
      datasets: [{
        label: 'Management fees',
        data: data.monthly,
        backgroundColor: 'rgba(188,140,255,0.65)',
        borderRadius: 3,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { ticks: AXIS_TICKS, grid: GRID },
        y: { ticks: { ...AXIS_TICKS, callback: v => fmtNum(v) }, grid: GRID },
      },
    },
  });

  let totalAmount = 0, totalFee = 0;
  document.getElementById('fees-tbody').innerHTML = data.transactions.map(t => {
    totalAmount += t.amount;
    totalFee += t.fee;
    return `<tr>
      <td>${t.date}</td>
      <td class="wrap">${escapeHtml(t.libelle)}</td>
      <td><span class="cat-label">${escapeHtml(t.compte_label)}</span><span class="cat-compte">${t.compte}</span></td>
      <td class="num">${fmtNum(t.amount)}</td>
      <td class="num">${Math.round(t.ratio * 100)}%</td>
      <td class="num purple-num">${fmtNum(t.fee)}</td>
    </tr>`;
  }).join('');
  document.getElementById('fees-tfoot').innerHTML = `
    <tr>
      <td colspan="3">Total (${data.transactions.length} transactions)</td>
      <td class="num">${fmtNum(totalAmount)}</td>
      <td></td>
      <td class="num purple-num">${fmtNum(totalFee)}</td>
    </tr>`;
}
