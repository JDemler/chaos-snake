(() => {
    const mainCanvas = document.getElementById('main');
    const mainCtx = mainCanvas.getContext('2d');
    const thumbsContainer = document.getElementById('thumbs');
    const statusEl = document.getElementById('status');
    const lengthEl = document.getElementById('length');
    const fieldCountEl = document.getElementById('fieldcount');
    const playerCountEl = document.getElementById('playercount');
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
        you: '',
    };

    // FieldID -> {canvas, ctx, label}
    const thumbs = new Map();

    let ws = null;
    let reconnectDelay = 500;
    let lastName = '';

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
            state.you = '';
            joinDiv.hidden = !!lastName;
            dpad.hidden = true;
            clearThumbs();
            renderAll();
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
            state.fields.set(f.id, { id: f.id, pellet: f.pellet });
        }
        state.snakes.clear();
        for (const s of msg.snakes) state.snakes.set(s.id, cloneSnake(s));

        if (state.you) {
            joinDiv.hidden = true;
            dpad.hidden = false;
        } else {
            joinDiv.hidden = false;
            dpad.hidden = true;
        }
        rebuildThumbs();
        updateLength();
        renderAll();
    }

    function applyDelta(msg) {
        if (msg.field_joins) {
            for (const f of msg.field_joins) {
                state.fields.set(f.id, { id: f.id, pellet: f.pellet });
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
                if (f) f.pellet = p.p;
            }
        }
        rebuildThumbs();
        updateLength();
        renderAll();
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

    function renderAll() {
        const mainID = currentFieldID();
        if (mainID) {
            renderField(mainCtx, mainID, MAIN_TILE, true);
        } else {
            mainCtx.fillStyle = '#000';
            mainCtx.fillRect(0, 0, mainCanvas.width, mainCanvas.height);
        }
        for (const [id, entry] of thumbs) {
            renderField(entry.ctx, id, THUMB_TILE, false);
        }
    }

    function renderField(ctx, fieldID, tile, drawGrid) {
        const w = state.size.w * tile;
        const h = state.size.h * tile;
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, w, h);

        if (drawGrid) {
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

        // pellet
        const f = state.fields.get(fieldID);
        if (f && f.pellet && f.pellet[0] >= 0) {
            ctx.fillStyle = '#e22';
            const inset = Math.max(1, Math.floor(tile * 0.15));
            ctx.fillRect(
                f.pellet[0] * tile + inset,
                f.pellet[1] * tile + inset,
                tile - inset * 2,
                tile - inset * 2,
            );
        }

        // snakes
        for (const s of state.snakes.values()) {
            for (let i = 0; i < s.body.length; i++) {
                const t = s.body[i];
                if (t.f !== fieldID) continue;
                ctx.fillStyle = i === 0 ? brighten(s.color) : s.color;
                const inset = tile >= 8 ? 1 : 0;
                ctx.fillRect(
                    t.p[0] * tile + inset,
                    t.p[1] * tile + inset,
                    tile - inset * 2,
                    tile - inset * 2,
                );
            }
            if (s.id === state.you && tile >= 8) {
                const head = s.body[0];
                if (head && head.f === fieldID) {
                    ctx.strokeStyle = '#fff';
                    ctx.lineWidth = 2;
                    ctx.strokeRect(head.p[0] * tile + 1, head.p[1] * tile + 1, tile - 2, tile - 2);
                }
            }
        }
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
