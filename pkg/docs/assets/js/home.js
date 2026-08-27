// Index page: live filter over route rows plus the "/" focus shortcut.

const q = document.getElementById('q');

function filterRoutes() {
    const query = q.value.trim().toLowerCase();
    document.querySelectorAll('.namespace').forEach(ns => {
        let visible = 0;
        ns.querySelectorAll('.route').forEach(r => {
            const hit = !query || r.getAttribute('data-k').includes(query);
            r.style.display = hit ? '' : 'none';
            if (hit) visible++;
        });
        ns.style.display = visible > 0 ? '' : 'none';
    });
}

q.addEventListener('input', filterRoutes);

document.addEventListener('keydown', e => {
    const tag = (document.activeElement || {}).tagName;
    if (e.key === '/' && tag !== 'INPUT' && tag !== 'TEXTAREA') {
        e.preventDefault();
        q.focus();
    } else if (e.key === 'Escape' && tag === 'INPUT') {
        q.blur();
    }
});
