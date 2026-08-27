// Route detail page: Try panel — fetch the feed URL and show status,
// timing and highlighted XML output.

import { escHTML, hlXML, fmtXML } from './xml.js';

const form = document.querySelector('form.try-row');
const input = document.getElementById('tryPath');
const st = document.getElementById('tryStatus');
const out = document.getElementById('tryOut');

if (form && input && st && out) {
    form.addEventListener('submit', e => {
        e.preventDefault();
        runTry();
    });
}

function runTry() {
    const path = input.value.trim();
    out.hidden = false;
    st.hidden = false;
    st.className = '';
    st.textContent = 'LOADING';
    out.textContent = '';
    const started = performance.now();
    fetch(path).then(async r => {
        const ms = Math.round(performance.now() - started);
        st.className = r.ok ? 'yes' : 'no';
        st.textContent = r.status + ' ' + r.statusText + ' · ' + ms + 'MS';
        out.innerHTML = hlXML(fmtXML(await r.text()));
    }).catch(e => {
        st.className = 'no';
        st.textContent = 'FETCH FAILED';
        out.textContent = String(e);
    });
}
