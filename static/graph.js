(function () {
  const q = (s) => document.querySelector(s);
  const chartEl = q('#chart');
  const fromEl = q('#from');
  const toEl = q('#to');
  const applyBtn = q('#apply');
  const resetBtn = q('#reset');

  const tDaily = q('#toggle-daily');
  const tCum = q('#toggle-cum');
  const tBudget = q('#toggle-budget');
  const tSaldo = q('#toggle-saldo');

  let u = null;
  let raw = null;

  function fmtDate(d) {
    const yy = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    const dd = String(d.getDate()).padStart(2, '0');
    return `${yy}-${mm}-${dd}`;
  }

  function setDefaultsWindow(from, to) {
    fromEl.value = fmtDate(from);
    toEl.value = fmtDate(to);
  }

  function fetchData() {
    const params = new URLSearchParams();
    if (fromEl.value) params.set('from', fromEl.value);
    if (toEl.value) params.set('to', toEl.value);

    return fetch(`/expenses/graph-data?${params}`).then(r => r.json());
  }

  function toUplotSeries(data) {
    // x in ms timestamps
    const x = data.points.map(p => new Date(p.date + 'T00:00:00Z').getTime());
    const daily = data.points.map(p => p.spend);
    const cum = data.points.map(p => p.cumulative);
    const budget = data.points.map(p => p.budget_cum);
    const saldo = data.points.map(p => p.saldo);
    return { x, daily, cum, budget, saldo, meta: data };
  }

  function buildChart(series) {
    if (u) {
      u.destroy();
      u = null;
    }

    const opts = {
      width: chartEl.clientWidth,
      height: chartEl.clientHeight,
      scales: { x: { time: true } },
      series: [
        { label: 'Date' },
        { label: 'Daily', stroke: '#4e79a7', width: 2, points: { show: false } },
        { label: 'Cumulative', stroke: '#f28e2b', width: 2, points: { show: false } },
        { label: 'Budget', stroke: '#76b7b2', width: 2, dash: [6, 6], points: { show: false } },
        { label: 'Saldo', stroke: '#e15759', width: 2, points: { show: false } },
      ],
      legend: { show: true },
      axes: [
        { },
        { values: (u, vals) => vals.map(v => v.toFixed(0)) },
      ],
      hooks: {
        ready: [
          (u) => {
            // Drag to zoom
            let sx = 0, sy = 0, ex = 0, ey = 0, selecting = false;
            const over = u.over;
            over.addEventListener('mousedown', (e) => {
              selecting = true; sx = e.clientX; sy = e.clientY;
            });
            window.addEventListener('mousemove', (e) => { if (selecting) { ex = e.clientX; ey = e.clientY; }});
            window.addEventListener('mouseup', (e) => {
              if (!selecting) return;
              selecting = false;
              const l = Math.min(sx, ex), r = Math.max(sx, ex);
              if (Math.abs(r - l) < 8) return;
              const left = u.posToVal(l, 'x');
              const right = u.posToVal(r, 'x');
              u.setScale('x', { min: left, max: right });
            });

            // Wheel to zoom
            over.addEventListener('wheel', (e) => {
              e.preventDefault();
              const factor = e.deltaY < 0 ? 0.9 : 1.1;
              const [min, max] = u.getScale('x');
              const x = u.posToVal(e.clientX, 'x');
              const newMin = x - (x - min) * factor;
              const newMax = x + (max - x) * factor;
              u.setScale('x', { min: newMin, max: newMax });
            }, { passive: false });
          }
        ],
        setSize: [
          (u) => {
            // keep y axis integers
          }
        ]
      }
    };

    const data = [
      series.x,
      series.daily,
      series.cum,
      series.budget,
      series.saldo,
    ];

    u = new uPlot(opts, data, chartEl);

    // Visibility toggles
    const updateVis = () => {
      u.setSeries(1, { show: tDaily.checked });
      u.setSeries(2, { show: tCum.checked });
      u.setSeries(3, { show: tBudget.checked });
      u.setSeries(4, { show: tSaldo.checked });
    };
    [tDaily, tCum, tBudget, tSaldo].forEach(cb => cb.addEventListener('change', updateVis));
    updateVis();

    // Resize handler
    const onResize = () => {
      u.setSize({ width: chartEl.clientWidth, height: chartEl.clientHeight });
    };
    window.addEventListener('resize', onResize);
  }

  function init() {
    // Default: last 90 days
    const now = new Date();
    const from = new Date(now.getTime() - 90 * 86400000);
    setDefaultsWindow(from, now);

    applyBtn.addEventListener('click', () => {
      fetchData().then(data => {
        raw = data;
        const s = toUplotSeries(data);
        buildChart(s);
      });
    });

    resetBtn.addEventListener('click', () => {
      if (!raw) return;
      const first = raw.points[0]?.date;
      const last = raw.points[raw.points.length - 1]?.date;
      if (!first || !last) return;
      fromEl.value = first;
      toEl.value = last;
      fetchData().then(data => {
        raw = data;
        const s = toUplotSeries(data);
        buildChart(s);
      });
    });

    // Initial load
    fetchData().then(data => {
      raw = data;
      // Set actual window from response if empty inputs
      if (!fromEl.value || !toEl.value) {
        fromEl.value = data.from;
        toEl.value = data.to;
      }
      const s = toUplotSeries(data);
      buildChart(s);
    });
  }

  document.addEventListener('DOMContentLoaded', init);
})();


