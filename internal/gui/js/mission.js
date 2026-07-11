'use strict';

import { MONTHS, fmtNum, fmtEur, fetchJSON, mkChart, AXIS_TICKS, GRID } from './util.js';

export async function showMission() {
  const data = await fetchJSON('/api/mission');
  const current = (data.rows || []).find(r => r.year === data.year);

  renderFunnel(data, current);
  renderKpis(data, current);
  renderProfit(data, current);
  renderChart(data, current);
  renderTable(data);
}

// The four levels, from the most abstract to the most real. Each one shows
// how much of the level above it is secured, and what remains to be done.
function renderFunnel(data, r) {
  document.getElementById('mission-year').textContent = data.year;
  document.getElementById('mission-empty').hidden = data.has_estimate;
  document.getElementById('mission-status').textContent = data.fetched_at
    ? `Pipeline actualisé : ${new Date(data.fetched_at).toLocaleString('fr-FR')}`
    : '';

  const funnel = document.getElementById('mission-funnel');
  if (!r) {
    funnel.innerHTML = `<p class="hint">Aucun objectif dans le plan pour ${data.year}.</p>`;
    return;
  }
  const pct = (a, b) => b ? `${(a / b * 100).toFixed(0)}%` : '-';
  const levels = [
    {
      cls: 'objectif', label: 'Objectif', value: r.objective,
      sub: 'l\'ambition (rules.yml)',
      note: `marge cible ${r.margin}% · résultat visé ${fmtEur(r.objective * r.margin / 100)}`,
    },
    {
      cls: 'attio', label: 'Projection Attio', value: r.projection,
      sub: 'facturé + pipeline pondéré',
      note: `pipeline ${fmtEur(r.pipeline)} · reste à vendre ${gap(r.reste_vendre)}`,
    },
    {
      cls: 'ca', label: 'CA facturé', value: r.ca,
      sub: 'signé et facturé, pas encore payé',
      note: `${pct(r.ca, r.objective)} de l'objectif · reste à facturer ${gap(r.reste_facturer)}`,
    },
    {
      cls: 'cash', label: 'Encaissé', value: r.cash,
      sub: 'sur le compte en banque',
      note: `${pct(r.cash, r.ca)} du facturé · reste à encaisser ${gap(r.reste_encaisser)}`,
    },
  ];
  const scale = Math.max(r.objective, r.projection);
  funnel.innerHTML = levels.map(l => `
    <div class="funnel-row">
      <div class="funnel-head">
        <span class="funnel-label">${l.label}</span>
        <span class="funnel-sub">${l.sub}</span>
      </div>
      <div class="funnel-track">
        <div class="funnel-fill ${l.cls}" style="width:${(Math.max(0, l.value) / scale * 100).toFixed(2)}%"></div>
        <span class="funnel-value">${fmtEur(l.value)}</span>
      </div>
      <div class="funnel-note">${l.note}</div>
    </div>`).join('');
}

function gap(v) {
  return v > 0 ? `<span class="debit">${fmtEur(v)}</span>` : '<span class="credit">aucun</span>';
}

function renderKpis(data, r) {
  const el = document.getElementById('mission-kpis');
  if (!r) {
    el.innerHTML = '';
    return;
  }
  el.innerHTML = `
    <div class="kpi-card red">
      <div class="label">Reste à vendre</div>
      <div class="value">${r.reste_vendre > 0 ? fmtEur(r.reste_vendre) : 'couvert'}</div>
      <div class="sub">objectif non couvert, même par le pipeline</div>
    </div>
    <div class="kpi-card blue">
      <div class="label">Reste à facturer</div>
      <div class="value">${r.reste_facturer > 0 ? fmtEur(r.reste_facturer) : 'couvert'}</div>
      <div class="sub">objectif moins le CA facturé</div>
    </div>
    <div class="kpi-card purple">
      <div class="label">Reste à encaisser</div>
      <div class="value">${r.reste_encaisser > 0 ? fmtEur(r.reste_encaisser) : 'à jour'}</div>
      <div class="sub">facturé mais pas encore payé</div>
    </div>
    <div class="kpi-card">
      <div class="label">Rythme nécessaire</div>
      <div class="value">${r.reste_vendre > 0 ? `${fmtEur(data.run_rate)}/mois` : '-'}</div>
      <div class="sub">de ventes sur les ${data.months_left} mois restants</div>
    </div>`;
}

// From the CA to the shareholder's pocket: target vs actual net margin,
// gross profit, estimated IS (scale in rules.yml, docs/IS_CHEAT_SHEET.md),
// net profit (with the management fees added back as the true profit), and
// the dividend (payout x net) vs its objective.
function renderProfit(data, r) {
  document.getElementById('mission-year3').textContent = data.year;
  const el = document.getElementById('mission-profit');
  if (!r) {
    el.innerHTML = '';
    return;
  }
  const profitTarget = r.objective * r.margin / 100;
  const marginOk = r.net_margin >= r.margin;
  const divPct = r.dividend_target ? (r.dividend / r.dividend_target * 100).toFixed(0) : '-';
  el.innerHTML = `
    <div class="kpi-card ${marginOk ? 'green' : 'red'}">
      <div class="label">Marge nette</div>
      <div class="value">${r.net_margin.toFixed(1)}%</div>
      <div class="sub">cible ${r.margin}% (résultat visé ${fmtEur(profitTarget)})</div>
    </div>
    <div class="kpi-card blue">
      <div class="label">Résultat brut</div>
      <div class="value">${fmtEur(r.gross_profit)}</div>
      <div class="sub">CA facturé moins charges, avant IS</div>
    </div>
    <div class="kpi-card purple">
      <div class="label">IS estimé</div>
      <div class="value">${fmtEur(r.is)}</div>
      <div class="sub">barème rules.yml · docs/IS_CHEAT_SHEET.md</div>
    </div>
    <div class="kpi-card ${r.net_profit >= 0 ? 'green' : 'red'}">
      <div class="label">Résultat net</div>
      <div class="value">${fmtEur(r.net_profit)}</div>
      <div class="sub">management fees réintégrés : ${fmtEur(r.net_profit_fees)}</div>
    </div>
    <div class="kpi-card">
      <div class="label">Dividendes</div>
      <div class="value">${fmtEur(r.dividend)}</div>
      <div class="sub">objectif ${fmtEur(r.dividend_target)} · ${divPct}% atteint</div>
    </div>`;
}

// Cumulative encaisse and facture (up to the current month), cumulative
// projection (facture + pipeline) and the linear pace toward the objectif.
function renderChart(data, r) {
  document.getElementById('mission-year2').textContent = data.year;
  const cumCash = [], cumCA = [], cumProj = [];
  let cash = 0, ca = 0, proj = 0;
  for (let i = 0; i < 12; i++) {
    cash += data.monthly_cash[i];
    ca += data.monthly_ca[i];
    proj += data.monthly_ca[i] + data.monthly_pipeline[i];
    cumCash.push(i < data.month ? cash : null);
    cumCA.push(i < data.month ? ca : null);
    cumProj.push(proj);
  }
  const datasets = [
    {
      label: 'Encaissé cumulé',
      data: cumCash,
      borderColor: '#3fb950',
      backgroundColor: 'rgba(63,185,80,0.25)',
      fill: true,
      tension: 0.2,
    },
    {
      label: 'CA facturé cumulé',
      data: cumCA,
      borderColor: '#d29922',
      backgroundColor: 'rgba(210,153,34,0.10)',
      fill: true,
      tension: 0.2,
    },
    {
      label: 'Projection (facturé + pipeline)',
      data: cumProj,
      borderColor: '#58a6ff',
      borderDash: [6, 4],
      pointRadius: 0,
      tension: 0.2,
    },
  ];
  if (r) {
    datasets.push({
      label: 'Rythme objectif',
      data: MONTHS.map((_, i) => r.objective / 12 * (i + 1)),
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
    const pct = r.ca / r.objective * 100;
    const projPct = Math.min(r.projection / r.objective * 100, 100);
    const color = pct >= 100 ? 'var(--green)' : pct >= 60 ? 'var(--yellow)' : 'var(--red)';
    return `<tr>
      <td><strong>${r.year}</strong></td>
      <td class="num">${fmtEur(r.objective)}</td>
      <td class="num">${r.pipeline ? fmtEur(r.pipeline) : '<span class="zero">-</span>'}</td>
      <td class="num">${r.ca ? fmtEur(r.ca) : '<span class="zero">-</span>'}</td>
      <td class="num">${r.cash ? fmtEur(r.cash) : '<span class="zero">-</span>'}</td>
      <td class="num">${gap(r.reste_vendre)}</td>
      <td class="num">${gap(r.reste_facturer)}</td>
      <td class="num">${r.ca ? gap(r.reste_encaisser) : '<span class="zero">-</span>'}</td>
      <td class="progress-col">
        <div class="progress" title="${pct.toFixed(0)}% facturé, ${(r.projection / r.objective * 100).toFixed(0)}% projeté">
          <div class="progress-proj" style="width:${projPct.toFixed(1)}%"></div>
          <div class="progress-fill" style="width:${Math.min(pct, 100).toFixed(1)}%;background:${color}"></div>
        </div><span class="progress-pct">${pct.toFixed(0)}%</span>
      </td>
    </tr>`;
  }).join('');
}
