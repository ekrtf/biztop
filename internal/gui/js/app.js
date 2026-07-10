'use strict';

// Router and navigation. Routes:
//   #/compta/pilotage/<year>
//   #/compta/transactions/<year>/<compte?>/<month?>
//   #/clients
//   #/objectifs
//   #/fees/<year>

import { getYears } from './util.js';
import { showPilotage } from './pilotage.js';
import { showTransactions } from './transactions.js';
import { showClients } from './clients.js';
import { showObjectifs } from './objectifs.js';
import { showFees } from './fees.js';

const TABS = ['compta', 'clients', 'objectifs', 'fees'];
const SUBS = ['pilotage', 'transactions'];

const state = { tab: 'compta', sub: 'pilotage', year: null };

function parseRoute() {
  const p = location.hash.replace(/^#\/?/, '').split('/').filter(Boolean);
  const tab = TABS.includes(p[0]) ? p[0] : 'compta';
  if (tab === 'compta') {
    const sub = SUBS.includes(p[1]) ? p[1] : 'pilotage';
    return {
      tab, sub,
      year: parseInt(p[2], 10) || null,
      compte: p[3] || '',
      month: parseInt(p[4], 10) || 0,
    };
  }
  if (tab === 'fees') return { tab, year: parseInt(p[1], 10) || null };
  return { tab };
}

function hashFor(tab, sub, year) {
  if (tab === 'compta') return `#/compta/${sub}/${year}`;
  if (tab === 'fees') return `#/fees/${year}`;
  return `#/${tab}`;
}

function setActive(selector, dataKey, value) {
  document.querySelectorAll(selector).forEach(btn => {
    btn.classList.toggle('active', btn.dataset[dataKey] === value);
  });
}

function renderYearNav(years, active, visible) {
  const nav = document.getElementById('year-nav');
  nav.hidden = !visible;
  if (!visible) return;
  nav.replaceChildren(...years.map(y => {
    const btn = document.createElement('button');
    btn.className = 'year-btn' + (y === active ? ' active' : '');
    btn.textContent = y;
    btn.addEventListener('click', () => { location.hash = hashFor(state.tab, state.sub, y); });
    return btn;
  }));
}

async function route() {
  const r = parseRoute();
  state.tab = r.tab;
  if (r.sub) state.sub = r.sub;

  const years = await getYears();
  state.year = r.year || state.year || years[years.length - 1];

  setActive('.tab-btn', 'tab', state.tab);
  for (const tab of TABS) {
    document.getElementById(`tab-${tab}`).hidden = tab !== state.tab;
  }
  renderYearNav(years, state.year, state.tab === 'compta' || state.tab === 'fees');

  if (state.tab === 'compta') {
    setActive('.subtab-btn', 'sub', state.sub);
    document.getElementById('view-pilotage').hidden = state.sub !== 'pilotage';
    document.getElementById('view-transactions').hidden = state.sub !== 'transactions';
    if (state.sub === 'pilotage') await showPilotage(state.year);
    else await showTransactions(state.year, r.compte, r.month);
  } else if (state.tab === 'clients') {
    await showClients();
  } else if (state.tab === 'objectifs') {
    await showObjectifs();
  } else {
    await showFees(state.year);
  }
}

document.getElementById('main-tabs').addEventListener('click', e => {
  const btn = e.target.closest('.tab-btn');
  if (btn) location.hash = hashFor(btn.dataset.tab, state.sub, state.year);
});
document.querySelector('.subtabs').addEventListener('click', e => {
  const btn = e.target.closest('.subtab-btn');
  if (btn) location.hash = hashFor('compta', btn.dataset.sub, state.year);
});

window.addEventListener('hashchange', route);
route();
