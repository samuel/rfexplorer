"use strict";

// Same stops as trace.DefaultGradient.
const gradient = [
  [0x00, 0x00, 0x00], [0x00, 0x00, 0x20], [0x00, 0x00, 0x30], [0x00, 0x00, 0x50],
  [0x00, 0x00, 0x91], [0x1e, 0x90, 0xff], [0xff, 0xff, 0x00], [0xfe, 0x6d, 0x16],
  [0xff, 0x00, 0x00], [0xc6, 0x00, 0x00], [0x9f, 0x00, 0x00], [0x75, 0x00, 0x00],
  [0x4a, 0x00, 0x00],
];

function gradientAt(v) {
  if (!(v > 0)) return gradient[0];
  if (v >= 1) return gradient[gradient.length - 1];
  const pos = v * (gradient.length - 1);
  const i = Math.floor(pos), t = pos - i;
  const a = gradient[i], b = gradient[i + 1];
  return [0, 1, 2].map((k) => Math.round(a[k] + (b[k] - a[k]) * t));
}

const $ = (id) => document.getElementById(id);
const spectrum = $("spectrum"), waterfall = $("waterfall");
const sctx = spectrum.getContext("2d"), wctx = waterfall.getContext("2d");

let config = null;
let sweep = null;
let maxHold = null;
let socket = null;

const mhz = (hz) => (hz / 1e6).toFixed(3);

function setStatus(text, isError) {
  const el = $("status");
  el.textContent = text;
  el.classList.toggle("error", !!isError);
}

function resize() {
  const w = Math.max(1, Math.round(spectrum.clientWidth * devicePixelRatio));
  if (spectrum.width !== w) spectrum.width = w;
  draw();
}

function levelY(v, h) {
  const top = config ? config.amp_top : 0, bottom = config ? config.amp_bottom : -120;
  const a = Math.min(Math.max((v - bottom) / (top - bottom), 0), 1);
  return (h - 1) * (1 - a);
}

function stroke(samples, color, width) {
  const w = spectrum.width, h = spectrum.height, n = samples.length;
  sctx.beginPath();
  for (let i = 0; i < n; i++) {
    const x = n > 1 ? (i * (w - 1)) / (n - 1) : 0;
    const y = levelY(samples[i], h);
    if (i === 0) sctx.moveTo(x, y); else sctx.lineTo(x, y);
  }
  sctx.strokeStyle = color;
  sctx.lineWidth = width;
  sctx.stroke();
}

function draw() {
  const w = spectrum.width, h = spectrum.height;
  sctx.fillStyle = "#000";
  sctx.fillRect(0, 0, w, h);
  sctx.strokeStyle = "rgba(128,128,128,0.35)";
  sctx.lineWidth = 1;
  for (let i = 1; i < 10; i++) {
    const x = Math.round((i * w) / 10) + 0.5;
    sctx.beginPath(); sctx.moveTo(x, 0); sctx.lineTo(x, h); sctx.stroke();
  }
  sctx.fillStyle = "#aaa";
  sctx.font = `${11 * devicePixelRatio}px system-ui`;
  for (let i = 0; i <= 5; i++) {
    const y = Math.round((i * (h - 1)) / 5) + 0.5;
    sctx.beginPath(); sctx.moveTo(0, y); sctx.lineTo(w, y); sctx.stroke();
    if (config) {
      const dbm = config.amp_top - (i * (config.amp_top - config.amp_bottom)) / 5;
      sctx.fillText(`${dbm.toFixed(0)} dBm`, 4, Math.min(y + 14 * devicePixelRatio, h - 4));
    }
  }
  if (!sweep) return;
  if (maxHold) stroke(maxHold, "#d03030", 1.5 * devicePixelRatio);
  stroke(sweep.samples, "#8080ff", 2 * devicePixelRatio);

  // Peak marker on the max hold trace.
  const trace = maxHold || sweep.samples;
  let peak = 0;
  for (let i = 1; i < trace.length; i++) if (trace[i] > trace[peak]) peak = i;
  const n = trace.length;
  const x = n > 1 ? (peak * (w - 1)) / (n - 1) : 0;
  const y = levelY(trace[peak], h);
  sctx.fillStyle = "#ffe000";
  sctx.beginPath();
  sctx.moveTo(x, y - 3); sctx.lineTo(x - 7, y - 13); sctx.lineTo(x + 7, y - 13);
  sctx.fill();
  const label = `${mhz(sweep.start + peak * sweep.step)} MHz  ${trace[peak].toFixed(1)} dBm`;
  const tw = sctx.measureText(label).width;
  sctx.fillText(label, Math.min(Math.max(x - tw / 2, 4), w - tw - 4), Math.max(y - 18, 14 * devicePixelRatio));
}

function addWaterfallRow(samples) {
  const n = samples.length;
  if (waterfall.width !== n) {
    waterfall.width = n;
    wctx.clearRect(0, 0, n, waterfall.height);
  }
  // Scroll down one row and draw the newest sweep at the top.
  wctx.drawImage(waterfall, 0, 0, n, waterfall.height - 1, 0, 1, n, waterfall.height - 1);
  const row = wctx.createImageData(n, 1);
  const top = config ? config.amp_top : 0, bottom = config ? config.amp_bottom : -120;
  for (let i = 0; i < n; i++) {
    const c = gradientAt((samples[i] - bottom) / (top - bottom));
    row.data.set([c[0], c[1], c[2], 255], i * 4);
  }
  wctx.putImageData(row, 0, 0);
}

function clearTraces() {
  maxHold = null;
  wctx.clearRect(0, 0, waterfall.width, waterfall.height);
}

function showConfig(c) {
  const changed = !config || config.start !== c.start || config.step !== c.step ||
    config.points !== c.points || config.amp_top !== c.amp_top || config.amp_bottom !== c.amp_bottom;
  config = c;
  if (changed) clearTraces();
  $("f-start").textContent = mhz(c.start);
  $("f-center").textContent = mhz((c.start + c.stop) / 2);
  $("f-stop").textContent = mhz(c.stop);
  if (document.activeElement !== $("start")) $("start").value = mhz(c.start);
  if (document.activeElement !== $("stop")) $("stop").value = mhz(c.stop);
  $("calculator").value = c.calculator;
  const rows = [
    ["Module", c.module], ["Points", c.points], ["Step", `${(c.step / 1e3).toFixed(1)} kHz`],
    ["RBW", `${(c.rbw / 1e3).toFixed(0)} kHz`], ["Amplitude", `${c.amp_bottom} to ${c.amp_top} dBm`],
    ...c.boards.map((b) => [
      `${b.module === c.module ? "▸ " : ""}${b.model}`,
      b.max_freq ? `${mhz(b.min_freq)}–${mhz(b.max_freq)}, span ${mhz(b.max_span)}` : "range unknown",
    ]),
  ];
  const dl = $("config");
  dl.replaceChildren(...rows.flatMap(([k, v]) => {
    const dt = document.createElement("dt"), dd = document.createElement("dd");
    dt.textContent = k; dd.textContent = v;
    return [dt, dd];
  }));
  draw();
}

function handle(msg) {
  switch (msg.type) {
    case "info":
      $("device").textContent = `${msg.main}` + (msg.expansion !== "None" ? ` + ${msg.expansion}` : "") +
        ` · firmware ${msg.firmware}`;
      break;
    case "config":
      showConfig(msg);
      break;
    case "sweep":
      if (sweep && (sweep.start !== msg.start || sweep.step !== msg.step ||
          sweep.samples.length !== msg.samples.length)) clearTraces();
      sweep = msg;
      if (!maxHold || maxHold.length !== msg.samples.length) maxHold = msg.samples.slice();
      else for (let i = 0; i < maxHold.length; i++) maxHold[i] = Math.max(maxHold[i], msg.samples[i]);
      addWaterfallRow(msg.samples);
      draw();
      break;
    case "result":
      setStatus(msg.error ? `${msg.cmd}: ${msg.error}` : "", !!msg.error);
      break;
  }
}

function send(cmd) {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    setStatus("Not connected", true);
    return;
  }
  setStatus(`${cmd.cmd}…`);
  socket.send(JSON.stringify(cmd));
}

function connect() {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  socket = new WebSocket(`${proto}//${location.host}/ws`);
  socket.onopen = () => setStatus("");
  socket.onmessage = (e) => handle(JSON.parse(e.data));
  socket.onclose = (e) => {
    setStatus(`Disconnected${e.reason ? ": " + e.reason : ""}; retrying…`, true);
    setTimeout(connect, 2000);
  };
}

async function loadPresets() {
  const ul = $("presets");
  try {
    const res = await fetch("api/presets");
    if (!res.ok) throw new Error(await res.text());
    const presets = await res.json();
    ul.replaceChildren(...presets.map((p) => {
      const li = document.createElement("li"), b = document.createElement("button");
      b.type = "button";
      b.textContent = `${p.index + 1}. ${p.name}`;
      b.title = `${p.start} – ${p.stop}`;
      b.onclick = () => send({ cmd: "sweep", start: p.start, stop: p.stop, amp_top: p.amp_top, amp_bottom: p.amp_bottom });
      li.append(b);
      return li;
    }));
    if (!presets.length) ul.innerHTML = "<li>None</li>";
  } catch (err) {
    ul.innerHTML = "";
    const li = document.createElement("li");
    li.textContent = `Could not read presets: ${err.message}`;
    ul.append(li);
  }
}

$("range").onsubmit = (e) => {
  e.preventDefault();
  send({ cmd: "sweep", start: `${$("start").value}MHz`, stop: `${$("stop").value}MHz` });
};
$("calculator").onchange = (e) => send({ cmd: "calculator", mode: e.target.value });
$("hold").onclick = () => send({ cmd: "hold" });
$("resume").onclick = () => send({ cmd: "resume" });
$("reset").onclick = () => { maxHold = null; draw(); };
window.addEventListener("resize", resize);

resize();
connect();
loadPresets();
