'use strict';

import { MONTHS, fmtNum, fmtEur, fetchJSON, mkChart, AXIS_TICKS, GRID } from './util.js';

export async function showMission() {
  const data = await fetchJSON('/api/mission');
  const current = (data.rows || []).find(r => r.year === data.year);

  renderKpis(data, current);
  renderBar(data, current);
  renderChart(data, current);
  renderTable(data);
}

function renderKpis(data, r) {
  const el = document.getElementById('mission-kpis');
  if (!r) {
    el.innerHTML = `<div class="kpi-card"><div class="label">Objectif ${data.year}</div>
      <div class="value zero">-</div><div class="sub">aucun objectif dans le plan</div></div>`;
    return;
  }
  const pct = r.ca / r.objective_min * 100;
  const done = r.reste <= 0;
  el.innerHTML = `
    <div class="kpi-card">
      <div class="label">Objectif ${r.year}</div>
      <div class="value">${fmtEur(r.objective_min)}</div>
      <div class="sub">jusqu'à ${fmtEur(r.objective_max)}</div>
    </div>
    <div class="kpi-card green">
      <div class="label">CA réalisé (compta)</div>
      <div class="value">${fmtEur(r.ca)}</div>
      <div class="sub">${pct.toFixed(0)}% de l'objectif bas</div>
    </div>
    <div class="kpi-card blue">
      <div class="label">Pipeline Attio pondéré</div>
      <div class="value">${fmtEur(r.pipeline)}</div>
      <div class="sub">MRR projeté jusqu'à fin décembre</div>
    </div>
    <div class="kpi-card ${done ? 'green' : 'red'}">
      <div class="label">Reste à faire</div>
      <div class="value">${done ? 'Objectif couvert' : fmtEur(r.reste)}</div>
      <div class="sub">${fmtEur(r.reste_compta)} hors pipeline</div>
    </div>
    <div class="kpi-card purple">
      <div class="label">Rythme nécessaire</div>
      <div class="value">${done ? '-' : `${fmtEur(data.run_rate)}/mois`}</div>
      <div class="sub">sur les ${data.months_left} mois restants</div>
    </div>`;
}

// One horizontal bar: réalisé + pipeline + reste = objectif bas.
function renderBar(data, r) {
  document.getElementById('mission-year').textContent = data.year;
  document.getElementById('mission-empty').hidden = data.has_estimate;
  document.getElementById('mission-status').textContent = data.fetched_at
    ? `Pipeline actualisé : ${new Date(data.fetched_at).toLocaleString('fr-FR')}`
    : '';

  const bar = document.getElementById('mission-bar');
  const legend = document.getElementById('mission-legend');
  if (!r) {
    bar.innerHTML = '';
    legend.innerHTML = '';
    return;
  }
  const total = Math.max(r.objective_min, r.ca + r.pipeline);
  const seg = v => Math.max(0, v) / total * 100;
  bar.innerHTML = `<div class="recon-bar">
      <div class="recon-seg ca" style="width:${seg(r.ca).toFixed(2)}%"></div>
      <div class="recon-seg pipe" style="width:${seg(r.pipeline).toFixed(2)}%"></div>
    </div>`;
  legend.innerHTML = `
    <span><span class="sw sw-ca"></span>CA réalisé ${fmtEur(r.ca)}</span>
    <span><span class="sw sw-pipe"></span>Pipeline pondéré ${fmtEur(r.pipeline)}</span>
    <span><span class="sw sw-reste"></span>Reste à faire ${r.reste > 0 ? fmtEur(r.reste) : 'aucun'}</span>
    <span class="recon-target">Objectif bas ${fmtEur(r.objective_min)}</span>`;
}

// Cumulative realized CA (up to the current month), cumulative projection
// (realized + pipeline) and the linear pace toward the objectif bas.
function renderChart(data, r) {
  document.getElementById('mission-year2').textContent = data.year;
  const cumCA = [], cumProj = [];
  let ca = 0, proj = 0;
  for (let i = 0; i < 12; i++) {
    ca += data.monthly_ca[i];
    proj += data.monthly_ca[i] + data.monthly_pipeline[i];
    cumCA.push(i < data.month ? ca : null);
    cumProj.push(proj);
  }
  const datasets = [
    {
      label: 'CA réalisé cumulé',
      data: cumCA,
      borderColor: '#3fb950',
      backgroundColor: 'rgba(63,185,80,0.15)',
      fill: true,
      tension: 0.2,
    },
    {
      label: 'Projection (réalisé + pipeline)',
      data: cumProj,
      borderColor: '#58a6ff',
      borderDash: [6, 4],
      pointRadius: 0,
      tension: 0.2,
    },
  ];
  if (r) {
    datasets.push({
      label: 'Rythme objectif bas',
      data: MONTHS.map((_, i) => r.objective_min / 12 * (i + 1)),
      borderColor: '#8b949e',
      borderDash: [2, 4],
      pointRadius: 0,
    });
  }
  mkChart('chart-mission', {
    type: 'line',
    data: { labels: MONTHS, datasets },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { labels: { color: '#8b949e', font: { size: 11 } } } },
      scales: {
        x: { ticks: AXIS_TICKS, grid: GRID },
        y: { ticks: { ...AXIS_TICKS, callback: v => fmtNum(v) }, grid: GRID },
      },
    },
  });
}

function renderTable(data) {
  document.getElementById('mission-tbody').innerHTML = (data.rows || []).map(r => {
    const pct = r.ca / r.objective_min * 100;
    const projPct = Math.min(r.projection / r.objective_min * 100, 100);
    const color = pct >= 100 ? 'var(--green)' : pct >= 60 ? 'var(--yellow)' : 'var(--red)';
    const reste = v => v > 0
      ? `<span class="debit">${fmtEur(v)}</span>`
      : `<span class="credit">couvert (+${fmtEur(-v)})</span>`;
    return `<tr>
      <td><strong>${r.year}</strong></td>
      <td class="num">${fmtEur(r.objective_min)} - ${fmtEur(r.objective_max)}</td>
      <td class="num">${r.ca ? fmtEur(r.ca) : '<span class="zero">-</span>'}</td>
      <td class="num">${r.pipeline ? fmtEur(r.pipeline) : '<span class="zero">-</span>'}</td>
      <td class="num">${r.projection ? fmtEur(r.projection) : '<span class="zero">-</span>'}</td>
      <td class="num">${reste(r.reste_compta)}</td>
      <td class="num">${reste(r.reste)}</td>
      <td class="progress-col">
        <div class="progress" title="${pct.toFixed(0)}% réalisé, ${(r.projection / r.objective_min * 100).toFixed(0)}% projeté">
          <div class="progress-proj" style="width:${projPct.toFixed(1)}%"></div>
          <div class="progress-fill" style="width:${Math.min(pct, 100).toFixed(1)}%;background:${color}"></div>
        </div><span class="progress-pct">${pct.toFixed(0)}%</span>
      </td>
      <td class="num">${reste(r.reste_resultat)}</td>
    </tr>`;
  }).join('');
}
