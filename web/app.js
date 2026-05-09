(() => {
    const canvas = document.getElementById('canvas');
    const ctx = canvas.getContext('2d');
    const statusEl = document.getElementById('status');
    const lengthEl = document.getElementById('length');
    const joinDiv = document.getElementById('join');
    const joinForm = document.getElementById('joinForm');
    const nameInput = document.getElementById('name');
    const dpad = document.getElementById('dpad');

    const TILE = 20;

    const state = {
        field: { w: 30, h: 30, tick_hz: 10 },
        snakes: new Map(),
        pellet: null,
        you: '',
    };

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
            state.snakes.clear();
            state.pellet = null;
            state.you = '';
            joinDiv.hidden = !!lastName;
            dpad.hidden = true;
            render();
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
        state.field = msg.field;
        state.you = msg.you || '';
        state.pellet = msg.pellet;
        state.snakes.clear();
        for (const s of msg.snakes) state.snakes.set(s.id, cloneSnake(s));
        resizeCanvas();
        if (state.you) {
            joinDiv.hidden = true;
            dpad.hidden = false;
        } else {
            joinDiv.hidden = false;
            dpad.hidden = true;
        }
        updateLength();
        render();
    }

    function applyDelta(msg) {
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
                    s.body = [m.head];
                } else {
                    s.body.unshift(m.head);
                    if (!m.grew) s.body.pop();
                }
            }
        }
        if (msg.pellet) state.pellet = msg.pellet;
        updateLength();
        render();
    }

    function cloneSnake(s) {
        return {
            id: s.id,
            name: s.name,
            color: s.color,
            body: s.body.map(p => [p[0], p[1]]),
            dir: s.dir,
        };
    }

    function updateLength() {
        if (!state.you) { lengthEl.textContent = ''; return; }
        const me = state.snakes.get(state.you);
        lengthEl.textContent = me ? `length: ${me.body.length}` : '';
    }

    function resizeCanvas() {
        canvas.width = state.field.w * TILE;
        canvas.height = state.field.h * TILE;
    }

    function render() {
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        // grid
        ctx.strokeStyle = '#1a1a1a';
        ctx.lineWidth = 1;
        for (let x = 0; x <= state.field.w; x++) {
            ctx.beginPath();
            ctx.moveTo(x * TILE + 0.5, 0);
            ctx.lineTo(x * TILE + 0.5, state.field.h * TILE);
            ctx.stroke();
        }
        for (let y = 0; y <= state.field.h; y++) {
            ctx.beginPath();
            ctx.moveTo(0, y * TILE + 0.5);
            ctx.lineTo(state.field.w * TILE, y * TILE + 0.5);
            ctx.stroke();
        }

        // pellet
        if (state.pellet) {
            ctx.fillStyle = '#e22';
            ctx.fillRect(state.pellet[0] * TILE + 3, state.pellet[1] * TILE + 3, TILE - 6, TILE - 6);
        }

        // snakes
        for (const s of state.snakes.values()) {
            for (let i = 0; i < s.body.length; i++) {
                const [x, y] = s.body[i];
                ctx.fillStyle = i === 0 ? brighten(s.color) : s.color;
                ctx.fillRect(x * TILE + 1, y * TILE + 1, TILE - 2, TILE - 2);
            }
            if (s.id === state.you && s.body.length > 0) {
                const [x, y] = s.body[0];
                ctx.strokeStyle = '#fff';
                ctx.lineWidth = 2;
                ctx.strokeRect(x * TILE + 1, y * TILE + 1, TILE - 2, TILE - 2);
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
