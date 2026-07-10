'use strict';

export const MONTHS = ['janv.', 'févr.', 'mars', 'avr.', 'mai', 'juin',
                       'juil.', 'août', 'sept.', 'oct.', 'nov.', 'déc.'];

const numFmt = new Intl.NumberFormat('fr-FR', { maximumFractionDigits: 0 });
const eurFmt = new Intl.NumberFormat('fr-FR', {
  style: 'currency', currency: 'EUR', maximumFractionDigits: 0,
});

export function fmtNum(v) { return numFmt.format(v); }
export function fmtEur(v) { return eurFmt.format(v); }

export function escapeHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export async function fetchJSON(url, options) {
  const res = await fetch(url, options);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

// One Chart.js instance per canvas id, destroyed on re-render.
const charts = {};
export function mkChart(canvasId, config) {
  if (charts[canvasId]) charts[canvasId].destroy();
  const ctx = document.getElementById(canvasId).getContext('2d');
  charts[canvasId] = new Chart(ctx, config);
  return charts[canvasId];
}

export const AXIS_TICKS = { color: '#8b949e', font: { size: 10 } };
export const GRID = { color: '#21262d' };

// The exercice list, shared by every view.
let yearsCache = null;
export async function getYears() {
  if (!yearsCache) {
    const d = await fetchJSON('/api/data');
    yearsCache = d.years;
  }
  return yearsCache;
}
