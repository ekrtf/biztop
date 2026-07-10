'use strict';

import { fmtEur, escapeHtml, fetchJSON } from './util.js';

const DAVAI_TYPES = ['Projects', 'Maintenance & Hosting', 'Prezence', 'Bodacker'];
let bound = false;

export async function showObjectifs() {
  const data = await fetchJSON('/api/objectives');
  render(data);
  bindOnce();
}

// Weighted pipeline (amount x probability) per expected year.
function pipelineByYear(estimate) {
  const byYear = {};
  for (const deal of estimate?.deals || []) {
    const year = parseInt(String(deal.expected_month).slice(0, 4), 10);
    if (!year) continue;
    byYear[year] = (byYear[year] || 0) + deal.amount_eur * deal.probability;
  }
  return byYear;
}

function render(data) {
  const objectives = data.objectives || [];
  const actuals = data.actuals || {};
  const pipeline = pipelineByYear(data.estimate);

  const years = [...new Set([
    ...Object.keys(actuals).map(Number),
    ...objectives.map(o => o.year),
  ])].sort();

  document.getElementById('obj-tbody').innerHTML = years.map(year => {
    const o = objectives.find(x => x.year === year);
    const a = actuals[year];
    const pipe = pipeline[year] || 0;
    const projection = a || pipe ? (a?.ca || 0) + pipe : null;
    const pct = o && a ? a.ca / o.revenue_min * 100 : null;
    const projPct = o && projection !== null ? projection / o.revenue_min * 100 : null;
    return `<tr>
      <td><strong>${year}</strong></td>
      <td class="num">${o ? `${fmtEur(o.revenue_min)} - ${fmtEur(o.revenue_max)}` : '<span class="zero">-</span>'}</td>
      <td class="num">${a ? fmtEur(a.ca) : '<span class="zero">-</span>'}</td>
      <td class="progress-col">${pct !== null ? progressBar(pct, projPct) : ''}</td>
      <td class="num">${pipe ? fmtEur(pipe) : '<span class="zero">-</span>'}</td>
      <td class="num">${projection !== null && pipe ? fmtEur(projection) : '<span class="zero">-</span>'}</td>
      <td class="num">${o ? `${fmtEur(o.profit_min)} - ${fmtEur(o.profit_max)} (${o.margin_min}-${o.margin_max}%)` : '<span class="zero">-</span>'}</td>
      <td class="num ${a && a.resultat < 0 ? 'debit' : ''}">${a ? fmtEur(a.resultat) : '<span class="zero">-</span>'}</td>
      <td>${o ? escapeHtml(o.team) : ''}</td>
    </tr>`;
  }).join('');

  renderEstimate(data.estimate);
}

// Bar toward the low end of the objective; the hatched extension is the
// projection including the weighted Attio pipeline.
function progressBar(pct, projPct) {
  const real = Math.min(pct, 100);
  const proj = projPct === null ? real : Math.min(projPct, 100);
  const color = pct >= 100 ? 'var(--green)' : pct >= 60 ? 'var(--yellow)' : 'var(--red)';
  return `<div class="progress" title="${pct.toFixed(0)}% de l'objectif bas">
      <div class="progress-proj" style="width:${proj.toFixed(1)}%"></div>
      <div class="progress-fill" style="width:${real.toFixed(1)}%;background:${color}"></div>
    </div><span class="progress-pct">${pct.toFixed(0)}%</span>`;
}

function renderEstimate(estimate) {
  const empty = !estimate;
  document.getElementById('obj-empty').hidden = !empty;
  document.getElementById('obj-deals-table').hidden = empty;
  document.getElementById('obj-refresh-status').textContent = estimate?.fetched_at
    ? `Dernière actualisation : ${new Date(estimate.fetched_at).toLocaleString('fr-FR')}`
    : '';

  const byType = estimate?.by_type || {};
  document.getElementById('obj-type-cards').innerHTML = DAVAI_TYPES.map(t => `
    <div class="kpi-card blue">
      <div class="label">${t}</div>
      <div class="value">${fmtEur(byType[t] || 0)}</div>
    </div>`).join('');

  if (empty) return;
  let total = 0, weighted = 0;
  const deals = [...(estimate.deals || [])].sort((a, b) => b.amount_eur * b.probability - a.amount_eur * a.probability);
  document.getElementById('obj-deals-tbody').innerHTML = deals.map(d => {
    total += d.amount_eur;
    weighted += d.amount_eur * d.probability;
    return `<tr>
      <td class="wrap" title="${escapeHtml(d.notes || '')}">${escapeHtml(d.name)}</td>
      <td>${escapeHtml(d.type)}</td>
      <td class="num">${fmtEur(d.amount_eur)}</td>
      <td>${escapeHtml(d.expected_month)}</td>
      <td class="num">${Math.round(d.probability * 100)}%</td>
      <td class="num">${fmtEur(d.amount_eur * d.probability)}</td>
    </tr>`;
  }).join('');
  document.getElementById('obj-deals-tfoot').innerHTML = `
    <tr>
      <td colspan="2">Total (${deals.length} deals)</td>
      <td class="num">${fmtEur(total)}</td>
      <td></td><td></td>
      <td class="num">${fmtEur(weighted)}</td>
    </tr>`;
}

function bindOnce() {
  if (bound) return;
  bound = true;
  const btn = document.getElementById('obj-refresh');
  btn.addEventListener('click', async () => {
    const status = document.getElementById('obj-refresh-status');
    btn.disabled = true;
    status.textContent = 'Estimation en cours via codex, cela peut prendre quelques minutes...';
    try {
      await fetchJSON('/api/objectives/refresh', { method: 'POST' });
      render(await fetchJSON('/api/objectives'));
    } catch (err) {
      status.textContent = `Erreur : ${err.message.slice(0, 300)}`;
    } finally {
      btn.disabled = false;
    }
  });
}
