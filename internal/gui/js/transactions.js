'use strict';

import { MONTHS, fmtNum, escapeHtml, fetchJSON } from './util.js';

const txCache = new Map();
let currentYear = null;
let bound = false;

async function getTxs(year) {
  if (!txCache.has(year)) {
    txCache.set(year, await fetchJSON(`/api/transactions?year=${year}`));
  }
  return txCache.get(year);
}

export async function showTransactions(year, compte, month) {
  const data = await getTxs(year);
  currentYear = year;

  renderCompteOptions(data.transactions);
  renderMonthOptions();
  if (compte !== undefined) document.getElementById('tx-compte').value = compte || '';
  if (month !== undefined) document.getElementById('tx-month').value = month || '';
  bindOnce();
  applyFilters();
}

function renderCompteOptions(txs) {
  const select = document.getElementById('tx-compte');
  const previous = select.value;
  const seen = new Map();
  for (const t of txs) seen.set(t.compte, t.compte_label);
  const comptes = [...seen.entries()].sort((a, b) => a[0].localeCompare(b[0]));
  select.innerHTML = '<option value="">Toutes cat&eacute;gories</option>' +
    comptes.map(([c, label]) => `<option value="${c}">${c} · ${escapeHtml(label)}</option>`).join('');
  select.value = previous;
}

function renderMonthOptions() {
  const select = document.getElementById('tx-month');
  if (select.options.length) return;
  select.innerHTML = '<option value="">Tous les mois</option>' +
    MONTHS.map((m, i) => `<option value="${i + 1}">${m}</option>`).join('');
}

function bindOnce() {
  if (bound) return;
  bound = true;
  for (const id of ['tx-search', 'tx-compte', 'tx-month', 'tx-sens', 'tx-min', 'tx-max']) {
    document.getElementById(id).addEventListener('input', applyFilters);
  }
  document.getElementById('tx-reset').addEventListener('click', () => {
    for (const id of ['tx-search', 'tx-compte', 'tx-month', 'tx-sens', 'tx-min', 'tx-max']) {
      document.getElementById(id).value = '';
    }
    applyFilters();
  });
}

function readFilters() {
  return {
    q: document.getElementById('tx-search').value.trim().toLowerCase(),
    compte: document.getElementById('tx-compte').value,
    month: parseInt(document.getElementById('tx-month').value, 10) || 0,
    sens: document.getElementById('tx-sens').value,
    min: parseFloat(document.getElementById('tx-min').value),
    max: parseFloat(document.getElementById('tx-max').value),
  };
}

function matches(t, f) {
  if (f.q && !t.libelle.toLowerCase().includes(f.q)) return false;
  if (f.compte && t.compte !== f.compte) return false;
  if (f.month && parseInt(t.date.slice(3, 5), 10) !== f.month) return false;
  if (f.sens === 'debit' && !t.debit) return false;
  if (f.sens === 'credit' && !t.credit) return false;
  const amount = Math.max(t.debit, t.credit);
  if (!Number.isNaN(f.min) && amount < f.min) return false;
  if (!Number.isNaN(f.max) && amount > f.max) return false;
  return true;
}

async function applyFilters() {
  const data = await getTxs(currentYear);
  const f = readFilters();
  const rows = data.transactions.filter(t => matches(t, f));

  let totalDebit = 0, totalCredit = 0;
  const html = rows.map(t => {
    totalDebit += t.debit;
    totalCredit += t.credit;
    return `<tr>
      <td>${t.date}</td>
      <td class="wrap">${escapeHtml(t.libelle)}</td>
      <td><span class="cat-label">${escapeHtml(t.compte_label)}</span><span class="cat-compte">${t.compte}</span></td>
      <td class="num debit">${t.debit ? fmtNum(t.debit) : ''}</td>
      <td class="num credit">${t.credit ? fmtNum(t.credit) : ''}</td>
    </tr>`;
  }).join('');
  document.getElementById('tx-tbody').innerHTML = html;
  document.getElementById('tx-tfoot').innerHTML = `
    <tr>
      <td colspan="3">Total</td>
      <td class="num debit">${fmtNum(totalDebit)}</td>
      <td class="num credit">${fmtNum(totalCredit)}</td>
    </tr>`;
  document.getElementById('tx-count').textContent =
    `${rows.length} / ${data.transactions.length} transactions`;
}
