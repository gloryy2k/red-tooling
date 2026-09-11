package server

import (
	"html"
	"net/http"
	"strings"
)

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/login" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(loginPage()))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardPage(html.EscapeString(s.EngName))))
}

func loginPage() string {
	return `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>RT — Login</title>
<style>
:root{--bg:#0a0a1a;--surface:#141428;--border:#2a2a4a;--text:#e8e8f0;--muted:#888;--accent:#e94560;--accent-hover:#c73650}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;background:var(--bg);color:var(--text);display:flex;justify-content:center;align-items:center;min-height:100vh}
.box{background:var(--surface);padding:2.5rem;border-radius:16px;width:380px;border:1px solid var(--border)}
h1{color:var(--accent);margin:0 0 .5rem;font-size:1.4rem;font-weight:600}
p{color:var(--muted);font-size:.85rem;margin-bottom:1.5rem}
input{width:100%;padding:12px 14px;background:var(--bg);border:1px solid var(--border);color:var(--text);border-radius:8px;font-size:.9rem;margin-bottom:1rem;outline:none;transition:border .15s}
input:focus{border-color:var(--accent)}
button{width:100%;padding:12px;background:var(--accent);color:#fff;border:none;border-radius:8px;font-size:.9rem;cursor:pointer;font-weight:500;transition:background .15s}
button:hover{background:var(--accent-hover)}
.err{color:var(--accent);text-align:center;margin-top:.75rem;font-size:.85rem;display:none}
</style></head><body>
<div class="box">
<h1>RT Dashboard</h1>
<p>Enter your API key to access the dashboard</p>
<input type="password" id="key" placeholder="API Key (rt_key_...)" autofocus>
<button onclick="login()">Sign in</button>
<div class="err" id="err">Invalid API key</div>
</div>
<script>
function login(){
  const key=document.getElementById('key').value;
  if(!key)return;
  fetch('/api/overview',{headers:{'X-API-Key':key}}).then(r=>{
    if(r.ok){localStorage.setItem('rt_api_key',key);location.href='/';}
    else{document.getElementById('err').style.display='block';}
  });
}
document.getElementById('key').addEventListener('keypress',e=>{if(e.key==='Enter')login()});
</script></body></html>`
}

func dashboardPage(engName string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>RT — ` + engName + `</title>
<style>
:root{
  --bg:#0a0a1a;--surface:#141428;--surface-2:#1c1c35;--surface-3:#242445;
  --border:#2a2a4a;--border-hover:#3a3a6a;--text:#e8e8f0;--text-2:#b0b0c8;--muted:#666680;
  --accent:#e94560;--accent-bg:rgba(233,69,96,.12);--accent-hover:#c73650;
  --green:#22c55e;--green-bg:rgba(34,197,94,.12);
  --yellow:#eab308;--yellow-bg:rgba(234,179,8,.12);
  --blue:#3b82f6;--blue-bg:rgba(59,130,246,.12);
  --red:#ef4444;--red-bg:rgba(239,68,68,.12);
  --teal:#14b8a6;--teal-bg:rgba(20,184,166,.12);
  --purple:#a78bfa;--purple-bg:rgba(167,139,250,.12);
  --orange:#f97316;--orange-bg:rgba(249,115,22,.12);
  --radius:8px;--font-mono:'SF Mono','Cascadia Code','Fira Code',monospace;
}
[data-theme="light"]{
  --bg:#f5f5f8;--surface:#ffffff;--surface-2:#f0f0f5;--surface-3:#e8e8f0;
  --border:#d0d0dd;--border-hover:#b0b0c0;--text:#1a1a2e;--text-2:#4a4a60;--muted:#888899;
  --accent:#d6336c;--accent-bg:rgba(214,51,108,.08);--accent-hover:#b82d5e;
  --green:#16a34a;--green-bg:rgba(22,163,74,.08);
  --yellow:#ca8a04;--yellow-bg:rgba(202,138,4,.08);
  --blue:#2563eb;--blue-bg:rgba(37,99,235,.08);
  --red:#dc2626;--red-bg:rgba(220,38,38,.08);
  --teal:#0d9488;--teal-bg:rgba(13,148,136,.08);
  --purple:#7c3aed;--purple-bg:rgba(124,58,237,.08);
  --orange:#ea580c;--orange-bg:rgba(234,88,12,.08);
}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;background:var(--bg);color:var(--text);line-height:1.5;font-size:13px}
::-webkit-scrollbar{width:6px;height:6px}
::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:var(--border);border-radius:3px}

/* === SHELL LAYOUT === */
.shell{display:grid;grid-template-columns:220px minmax(0,1fr);grid-template-rows:48px minmax(0,1fr);height:100vh;overflow:hidden}
.topbar{grid-column:1/-1;display:flex;align-items:center;gap:12px;padding:0 16px;background:var(--surface);border-bottom:1px solid var(--border);z-index:20}
.topbar-brand{display:flex;align-items:center;gap:8px;font-weight:600;font-size:14px}
.topbar-brand svg{color:var(--accent)}
.topbar-eng{color:var(--text-2);font-size:12px;padding:2px 10px;background:var(--surface-2);border-radius:12px}
.topbar-status{font-size:11px;padding:2px 8px;border-radius:10px;font-weight:500}
.topbar-status.ok{background:var(--green-bg);color:var(--green)}
.topbar-status.warn{background:var(--yellow-bg);color:var(--yellow)}
.topbar-right{margin-left:auto;display:flex;align-items:center;gap:12px}
.topbar-search{display:flex;align-items:center;gap:6px;padding:5px 12px;background:var(--surface-2);border:1px solid var(--border);border-radius:var(--radius);color:var(--muted);font-size:12px;cursor:pointer;min-width:180px;transition:border .15s}
.topbar-search:hover{border-color:var(--border-hover)}
.topbar-btn{display:flex;align-items:center;gap:4px;padding:5px 10px;background:transparent;border:1px solid var(--border);border-radius:var(--radius);color:var(--text-2);font-size:12px;cursor:pointer;transition:all .15s}
.topbar-btn:hover{background:var(--surface-2);border-color:var(--border-hover)}

/* === SIDEBAR === */
.sidebar{background:var(--surface);border-right:1px solid var(--border);padding:8px 0;overflow-y:auto;display:flex;flex-direction:column}
.nav-section{padding:0 12px;margin-bottom:4px}
.nav-section-label{font-size:10px;font-weight:600;text-transform:uppercase;letter-spacing:.05em;color:var(--muted);padding:8px 8px 4px}
.nav-item{display:flex;align-items:center;gap:8px;padding:7px 12px;margin:1px 8px;border-radius:6px;color:var(--text-2);cursor:pointer;transition:all .12s;font-size:13px;text-decoration:none}
.nav-item:hover{background:var(--surface-2);color:var(--text)}
.nav-item.active{background:var(--accent-bg);color:var(--accent);font-weight:500}
.nav-item .badge{margin-left:auto;font-size:11px;min-width:20px;text-align:center;padding:1px 6px;border-radius:10px;background:var(--surface-2);color:var(--muted);font-weight:500;font-variant-numeric:tabular-nums}
.nav-item.active .badge{background:var(--accent);color:#fff}
.nav-sep{height:1px;background:var(--border);margin:8px 16px}
.sidebar-footer{margin-top:auto;padding:8px 12px;border-top:1px solid var(--border)}

/* === MAIN AREA === */
.main-wrap{position:relative;overflow:hidden;display:flex}
.main{flex:1;overflow-y:auto;padding:20px 24px}
.page{display:none}
.page.active{display:block}
.page-hdr{display:flex;align-items:center;justify-content:space-between;margin-bottom:16px}
.page-hdr h2{font-size:16px;font-weight:600}

/* === BUTTONS === */
.btn{display:inline-flex;align-items:center;gap:5px;padding:6px 14px;border-radius:var(--radius);border:1px solid var(--border);background:var(--surface);color:var(--text);font-size:12px;cursor:pointer;font-weight:500;transition:all .12s;white-space:nowrap}
.btn:hover{background:var(--surface-2);border-color:var(--border-hover)}
.btn.primary{background:var(--accent);color:#fff;border-color:transparent}
.btn.primary:hover{background:var(--accent-hover)}
.btn.danger{color:var(--red)}
.btn.danger:hover{background:var(--red-bg)}
.btn.sm{padding:4px 8px;font-size:11px}
.btn.ghost{border-color:transparent;background:transparent}
.btn.ghost:hover{background:var(--surface-2)}

/* === STATS === */
.stats{display:grid;grid-template-columns:repeat(4,1fr);gap:10px;margin-bottom:20px}
.stat{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:14px 16px}
.stat-label{font-size:11px;color:var(--muted);margin-bottom:2px;font-weight:500}
.stat-val{font-size:24px;font-weight:600;font-variant-numeric:tabular-nums;line-height:1.2}
.stat-sub{font-size:11px;color:var(--text-2);margin-top:2px}

/* === CARDS === */
.card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);margin-bottom:12px;overflow:hidden}
.card-hdr{display:flex;align-items:center;justify-content:space-between;padding:10px 14px;border-bottom:1px solid var(--border);font-weight:500;font-size:13px}
.card-hdr .link{color:var(--accent);cursor:pointer;font-size:12px;font-weight:400;text-decoration:none}
.card-hdr .link:hover{text-decoration:underline}

/* === TABLES === */
table{width:100%;border-collapse:collapse;font-size:12px}
th{text-align:left;padding:8px 14px;font-weight:500;color:var(--muted);font-size:11px;text-transform:uppercase;letter-spacing:.03em;border-bottom:1px solid var(--border);background:var(--surface-2)}
td{padding:8px 14px;border-bottom:1px solid var(--border);vertical-align:middle}
tr.clickable{cursor:pointer}
tr.clickable:hover td{background:var(--surface-2)}

/* === SEVERITY BADGES === */
.sev{display:inline-block;padding:2px 8px;border-radius:10px;font-size:11px;font-weight:600}
.sev-critical{background:var(--red-bg);color:var(--red)}
.sev-high{background:var(--orange-bg);color:var(--orange)}
.sev-medium{background:var(--yellow-bg);color:var(--yellow)}
.sev-low{background:var(--blue-bg);color:var(--blue)}
.sev-info{background:var(--surface-2);color:var(--muted)}

/* === STATUS === */
.status-confirmed{color:var(--green)}.status-false-positive{color:var(--muted);text-decoration:line-through}.status-unverified{color:var(--yellow)}

/* === TAGS === */
.tag{display:inline-block;padding:1px 7px;border-radius:4px;font-size:10px;background:var(--surface-2);color:var(--text-2);border:1px solid var(--border);margin-right:3px}
.tag.flag{background:var(--yellow-bg);color:var(--yellow);border-color:transparent}
.tag.cred{background:var(--red-bg);color:var(--red);border-color:transparent}
.tag.exploit{background:var(--accent-bg);color:var(--accent);border-color:transparent}

/* === DETAIL PANEL === */
.detail{position:absolute;top:0;right:0;width:400px;height:100%;background:var(--surface);border-left:1px solid var(--border);overflow-y:auto;transform:translateX(100%);transition:transform .2s ease;z-index:10}
.detail.open{transform:translateX(0)}
.detail-hdr{display:flex;align-items:center;justify-content:space-between;padding:14px 16px;border-bottom:1px solid var(--border);position:sticky;top:0;background:var(--surface);z-index:1}
.detail-hdr h3{font-size:14px;font-weight:600}
.detail-close{cursor:pointer;color:var(--muted);font-size:18px;padding:4px;border-radius:4px;transition:all .12s;background:none;border:none}
.detail-close:hover{background:var(--surface-2);color:var(--text)}
.detail-body{padding:16px}
.detail-field{margin-bottom:14px}
.detail-field .lbl{font-size:11px;color:var(--muted);margin-bottom:3px;font-weight:500;text-transform:uppercase;letter-spacing:.03em}
.detail-field .val{font-size:13px}
.detail-actions{display:flex;flex-wrap:wrap;gap:6px;padding:14px 16px;border-top:1px solid var(--border);position:sticky;bottom:0;background:var(--surface)}

/* === OUTPUT VIEWER === */
.output-box{background:var(--bg);border:1px solid var(--border);border-radius:6px;padding:10px 12px;font-family:var(--font-mono);font-size:11px;max-height:200px;overflow-y:auto;white-space:pre-wrap;color:var(--text-2);line-height:1.6;word-break:break-all}
.output-box.expanded{max-height:none}

/* === FILTER BAR === */
.filter-bar{display:flex;gap:6px;margin-bottom:14px;align-items:center;flex-wrap:wrap}
.filter-chip{padding:4px 12px;border-radius:14px;font-size:11px;border:1px solid var(--border);cursor:pointer;color:var(--text-2);background:var(--surface);transition:all .12s;font-weight:500}
.filter-chip:hover{border-color:var(--border-hover);background:var(--surface-2)}
.filter-chip.active{background:var(--accent-bg);color:var(--accent);border-color:var(--accent)}

/* === PROGRESS === */
.progress{height:4px;background:var(--surface-2);border-radius:2px;overflow:hidden}
.progress-bar{height:100%;background:var(--teal);border-radius:2px;transition:width .3s}

/* === MODAL === */
.modal-bg{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.6);z-index:100;display:flex;align-items:center;justify-content:center;animation:fadeIn .15s}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
.modal{background:var(--surface);border:1px solid var(--border);border-radius:12px;padding:1.5rem;width:500px;max-width:95vw;max-height:80vh;overflow-y:auto}
.modal h3{font-size:15px;font-weight:600;margin-bottom:1rem}
.modal label{display:block;font-size:11px;color:var(--muted);margin-bottom:4px;margin-top:12px;font-weight:500;text-transform:uppercase;letter-spacing:.03em}
.modal input,.modal select,.modal textarea{width:100%;padding:8px 12px;background:var(--bg);border:1px solid var(--border);color:var(--text);border-radius:6px;font-size:13px;outline:none;transition:border .15s}
.modal input:focus,.modal select:focus,.modal textarea:focus{border-color:var(--accent)}
.modal textarea{min-height:80px;resize:vertical;font-family:inherit}
.modal-actions{margin-top:1.25rem;display:flex;justify-content:flex-end;gap:8px}

/* === LIGHTBOX === */
.lightbox-bg{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.85);z-index:300;display:flex;align-items:center;justify-content:center;cursor:zoom-out}
.lightbox-bg img{max-width:90vw;max-height:90vh;border-radius:8px;box-shadow:0 8px 32px rgba(0,0,0,.5)}
.lightbox-close{position:fixed;top:16px;right:20px;color:#fff;font-size:28px;cursor:pointer;z-index:301;background:rgba(0,0,0,.4);width:36px;height:36px;border-radius:50%;display:flex;align-items:center;justify-content:center;line-height:1}
.lightbox-caption{position:fixed;bottom:20px;left:50%;transform:translateX(-50%);color:#fff;font-size:13px;z-index:301;background:rgba(0,0,0,.6);padding:6px 16px;border-radius:6px}

/* === PRESENCE === */
.presence-ind{display:flex;align-items:center;gap:6px;font-size:11px;color:var(--text-2);padding:3px 10px;background:var(--surface-2);border-radius:12px;cursor:default}
.presence-dot{width:7px;height:7px;border-radius:50%;background:var(--green);flex-shrink:0}
.presence-tooltip{position:absolute;top:100%;right:0;margin-top:6px;background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:8px 12px;font-size:11px;min-width:140px;box-shadow:0 4px 16px rgba(0,0,0,.3);display:none;z-index:30}
.presence-ind:hover .presence-tooltip{display:block}
.presence-tooltip div{padding:2px 0;color:var(--text-2)}

/* === COMMENTS === */
.comment-thread{margin-top:8px}
.comment-item{padding:8px 10px;margin-bottom:6px;background:var(--surface-2);border-radius:6px;border:1px solid var(--border)}
.comment-meta{display:flex;justify-content:space-between;align-items:center;margin-bottom:4px}
.comment-author{font-weight:600;font-size:11px;color:var(--accent)}
.comment-time{font-size:10px;color:var(--muted)}
.comment-text{font-size:12px;color:var(--text-2);white-space:pre-wrap;word-break:break-word}
.comment-del{background:none;border:none;color:var(--muted);cursor:pointer;font-size:10px;padding:2px 4px;border-radius:3px}
.comment-del:hover{color:var(--red);background:var(--red-bg)}
.comment-form{display:flex;gap:6px;margin-top:8px}
.comment-form textarea{flex:1;padding:6px 8px;background:var(--bg);border:1px solid var(--border);color:var(--text);border-radius:6px;font-size:12px;resize:vertical;min-height:36px;font-family:inherit}
.comment-form textarea:focus{outline:none;border-color:var(--accent)}
.comment-form button{align-self:flex-end}

/* === ATT&CK MAP === */
.attack-grid{display:flex;gap:2px;min-width:max-content;padding-bottom:12px}
.attack-col{min-width:130px;max-width:160px;flex:1}
.attack-hdr{padding:8px 10px;text-align:center;font-size:10px;font-weight:700;text-transform:uppercase;letter-spacing:.04em;color:#fff;border-radius:6px 6px 0 0;position:sticky;top:0}
.attack-cards{padding:4px;display:flex;flex-direction:column;gap:4px;min-height:60px;background:var(--surface);border:1px solid var(--border);border-top:none;border-radius:0 0 6px 6px}
.attack-card{padding:6px 8px;background:var(--surface-2);border:1px solid var(--border);border-radius:4px;cursor:pointer;transition:all .12s;font-size:11px}
.attack-card:hover{border-color:var(--accent);background:var(--accent-bg)}
.attack-card .ac-tech{color:var(--muted);font-size:9px;font-weight:600;margin-bottom:2px}
.attack-card .ac-title{font-weight:500;line-height:1.3}
.attack-card .ac-sev{margin-top:3px}
.attack-empty{padding:12px 8px;text-align:center;font-size:10px;color:var(--muted);font-style:italic}
.attack-legend{display:flex;gap:16px;margin-bottom:12px;align-items:center;flex-wrap:wrap}
.attack-legend .leg-item{display:flex;align-items:center;gap:5px;font-size:11px;color:var(--text-2)}
.attack-legend .leg-dot{width:10px;height:10px;border-radius:2px}

/* === TOPOLOGY === */
.topo-node{cursor:pointer}
.topo-node circle{transition:stroke .15s,r .15s}
.topo-node:hover circle{stroke:var(--accent);stroke-width:2.5}
.topo-node text{fill:var(--text);font-size:10px;text-anchor:middle;pointer-events:none}
.topo-edge{stroke:var(--border);stroke-width:1.5;opacity:.5}
.topo-legend{display:flex;gap:16px;margin-bottom:10px;flex-wrap:wrap;align-items:center}
.topo-legend .leg-item{display:flex;align-items:center;gap:5px;font-size:11px;color:var(--text-2)}
.topo-legend .leg-circle{width:12px;height:12px;border-radius:50%}
.attach-grid{display:flex;flex-wrap:wrap;gap:8px;margin-top:6px}
.attach-thumb{width:80px;height:60px;border-radius:6px;border:1px solid var(--border);overflow:hidden;cursor:pointer;transition:border-color .15s;position:relative}
.attach-thumb:hover{border-color:var(--accent)}
.attach-thumb img{width:100%;height:100%;object-fit:cover}
.attach-thumb .attach-icon{width:100%;height:100%;display:flex;align-items:center;justify-content:center;background:var(--bg);color:var(--muted);font-size:20px}
.attach-name{font-size:10px;color:var(--muted);text-align:center;margin-top:2px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:80px}

/* === TOAST === */
.toast-container{position:fixed;top:56px;right:16px;z-index:200;display:flex;flex-direction:column;gap:6px}
.toast{padding:10px 16px;background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);font-size:12px;animation:slideIn .2s ease,fadeOut .3s ease 3s forwards;max-width:320px;box-shadow:0 4px 12px rgba(0,0,0,.3)}
@keyframes slideIn{from{transform:translateX(100%);opacity:0}to{transform:translateX(0);opacity:1}}
@keyframes fadeOut{to{opacity:0;transform:translateY(-10px)}}

/* === SCOPE === */
.scope-row{display:flex;align-items:center;justify-content:space-between;padding:10px 14px;border-bottom:1px solid var(--border)}
.scope-row:hover{background:var(--surface-2)}
.scope-host{font-family:var(--font-mono);font-size:12px;font-weight:500}
.scope-tested{color:var(--green)}.scope-untested{color:var(--muted)}

/* === CHECKLIST === */
.check-section{margin-bottom:16px}
.check-section-hdr{display:flex;align-items:center;gap:8px;padding:6px 0;font-size:12px;font-weight:500;color:var(--text-2);cursor:pointer}
.check-item{display:flex;align-items:center;gap:10px;padding:6px 14px;border-bottom:1px solid var(--border);font-size:12px;cursor:pointer;transition:background .1s}
.check-item:hover{background:var(--surface-2)}
.check-item input{accent-color:var(--teal);width:16px;height:16px;cursor:pointer}
.check-item.done{color:var(--muted)}
.check-item.done .check-text{text-decoration:line-through}

/* === ACTIVITY === */
.activity-item{display:flex;gap:8px;padding:7px 0;border-bottom:1px solid var(--border);font-size:12px}
.activity-item:last-child{border-bottom:none}
.activity-time{color:var(--muted);font-size:11px;min-width:48px;font-variant-numeric:tabular-nums;font-family:var(--font-mono)}

/* === MONO TEXT === */
.mono{font-family:var(--font-mono);font-size:11px}
.mitre{font-family:var(--font-mono);font-size:11px;color:var(--blue)}
.link{color:var(--accent);cursor:pointer;text-decoration:none}
.link:hover{text-decoration:underline}

/* === GRID LAYOUTS === */
.grid-2{display:grid;grid-template-columns:minmax(0,1fr) 240px;gap:12px}
@media(max-width:900px){.grid-2{grid-template-columns:1fr}.stats{grid-template-columns:repeat(2,1fr)}}
.tpl-grid{display:grid;grid-template-columns:180px 1fr;gap:8px;height:calc(100vh - 140px)}
.tpl-edit-area{display:grid;grid-template-columns:1fr 1fr;gap:0;overflow:hidden;padding:0}
@media(max-width:800px){.tpl-grid{grid-template-columns:140px 1fr}}
</style></head><body>

<div class="shell">
<!-- TOPBAR -->
<div class="topbar">
  <div class="topbar-brand">
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
    RT
  </div>
  <span class="topbar-eng" id="eng-name">` + engName + `</span>
  <span class="topbar-status ok" id="topbar-chain">chain valid</span>
  <div class="topbar-right">
    <div class="topbar-search" id="search-trigger" onclick="showSearch()">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
      Search evidence...
    </div>
    <div class="topbar-btn" id="export-btn" onclick="showExport()">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
      Export
    </div>
    <button class="topbar-btn" onclick="toggleTheme()" title="Toggle dark/light mode">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
    <div class="presence-ind" id="presence-ind" style="display:none">
      <span class="presence-dot"></span>
      <span id="presence-count">0</span> online
      <div class="presence-tooltip" id="presence-list"></div>
    </div>
    <span class="topbar-status" id="ws-badge">offline</span>
  </div>
</div>

<!-- SIDEBAR -->
<div class="sidebar">
  <div class="nav-section">
    <div class="nav-section-label">Dashboard</div>
    <div class="nav-item active" data-page="overview">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
      Overview
    </div>
  </div>
  <div class="nav-section">
    <div class="nav-section-label">Evidence</div>
    <div class="nav-item" data-page="evidence">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
      Timeline <span class="badge" id="badge-evidence">0</span>
    </div>
    <div class="nav-item" data-page="findings">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
      Findings <span class="badge" id="badge-findings">0</span>
    </div>
    <div class="nav-item" data-page="creds">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
      Credentials <span class="badge" id="badge-creds">0</span>
    </div>
  </div>
  <div class="nav-section">
    <div class="nav-section-label">Tracking</div>
    <div class="nav-item" data-page="scope">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/></svg>
      Scope
    </div>
    <div class="nav-item" data-page="checklist">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
      Checklist
    </div>
    <div class="nav-item" data-page="sessions">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
      Sessions
    </div>
  </div>
  <div class="nav-sep"></div>
  <div class="nav-section">
    <div class="nav-item" data-page="attack">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"/><line x1="12" y1="22" x2="12" y2="15.5"/><polyline points="22 8.5 12 15.5 2 8.5"/></svg>
      ATT&amp;CK Map
    </div>
    <div class="nav-item" data-page="topology">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="6" cy="6" r="3"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="18" r="3"/><line x1="9" y1="6" x2="15" y2="6"/><line x1="6" y1="9" x2="6" y2="15"/><line x1="18" y1="9" x2="18" y2="15"/><line x1="9" y1="18" x2="15" y2="18"/></svg>
      Topology
    </div>
    <div class="nav-item" data-page="templates">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      Templates
    </div>
    <div class="nav-item" data-page="audit">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
      Audit log
    </div>
  </div>
  <div class="sidebar-footer">
    <div style="font-size:11px;color:var(--muted)">RT v2.0 — <span id="footer-time"></span></div>
  </div>
</div>

<!-- MAIN CONTENT -->
<div class="main-wrap">
<div class="main" id="main-scroll">

<!-- ===== OVERVIEW ===== -->
<div id="pg-overview" class="page active">
  <div class="stats" id="overview-stats"></div>
  <div class="grid-2">
    <div class="card">
      <div class="card-hdr"><span>Recent findings</span><a class="link" onclick="goPage('findings')">View all →</a></div>
      <table><thead><tr><th>Sev</th><th>Title</th><th>MITRE</th><th>Status</th></tr></thead><tbody id="overview-findings"></tbody></table>
    </div>
    <div class="card">
      <div class="card-hdr"><span>Activity</span></div>
      <div style="padding:8px 14px;max-height:400px;overflow-y:auto" id="overview-activity"></div>
    </div>
  </div>
  <div class="card">
    <div class="card-hdr"><span>Checklist progress</span><span id="overview-check-label" style="font-size:11px;color:var(--muted);font-weight:400"></span></div>
    <div style="padding:10px 14px">
      <div id="overview-check-cats" style="display:flex;gap:16px;font-size:11px;color:var(--text-2);margin-bottom:8px;flex-wrap:wrap"></div>
      <div class="progress" style="height:6px"><div class="progress-bar" id="overview-check-bar" style="width:0"></div></div>
    </div>
  </div>
</div>

<!-- ===== EVIDENCE ===== -->
<div id="pg-evidence" class="page">
  <div class="page-hdr"><h2>Evidence timeline</h2><span id="evidence-count" style="font-size:12px;color:var(--muted)"></span></div>
  <div class="filter-bar" id="evidence-filters">
    <div class="filter-chip active" data-filter="all">All</div>
    <div class="filter-chip" data-filter="flagged">Flagged</div>
    <div class="filter-chip" data-filter="milestone">Milestones</div>
    <div class="filter-chip" data-filter="cred">Creds found</div>
  </div>
  <div class="card"><table><thead><tr><th style="width:40px">#</th><th style="width:70px">Time</th><th>Command</th><th style="width:60px">Exit</th><th style="max-width:200px">Output</th><th>Tags</th><th style="width:30px"></th></tr></thead><tbody id="evidence-table"></tbody></table></div>
</div>

<!-- ===== FINDINGS ===== -->
<div id="pg-findings" class="page">
  <div class="page-hdr"><h2>Findings</h2><button class="btn primary" onclick="showNewFinding()">+ New finding</button></div>
  <div class="filter-bar" id="findings-filters"></div>
  <div id="findings-bulk-bar" style="display:none;margin-bottom:8px;padding:8px 12px;background:var(--surface);border:1px solid var(--accent);border-radius:var(--radius);display:none;align-items:center;gap:8px;font-size:13px"><span id="findings-bulk-count">0 selected</span><button class="btn sm primary" onclick="bulkVerifyFindings()">Verify selected</button><button class="btn sm danger" onclick="bulkDeleteFindings()">Delete selected</button><button class="btn sm ghost" onclick="clearFindingSelection()">Clear</button></div>
  <div class="card"><table><thead><tr><th style="width:30px"><input type="checkbox" id="findings-select-all" onchange="toggleAllFindings(this.checked)"></th><th style="width:30px">#</th><th style="width:70px">Sev</th><th>Title</th><th style="width:70px">MITRE</th><th style="width:80px">Status</th><th style="width:80px">Rec</th><th style="width:30px"></th></tr></thead><tbody id="findings-table"></tbody></table></div>
</div>

<!-- ===== CREDENTIALS ===== -->
<div id="pg-creds" class="page">
  <div class="page-hdr"><h2>Credentials</h2><button class="btn primary" onclick="showNewCred()">+ Add credential</button></div>
  <div id="creds-bulk-bar" style="display:none;margin-bottom:8px;padding:8px 12px;background:var(--surface);border:1px solid var(--accent);border-radius:var(--radius);align-items:center;gap:8px;font-size:13px"><span id="creds-bulk-count">0 selected</span><button class="btn sm danger" onclick="bulkDeleteCreds()">Delete selected</button><button class="btn sm ghost" onclick="clearCredSelection()">Clear</button></div>
  <div class="card"><table><thead><tr><th style="width:30px"><input type="checkbox" id="creds-select-all" onchange="toggleAllCreds(this.checked)"></th><th>Username</th><th>Secret</th><th>Type</th><th>Host</th><th>Source</th><th style="width:80px">Actions</th></tr></thead><tbody id="creds-table"></tbody></table></div>
</div>

<!-- ===== SCOPE ===== -->
<div id="pg-scope" class="page">
  <div class="page-hdr"><h2>Scope</h2><button class="btn primary" onclick="showAddScope()">+ Add hosts</button></div>
  <div id="scope-summary" style="margin-bottom:12px"></div>
  <div class="card" id="scope-list"></div>
</div>

<!-- ===== CHECKLIST ===== -->
<div id="pg-checklist" class="page">
  <div class="page-hdr">
    <h2>Checklist</h2>
    <div style="display:flex;gap:6px">
      <button class="btn" onclick="showLoadChecklist()">Load PTES</button>
      <button class="btn" onclick="showAddCheckItem()">+ Custom item</button>
    </div>
  </div>
  <div id="checklist-summary" style="margin-bottom:12px"></div>
  <div id="checklist-items"></div>
</div>

<!-- ===== SESSIONS ===== -->
<div id="pg-sessions" class="page">
  <div class="page-hdr"><h2>Sessions</h2></div>
  <div class="card"><table><thead><tr><th>ID</th><th>Name</th><th>Source</th><th>Status</th><th>Started</th><th>Operator</th></tr></thead><tbody id="sessions-table"></tbody></table></div>
</div>

<!-- ===== TEMPLATES ===== -->
<div id="pg-templates" class="page">
  <div class="page-hdr"><h2>Report Templates</h2><button class="btn sm primary" onclick="showNewTemplate()">+ New Template</button></div>
  <div class="tpl-grid">
    <div class="card" style="overflow-y:auto;padding:6px" id="template-list"></div>
    <div class="card tpl-edit-area">
      <div style="display:flex;flex-direction:column;border-right:1px solid var(--border)">
        <div style="padding:8px 12px;border-bottom:1px solid var(--border);font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.05em;color:var(--muted)">Editor</div>
        <textarea id="template-editor" style="flex:1;resize:none;border:none;background:var(--bg);color:var(--text);padding:12px;font-family:'Fira Code',monospace;font-size:13px;line-height:1.6;outline:none;overflow-y:auto" placeholder="Select a template to edit..." oninput="updateTemplatePreview()"></textarea>
      </div>
      <div style="display:flex;flex-direction:column">
        <div style="padding:8px 12px;border-bottom:1px solid var(--border);font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.05em;color:var(--muted)">Preview</div>
        <div id="template-preview" style="flex:1;overflow-y:auto;padding:12px;font-size:13px;line-height:1.6"></div>
      </div>
    </div>
  </div>
</div>

<!-- ===== AUDIT ===== -->
<!-- ===== TOPOLOGY ===== -->
<div id="pg-topology" class="page">
  <div class="page-hdr"><h2>Network Topology</h2></div>
  <div id="topo-canvas" style="background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);min-height:400px;position:relative;overflow:hidden"></div>
</div>

<!-- ===== ATT&CK MAP ===== -->
<div id="pg-attack" class="page">
  <div class="page-hdr"><h2>MITRE ATT&amp;CK Kill Chain</h2></div>
  <div id="attack-map" style="overflow-x:auto"></div>
</div>

<div id="pg-audit" class="page">
  <div class="page-hdr"><h2>Audit log</h2></div>
  <div class="filter-bar" id="audit-filters">
    <div class="filter-chip active" data-filter="all">All</div>
    <div class="filter-chip" data-filter="create">Creates</div>
    <div class="filter-chip" data-filter="update">Updates</div>
    <div class="filter-chip" data-filter="delete">Deletes</div>
  </div>
  <div class="card"><table><thead><tr><th style="width:70px">Time</th><th style="width:80px">Operator</th><th style="width:120px">Action</th><th>Target</th><th>Detail</th></tr></thead><tbody id="audit-table"></tbody></table></div>
</div>

</div>

<!-- DETAIL PANEL -->
<div class="detail" id="detail-panel">
  <div class="detail-hdr">
    <h3 id="detail-title"></h3>
    <button class="detail-close" onclick="closeDetail()">✕</button>
  </div>
  <div class="detail-body" id="detail-body"></div>
  <div class="detail-actions" id="detail-actions"></div>
</div>

</div><!-- /main-wrap -->
</div><!-- /shell -->

<div id="modal-root"></div>
<div id="lightbox-root"></div>
<div class="toast-container" id="toast-container"></div>

<script>
const KEY=localStorage.getItem('rt_api_key')||'';
const H={'X-API-Key':KEY,'Content-Type':'application/json'};
function esc(s){if(!s)return'';const d=document.createElement('div');d.textContent=s;return d.innerHTML}
function fmtTime(ts){try{const d=new Date(ts);return d.toLocaleTimeString('en-GB',{hour:'2-digit',minute:'2-digit',second:'2-digit'})}catch(e){return ts||''}}
function fmtDate(ts){try{return new Date(ts).toLocaleString('sv').replace('T',' ')}catch(e){return ts||''}}

// Theme
function toggleTheme(){
  const cur=document.documentElement.getAttribute('data-theme');
  const next=cur==='light'?'dark':'light';
  document.documentElement.setAttribute('data-theme',next);
  localStorage.setItem('rt_theme',next);
}
(function(){const t=localStorage.getItem('rt_theme');if(t)document.documentElement.setAttribute('data-theme',t)})();

// Navigation
let currentPage='overview';
document.querySelectorAll('.nav-item[data-page]').forEach(item=>{
  item.addEventListener('click',()=>{
    closeDetail();
    document.querySelectorAll('.nav-item').forEach(n=>n.classList.remove('active'));
    item.classList.add('active');
    document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
    currentPage=item.dataset.page;
    const pg=document.getElementById('pg-'+currentPage);
    if(pg)pg.classList.add('active');
    loadPage(currentPage);
  });
});
function goPage(p){document.querySelector('.nav-item[data-page="'+p+'"]').click()}

// API
async function api(path,opts){
  const r=await fetch(path,{headers:H,...(opts||{})});
  if(r.status===401){location.href='/login';return null}
  if(!r.ok){const e=await r.json().catch(()=>({error:'request failed'}));throw new Error(e.error||'failed')}
  return r.json();
}
async function post(path,body){return api(path,{method:'POST',body:JSON.stringify(body)})}
async function put(path,body){return api(path,{method:'PUT',body:JSON.stringify(body)})}
async function del(path){return api(path,{method:'DELETE'})}

// Toast
function toast(msg,type){
  const c=document.getElementById('toast-container');
  const t=document.createElement('div');
  t.className='toast';
  if(type==='success')t.style.borderLeft='3px solid var(--green)';
  else if(type==='error')t.style.borderLeft='3px solid var(--red)';
  else t.style.borderLeft='3px solid var(--accent)';
  t.textContent=msg;
  c.appendChild(t);
  setTimeout(()=>t.remove(),3500);
}

// Loading
async function loadPage(page){
  const loaders={overview:loadOverview,evidence:loadEvidence,findings:loadFindings,creds:loadCreds,scope:loadScope,checklist:loadChecklist,sessions:loadSessions,attack:loadAttackMap,topology:loadTopology,templates:loadTemplates,audit:loadAudit};
  if(loaders[page])try{await loaders[page]()}catch(e){console.error(e)}
}

function sevBadge(p){return '<span class="sev sev-'+esc(p)+'">'+esc(p)+'</span>'}
function statusSpan(v){return '<span class="status-'+esc(v)+'">'+esc(v)+'</span>'}
function tagSpan(t){
  let cls='tag';
  if(t.startsWith('auto:'))cls='tag flag';
  if(t==='cred-found'||t==='credential')cls='tag cred';
  if(t==='exploit'||t==='rce')cls='tag exploit';
  return '<span class="'+cls+'">'+esc(t)+'</span>';
}
function pct(a,b){return b===0?0:Math.round(a/b*100)}

// ===== DATA CACHE =====
let cachedFindings=[];let cachedEvidence=[];let cachedAudit=[];

// ===== OVERVIEW =====
async function loadOverview(){
  const [ov,f,cl]=await Promise.all([api('/api/overview'),api('/api/findings'),api('/api/checklist')]);
  if(!ov)return;
  const s=ov.stats;
  cachedFindings=f||[];
  document.getElementById('badge-evidence').textContent=s.TotalEvidence;
  document.getElementById('badge-findings').textContent=s.TotalFindings;
  document.getElementById('badge-creds').textContent=ov.cred_count;
  document.getElementById('overview-stats').innerHTML=
    statCard(s.TotalEvidence,'Evidence','','12 auto-flagged')+
    statCard(s.TotalFindings,'Findings','var(--accent)',s.Critical+' critical, '+s.High+' high')+
    statCard(ov.cred_count,'Credentials','','')+
    statCard(pct(ov.scope_tested,ov.scope_total)+'%','Scope tested','var(--teal)',ov.scope_tested+'/'+ov.scope_total+' hosts');
  document.getElementById('topbar-chain').textContent=ov.chain_intact?'chain valid':'CHAIN BROKEN';
  document.getElementById('topbar-chain').className='topbar-status '+(ov.chain_intact?'ok':'warn');

  document.getElementById('overview-findings').innerHTML=(f||[]).slice(0,6).map(x=>
    '<tr class="clickable" onclick="goPage(\'findings\')"><td>'+sevBadge(x.Priority)+'</td><td>'+esc(x.Title)+'</td><td class="mitre">'+(x.Mitre||[]).join(', ')+'</td><td>'+statusSpan(x.Verified)+'</td></tr>'
  ).join('')||'<tr><td colspan="4" style="color:var(--muted);text-align:center;padding:20px">No findings yet</td></tr>';

  // Activity from timeline
  const ev=await api('/api/timeline');
  cachedEvidence=ev||[];
  document.getElementById('overview-activity').innerHTML=(ev||[]).slice(0,20).map(e=>
    '<div class="activity-item"><span class="activity-time">'+fmtTime(e.timestamp)+'</span><span>'+esc(e.action)+': <strong class="mono">'+esc((e.input||'').substring(0,60))+'</strong></span></div>'
  ).join('')||'<div style="color:var(--muted);padding:12px">No activity yet</div>';

  // Checklist
  if(cl){
    document.getElementById('overview-check-label').textContent=cl.done+'/'+cl.total+' complete';
    document.getElementById('overview-check-bar').style.width=pct(cl.done,cl.total)+'%';
    const cats={};
    (cl.items||[]).forEach(it=>{
      if(!cats[it.category])cats[it.category]={total:0,done:0};
      cats[it.category].total++;
      if(it.done)cats[it.category].done++;
    });
    document.getElementById('overview-check-cats').innerHTML=Object.entries(cats).map(([k,v])=>
      '<span>'+esc(k)+': '+v.done+'/'+v.total+'</span>'
    ).join('');
  }
}
function statCard(val,label,color,sub){
  return '<div class="stat"><div class="stat-label">'+esc(label)+'</div><div class="stat-val"'+(color?' style="color:'+color+'"':'')+'>'+(val)+'</div>'+(sub?'<div class="stat-sub">'+esc(sub)+'</div>':'')+'</div>';
}

// ===== EVIDENCE =====
async function loadEvidence(){
  const ev=await api('/api/timeline');
  if(!ev)return;
  cachedEvidence=ev;
  document.getElementById('evidence-count').textContent=ev.length+' entries';
  renderEvidence(ev);
}
function renderEvidence(ev){
  document.getElementById('evidence-table').innerHTML=(ev||[]).map(e=>
    '<tr class="clickable" onclick="showEvidenceDetail('+e.id+')"><td>'+e.id+'</td><td style="font-variant-numeric:tabular-nums" class="mono">'+fmtTime(e.timestamp)+'</td><td class="mono" style="max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(e.input)+'</td><td>'+(e.exit_code===0?'<span style="color:var(--green)">0</span>':'<span style="color:var(--red)">'+e.exit_code+'</span>')+'</td><td class="mono" style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:10px;color:var(--text-2)">'+esc((e.output||'').substring(0,80))+'</td><td>'+(e.tags||[]).map(t=>tagSpan(t)).join('')+'</td><td><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--muted)" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg></td></tr>'
  ).join('')||'<tr><td colspan="7" style="color:var(--muted);text-align:center;padding:20px">No evidence</td></tr>';
}

// Evidence filters
document.getElementById('evidence-filters').addEventListener('click',e=>{
  const chip=e.target.closest('.filter-chip');
  if(!chip)return;
  document.querySelectorAll('#evidence-filters .filter-chip').forEach(c=>c.classList.remove('active'));
  chip.classList.add('active');
  const f=chip.dataset.filter;
  if(f==='all')renderEvidence(cachedEvidence);
  else if(f==='flagged')renderEvidence(cachedEvidence.filter(e=>(e.tags||[]).some(t=>t.startsWith('auto:'))));
  else if(f==='milestone')renderEvidence(cachedEvidence.filter(e=>e.action==='milestone'));
  else if(f==='cred')renderEvidence(cachedEvidence.filter(e=>(e.tags||[]).some(t=>t==='cred-found'||t==='credential')));
});

async function showEvidenceDetail(id){
  const ev=await api('/api/evidence/'+id);
  if(!ev)return;
  const att=await loadAttachments(id);
  const body=document.getElementById('detail-body');
  body.innerHTML=
    '<div class="detail-field"><div class="lbl">Command</div><div class="val mono" style="word-break:break-all">'+esc(ev.Input)+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Exit code</div><div class="val">'+(ev.ExitCode===0?'<span style="color:var(--green)">0</span>':'<span style="color:var(--red)">'+ev.ExitCode+'</span>')+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Timestamp</div><div class="val">'+fmtDate(ev.Timestamp)+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Tags</div><div class="val">'+((ev.Tags||[]).map(t=>tagSpan(t)).join(' ')||'<span style="color:var(--muted)">none</span>')+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Hash</div><div class="val mono" style="font-size:10px;color:var(--muted);word-break:break-all">'+esc(ev.Hash)+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Attachments ('+att.length+')</div><div class="val">'+renderAttachments(att)+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Output</div><div class="output-box" id="ev-output">'+esc(ev.Output||'(no output)')+'</div>'+
    '<button class="btn sm ghost" style="margin-top:4px" onclick="document.getElementById(\'ev-output\').classList.toggle(\'expanded\');this.textContent=this.textContent===\'Expand\'?\'Collapse\':\'Expand\'">Expand</button></div>';
  document.getElementById('detail-title').textContent='Evidence #'+id;
  document.getElementById('detail-actions').innerHTML=
    '<button class="btn sm" onclick="toast(\'Tag added\',\'success\')">+ Tag</button>'+
    '<button class="btn sm" onclick="toast(\'Note added\',\'success\')">+ Note</button>'+
    '<button class="btn sm danger" onclick="toast(\'Redacted\',\'success\')">Redact</button>';
  openDetail();
}

// ===== FINDINGS =====
async function loadFindings(){
  const f=await api('/api/findings');
  if(!f)return;
  cachedFindings=f;
  const crit=f.filter(x=>x.Priority==='critical').length;
  const high=f.filter(x=>x.Priority==='high').length;
  const uv=f.filter(x=>x.Verified==='unverified').length;
  document.getElementById('findings-filters').innerHTML=
    '<div class="filter-chip active" data-filter="all">All ('+f.length+')</div>'+
    (crit?'<div class="filter-chip" data-filter="critical" style="background:var(--red-bg);color:var(--red);border-color:transparent">Critical ('+crit+')</div>':'')+
    (high?'<div class="filter-chip" data-filter="high" style="background:var(--orange-bg);color:var(--orange);border-color:transparent">High ('+high+')</div>':'')+
    (uv?'<div class="filter-chip" data-filter="unverified">Unverified ('+uv+')</div>':'');
  renderFindings(f);

  document.getElementById('findings-filters').addEventListener('click',e=>{
    const chip=e.target.closest('.filter-chip');
    if(!chip)return;
    document.querySelectorAll('#findings-filters .filter-chip').forEach(c=>c.classList.remove('active'));
    chip.classList.add('active');
    const fl=chip.dataset.filter;
    if(fl==='all')renderFindings(cachedFindings);
    else if(fl==='critical')renderFindings(cachedFindings.filter(x=>x.Priority==='critical'));
    else if(fl==='high')renderFindings(cachedFindings.filter(x=>x.Priority==='high'));
    else if(fl==='unverified')renderFindings(cachedFindings.filter(x=>x.Verified==='unverified'));
  });
}
function renderFindings(f){
  document.getElementById('findings-table').innerHTML=(f||[]).map(x=>
    '<tr class="clickable"><td onclick="event.stopPropagation()"><input type="checkbox" class="finding-cb" value="'+x.ID+'" onchange="updateFindingBulkBar()"></td><td onclick="showFindingDetail('+x.ID+')">'+x.ID+'</td><td onclick="showFindingDetail('+x.ID+')">'+sevBadge(x.Priority)+'</td><td onclick="showFindingDetail('+x.ID+')">'+esc(x.Title)+'</td><td onclick="showFindingDetail('+x.ID+')" class="mitre">'+(x.Mitre||[]).slice(0,2).join(', ')+'</td><td onclick="showFindingDetail('+x.ID+')">'+statusSpan(x.Verified)+'</td><td onclick="showFindingDetail('+x.ID+')">'+(x.Recommendation?'<span style="color:var(--green)">✓</span>':'<span style="color:var(--muted)">—</span>')+'</td><td onclick="showFindingDetail('+x.ID+')"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--muted)" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg></td></tr>'
  ).join('')||'<tr><td colspan="8" style="color:var(--muted);text-align:center;padding:20px">No findings</td></tr>';
  updateFindingBulkBar();
}

async function showFindingDetail(id){
  const f=await api('/api/findings/'+id);
  if(!f)return;
  const att=await loadFindingAttachments(id);
  const body=document.getElementById('detail-body');
  body.innerHTML=
    '<div style="margin-bottom:12px">'+sevBadge(f.Priority)+' '+(f.Mitre||[]).map(m=>'<span class="mitre" style="margin-left:6px">'+esc(m)+'</span>').join('')+'</div>'+
    '<div class="detail-field"><div class="lbl">Title</div><div class="val" style="font-weight:500">'+esc(f.Title)+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Description</div><div class="val" style="color:var(--text-2)">'+(esc(f.Description)||'<em style="color:var(--muted)">No description</em>')+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Status</div><div class="val">'+statusSpan(f.Verified)+(f.VerifiedBy?' by '+esc(f.VerifiedBy):'')+(f.VerifiedAt?' at '+fmtDate(f.VerifiedAt):'')+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Verification note</div><div class="val" style="color:var(--text-2)">'+(esc(f.Notes)||'<em style="color:var(--muted)">—</em>')+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Linked evidence</div><div class="val">'+(f.EvidenceIDs&&f.EvidenceIDs.length?f.EvidenceIDs.map(id=>'<a class="link" onclick="showEvidenceDetail('+id+')">#'+id+'</a>').join(', '):'<em style="color:var(--muted)">none</em>')+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Screenshots & Attachments ('+att.length+')</div><div class="val">'+renderAttachments(att)+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Recommendation</div><div class="val" style="color:var(--text-2)">'+(esc(f.Recommendation)||'<em style="color:var(--muted)">None yet</em>')+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Created</div><div class="val">'+fmtDate(f.CreatedAt)+'</div></div>'+
    '<div class="detail-field"><div class="lbl">Comments</div><div class="val"><div id="comment-thread-'+id+'" class="comment-thread"></div>'+
    '<div class="comment-form"><textarea id="comment-input-'+id+'" placeholder="Add a comment..." rows="1"></textarea><button class="btn sm primary" onclick="addComment('+id+')">Post</button></div></div></div>';
  loadComments(id);
  document.getElementById('detail-title').textContent='Finding #'+id;
  document.getElementById('detail-actions').innerHTML=
    '<button class="btn sm primary" onclick="showVerify('+id+')">Verify</button>'+
    '<button class="btn sm" onclick="showRecommend('+id+')">Recommend</button>'+
    '<button class="btn sm" onclick="showEditFinding('+id+')">Edit</button>'+
    '<button class="btn sm danger" onclick="deleteFinding('+id+')">Delete</button>';
  openDetail();
}

function showNewFinding(){
  showModal('New finding',
    '<label>Title</label><input id="m-title" placeholder="Finding title">'+
    '<label>Priority</label><select id="m-priority"><option>critical</option><option>high</option><option selected>medium</option><option>low</option><option>info</option></select>'+
    '<label>Description</label><textarea id="m-desc" placeholder="Describe the vulnerability"></textarea>'+
    '<label>MITRE ATT&CK</label><input id="m-mitre" placeholder="T1190, T1210">'
  ,async()=>{
    await post('/api/findings',{title:gv('m-title'),priority:gv('m-priority'),description:gv('m-desc'),mitre:gv('m-mitre').split(',').map(s=>s.trim()).filter(Boolean)});
    closeModal();loadFindings();toast('Finding created','success');
  });
}
function showVerify(id){
  showModal('Verify finding #'+id,
    '<label>Status</label><select id="m-vstatus"><option>confirmed</option><option>false-positive</option></select>'+
    '<label>Note</label><textarea id="m-vnote" placeholder="How was this verified?"></textarea>'
  ,async()=>{
    await post('/api/findings/'+id+'/verify',{status:gv('m-vstatus'),note:gv('m-vnote')});
    closeModal();loadFindings();showFindingDetail(id);toast('Finding verified','success');
  });
}
function showRecommend(id){
  showModal('Add recommendation — Finding #'+id,
    '<label>Recommendation</label><textarea id="m-rec" placeholder="Specific remediation steps..."></textarea>'
  ,async()=>{
    await post('/api/findings/'+id+'/recommend',{recommendation:gv('m-rec')});
    closeModal();loadFindings();showFindingDetail(id);toast('Recommendation added','success');
  });
}
async function showEditFinding(id){
  const f=await api('/api/findings/'+id);
  if(!f)return;
  showModal('Edit finding #'+id,
    '<label>Title</label><input id="m-etitle" value="'+esc(f.Title)+'">'+
    '<label>Priority</label><select id="m-epri"><option'+(f.Priority==='critical'?' selected':'')+'>critical</option><option'+(f.Priority==='high'?' selected':'')+'>high</option><option'+(f.Priority==='medium'?' selected':'')+'>medium</option><option'+(f.Priority==='low'?' selected':'')+'>low</option><option'+(f.Priority==='info'?' selected':'')+'>info</option></select>'+
    '<label>Description</label><textarea id="m-edesc">'+esc(f.Description)+'</textarea>'+
    '<label>MITRE</label><input id="m-emitre" value="'+(f.Mitre||[]).join(', ')+'">'
  ,async()=>{
    await put('/api/findings/'+id,{title:gv('m-etitle'),priority:gv('m-epri'),description:gv('m-edesc'),mitre:gv('m-emitre').split(',').map(s=>s.trim()).filter(Boolean)});
    closeModal();loadFindings();showFindingDetail(id);toast('Finding updated','success');
  });
}
async function deleteFinding(id){
  if(!confirm('Delete finding #'+id+'?'))return;
  await del('/api/findings/'+id);
  closeDetail();loadFindings();toast('Finding deleted','success');
}

// ===== CREDENTIALS =====
async function loadCreds(){
  const c=await api('/api/creds');
  if(!c)return;
  document.getElementById('creds-table').innerHTML=(c||[]).map(x=>
    '<tr><td><input type="checkbox" class="cred-cb" value="'+x.id+'" onchange="updateCredBulkBar()"></td>'+
    '<td style="font-weight:500" class="mono">'+esc(x.Username)+'</td>'+
    '<td><span class="mono" id="cred-secret-'+x.id+'" style="letter-spacing:2px">••••••••</span> '+
    '<button class="btn sm ghost" onclick="revealCred('+x.id+')" title="Reveal">👁</button> '+
    '<button class="btn sm ghost" onclick="copyCred('+x.id+')" title="Copy">📋</button></td>'+
    '<td>'+esc(x.CredType)+'</td>'+
    '<td class="mono">'+esc(x.Host)+'</td>'+
    '<td>'+(x.SourceEvidenceID?'<a class="link" onclick="goPage(\'evidence\');setTimeout(()=>showEvidenceDetail('+x.SourceEvidenceID+'),300)">#'+x.SourceEvidenceID+'</a>':'—')+'</td>'+
    '<td><button class="btn sm danger" onclick="deleteCred('+x.id+')">Delete</button></td></tr>'
  ).join('')||'<tr><td colspan="7" style="color:var(--muted);text-align:center;padding:20px">No credentials</td></tr>';
  updateCredBulkBar();
}
async function revealCred(id){
  try{
    const r=await api('/api/creds/'+id+'/reveal');
    const el=document.getElementById('cred-secret-'+id);
    if(el.dataset.revealed){el.textContent='••••••••';el.style.letterSpacing='2px';delete el.dataset.revealed}
    else{el.textContent=r.secret;el.style.letterSpacing='normal';el.dataset.revealed='1'}
  }catch(e){toast('Failed to reveal: '+e.message,'error')}
}
async function copyCred(id){
  try{
    const r=await api('/api/creds/'+id+'/reveal');
    await navigator.clipboard.writeText(r.secret);
    toast('Copied to clipboard','success');
  }catch(e){toast('Failed: '+e.message,'error')}
}
async function deleteCred(id){
  if(!confirm('Delete credential #'+id+'?'))return;
  await del('/api/creds/'+id);
  loadCreds();toast('Credential deleted','success');
}
// ===== BULK ACTIONS =====
function getSelectedFindings(){return[...document.querySelectorAll('.finding-cb:checked')].map(c=>parseInt(c.value))}
function getSelectedCreds(){return[...document.querySelectorAll('.cred-cb:checked')].map(c=>parseInt(c.value))}
function updateFindingBulkBar(){
  const sel=getSelectedFindings();
  const bar=document.getElementById('findings-bulk-bar');
  bar.style.display=sel.length?'flex':'none';
  document.getElementById('findings-bulk-count').textContent=sel.length+' selected';
}
function updateCredBulkBar(){
  const sel=getSelectedCreds();
  const bar=document.getElementById('creds-bulk-bar');
  bar.style.display=sel.length?'flex':'none';
  document.getElementById('creds-bulk-count').textContent=sel.length+' selected';
}
function toggleAllFindings(checked){document.querySelectorAll('.finding-cb').forEach(c=>c.checked=checked);updateFindingBulkBar()}
function toggleAllCreds(checked){document.querySelectorAll('.cred-cb').forEach(c=>c.checked=checked);updateCredBulkBar()}
function clearFindingSelection(){document.getElementById('findings-select-all').checked=false;toggleAllFindings(false)}
function clearCredSelection(){document.getElementById('creds-select-all').checked=false;toggleAllCreds(false)}
async function bulkVerifyFindings(){
  const ids=getSelectedFindings();
  if(!ids.length)return;
  if(!confirm('Verify '+ids.length+' finding(s) as confirmed?'))return;
  for(const id of ids){await post('/api/findings/'+id+'/verify',{status:'confirmed',note:'Bulk verified from dashboard'})}
  clearFindingSelection();loadFindings();toast(ids.length+' findings verified','success');
}
async function bulkDeleteFindings(){
  const ids=getSelectedFindings();
  if(!ids.length)return;
  if(!confirm('Delete '+ids.length+' finding(s)? This cannot be undone.'))return;
  for(const id of ids){await del('/api/findings/'+id)}
  clearFindingSelection();loadFindings();toast(ids.length+' findings deleted','success');
}
async function bulkDeleteCreds(){
  const ids=getSelectedCreds();
  if(!ids.length)return;
  if(!confirm('Delete '+ids.length+' credential(s)? This cannot be undone.'))return;
  for(const id of ids){await del('/api/creds/'+id)}
  clearCredSelection();loadCreds();toast(ids.length+' credentials deleted','success');
}

function showNewCred(){
  showModal('Add credential',
    '<label>Username</label><input id="m-cuser" placeholder="admin">'+
    '<label>Secret</label><input id="m-csecret" placeholder="password or hash">'+
    '<label>Type</label><select id="m-ctype"><option>password</option><option>hash</option><option>token</option><option>key</option></select>'+
    '<label>Host</label><input id="m-chost" placeholder="10.10.10.1">'
  ,async()=>{
    await post('/api/creds',{username:gv('m-cuser'),secret:gv('m-csecret'),secret_type:gv('m-ctype'),host:gv('m-chost')});
    closeModal();loadCreds();toast('Credential added','success');
  });
}

// ===== SCOPE =====
async function loadScope(){
  const d=await api('/api/scope');
  if(!d)return;
  const hosts=d.hosts||[];
  const p=pct(d.tested,d.total);
  document.getElementById('scope-summary').innerHTML=
    '<div style="display:flex;align-items:center;gap:12px"><span style="font-size:20px;font-weight:600;color:var(--teal)">'+p+'%</span><div style="flex:1"><div class="progress" style="height:6px"><div class="progress-bar" style="width:'+p+'%"></div></div></div><span style="font-size:12px;color:var(--muted)">'+d.tested+'/'+d.total+' hosts tested</span></div>';
  document.getElementById('scope-list').innerHTML=hosts.map(h=>
    '<div class="scope-row"><div><span class="scope-host">'+esc(h.Host)+'</span></div><div style="display:flex;align-items:center;gap:12px">'+
    (h.Tested?'<span class="scope-tested" style="font-size:12px">✓ Tested</span>':'<span class="scope-untested" style="font-size:12px">○ Untested</span>')+
    (h.Tested?'':'<button class="btn sm primary" onclick="markTested(\''+esc(h.Host)+'\')">Mark tested</button>')+
    '</div></div>'
  ).join('')||'<div style="padding:20px;text-align:center;color:var(--muted)">No hosts in scope</div>';
}
function showAddScope(){
  showModal('Add scope hosts',
    '<label>Hosts (comma-separated)</label><textarea id="m-hosts" placeholder="10.10.10.1, 10.10.10.2"></textarea>'
  ,async()=>{
    await post('/api/scope',{hosts:gv('m-hosts')});
    closeModal();loadScope();toast('Hosts added','success');
  });
}
async function markTested(host){
  try{await post('/api/scope/tested',{host});loadScope();toast(host+' marked tested','success')}catch(e){toast('Error: '+e.message,'error')}
}

// ===== CHECKLIST =====
async function loadChecklist(){
  const d=await api('/api/checklist');
  if(!d)return;
  const items=d.items||[];
  const p=pct(d.done,d.total);
  document.getElementById('checklist-summary').innerHTML=
    '<div style="display:flex;align-items:center;gap:12px"><span style="font-size:20px;font-weight:600;color:var(--teal)">'+d.done+'/'+d.total+'</span><div style="flex:1"><div class="progress" style="height:6px"><div class="progress-bar" style="width:'+p+'%"></div></div></div><span style="font-size:12px;color:var(--muted)">'+p+'% complete</span></div>';

  const cats={};
  items.forEach(it=>{
    if(!cats[it.category])cats[it.category]=[];
    cats[it.category].push(it);
  });
  let html='';
  Object.entries(cats).forEach(([cat,list])=>{
    const done=list.filter(i=>i.done).length;
    const cp=pct(done,list.length);
    html+='<div class="check-section"><div class="card"><div class="card-hdr"><span>'+esc(cat)+'</span><span style="font-size:11px;color:var(--muted);font-weight:400">'+done+'/'+list.length+'</span></div>';
    html+='<div style="padding:6px 14px"><div class="progress"><div class="progress-bar" style="width:'+cp+'%"></div></div></div>';
    list.forEach(it=>{
      html+='<div class="check-item'+(it.done?' done':'')+'"><input type="checkbox" '+(it.done?'checked':'')+' onchange="toggleCheck('+it.id+',this.checked)"><span class="check-text">'+esc(it.item)+'</span>'+(it.done_at?'<span style="margin-left:auto;font-size:10px;color:var(--muted)">'+fmtTime(it.done_at)+'</span>':'')+'</div>';
    });
    html+='</div></div>';
  });
  document.getElementById('checklist-items').innerHTML=html||'<div style="padding:20px;text-align:center;color:var(--muted)">No checklist loaded</div>';
}
async function toggleCheck(id,checked){
  try{await post('/api/checklist/toggle',{id,checked});loadChecklist()}catch(e){toast('Error: '+e.message,'error')}
}
function showLoadChecklist(){
  showModal('Load checklist preset',
    '<label>Preset</label><select id="m-preset"><option value="ptes">PTES (27 items)</option></select>'
  ,async()=>{
    await post('/api/checklist',{preset:gv('m-preset')});
    closeModal();loadChecklist();toast('Checklist loaded','success');
  });
}
function showAddCheckItem(){
  showModal('Add custom checklist item',
    '<label>Category</label><input id="m-cat" placeholder="Custom">'+
    '<label>Item</label><input id="m-item" placeholder="Task description">'
  ,async()=>{
    await post('/api/checklist',{category:gv('m-cat'),item:gv('m-item')});
    closeModal();loadChecklist();toast('Item added','success');
  });
}

// ===== SESSIONS =====
async function loadSessions(){
  const s=await api('/api/sessions');
  if(!s)return;
  document.getElementById('sessions-table').innerHTML=(s||[]).map(x=>
    '<tr><td class="mono" style="font-size:11px">'+esc((x.ID||'').substring(0,12))+'</td><td style="font-weight:500">'+esc(x.Name)+'</td><td>'+esc(x.Source)+'</td><td>'+(x.Status==='active'?'<span style="color:var(--green)">● active</span>':'<span style="color:var(--muted)">stopped</span>')+'</td><td>'+fmtDate(x.StartedAt)+'</td><td>'+esc(x.Operator||'default')+'</td></tr>'
  ).join('')||'<tr><td colspan="6" style="color:var(--muted);text-align:center;padding:20px">No sessions</td></tr>';
}

// ===== TEMPLATES =====
let cachedTemplates=[];
let activeTemplateId=null;
async function loadTemplates(){
  const list=await api('/api/templates');
  if(!list)return;
  cachedTemplates=list;
  renderTemplateList(list);
  if(list.length&&!activeTemplateId)selectTemplate(list[0].id);
}
function renderTemplateList(list){
  document.getElementById('template-list').innerHTML=list.map(t=>
    '<div style="padding:8px 10px;border-radius:6px;cursor:pointer;font-size:13px;margin-bottom:2px;'+(t.id===activeTemplateId?'background:var(--accent);color:#fff':'color:var(--text)')
    +'" onclick="selectTemplate('+t.id+')"><div style="font-weight:500">'+esc(t.name)+'</div><div style="font-size:10px;'+(t.id===activeTemplateId?'color:rgba(255,255,255,.7)':'color:var(--muted)')+'">'+t.format+' · '+fmtDate(t.updated_at)+'</div></div>'
  ).join('')+'<div style="padding:8px 10px;margin-top:8px;border-top:1px solid var(--border);font-size:11px;color:var(--muted)">'+list.length+' template(s)</div>';
}
async function selectTemplate(id){
  activeTemplateId=id;
  const t=await api('/api/templates/'+id);
  if(!t)return;
  document.getElementById('template-editor').value=t.content;
  updateTemplatePreview();
  renderTemplateList(cachedTemplates);
}
function updateTemplatePreview(){
  const md=document.getElementById('template-editor').value;
  document.getElementById('template-preview').innerHTML=simpleMarkdown(md);
}
function simpleMarkdown(md){
  return md.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    .replace(/^### (.+)$/gm,'<h3 style="margin:12px 0 4px;font-size:14px">$1</h3>')
    .replace(/^## (.+)$/gm,'<h2 style="margin:16px 0 6px;font-size:16px">$1</h2>')
    .replace(/^# (.+)$/gm,'<h1 style="margin:20px 0 8px;font-size:20px">$1</h1>')
    .replace(/\*\*(.+?)\*\*/g,'<strong>$1</strong>')
    .replace(/\*(.+?)\*/g,'<em>$1</em>')
    .replace(/^\|(.+)$/gm,function(m){return'<div style="font-family:monospace;font-size:11px;color:var(--muted)">'+m+'</div>'})
    .replace(/^- (.+)$/gm,'<div style="padding-left:16px">• $1</div>')
    .replace(/\{\{[^}]+\}\}/g,'<span style="color:var(--accent);font-style:italic">$&</span>')
    .replace(/\n\n/g,'<br><br>')
    .replace(/\n/g,'<br>');
}
async function saveTemplate(){
  if(!activeTemplateId)return;
  const content=document.getElementById('template-editor').value;
  const t=cachedTemplates.find(x=>x.id===activeTemplateId);
  await api('/api/templates/'+activeTemplateId,{method:'PUT',body:JSON.stringify({name:t?t.name:'',content:content})});
  toast('Template saved','success');
  loadTemplates();
}
function showNewTemplate(){
  showModal('New Template',
    '<label>Name</label><input id="m-tname" placeholder="My Report Template">'+
    '<label>Format</label><select id="m-tformat"><option value="markdown">Markdown</option><option value="html">HTML</option></select>',
    async()=>{
      await post('/api/templates',{name:gv('m-tname'),format:gv('m-tformat'),content:'# '+gv('m-tname')+'\n\nStart writing your template here.\n'});
      closeModal();loadTemplates();toast('Template created','success');
    }
  );
}
document.addEventListener('keydown',e=>{
  if((e.ctrlKey||e.metaKey)&&e.key==='s'&&activeTemplateId&&document.getElementById('pg-templates').classList.contains('active')){
    e.preventDefault();saveTemplate();
  }
});

// Template scroll sync (VSCode-like)
(function(){
  const ed=document.getElementById('template-editor');
  const pv=document.getElementById('template-preview');
  let syncing=false;
  ed.addEventListener('scroll',()=>{
    if(syncing)return;syncing=true;
    const pct=ed.scrollTop/(ed.scrollHeight-ed.clientHeight||1);
    pv.scrollTop=pct*(pv.scrollHeight-pv.clientHeight);
    syncing=false;
  });
  pv.addEventListener('scroll',()=>{
    if(syncing)return;syncing=true;
    const pct=pv.scrollTop/(pv.scrollHeight-pv.clientHeight||1);
    ed.scrollTop=pct*(ed.scrollHeight-ed.clientHeight);
    syncing=false;
  });
})();

// ===== AUDIT =====
async function loadAudit(){
  const a=await api('/api/audit');
  if(!a)return;
  cachedAudit=a;
  renderAudit(a);
}
function renderAudit(a){
  document.getElementById('audit-table').innerHTML=(a||[]).map(x=>{
    let detail='';
    try{
      const d=typeof x.Detail==='string'?JSON.parse(x.Detail):x.Detail;
      if(d&&typeof d==='object')detail=Object.entries(d).map(([k,v])=>esc(k)+': '+esc(String(v))).join(', ');
      else detail=esc(String(x.Detail||''));
    }catch(e){detail=esc(String(x.Detail||''))}
    return '<tr><td class="mono" style="font-variant-numeric:tabular-nums">'+fmtTime(x.Timestamp)+'</td><td>'+esc(x.Operator)+'</td><td><span class="tag">'+esc(x.Action)+'</span></td><td>'+esc(x.TargetType+(x.TargetID?' #'+x.TargetID:''))+'</td><td style="color:var(--text-2);font-size:11px;max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+detail+'</td></tr>';
  }).join('')||'<tr><td colspan="5" style="color:var(--muted);text-align:center;padding:20px">No audit entries</td></tr>';
}
document.getElementById('audit-filters').addEventListener('click',e=>{
  const chip=e.target.closest('.filter-chip');
  if(!chip)return;
  document.querySelectorAll('#audit-filters .filter-chip').forEach(c=>c.classList.remove('active'));
  chip.classList.add('active');
  const f=chip.dataset.filter;
  if(f==='all')renderAudit(cachedAudit);
  else renderAudit(cachedAudit.filter(x=>x.Action&&x.Action.includes(f)));
});

// ===== DETAIL PANEL =====
function openDetail(){document.getElementById('detail-panel').classList.add('open')}
function closeDetail(){document.getElementById('detail-panel').classList.remove('open')}

// ===== MODAL =====
function showModal(title,body,onSubmit){
  document.getElementById('modal-root').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h3>'+title+'</h3>'+body+'<div class="modal-actions"><button class="btn" onclick="closeModal()">Cancel</button><button class="btn primary" id="modal-submit">Save</button></div></div></div>';
  document.getElementById('modal-submit').onclick=async()=>{try{await onSubmit()}catch(e){toast('Error: '+e.message,'error')}};
}
function closeModal(){document.getElementById('modal-root').innerHTML=''}
function gv(id){return document.getElementById(id).value}

// ===== LIGHTBOX =====
function openLightbox(src,caption){
  document.getElementById('lightbox-root').innerHTML=
    '<div class="lightbox-bg" onclick="closeLightbox()">'+
    '<div class="lightbox-close" onclick="closeLightbox()">×</div>'+
    '<img src="'+src+'" alt="Screenshot">'+
    (caption?'<div class="lightbox-caption">'+esc(caption)+'</div>':'')+
    '</div>';
}
function closeLightbox(){document.getElementById('lightbox-root').innerHTML=''}
function renderAttachments(list){
  if(!list||!list.length)return'<span style="color:var(--muted)">No attachments</span>';
  return'<div class="attach-grid">'+list.map(a=>{
    const isImg=a.Filetype==='screenshot';
    const src='/api/attachments/'+a.ID;
    if(isImg)return'<div><div class="attach-thumb" onclick="openLightbox(\''+src+'\',\''+esc(a.Caption||a.Filename)+'\')"><img src="'+src+'" loading="lazy"></div><div class="attach-name">'+esc(a.Filename)+'</div></div>';
    return'<div><div class="attach-thumb" onclick="window.open(\''+src+'\',\'_blank\')"><div class="attach-icon">📎</div></div><div class="attach-name">'+esc(a.Filename)+'</div></div>';
  }).join('')+'</div>';
}
async function loadAttachments(evID){
  const list=await api('/api/attachments?evidence_id='+evID);
  return list||[];
}
async function loadFindingAttachments(findingID){
  const list=await api('/api/attachments?finding_id='+findingID);
  return list||[];
}

// ===== EXPORT =====
function showExport(){
  showModal('Export',
    '<label>Format</label><select id="m-fmt"><option value="html">HTML Report</option><option value="md">Markdown Report</option><option value="exec-html">Executive Summary (HTML)</option><option value="tech-html">Technical Details (HTML)</option></select>'+
    '<label>Options</label><div style="margin-top:6px"><label style="display:flex;align-items:center;gap:6px;font-size:12px;text-transform:none;color:var(--text)"><input type="checkbox" id="m-verified"> Verified findings only</label></div>'
  ,async()=>{
    const fmt=gv('m-fmt');const vo=document.getElementById('m-verified').checked;
    let url='/api/report?format='+(fmt.includes('html')?'html':'md');
    if(vo)url+='&verified=true';
    if(fmt==='exec-html')url+='&exec=true';
    if(fmt==='tech-html')url+='&tech=true';
    window.open(url+(KEY?'&api_key='+KEY:''));
    closeModal();
  });
}

// ===== SEARCH =====
function showSearch(){
  showModal('Search evidence',
    '<label>Keyword</label><input id="m-search" placeholder="Search commands, output, tags..." autofocus>'
  ,async()=>{
    const q=gv('m-search').toLowerCase();
    closeModal();goPage('evidence');
    setTimeout(()=>{
      const filtered=cachedEvidence.filter(e=>(e.input||'').toLowerCase().includes(q)||(e.output||'').toLowerCase().includes(q)||(e.tags||[]).some(t=>t.toLowerCase().includes(q)));
      renderEvidence(filtered);
      document.getElementById('evidence-count').textContent=filtered.length+' results for "'+q+'"';
    },100);
  });
}

// ===== TOPOLOGY =====
async function loadTopology(){
  var scopeData=await api('/api/scope')||{};
  var scope=(scopeData.hosts)||[];
  var creds=await api('/api/creds')||[];
  var findings=await api('/api/findings')||[];
  var canvas=document.getElementById('topo-canvas');
  var W=canvas.offsetWidth||700,H=Math.max(400,scope.length*40+100);
  canvas.style.height=H+'px';

  var nodes=[],edges=[];
  var hostSet={};
  scope.forEach(s=>{hostSet[s.Host]=s});
  creds.forEach(c=>{if(c.Host&&!hostSet[c.Host])hostSet[c.Host]={Host:c.Host,Tested:0}});

  var hosts=Object.values(hostSet);
  if(!hosts.length){
    canvas.innerHTML='<div style="display:flex;align-items:center;justify-content:center;height:100%;color:var(--muted);font-size:13px">No hosts in scope. Add hosts with <code style="margin:0 4px">rt scope</code></div>';
    return;
  }

  var centerX=W/2,centerY=H/2;
  var radius=Math.min(W,H)/2-60;
  if(radius<80)radius=80;
  hosts.forEach(function(h,i){
    var angle=(2*Math.PI*i/hosts.length)-Math.PI/2;
    var x=centerX+radius*Math.cos(angle);
    var y=centerY+radius*Math.sin(angle);
    var hostCreds=creds.filter(c=>c.Host===h.Host).length;
    var hostFindings=findings.filter(f=>(f.Mitre||[]).length>0).length;
    nodes.push({id:h.Host,x:x,y:y,tested:h.Tested,creds:hostCreds,host:h});
  });

  // Edges: connect hosts that share credentials (same username)
  var userHosts={};
  creds.forEach(c=>{
    if(!c.Host||!c.Username)return;
    if(!userHosts[c.Username])userHosts[c.Username]=new Set();
    userHosts[c.Username].add(c.Host);
  });
  Object.values(userHosts).forEach(hset=>{
    var arr=Array.from(hset);
    for(var i=0;i<arr.length;i++){
      for(var j=i+1;j<arr.length;j++){
        edges.push({from:arr[i],to:arr[j]});
      }
    }
  });

  var svg='<svg width="'+W+'" height="'+H+'" xmlns="http://www.w3.org/2000/svg">';

  // Edges
  edges.forEach(e=>{
    var n1=nodes.find(n=>n.id===e.from);
    var n2=nodes.find(n=>n.id===e.to);
    if(n1&&n2)svg+='<line class="topo-edge" x1="'+n1.x+'" y1="'+n1.y+'" x2="'+n2.x+'" y2="'+n2.y+'"/>';
  });

  // Nodes
  nodes.forEach(n=>{
    var r=n.creds>0?22:16;
    var fill=n.tested?'var(--green)':'var(--yellow)';
    if(n.creds>0)fill='var(--red)';
    var strokeCol='var(--border)';
    svg+='<g class="topo-node" onclick="topoClick(\''+esc(n.id)+'\')">';
    svg+='<circle cx="'+n.x+'" cy="'+n.y+'" r="'+r+'" fill="'+fill+'" fill-opacity=".15" stroke="'+strokeCol+'" stroke-width="1.5"/>';
    svg+='<circle cx="'+n.x+'" cy="'+n.y+'" r="4" fill="'+fill+'"/>';
    svg+='<text x="'+n.x+'" y="'+(n.y+r+14)+'">'+esc(n.id)+'</text>';
    if(n.creds>0)svg+='<text x="'+(n.x+r-2)+'" y="'+(n.y-r+6)+'" font-size="9" fill="var(--red)">'+n.creds+' cred'+(n.creds>1?'s':'')+'</text>';
    svg+='</g>';
  });

  svg+='</svg>';

  var legend='<div class="topo-legend">';
  legend+='<div class="leg-item"><strong>'+hosts.length+'</strong> hosts</div>';
  legend+='<div class="leg-item"><div class="leg-circle" style="background:var(--green);opacity:.4"></div> Tested</div>';
  legend+='<div class="leg-item"><div class="leg-circle" style="background:var(--yellow);opacity:.4"></div> Untested</div>';
  legend+='<div class="leg-item"><div class="leg-circle" style="background:var(--red);opacity:.4"></div> Has credentials</div>';
  legend+='<div class="leg-item">Lines = shared credentials</div>';
  legend+='</div>';
  canvas.innerHTML=legend+svg;
}
function topoClick(host){
  goPage('scope');
}

// ===== ATT&CK MAP =====
var TACTICS=[
  {id:'TA0043',name:'Reconnaissance',color:'#6366f1'},
  {id:'TA0042',name:'Resource Dev',color:'#8b5cf6'},
  {id:'TA0001',name:'Initial Access',color:'#ec4899'},
  {id:'TA0002',name:'Execution',color:'#ef4444'},
  {id:'TA0003',name:'Persistence',color:'#f97316'},
  {id:'TA0004',name:'Priv Escalation',color:'#eab308'},
  {id:'TA0005',name:'Defense Evasion',color:'#84cc16'},
  {id:'TA0006',name:'Credential Access',color:'#22c55e'},
  {id:'TA0007',name:'Discovery',color:'#14b8a6'},
  {id:'TA0008',name:'Lateral Movement',color:'#06b6d4'},
  {id:'TA0009',name:'Collection',color:'#3b82f6'},
  {id:'TA0011',name:'C2',color:'#6366f1'},
  {id:'TA0010',name:'Exfiltration',color:'#a78bfa'},
  {id:'TA0040',name:'Impact',color:'#f43f5e'}
];
var TECHNIQUE_TACTIC={
  'T1595':'TA0043','T1592':'TA0043','T1589':'TA0043','T1590':'TA0043','T1591':'TA0043',
  'T1583':'TA0042','T1584':'TA0042','T1587':'TA0042','T1588':'TA0042','T1608':'TA0042',
  'T1190':'TA0001','T1133':'TA0001','T1078':'TA0001','T1566':'TA0001','T1195':'TA0001','T1199':'TA0001','T1091':'TA0001',
  'T1059':'TA0002','T1053':'TA0002','T1203':'TA0002','T1047':'TA0002','T1569':'TA0002',
  'T1098':'TA0003','T1136':'TA0003','T1543':'TA0003','T1053':'TA0003','T1505':'TA0003',
  'T1548':'TA0004','T1134':'TA0004','T1068':'TA0004','T1078':'TA0004','T1055':'TA0004',
  'T1140':'TA0005','T1070':'TA0005','T1036':'TA0005','T1027':'TA0005','T1218':'TA0005','T1562':'TA0005',
  'T1110':'TA0006','T1003':'TA0006','T1552':'TA0006','T1555':'TA0006','T1556':'TA0006','T1558':'TA0006','T1539':'TA0006',
  'T1087':'TA0007','T1046':'TA0007','T1135':'TA0007','T1082':'TA0007','T1083':'TA0007','T1018':'TA0007',
  'T1021':'TA0008','T1550':'TA0008','T1080':'TA0008','T1563':'TA0008','T1570':'TA0008',
  'T1560':'TA0009','T1119':'TA0009','T1005':'TA0009','T1039':'TA0009','T1074':'TA0009',
  'T1071':'TA0011','T1105':'TA0011','T1572':'TA0011','T1090':'TA0011','T1219':'TA0011',
  'T1041':'TA0010','T1048':'TA0010','T1567':'TA0010',
  'T1486':'TA0040','T1489':'TA0040','T1490':'TA0040','T1498':'TA0040','T1496':'TA0040',
  'T1557':'TA0006','T1040':'TA0006',
  'T1134':'TA0004','T1055':'TA0004',
  'T1082':'TA0007','T1049':'TA0007',
  'T1569':'TA0002','T1059':'TA0002',
  'T1562':'TA0005','T1553':'TA0005',
  'T1547':'TA0003','T1546':'TA0003',
  'T1071':'TA0011',
  'T1485':'TA0040','T1491':'TA0040',
  'T1557':'TA0006','T1040':'TA0006',
  'T1021':'TA0008',
  'T1560':'TA0009',
  'T1041':'TA0010',
  'T1190':'TA0001','T1133':'TA0001'
};
function getTacticForTechnique(tid){
  var base=tid.replace(/\.\d+$/,'');
  return TECHNIQUE_TACTIC[base]||null;
}
async function loadAttackMap(){
  var findings=await api('/api/findings');
  if(!findings)findings=[];
  var mapped={};
  TACTICS.forEach(t=>{mapped[t.id]=[]});
  findings.forEach(f=>{
    if(!f.Mitre||!f.Mitre.length)return;
    f.Mitre.forEach(tid=>{
      var tactic=getTacticForTechnique(tid);
      if(tactic&&mapped[tactic]){
        mapped[tactic].push({technique:tid,title:f.Title,priority:f.Priority,verified:f.Verified,id:f.ID});
      }
    });
  });
  var totalMapped=0;
  TACTICS.forEach(t=>{totalMapped+=mapped[t.id].length});
  var html='<div class="attack-legend">';
  html+='<div class="leg-item"><strong>'+totalMapped+'</strong> techniques mapped from <strong>'+findings.length+'</strong> findings</div>';
  html+='<div class="leg-item"><div class="leg-dot" style="background:var(--red)"></div> Critical</div>';
  html+='<div class="leg-item"><div class="leg-dot" style="background:var(--orange)"></div> High</div>';
  html+='<div class="leg-item"><div class="leg-dot" style="background:var(--yellow)"></div> Medium</div>';
  html+='<div class="leg-item"><div class="leg-dot" style="background:var(--blue)"></div> Low</div>';
  html+='</div>';
  html+='<div class="attack-grid">';
  TACTICS.forEach(t=>{
    var items=mapped[t.id];
    html+='<div class="attack-col">';
    html+='<div class="attack-hdr" style="background:'+t.color+'">'+esc(t.name)+'</div>';
    html+='<div class="attack-cards">';
    if(!items.length){
      html+='<div class="attack-empty">No findings</div>';
    }else{
      items.forEach(item=>{
        html+='<div class="attack-card" onclick="goPage(\'findings\');setTimeout(()=>showFindingDetail('+item.id+'),300)">';
        html+='<div class="ac-tech">'+esc(item.technique)+'</div>';
        html+='<div class="ac-title">'+esc(item.title)+'</div>';
        html+='<div class="ac-sev">'+sevBadge(item.priority)+' '+statusSpan(item.verified)+'</div>';
        html+='</div>';
      });
    }
    html+='</div></div>';
  });
  html+='</div>';
  document.getElementById('attack-map').innerHTML=html;
}

// ===== COMMENTS =====
async function loadComments(findingID){
  const cs=await api('/api/findings/'+findingID+'/comments');
  const el=document.getElementById('comment-thread-'+findingID);
  if(!el)return;
  if(!cs||!cs.length){el.innerHTML='<div style="color:var(--muted);font-size:11px">No comments yet</div>';return;}
  el.innerHTML=cs.map(c=>
    '<div class="comment-item"><div class="comment-meta"><span class="comment-author">'+esc(c.operator)+'</span><span><span class="comment-time">'+fmtDate(c.created_at)+'</span> <button class="comment-del" onclick="deleteComment('+c.id+','+c.finding_id+')" title="Delete">x</button></span></div><div class="comment-text">'+esc(c.content)+'</div></div>'
  ).join('');
}
async function addComment(findingID){
  const inp=document.getElementById('comment-input-'+findingID);
  if(!inp)return;
  const txt=inp.value.trim();
  if(!txt)return;
  await post('/api/findings/'+findingID+'/comments',{content:txt});
  inp.value='';
  loadComments(findingID);
}
async function deleteComment(commentID,findingID){
  if(!confirm('Delete comment?'))return;
  await del('/api/findings/'+findingID+'/comments/'+commentID);
  loadComments(findingID);
}

// ===== PRESENCE =====
var wsRef=null;
function updatePresence(operators){
  var ind=document.getElementById('presence-ind');
  var cnt=document.getElementById('presence-count');
  var list=document.getElementById('presence-list');
  if(!operators||!operators.length){ind.style.display='none';return;}
  ind.style.display='flex';
  cnt.textContent=operators.length;
  list.innerHTML=operators.map(o=>'<div>'+esc(o)+'</div>').join('');
}

// ===== WEBSOCKET =====
function connectWS(){
  const proto=location.protocol==='https:'?'wss:':'ws:';
  const url=proto+'//'+location.host+'/ws/live'+(KEY?'?api_key='+KEY:'');
  const ws=new WebSocket(url);
  const badge=document.getElementById('ws-badge');
  ws.onopen=()=>{
    badge.textContent='live';badge.className='topbar-status ok';
    wsRef=ws;
    ws.send(JSON.stringify({type:'presence',operator:'operator'}));
  };
  ws.onclose=()=>{badge.textContent='offline';badge.className='topbar-status warn';wsRef=null;updatePresence([]);setTimeout(connectWS,3000)};
  ws.onmessage=(e)=>{
    try{
      const d=JSON.parse(e.data);
      if(d.type==='presence'){updatePresence(d.operators);return;}
      if(d.type==='comment.new'){
        var ct=document.getElementById('comment-thread-'+d.finding_id);
        if(ct)loadComments(d.finding_id);
        return;
      }
      if(d.type==='evidence')toast('New evidence: '+d.input,'');
      else if(d.type&&d.type.startsWith('finding.'))toast('Finding '+d.type.split('.')[1]+' #'+d.id,'');
      else if(d.type&&d.type.startsWith('cred.'))toast('Credential '+d.type.split('.')[1],'');
      if(currentPage==='overview')loadOverview();
      else if(currentPage==='evidence'&&d.type==='evidence')loadEvidence();
      else if(currentPage==='findings'&&d.type&&d.type.startsWith('finding.'))loadFindings();
      else if(currentPage==='creds'&&d.type&&d.type.startsWith('cred.'))loadCreds();
    }catch(err){}
  };
}
connectWS();

// ===== KEYBOARD SHORTCUTS =====
var shortcutPages=['overview','evidence','findings','creds','scope','checklist','sessions','attack','topology'];
document.addEventListener('keydown',function(e){
  if(e.target.tagName==='INPUT'||e.target.tagName==='TEXTAREA'||e.target.tagName==='SELECT'||e.target.isContentEditable)return;
  if(e.key==='Escape'){
    if(document.getElementById('modal-root').innerHTML){closeModal();return;}
    if(document.getElementById('detail-panel').classList.contains('open')){closeDetail();return;}
    return;
  }
  if(e.key==='/'){e.preventDefault();showSearch();return;}
  if(e.key==='?'){showShortcuts();return;}
  if(e.key==='n'&&!e.ctrlKey&&!e.metaKey){
    if(currentPage==='findings'){showNewFinding();return;}
    if(currentPage==='creds'){showNewCred();return;}
  }
  var num=parseInt(e.key);
  if(num>=1&&num<=shortcutPages.length){
    goPage(shortcutPages[num-1]);return;
  }
});
function showShortcuts(){
  showModal('Keyboard shortcuts',
    '<table style="width:100%;font-size:12px">'+
    '<tr><td style="padding:4px 8px;font-family:var(--font-mono);color:var(--accent)">1-9</td><td style="padding:4px 8px">Navigate pages</td></tr>'+
    '<tr><td style="padding:4px 8px;font-family:var(--font-mono);color:var(--accent)">/</td><td style="padding:4px 8px">Focus search</td></tr>'+
    '<tr><td style="padding:4px 8px;font-family:var(--font-mono);color:var(--accent)">n</td><td style="padding:4px 8px">New finding / credential</td></tr>'+
    '<tr><td style="padding:4px 8px;font-family:var(--font-mono);color:var(--accent)">Esc</td><td style="padding:4px 8px">Close panel / modal</td></tr>'+
    '<tr><td style="padding:4px 8px;font-family:var(--font-mono);color:var(--accent)">?</td><td style="padding:4px 8px">Show shortcuts</td></tr>'+
    '<tr><td style="padding:4px 8px;font-family:var(--font-mono);color:var(--accent)">Ctrl+S</td><td style="padding:4px 8px">Save template (in editor)</td></tr>'+
    '</table>'
  ,function(){closeModal()});
  document.getElementById('modal-submit').style.display='none';
}

// Footer clock
setInterval(()=>{document.getElementById('footer-time').textContent=new Date().toLocaleTimeString('en-GB')},1000);

// Initial load
loadOverview();
</script></body></html>`)
	return b.String()
}
