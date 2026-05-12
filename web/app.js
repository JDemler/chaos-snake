(() => {
    const mainCanvas = document.getElementById('main');
    const mainCtx = mainCanvas.getContext('2d');
    const thumbsContainer = document.getElementById('thumbs');
    const statusEl = document.getElementById('status');
    const lengthEl = document.getElementById('length');
    const fieldCountEl = document.getElementById('fieldcount');
    const playerCountEl = document.getElementById('playercount');
    const leadersEl = document.getElementById('leaders');
    const kingEl = document.getElementById('king');
    const joinDiv = document.getElementById('join');
    const joinForm = document.getElementById('joinForm');
    const nameInput = document.getElementById('name');
    const dpad = document.getElementById('dpad');

    const MAIN_TILE = 20;
    const THUMB_TILE = 4;

    const state = {
        size: { w: 30, h: 30 },
        tickHz: 10,
        fields: new Map(),  // FieldID -> { id, pellet }
        snakes: new Map(),  // SnakeID -> { id, name, color, body[Tile], dir }
        leaderboard: [],    // [{ name, peak }] sorted desc
        you: '',
    };

    const LEADER_LIMIT = 10;

    // FieldID -> {canvas, ctx, label}
    const thumbs = new Map();

    let ws = null;
    let reconnectDelay = 500;
    let lastName = '';

    // Set when the local player's snake crosses a field edge; the entered edge
    // of the new main field flashes briefly to make the teleport read as
    // continuous motion rather than re-spawn.
    let lastCrossing = null; // { edge: 'left'|'right'|'top'|'bottom', start: ms }
    let mainRAF = null;

    function connect() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${proto}//${location.host}/ws`);

        ws.onopen = () => {
            statusEl.textContent = 'connected';
            reconnectDelay = 500;
            if (lastName) send({ type: 'join', name: lastName });
        };

        ws.onmessage = (e) => {
            try { handleMessage(JSON.parse(e.data)); }
            catch (err) { console.error('bad message', err); }
        };

        ws.onclose = () => {
            statusEl.textContent = 'disconnected, reconnecting…';
            state.fields.clear();
            state.snakes.clear();
            state.leaderboard = [];
            state.you = '';
            lastCrossing = null;
            joinDiv.hidden = !!lastName;
            dpad.hidden = true;
            clearThumbs();
            renderThumbs();
            renderLeaderboard();
            ws = null;
            setTimeout(connect, reconnectDelay);
            reconnectDelay = Math.min(reconnectDelay * 2, 5000);
        };

        ws.onerror = () => {};
    }

    function send(obj) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify(obj));
        }
    }

    function handleMessage(msg) {
        if (msg.type === 'snapshot') applySnapshot(msg);
        else if (msg.type === 'delta') applyDelta(msg);
    }

    function applySnapshot(msg) {
        state.size = msg.field_size;
        state.tickHz = msg.field_size.tick_hz;
        state.you = msg.you || '';
        state.fields.clear();
        for (const f of msg.fields) {
            state.fields.set(f.id, { id: f.id, pellets: (f.pellets || []).map(p => [p[0], p[1]]) });
        }
        state.snakes.clear();
        for (const s of msg.snakes) state.snakes.set(s.id, cloneSnake(s));
        state.leaderboard = (msg.leaderboard || []).map(e => ({ name: e.name, peak: e.peak }));

        if (state.you) {
            joinDiv.hidden = true;
            dpad.hidden = false;
        } else {
            joinDiv.hidden = false;
            dpad.hidden = true;
        }
        rebuildThumbs();
        updateLength();
        renderLeaderboard();
        renderThumbs();
        startMainLoop();
    }

    function applyDelta(msg) {
        if (msg.field_joins) {
            for (const f of msg.field_joins) {
                state.fields.set(f.id, { id: f.id, pellets: (f.pellets || []).map(p => [p[0], p[1]]) });
            }
        }
        if (msg.field_leaves) {
            for (const id of msg.field_leaves) state.fields.delete(id);
        }
        if (msg.joins) {
            for (const s of msg.joins) state.snakes.set(s.id, cloneSnake(s));
        }
        if (msg.leaves) {
            for (const id of msg.leaves) state.snakes.delete(id);
        }
        if (msg.moves) {
            for (const m of msg.moves) {
                const s = state.snakes.get(m.id);
                if (!s) continue;
                const prevHead = s.body[0];
                if (m.id === state.you && !m.dead && prevHead && prevHead.f !== m.head.f) {
                    // Own snake crossed a field edge. The new head lands on the
                    // opposite edge of the destination field; figure out which.
                    const x = m.head.p[0], y = m.head.p[1];
                    let edge = null;
                    if (x === 0) edge = 'left';
                    else if (x === state.size.w - 1) edge = 'right';
                    else if (y === 0) edge = 'top';
                    else if (y === state.size.h - 1) edge = 'bottom';
                    if (edge) lastCrossing = { edge, start: performance.now() };
                }
                if (m.dead) {
                    s.body = [{ f: m.head.f, p: [m.head.p[0], m.head.p[1]] }];
                } else {
                    s.body.unshift({ f: m.head.f, p: [m.head.p[0], m.head.p[1]] });
                    if (!m.grew) s.body.pop();
                }
            }
        }
        if (msg.pellets) {
            for (const p of msg.pellets) {
                const f = state.fields.get(p.f);
                if (f) f.pellets = (p.ps || []).map(q => [q[0], q[1]]);
            }
        }
        rebuildThumbs();
        updateLength();
        renderLeaderboard();
        renderThumbs();
    }

    function cloneSnake(s) {
        return {
            id: s.id,
            name: s.name,
            color: s.color,
            body: s.body.map(t => ({ f: t.f, p: [t.p[0], t.p[1]] })),
            dir: s.dir,
        };
    }

    function updateLength() {
        if (!state.you) { lengthEl.textContent = ''; }
        else {
            const me = state.snakes.get(state.you);
            lengthEl.textContent = me ? `length: ${me.body.length}` : '';
        }
        fieldCountEl.textContent = state.fields.size > 0
            ? `fields: ${state.fields.size}`
            : '';
        playerCountEl.textContent = `players: ${state.snakes.size}`;
    }

    // currentKing returns { name, length } of the longest snake currently
    // alive, or null if no snakes exist. Ties broken by name ascending.
    function currentKing() {
        let best = null;
        for (const s of state.snakes.values()) {
            const len = s.body.length;
            if (!best || len > best.length ||
                (len === best.length && s.name < best.name)) {
                best = { name: s.name, length: len };
            }
        }
        return best;
    }

    function renderLeaderboard() {
        const king = currentKing();
        if (king) {
            kingEl.textContent = `👑 ${king.name} — ${king.length}`;
        } else {
            kingEl.textContent = '';
        }
        leadersEl.innerHTML = '';
        const top = state.leaderboard.slice(0, LEADER_LIMIT);
        for (const e of top) {
            const li = document.createElement('li');
            li.className = 'leader';
            if (king && e.name === king.name) li.classList.add('is-king');
            const nameSpan = document.createElement('span');
            nameSpan.className = 'leader-name';
            nameSpan.textContent = e.name;
            const peakSpan = document.createElement('span');
            peakSpan.className = 'leader-peak';
            peakSpan.textContent = e.peak;
            li.appendChild(nameSpan);
            li.appendChild(peakSpan);
            leadersEl.appendChild(li);
        }
    }

    // currentFieldID returns the field the player's head is on (when joined),
    // or the lexicographically-first field otherwise.
    function currentFieldID() {
        if (state.you) {
            const me = state.snakes.get(state.you);
            if (me && me.body.length > 0) return me.body[0].f;
        }
        const ids = [...state.fields.keys()].sort();
        return ids[0] || null;
    }

    function rebuildThumbs() {
        const main = currentFieldID();
        const wantThumbIDs = new Set(
            [...state.fields.keys()].filter(id => id !== main)
        );
        // Remove obsolete thumbnails.
        for (const [id, entry] of thumbs) {
            if (!wantThumbIDs.has(id)) {
                entry.wrapper.remove();
                thumbs.delete(id);
            }
        }
        // Add new thumbnails.
        for (const id of wantThumbIDs) {
            if (thumbs.has(id)) continue;
            const wrapper = document.createElement('div');
            wrapper.className = 'thumb';
            const label = document.createElement('div');
            label.className = 'thumb-label';
            label.textContent = id;
            const canvas = document.createElement('canvas');
            canvas.width = state.size.w * THUMB_TILE;
            canvas.height = state.size.h * THUMB_TILE;
            wrapper.appendChild(canvas);
            wrapper.appendChild(label);
            thumbsContainer.appendChild(wrapper);
            thumbs.set(id, { wrapper, canvas, ctx: canvas.getContext('2d') });
        }
        // Resize main canvas to match field size.
        const mainW = state.size.w * MAIN_TILE;
        const mainH = state.size.h * MAIN_TILE;
        if (mainCanvas.width !== mainW) mainCanvas.width = mainW;
        if (mainCanvas.height !== mainH) mainCanvas.height = mainH;
    }

    function clearThumbs() {
        for (const entry of thumbs.values()) entry.wrapper.remove();
        thumbs.clear();
    }

    function startMainLoop() {
        if (mainRAF !== null) return;
        const loop = (now) => {
            renderMain(now);
            mainRAF = requestAnimationFrame(loop);
        };
        mainRAF = requestAnimationFrame(loop);
    }

    function renderMain(now) {
        const mainID = currentFieldID();
        if (mainID) {
            renderField(mainCtx, mainID, MAIN_TILE, true, now);
        } else {
            mainCtx.fillStyle = '#000';
            mainCtx.fillRect(0, 0, mainCanvas.width, mainCanvas.height);
        }
    }

    function renderThumbs() {
        for (const [id, entry] of thumbs) {
            renderField(entry.ctx, id, THUMB_TILE, false, 0);
        }
    }

    function renderField(ctx, fieldID, tile, isMain, now) {
        const w = state.size.w * tile;
        const h = state.size.h * tile;
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, w, h);

        if (isMain) {
            ctx.strokeStyle = '#1a1a1a';
            ctx.lineWidth = 1;
            for (let x = 0; x <= state.size.w; x++) {
                ctx.beginPath();
                ctx.moveTo(x * tile + 0.5, 0);
                ctx.lineTo(x * tile + 0.5, h);
                ctx.stroke();
            }
            for (let y = 0; y <= state.size.h; y++) {
                ctx.beginPath();
                ctx.moveTo(0, y * tile + 0.5);
                ctx.lineTo(w, y * tile + 0.5);
                ctx.stroke();
            }
        }

        // pellets
        const f = state.fields.get(fieldID);
        if (f && f.pellets) {
            ctx.fillStyle = '#e22';
            const inset = Math.max(1, Math.floor(tile * 0.15));
            for (const p of f.pellets) {
                ctx.fillRect(
                    p[0] * tile + inset,
                    p[1] * tile + inset,
                    tile - inset * 2,
                    tile - inset * 2,
                );
            }
        }

        // Snakes: other players first, own snake last with a halo so the
        // player can pick themselves out without hunting.
        const inset = tile >= 8 ? 1 : 0;
        for (const s of state.snakes.values()) {
            if (s.id === state.you) continue;
            drawSnakeBody(ctx, s, fieldID, tile, inset);
        }
        const me = state.snakes.get(state.you);
        if (me) {
            ctx.save();
            ctx.shadowBlur = isMain ? tile * 0.8 : tile * 0.6;
            ctx.shadowColor = '#9cf';
            drawSnakeBody(ctx, me, fieldID, tile, inset);
            ctx.restore();
            if (tile >= 8) {
                const head = me.body[0];
                if (head && head.f === fieldID) {
                    ctx.strokeStyle = '#fff';
                    ctx.lineWidth = 2;
                    ctx.strokeRect(head.p[0] * tile + 1, head.p[1] * tile + 1, tile - 2, tile - 2);
                }
            }
        }

        if (isMain) drawCrossingFlash(ctx, w, h, tile, now);
    }

    function drawSnakeBody(ctx, s, fieldID, tile, inset) {
        for (let i = 0; i < s.body.length; i++) {
            const t = s.body[i];
            if (t.f !== fieldID) continue;
            ctx.fillStyle = i === 0 ? brighten(s.color) : s.color;
            ctx.fillRect(
                t.p[0] * tile + inset,
                t.p[1] * tile + inset,
                tile - inset * 2,
                tile - inset * 2,
            );
        }
    }

    function drawCrossingFlash(ctx, w, h, tile, now) {
        if (!lastCrossing) return;
        const duration = 500;
        const elapsed = now - lastCrossing.start;
        if (elapsed >= duration) { lastCrossing = null; return; }
        const alpha = 1 - elapsed / duration;
        const band = tile * 3;
        const lit = `rgba(150, 220, 255, ${alpha})`;
        const dim = 'rgba(150, 220, 255, 0)';
        ctx.save();
        let grad;
        switch (lastCrossing.edge) {
            case 'left':
                grad = ctx.createLinearGradient(0, 0, band, 0);
                grad.addColorStop(0, lit); grad.addColorStop(1, dim);
                ctx.fillStyle = grad;
                ctx.fillRect(0, 0, band, h);
                break;
            case 'right':
                grad = ctx.createLinearGradient(w - band, 0, w, 0);
                grad.addColorStop(0, dim); grad.addColorStop(1, lit);
                ctx.fillStyle = grad;
                ctx.fillRect(w - band, 0, band, h);
                break;
            case 'top':
                grad = ctx.createLinearGradient(0, 0, 0, band);
                grad.addColorStop(0, lit); grad.addColorStop(1, dim);
                ctx.fillStyle = grad;
                ctx.fillRect(0, 0, w, band);
                break;
            case 'bottom':
                grad = ctx.createLinearGradient(0, h - band, 0, h);
                grad.addColorStop(0, dim); grad.addColorStop(1, lit);
                ctx.fillStyle = grad;
                ctx.fillRect(0, h - band, w, band);
                break;
        }
        ctx.restore();
    }

    function brighten(hex) {
        const r = Math.min(255, parseInt(hex.slice(1, 3), 16) + 40);
        const g = Math.min(255, parseInt(hex.slice(3, 5), 16) + 40);
        const b = Math.min(255, parseInt(hex.slice(5, 7), 16) + 40);
        return `rgb(${r},${g},${b})`;
    }

    const keymap = {
        ArrowUp: 'up', ArrowRight: 'right', ArrowDown: 'down', ArrowLeft: 'left',
        KeyW: 'up', KeyD: 'right', KeyS: 'down', KeyA: 'left',
    };

    function sendDir(dir) {
        if (!state.you) return;
        send({ type: 'input', dir });
    }

    document.addEventListener('keydown', (e) => {
        const dir = keymap[e.code];
        if (dir) {
            e.preventDefault();
            sendDir(dir);
        }
    });

    dpad.addEventListener('pointerdown', (e) => {
        const b = e.target.closest('button[data-dir]');
        if (b) {
            e.preventDefault();
            sendDir(b.dataset.dir);
        }
    });

    joinForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const name = nameInput.value.trim() || 'anon';
        lastName = name;
        send({ type: 'join', name });
    });

    connect();
})();
