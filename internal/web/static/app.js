// F1 Dashboard — app.js
// Talks to the F1 API and renders Chart.js charts.

const API = ''  // same origin; set to 'http://localhost:8080' if needed

const COLORS = [
  '#e10600','#0090ff','#00d2be','#ff8c00','#39b54a',
  '#006f62','#2b4562','#f596c8','#ff7f0f','#fff500',
  '#0082fa','#dc0000','#005aff','#52e252','#b6babd',
  '#fe86bc','#469bff','#9b0000','#4ade80','#00a0dd',
]

const CHART_DEFAULTS = {
  responsive: true,
  plugins: { legend: { labels: { color: '#ccc', boxWidth: 12, padding: 16 } } },
  scales: {
    x: { ticks: { color: '#9ca3af' }, grid: { color: '#2e2e3e' } },
    y: { ticks: { color: '#9ca3af' }, grid: { color: '#2e2e3e' } },
  },
}

// Active Chart.js instances — destroyed before re-render.
const charts = {}

function destroyChart(id) {
  if (charts[id]) { charts[id].destroy(); delete charts[id] }
}

async function apiFetch(path) {
  const res = await fetch(API + path)
  if (!res.ok) throw new Error(`API error ${res.status}: ${path}`)
  const json = await res.json()
  return json.data ?? json
}

// ── Tab navigation ────────────────────────────────────────────────────────────

function showTab(name) {
  document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'))
  document.getElementById('tab-' + name).classList.add('active')
  event.target.classList.add('active')
  if (name === 'live') startLivePolling()
  else stopLivePolling()
}

// ── Championship Points Progression ──────────────────────────────────────────

async function loadPointsProgression() {
  const season = document.getElementById('pp-season').value
  try {
    const d = await apiFetch(`/stats/points-progression?season=${season}`)
    renderProgressionChart(d)
  } catch (e) { alert(e.message) }
}

function renderProgressionChart(d) {
  destroyChart('progression')
  const ctx = document.getElementById('chart-progression').getContext('2d')
  // Show top 10 drivers by final points.
  const sorted = [...d.drivers].sort((a, b) => {
    const ap = a.points[a.points.length - 1] ?? 0
    const bp = b.points[b.points.length - 1] ?? 0
    return bp - ap
  }).slice(0, 10)

  charts['progression'] = new Chart(ctx, {
    type: 'line',
    data: {
      labels: d.race_names.map((n, i) => `R${d.rounds[i]}: ${n.replace(' Grand Prix','').replace(' GP','')}`),
      datasets: sorted.map((drv, i) => ({
        label: drv.name,
        data: drv.points,
        borderColor: COLORS[i],
        backgroundColor: COLORS[i] + '22',
        borderWidth: 2,
        pointRadius: 3,
        tension: 0.3,
        fill: false,
      })),
    },
    options: {
      ...CHART_DEFAULTS,
      plugins: {
        ...CHART_DEFAULTS.plugins,
        tooltip: { mode: 'index', intersect: false },
      },
      scales: {
        x: { ...CHART_DEFAULTS.scales.x, ticks: { color: '#9ca3af', maxRotation: 45 } },
        y: { ...CHART_DEFAULTS.scales.y, title: { display: true, text: 'Points', color: '#9ca3af' } },
      },
    },
  })
}

// ── Constructor Dominance ─────────────────────────────────────────────────────

async function loadConstructorWins() {
  const from = document.getElementById('cw-from').value
  const to   = document.getElementById('cw-to').value
  try {
    const d = await apiFetch(`/stats/constructor-wins?from=${from}&to=${to}`)
    renderConstructorWins(d)
  } catch (e) { alert(e.message) }
}

function renderConstructorWins(d) {
  destroyChart('constructor-wins')
  const ctx = document.getElementById('chart-constructor-wins').getContext('2d')

  // Sort constructors by total wins descending.
  const sorted = [...d.constructors].sort((a, b) => {
    return b.wins.reduce((s, v) => s + v, 0) - a.wins.reduce((s, v) => s + v, 0)
  })

  charts['constructor-wins'] = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: d.years,
      datasets: sorted.map((c, i) => ({
        label: c.name,
        data: c.wins,
        backgroundColor: COLORS[i] + 'cc',
        stack: 'wins',
      })),
    },
    options: {
      ...CHART_DEFAULTS,
      scales: {
        x: { ...CHART_DEFAULTS.scales.x, stacked: true },
        y: { ...CHART_DEFAULTS.scales.y, stacked: true,
             title: { display: true, text: 'Race Wins', color: '#9ca3af' } },
      },
    },
  })
}

// ── Driver Career Arc ─────────────────────────────────────────────────────────

async function loadCareer() {
  const ref = document.getElementById('career-ref').value.trim()
  if (!ref) return
  try {
    const d = await apiFetch(`/drivers/${ref}/career`)
    document.getElementById('career-name').textContent = d.driver.name
    renderCareerChart(d.career)
  } catch (e) { alert(e.message) }
}

function renderCareerChart(career) {
  destroyChart('career')
  const ctx = document.getElementById('chart-career').getContext('2d')
  const seasons = career.map(s => s.season)

  charts['career'] = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: seasons,
      datasets: [
        { label: 'Wins',    data: career.map(s => s.wins),    backgroundColor: '#e10600cc', yAxisID: 'y' },
        { label: 'Podiums', data: career.map(s => s.podiums), backgroundColor: '#0090ffcc', yAxisID: 'y' },
        { label: 'Points',  data: career.map(s => s.points),  type: 'line',
          borderColor: '#00d2be', backgroundColor: 'transparent',
          borderWidth: 2, pointRadius: 3, tension: 0.3, yAxisID: 'y2' },
      ],
    },
    options: {
      ...CHART_DEFAULTS,
      scales: {
        x: CHART_DEFAULTS.scales.x,
        y:  { ...CHART_DEFAULTS.scales.y, position: 'left',  title: { display: true, text: 'Count', color: '#9ca3af' } },
        y2: { ...CHART_DEFAULTS.scales.y, position: 'right', title: { display: true, text: 'Points', color: '#9ca3af' },
              grid: { drawOnChartArea: false } },
      },
    },
  })
}

// ── Head-to-Head ──────────────────────────────────────────────────────────────

async function loadH2H() {
  const d1  = document.getElementById('h2h-d1').value.trim()
  const d2  = document.getElementById('h2h-d2').value.trim()
  const szn = document.getElementById('h2h-season').value
  if (!d1 || !d2) return
  try {
    const path = `/stats/compare?d1=${d1}&d2=${d2}` + (szn ? `&season=${szn}` : '')
    const data = await apiFetch(path)
    renderH2H(data)
  } catch (e) { alert(e.message) }
}

function renderH2H(data) {
  const { driver1: d1, driver2: d2, races_together } = data
  document.getElementById('h2h-content').classList.remove('hidden')
  document.getElementById('h2h-name1').textContent = d1.name
  document.getElementById('h2h-name2').textContent = d2.name

  const metrics = [
    { label: 'Races Ahead',    v1: d1.ahead,        v2: d2.ahead },
    { label: 'Wins',           v1: d1.wins,          v2: d2.wins },
    { label: 'Podiums',        v1: d1.podiums,       v2: d2.podiums },
    { label: 'Avg Position',   v1: d1.avg_position,  v2: d2.avg_position, lowerBetter: true },
    { label: 'Points',         v1: d1.points,        v2: d2.points },
    { label: 'Poles',          v1: d1.poles,         v2: d2.poles },
  ]

  document.getElementById('h2h-stats').innerHTML = metrics.map(m => {
    const d1Wins = m.lowerBetter ? m.v1 < m.v2 : m.v1 > m.v2
    const d2Wins = m.lowerBetter ? m.v2 < m.v1 : m.v2 > m.v1
    return `
      <div class="stat-box">
        <div class="flex justify-between items-end gap-2">
          <span class="stat-value" style="color:${d1Wins ? '#e10600' : '#9ca3af'}">${m.v1}</span>
          <span class="stat-label">${m.label}</span>
          <span class="stat-value" style="color:${d2Wins ? '#0090ff' : '#9ca3af'}">${m.v2}</span>
        </div>
      </div>`
  }).join('')

  destroyChart('h2h')
  const ctx = document.getElementById('chart-h2h').getContext('2d')
  charts['h2h'] = new Chart(ctx, {
    type: 'radar',
    data: {
      labels: ['Wins', 'Podiums', 'Poles', 'Points (÷10)', 'Races Ahead'],
      datasets: [
        { label: d1.name,
          data: [d1.wins, d1.podiums, d1.poles, d1.points / 10, d1.ahead],
          borderColor: '#e10600', backgroundColor: '#e1060033', borderWidth: 2 },
        { label: d2.name,
          data: [d2.wins, d2.podiums, d2.poles, d2.points / 10, d2.ahead],
          borderColor: '#0090ff', backgroundColor: '#0090ff33', borderWidth: 2 },
      ],
    },
    options: {
      plugins: { legend: { labels: { color: '#ccc' } } },
      scales: { r: { ticks: { color: '#9ca3af', backdropColor: 'transparent' },
                     grid: { color: '#2e2e3e' }, pointLabels: { color: '#ccc' } } },
    },
  })
}

// ── Circuit Form Guide ────────────────────────────────────────────────────────

async function loadCircuitForm() {
  const ref = document.getElementById('circuit-ref').value.trim()
  if (!ref) return
  try {
    const d = await apiFetch(`/circuits/${ref}/form`)
    document.getElementById('circuit-name').textContent = d.circuit.name
    renderCircuitTable(d.form ?? [])
  } catch (e) { alert(e.message) }
}

function renderCircuitTable(form) {
  if (!form.length) {
    document.getElementById('circuit-table').innerHTML = '<p class="text-gray-500 text-sm">No data yet — run the seeder first.</p>'
    return
  }
  document.getElementById('circuit-table').innerHTML = `
    <table>
      <thead><tr>
        <th>#</th><th>Driver</th><th>Races</th><th>Wins</th><th>Podiums</th><th>Avg Finish</th>
      </tr></thead>
      <tbody>${form.map((r, i) => `
        <tr class="timing-row">
          <td class="text-gray-500">${i + 1}</td>
          <td class="font-semibold">${r.name}</td>
          <td>${r.races}</td>
          <td class="font-bold" style="color:${r.wins > 0 ? '#e10600' : '#fff'}">${r.wins}</td>
          <td>${r.podiums}</td>
          <td>${r.avg_position}</td>
        </tr>`).join('')}
      </tbody>
    </table>`
}

// ── Grid vs Finish Heatmap ────────────────────────────────────────────────────

async function loadGridVsFinish() {
  const season = document.getElementById('gvf-season').value
  try {
    const path = season ? `/stats/grid-vs-finish?season=${season}` : '/stats/grid-vs-finish'
    const d = await apiFetch(path)
    renderHeatmap(d.data ?? [])
  } catch (e) { alert(e.message) }
}

function renderHeatmap(data) {
  if (!data.length) {
    document.getElementById('heatmap-container').innerHTML = '<p class="text-gray-500 text-sm">No data.</p>'
    return
  }

  const maxPos = 20
  // Build grid[gridPos][finishPos] = count
  const grid = {}
  let maxCount = 1
  data.forEach(({ grid: g, position: p, count: c }) => {
    if (g > maxPos || p > maxPos) return
    if (!grid[g]) grid[g] = {}
    grid[g][p] = c
    if (c > maxCount) maxCount = c
  })

  const positions = Array.from({ length: maxPos }, (_, i) => i + 1)

  let html = '<div style="overflow-x:auto"><table style="border-collapse:separate;border-spacing:2px">'
  html += '<thead><tr><th style="color:#9ca3af;font-size:10px;padding:2px 6px">Start ↓ / Finish →</th>'
  positions.forEach(p => { html += `<th style="color:#9ca3af;font-size:10px;padding:2px 4px">${p}</th>` })
  html += '</tr></thead><tbody>'

  positions.forEach(g => {
    html += `<tr><td style="color:#9ca3af;font-size:10px;padding:2px 6px;font-weight:700">P${g}</td>`
    positions.forEach(p => {
      const count = (grid[g] && grid[g][p]) || 0
      const intensity = count / maxCount
      const isDiag = g === p
      const bg = count === 0
        ? '#1a1a2a'
        : isDiag
          ? `rgba(0, 210, 190, ${0.15 + intensity * 0.85})`
          : `rgba(225, 6, 0, ${0.1 + intensity * 0.9})`
      const textColor = intensity > 0.5 ? '#fff' : count > 0 ? '#ccc' : '#333'
      html += `<td style="background:${bg};width:28px;height:28px;text-align:center;font-size:9px;font-weight:700;color:${textColor};border-radius:3px">${count || ''}</td>`
    })
    html += '</tr>'
  })
  html += '</tbody></table></div>'
  document.getElementById('heatmap-container').innerHTML = html
}

// ── Live: Track Map ───────────────────────────────────────────────────────────

let liveTimer = null
let gapChart = null
let gapHistory = {}   // ref → [gap, gap, ...]

function startLivePolling() {
  document.getElementById('live-badge').classList.remove('hidden')
  pollLive()
  liveTimer = setInterval(pollLive, 2000)
}

function stopLivePolling() {
  document.getElementById('live-badge').classList.add('hidden')
  if (liveTimer) { clearInterval(liveTimer); liveTimer = null }
}

async function pollLive() {
  try {
    const [pos, timing, rc] = await Promise.all([
      apiFetch('/live/positions'),
      apiFetch('/live/timing'),
      apiFetch('/live/race-control'),
    ])
    drawTrackMap(pos)
    renderTimingTower(timing)
    updateGapChart(timing)
    renderRaceControl(rc)
  } catch (_) {
    // No live session — show placeholder.
    renderTimingTower(null)
  }
}

function drawTrackMap(posData) {
  const canvas = document.getElementById('track-canvas')
  const ctx = canvas.getContext('2d')
  const W = canvas.width, H = canvas.height

  // Position.z entries: posData.Position = [{Entries: { "1": {X,Y,Status}, ... }}]
  const entries = posData?.Position?.[posData.Position.length - 1]?.Entries
  if (!entries) return

  ctx.clearRect(0, 0, W, H)

  const positions = Object.entries(entries)
    .map(([num, v]) => ({ num, x: v.X, y: v.Y, status: v.Status }))
    .filter(p => p.x !== undefined)

  if (!positions.length) return

  // Normalise X/Y to canvas.
  const xs = positions.map(p => p.x), ys = positions.map(p => p.y)
  const minX = Math.min(...xs), maxX = Math.max(...xs)
  const minY = Math.min(...ys), maxY = Math.max(...ys)
  const pad = 40
  const scaleX = (W - pad * 2) / (maxX - minX || 1)
  const scaleY = (H - pad * 2) / (maxY - minY || 1)
  const scale = Math.min(scaleX, scaleY)

  positions.forEach((p, i) => {
    const cx = pad + (p.x - minX) * scale
    const cy = H - pad - (p.y - minY) * scale

    // Car dot.
    ctx.beginPath()
    ctx.arc(cx, cy, 7, 0, Math.PI * 2)
    ctx.fillStyle = COLORS[i % COLORS.length]
    ctx.fill()

    // Car number label.
    ctx.fillStyle = '#fff'
    ctx.font = 'bold 8px sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText(p.num, cx, cy + 3)
  })
}

function renderTimingTower(timing) {
  const el = document.getElementById('timing-tower')
  const lines = timing?.Lines
  if (!lines) {
    el.innerHTML = '<p class="text-gray-500 text-sm">No active session.</p>'
    return
  }

  const drivers = Object.entries(lines)
    .map(([num, d]) => ({ num, ...d }))
    .filter(d => d.Position)
    .sort((a, b) => parseInt(a.Position) - parseInt(b.Position))

  el.innerHTML = `
    <table>
      <thead><tr>
        <th>Pos</th><th>Driver</th><th>Gap</th><th>Last Lap</th><th>Pits</th>
      </tr></thead>
      <tbody>${drivers.map(d => `
        <tr class="timing-row">
          <td class="font-bold">${d.Position}</td>
          <td>${d.RacingNumber ?? d.num}</td>
          <td class="text-gray-400">${d.GapToLeader ?? '—'}</td>
          <td>${d.LastLapTime?.Value ?? '—'}</td>
          <td>${d.NumberOfPitStops ?? 0}</td>
        </tr>`).join('')}
      </tbody>
    </table>`
}

function updateGapChart(timing) {
  const lines = timing?.Lines
  if (!lines) return

  const drivers = Object.entries(lines)
    .map(([num, d]) => ({ num, ...d }))
    .filter(d => d.Position && d.GapToLeader)
    .sort((a, b) => parseInt(a.Position) - parseInt(b.Position))
    .slice(0, 10)

  const labels = Object.keys(gapHistory)
  if (labels.length > 30) {
    Object.keys(gapHistory).forEach(k => gapHistory[k].shift())
  }

  drivers.forEach(d => {
    const gap = parseFloat(d.GapToLeader?.replace('+', '')) || 0
    if (!gapHistory[d.num]) gapHistory[d.num] = []
    gapHistory[d.num].push(gap)
  })

  const ticks = gapHistory[drivers[0]?.num]?.length || 0
  const tickLabels = Array.from({ length: ticks }, (_, i) => i + 1)

  if (!gapChart) {
    const ctx = document.getElementById('chart-gap').getContext('2d')
    gapChart = new Chart(ctx, {
      type: 'line',
      data: { labels: tickLabels, datasets: [] },
      options: {
        ...CHART_DEFAULTS,
        animation: false,
        scales: {
          x: CHART_DEFAULTS.scales.x,
          y: { ...CHART_DEFAULTS.scales.y, reverse: false,
               title: { display: true, text: 'Gap to Leader (s)', color: '#9ca3af' } },
        },
      },
    })
  }

  gapChart.data.labels = tickLabels
  gapChart.data.datasets = drivers.map((d, i) => ({
    label: `#${d.num}`,
    data: gapHistory[d.num] ?? [],
    borderColor: COLORS[i],
    backgroundColor: 'transparent',
    borderWidth: 2,
    pointRadius: 0,
    tension: 0.3,
  }))
  gapChart.update('none')
}

function renderRaceControl(data) {
  const el = document.getElementById('race-control')
  const messages = data?.Messages
  if (!messages || !Object.keys(messages).length) {
    el.innerHTML = '<p class="text-gray-500 text-sm">No messages.</p>'
    return
  }

  const msgs = Object.values(messages)
    .sort((a, b) => b.Utc > a.Utc ? 1 : -1)
    .slice(0, 10)

  const flagColor = flag => ({
    'GREEN FLAG': '#4ade80', 'YELLOW FLAG': '#facc15', 'RED FLAG': '#e10600',
    'SAFETY CAR': '#facc15', 'VIRTUAL SAFETY CAR': '#fb923c', 'CHEQUERED FLAG': '#fff',
  }[flag] ?? '#9ca3af')

  el.innerHTML = msgs.map(m => `
    <div class="flex items-start gap-3 p-3 rounded-lg" style="background:#1e1e2e;border:1px solid #2e2e3e">
      <div class="w-2 h-2 rounded-full mt-1.5 flex-shrink-0"
           style="background:${flagColor(m.Flag ?? '')}"></div>
      <div>
        <div class="font-semibold text-sm">${m.Message ?? ''}</div>
        <div class="text-xs text-gray-500 mt-0.5">${m.Utc ? new Date(m.Utc).toLocaleTimeString() : ''}</div>
      </div>
    </div>`).join('')
}

// ── Init ──────────────────────────────────────────────────────────────────────

// Auto-load the season chart on page load.
window.addEventListener('DOMContentLoaded', () => {
  loadPointsProgression()
  loadConstructorWins()
})
