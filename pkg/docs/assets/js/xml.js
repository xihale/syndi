// Display helpers for the route detail Try panel output.

export function escHTML(t) {
    return t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export function hlXML(t) {
    const s = escHTML(t);
    if (!t.trimStart().startsWith('<')) return s;
    return s
        .replace(/&lt;!--[\s\S]*?--&gt;/g, m => '<span class="x-cmt">' + m + '</span>')
        .replace(/&lt;\?[\s\S]*?\?&gt;/g, m => '<span class="x-cmt">' + m + '</span>')
        .replace(/&lt;(\/?)([A-Za-z][\w:.-]*)((?:\s+[A-Za-z_:][\w:.-]*(?:="[^"]*")?)*)\s*(\/?)&gt;/g,
            function(m, slash, name, attrs, selfc) {
                attrs = attrs.replace(/([A-Za-z_:][\w:.-]*)(=)("[^"]*")/g,
                    (a, an, eq, v) => '<span class="x-attr">' + an + '</span>=<span class="x-str">' + v + '</span>');
                return '&lt;' + slash + '<span class="x-tag">' + name + '</span>' + attrs + selfc + '&gt;';
            });
}

export function fmtXML(t) {
    t = t.replace(/^\s*<\?xml[^>]*\?>/, '').trimStart();
    if (!t.startsWith('<')) return t;
    const lines = t.replace(/></g, '>\n<').split('\n');
    let depth = 0;
    return lines.map(raw => {
        const l = raw.trim();
        if (/^<\//.test(l)) depth = Math.max(depth - 1, 0);
        const out = '  '.repeat(depth) + l;
        if (/^<[a-zA-Z]/.test(l) && !/<\/[a-zA-Z][\w:-]*>$/.test(l) && !/\/>$/.test(l)) depth++;
        return out;
    }).join('\n');
}
