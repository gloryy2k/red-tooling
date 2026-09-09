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
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f0f23;color:#e0e0e0;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.box{background:#1a1a2e;padding:2rem;border-radius:12px;width:360px;box-shadow:0 8px 32px rgba(0,0,0,.4)}
h1{color:#e94560;margin:0 0 1.5rem;font-size:1.5rem;text-align:center}
input{width:100%;padding:12px;box-sizing:border-box;background:#16213e;border:1px solid #333;color:#e0e0e0;border-radius:6px;font-size:1rem;margin-bottom:1rem}
button{width:100%;padding:12px;background:#e94560;color:#fff;border:none;border-radius:6px;font-size:1rem;cursor:pointer}
button:hover{background:#c73650}
.err{color:#e94560;text-align:center;margin-top:.5rem;display:none}
</style></head><body>
<div class="box">
<h1>RT Dashboard</h1>
<input type="password" id="key" placeholder="API Key (rt_key_...)" autofocus>
<button onclick="login()">Login</button>
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
:root{--bg:#0f0f23;--surface:#1a1a2e;--surface-2:#16213e;--border:#2a2a4a;--text:#e0e0e0;--muted:#888;--accent:#e94560;--green:#28a745;--yellow:#f0a500;--blue:#17a2b8;--red:#dc3545;--teal:#00d4aa;--purple:#a78bfa}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:var(--bg);color:var(--text);line-height:1.5}
nav{background:var(--surface);border-bottom:1px solid var(--border);padding:.5rem 1rem;display:flex;align-items:center;gap:.5rem;position:sticky;top:0;z-index:100;flex-wrap:wrap}
nav .brand{color:var(--accent);font-weight:bold;font-size:1.2rem;text-decoration:none;margin-right:.5rem}
nav a{color:var(--muted);text-decoration:none;padding:.25rem .5rem;border-radius:4px;font-size:.85rem}
nav a:hover,nav a.active{color:var(--text);background:var(--border)}
.container{max-width:1400px;margin:0 auto;padding:1rem}
h2{color:var(--accent);margin-bottom:1rem;font-size:1.2rem}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(130px,1fr));gap:.75rem;margin-bottom:1.5rem}
.stat{background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:.75rem;text-align:center}
.stat .n{font-size:1.8rem;font-weight:bold;display:block}
.stat .l{font-size:.7rem;text-transform:uppercase;color:var(--muted);margin-top:.2rem}
.critical{color:var(--red)}.high{color:var(--accent)}.medium{color:var(--yellow)}.low{color:var(--blue)}.info{color:var(--muted)}
.confirmed{color:var(--green)}.false-positive{color:var(--muted)}.unverified{color:var(--yellow)}
table{width:100%;border-collapse:collapse;background:var(--surface);border-radius:8px;overflow:hidden;margin-bottom:1rem}
th{background:var(--surface-2);padding:8px 10px;text-align:left;font-size:.75rem;text-transform:uppercase;color:var(--muted);border-bottom:1px solid var(--border)}
td{padding:6px 10px;border-bottom:1px solid var(--border);font-size:.85rem;max-width:350px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
tr:hover{background:rgba(233,69,96,.05)}
code{background:var(--surface-2);padding:2px 6px;border-radius:3px;font-size:.85em}
.badge{display:inline-block;padding:2px 8px;border-radius:10px;font-size:.75rem;font-weight:bold}
.badge.critical{background:rgba(220,53,69,.2)}.badge.high{background:rgba(233,69,96,.2)}
.badge.medium{background:rgba(240,165,0,.2)}.badge.low{background:rgba(23,162,184,.2)}
.live-dot{width:8px;height:8px;background:var(--green);border-radius:50%;display:inline-block;margin-right:4px;animation:pulse 2s infinite}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.3}}
.page{display:none}.page.active{display:block}
.feed-item{background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:.75rem 1rem;margin-bottom:.5rem;border-left:3px solid var(--border)}
.feed-item.critical{border-left-color:var(--red)}.feed-item.high{border-left-color:var(--accent)}
.feed-item .ts{color:var(--muted);font-size:.8rem}.feed-item .cmd{font-weight:bold}
.feed-item .out{color:var(--muted);font-size:.85rem;white-space:pre-wrap;max-height:80px;overflow:hidden}
.tags span{background:var(--border);padding:1px 6px;border-radius:3px;font-size:.75rem;margin-right:4px}
#ws-status{font-size:.75rem;margin-left:auto}
.connected{color:var(--green)}.disconnected{color:var(--red)}
.btn{padding:6px 14px;border:none;border-radius:6px;font-size:.85rem;cursor:pointer;font-weight:500;display:inline-flex;align-items:center;gap:4px}
.btn-primary{background:var(--accent);color:#fff}.btn-primary:hover{background:#c73650}
.btn-sm{padding:4px 10px;font-size:.75rem}
.btn-teal{background:var(--teal);color:#000}.btn-teal:hover{opacity:.8}
.btn-outline{background:transparent;border:1px solid var(--border);color:var(--text)}.btn-outline:hover{background:var(--border)}
.modal-bg{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.6);z-index:200;display:flex;align-items:center;justify-content:center}
.modal{background:var(--surface);border:1px solid var(--border);border-radius:12px;padding:1.5rem;width:480px;max-width:95vw;max-height:80vh;overflow-y:auto}
.modal h3{color:var(--accent);margin-bottom:1rem}
.modal label{display:block;font-size:.85rem;color:var(--muted);margin-bottom:.25rem;margin-top:.75rem}
.modal input,.modal select,.modal textarea{width:100%;padding:8px 10px;background:var(--surface-2);border:1px solid var(--border);color:var(--text);border-radius:6px;font-size:.9rem}
.modal textarea{min-height:80px;resize:vertical}
.modal-actions{margin-top:1rem;display:flex;justify-content:flex-end;gap:.5rem}
.scope-bar{height:8px;background:var(--surface-2);border-radius:4px;overflow:hidden;margin:.5rem 0}
.scope-bar-fill{height:100%;background:var(--teal);border-radius:4px;transition:width .3s}
.check-item{display:flex;align-items:center;gap:.75rem;padding:.5rem .75rem;background:var(--surface);border:1px solid var(--border);border-radius:6px;margin-bottom:.4rem}
.check-item input[type=checkbox]{width:18px;height:18px;accent-color:var(--teal)}
.check-item.done .check-text{text-decoration:line-through;color:var(--muted)}
.check-cat{font-size:.7rem;text-transform:uppercase;color:var(--purple);font-weight:bold;margin-top:1rem;margin-bottom:.4rem}
.toolbar{display:flex;gap:.5rem;margin-bottom:1rem;flex-wrap:wrap;align-items:center}
</style></head><body>
<nav>
<a href="#" class="brand">RT</a>
<a href="#" data-page="overview" class="active">Overview</a>
<a href="#" data-page="live">Live</a>
<a href="#" data-page="findings">Findings</a>
<a href="#" data-page="timeline">Timeline</a>
<a href="#" data-page="scope">Scope</a>
<a href="#" data-page="checklist">Checklist</a>
<a href="#" data-page="creds">Creds</a>
<a href="#" data-page="sessions">Sessions</a>
<a href="#" data-page="audit">Audit</a>
<span id="ws-status" class="disconnected">disconnected</span>
</nav>
<div class="container">

<div id="overview" class="page active">
<h2>` + engName + ` — Overview</h2>
<div class="stats" id="stats-grid"></div>
<h2>Top Findings</h2>
<table><thead><tr><th>#</th><th>Priority</th><th>Status</th><th>Title</th></tr></thead><tbody id="top-findings"></tbody></table>
</div>

<div id="live" class="page">
<h2><span class="live-dot"></span> Live Feed</h2>
<div id="live-feed"></div>
</div>

<div id="findings" class="page">
<div class="toolbar">
<h2 style="margin:0">Findings</h2>
<button class="btn btn-primary" onclick="showNewFinding()">+ New Finding</button>
</div>
<table><thead><tr><th>ID</th><th>Priority</th><th>Status</th><th>Title</th><th>MITRE</th><th>Actions</th></tr></thead><tbody id="findings-table"></tbody></table>
</div>

<div id="timeline" class="page">
<h2>Timeline</h2>
<table><thead><tr><th>Time</th><th>Action</th><th>Input</th><th>Output</th><th>Tags</th></tr></thead><tbody id="timeline-table"></tbody></table>
</div>

<div id="scope" class="page">
<div class="toolbar">
<h2 style="margin:0">Scope</h2>
<button class="btn btn-primary" onclick="showAddScope()">+ Add Hosts</button>
</div>
<div id="scope-summary"></div>
<table><thead><tr><th>Host</th><th>Status</th><th>Tested By</th><th>Session</th><th>Actions</th></tr></thead><tbody id="scope-table"></tbody></table>
</div>

<div id="checklist" class="page">
<div class="toolbar">
<h2 style="margin:0">Checklist</h2>
<button class="btn btn-primary" onclick="showLoadChecklist()">Load PTES</button>
<button class="btn btn-outline" onclick="showAddCheckItem()">+ Custom Item</button>
</div>
<div id="checklist-summary"></div>
<div id="checklist-items"></div>
</div>

<div id="creds" class="page">
<h2>Credentials</h2>
<table><thead><tr><th>Username</th><th>Type</th><th>Host</th><th>Source</th><th>Found At</th></tr></thead><tbody id="creds-table"></tbody></table>
</div>

<div id="sessions" class="page">
<h2>Sessions</h2>
<table><thead><tr><th>ID</th><th>Name</th><th>Source</th><th>Status</th><th>Started</th></tr></thead><tbody id="sessions-table"></tbody></table>
</div>

<div id="audit" class="page">
<h2>Audit Log</h2>
<table><thead><tr><th>Time</th><th>Operator</th><th>Action</th><th>Target</th><th>Detail</th></tr></thead><tbody id="audit-table"></tbody></table>
</div>

</div>
<div id="modal-root"></div>
<script>
const KEY=localStorage.getItem('rt_api_key')||'';
const H={'X-API-Key':KEY,'Content-Type':'application/json'};
function esc(s){if(!s)return'';const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
function fmtTime(ts){try{return new Date(ts).toLocaleString('sv',{hour:'2-digit',minute:'2-digit',second:'2-digit'})}catch(e){return ts}}

// Navigation
document.querySelectorAll('nav a[data-page]').forEach(a=>{
  a.addEventListener('click',e=>{
    e.preventDefault();
    document.querySelectorAll('nav a').forEach(x=>x.classList.remove('active'));
    a.classList.add('active');
    document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
    document.getElementById(a.dataset.page).classList.add('active');
    loadPage(a.dataset.page);
  });
});

async function api(path,opts){
  const r=await fetch(path,{headers:H,...(opts||{})});
  if(r.status===401){location.href='/login';return null;}
  if(!r.ok){const e=await r.json().catch(()=>({error:'request failed'}));throw new Error(e.error||'failed');}
  return r.json();
}
async function post(path,body){return api(path,{method:'POST',body:JSON.stringify(body)})}

async function loadPage(page){
  const loaders={overview:loadOverview,findings:loadFindings,timeline:loadTimeline,sessions:loadSessions,audit:loadAudit,scope:loadScope,checklist:loadChecklist,creds:loadCreds};
  if(loaders[page])loaders[page]();
}

function badge(p){return '<span class="badge '+esc(p)+'">'+esc(p)+'</span>'}
function stat(n,l,c){return '<div class="stat"><span class="n '+(c||'')+'">'+n+'</span><span class="l">'+esc(l)+'</span></div>'}
function pct(a,b){return b===0?0:Math.round(a/b*100)}

// ===== OVERVIEW =====
async function loadOverview(){
  const d=await api('/api/overview');
  if(!d)return;
  const s=d.stats;
  document.getElementById('stats-grid').innerHTML=
    stat(s.Critical,'Critical','critical')+stat(s.High,'High','high')+
    stat(s.Medium,'Medium','medium')+stat(s.Low,'Low','low')+
    stat(s.TotalFindings,'Findings','')+stat(s.Confirmed,'Confirmed','confirmed')+
    stat(s.TotalEvidence,'Evidence','')+stat(d.session_count,'Sessions','')+
    stat(d.cred_count,'Credentials','')+
    stat(pct(d.scope_tested,d.scope_total)+'%','Scope Tested','teal')+
    stat(d.checklist_done,'Checklist Done','');
  const f=await api('/api/findings');
  if(!f)return;
  document.getElementById('top-findings').innerHTML=(f||[]).slice(0,10).map((x,i)=>
    '<tr><td>'+(i+1)+'</td><td>'+badge(x.Priority)+'</td><td><span class="'+esc(x.Verified)+'">'+esc(x.Verified)+'</span></td><td>'+esc(x.Title)+'</td></tr>'
  ).join('');
}

// ===== FINDINGS (with CRUD) =====
async function loadFindings(){
  const f=await api('/api/findings');
  if(!f)return;
  document.getElementById('findings-table').innerHTML=(f||[]).map(x=>
    '<tr><td>'+x.ID+'</td><td>'+badge(x.Priority)+'</td><td><span class="'+esc(x.Verified)+'">'+esc(x.Verified)+'</span></td><td>'+esc(x.Title)+'</td><td>'+
    (x.Mitre||[]).map(m=>'<code>'+esc(m)+'</code>').join(' ')+'</td><td>'+
    '<button class="btn btn-sm btn-outline" onclick="showVerify('+x.ID+')">Verify</button> '+
    '<button class="btn btn-sm btn-outline" onclick="showRecommend('+x.ID+')">Recommend</button>'+
    '</td></tr>'
  ).join('');
}

function showNewFinding(){
  showModal('New Finding',
    '<label>Title</label><input id="m-title" placeholder="Finding title">'+
    '<label>Priority</label><select id="m-priority"><option>critical</option><option>high</option><option selected>medium</option><option>low</option><option>info</option></select>'+
    '<label>Description</label><textarea id="m-desc" placeholder="Description"></textarea>'+
    '<label>MITRE ATT&CK IDs (comma-separated)</label><input id="m-mitre" placeholder="T1190, T1210">'
  ,async()=>{
    await post('/api/findings',{
      title:gv('m-title'),priority:gv('m-priority'),description:gv('m-desc'),
      mitre:gv('m-mitre').split(',').map(s=>s.trim()).filter(Boolean)
    });
    closeModal();loadFindings();
  });
}

function showVerify(id){
  showModal('Verify Finding #'+id,
    '<label>Status</label><select id="m-vstatus"><option>confirmed</option><option>false-positive</option><option>unverified</option></select>'+
    '<label>Note</label><textarea id="m-vnote"></textarea>'
  ,async()=>{
    await post('/api/findings/'+id+'/verify',{status:gv('m-vstatus'),note:gv('m-vnote')});
    closeModal();loadFindings();
  });
}

function showRecommend(id){
  showModal('Add Recommendation — Finding #'+id,
    '<label>Recommendation</label><textarea id="m-rec" placeholder="Recommended remediation steps..."></textarea>'
  ,async()=>{
    await post('/api/findings/'+id+'/recommend',{recommendation:gv('m-rec')});
    closeModal();loadFindings();
  });
}

// ===== SCOPE (with CRUD) =====
async function loadScope(){
  const d=await api('/api/scope');
  if(!d)return;
  const hosts=d.hosts||[];
  document.getElementById('scope-summary').innerHTML=
    '<div style="display:flex;align-items:center;gap:1rem;margin-bottom:1rem">'+
    '<span style="font-size:1.5rem;font-weight:bold;color:var(--teal)">'+pct(d.tested,d.total)+'%</span>'+
    '<div style="flex:1"><div class="scope-bar"><div class="scope-bar-fill" style="width:'+pct(d.tested,d.total)+'%"></div></div></div>'+
    '<span style="color:var(--muted);font-size:.85rem">'+d.tested+'/'+d.total+' tested</span></div>';
  document.getElementById('scope-table').innerHTML=hosts.map(h=>
    '<tr><td><code>'+esc(h.Host)+'</code></td><td>'+(h.Tested?'<span style="color:var(--green)">Tested</span>':'<span style="color:var(--yellow)">Untested</span>')+
    '</td><td>'+esc(h.TestedAt||'')+'</td><td>'+esc(h.SessionID||'')+'</td><td>'+
    (h.Tested?'':'<button class="btn btn-sm btn-teal" onclick="markTested(\''+esc(h.Host)+'\')">Mark Tested</button>')+
    '</td></tr>'
  ).join('');
}

function showAddScope(){
  showModal('Add Scope Hosts',
    '<label>Hosts (comma-separated IPs, CIDRs, domains)</label>'+
    '<textarea id="m-hosts" placeholder="10.10.10.1, 10.10.10.2, 192.168.1.0/24"></textarea>'
  ,async()=>{
    await post('/api/scope',{hosts:gv('m-hosts')});
    closeModal();loadScope();
  });
}

async function markTested(host){
  try{await post('/api/scope/tested',{host});loadScope();}catch(e){alert('Error: '+e.message)}
}

// ===== CHECKLIST (with toggle) =====
async function loadChecklist(){
  const d=await api('/api/checklist');
  if(!d)return;
  const items=d.items||[];
  document.getElementById('checklist-summary').innerHTML=
    '<div style="display:flex;align-items:center;gap:1rem;margin-bottom:1rem">'+
    '<span style="font-size:1.5rem;font-weight:bold;color:var(--teal)">'+d.done+'/'+d.total+'</span>'+
    '<div style="flex:1"><div class="scope-bar"><div class="scope-bar-fill" style="width:'+pct(d.done,d.total)+'%"></div></div></div>'+
    '<span style="color:var(--muted);font-size:.85rem">'+pct(d.done,d.total)+'% complete</span></div>';
  let html='';
  let lastCat='';
  items.forEach(it=>{
    if(it.category!==lastCat){html+='<div class="check-cat">'+esc(it.category)+'</div>';lastCat=it.category;}
    const done=it.done;
    html+='<div class="check-item'+(done?' done':'')+'">'+
      '<input type="checkbox" '+(done?'checked':'')+' onchange="toggleCheck('+it.id+',this.checked)">'+
      '<span class="check-text">'+esc(it.item)+'</span>'+
      (it.done_at?'<span style="margin-left:auto;font-size:.75rem;color:var(--muted)">'+fmtTime(it.done_at)+'</span>':'')+
      '</div>';
  });
  document.getElementById('checklist-items').innerHTML=html||'<p style="color:var(--muted)">No checklist loaded. Click "Load PTES" to get started.</p>';
}

async function toggleCheck(id,checked){
  try{await post('/api/checklist/toggle',{id,checked});loadChecklist();}catch(e){alert('Error: '+e.message)}
}

function showLoadChecklist(){
  showModal('Load Checklist Preset',
    '<label>Preset</label><select id="m-preset"><option value="ptes">PTES (27 items)</option></select>'
  ,async()=>{
    await post('/api/checklist',{preset:gv('m-preset')});
    closeModal();loadChecklist();
  });
}

function showAddCheckItem(){
  showModal('Add Custom Checklist Item',
    '<label>Category</label><input id="m-cat" placeholder="Custom">'+
    '<label>Item</label><input id="m-item" placeholder="Task description">'
  ,async()=>{
    await post('/api/checklist',{category:gv('m-cat'),item:gv('m-item')});
    closeModal();loadChecklist();
  });
}

// ===== CREDENTIALS =====
async function loadCreds(){
  const c=await api('/api/creds');
  if(!c)return;
  document.getElementById('creds-table').innerHTML=(c||[]).map(x=>
    '<tr><td><code>'+esc(x.Username)+'</code></td><td>'+esc(x.CredType)+'</td><td><code>'+esc(x.Host||'')+'</code></td><td>'+esc(x.Source)+'</td><td>'+fmtTime(x.FoundAt)+'</td></tr>'
  ).join('');
}

// ===== TIMELINE =====
async function loadTimeline(){
  const ev=await api('/api/timeline');
  if(!ev)return;
  document.getElementById('timeline-table').innerHTML=(ev||[]).map(e=>
    '<tr><td>'+fmtTime(e.timestamp)+'</td><td>'+esc(e.action)+'</td><td><code>'+esc(e.input)+'</code></td><td>'+esc((e.output||'').substring(0,100))+'</td><td class="tags">'+
    (e.tags||[]).map(t=>'<span>'+esc(t)+'</span>').join('')+'</td></tr>'
  ).join('');
}

// ===== SESSIONS =====
async function loadSessions(){
  const s=await api('/api/sessions');
  if(!s)return;
  document.getElementById('sessions-table').innerHTML=(s||[]).map(x=>
    '<tr><td><code>'+esc((x.ID||'').substring(0,12))+'</code></td><td>'+esc(x.Name)+'</td><td>'+esc(x.Source)+'</td><td>'+esc(x.Status)+'</td><td>'+fmtTime(x.StartedAt)+'</td></tr>'
  ).join('');
}

// ===== AUDIT =====
async function loadAudit(){
  const a=await api('/api/audit');
  if(!a)return;
  document.getElementById('audit-table').innerHTML=(a||[]).map(x=>
    '<tr><td>'+fmtTime(x.Timestamp)+'</td><td>'+esc(x.Operator)+'</td><td>'+esc(x.Action)+'</td><td>'+esc(x.TargetType+':'+x.TargetID)+'</td><td>'+esc(JSON.stringify(x.Detail).substring(0,60))+'</td></tr>'
  ).join('');
}

// ===== MODAL SYSTEM =====
function showModal(title,body,onSubmit){
  const root=document.getElementById('modal-root');
  root.innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal">'+
    '<h3>'+esc(title)+'</h3>'+body+
    '<div class="modal-actions"><button class="btn btn-outline" onclick="closeModal()">Cancel</button>'+
    '<button class="btn btn-primary" id="modal-submit">Save</button></div></div></div>';
  document.getElementById('modal-submit').onclick=async()=>{
    try{await onSubmit();}catch(e){alert('Error: '+e.message)}
  };
}
function closeModal(){document.getElementById('modal-root').innerHTML='';}
function gv(id){return document.getElementById(id).value;}

// ===== WEBSOCKET =====
function connectWS(){
  const proto=location.protocol==='https:'?'wss:':'ws:';
  const url=proto+'//'+location.host+'/ws/live'+(KEY?'?api_key='+KEY:'');
  const ws=new WebSocket(url);
  const status=document.getElementById('ws-status');
  ws.onopen=()=>{status.textContent='connected';status.className='connected'};
  ws.onclose=()=>{status.textContent='disconnected';status.className='disconnected';setTimeout(connectWS,3000)};
  ws.onmessage=(e)=>{
    try{
      const d=JSON.parse(e.data);
      if(d.type==='evidence'){
        const feed=document.getElementById('live-feed');
        const item=document.createElement('div');
        item.className='feed-item '+(d.priority||'');
        item.innerHTML='<div class="ts">'+fmtTime(d.timestamp)+'</div><div class="cmd">'+esc(d.input)+'</div><div class="out">'+esc((d.output||'').substring(0,300))+'</div>';
        feed.prepend(item);
        if(feed.children.length>100)feed.lastChild.remove();
      }
    }catch(err){}
  };
}
connectWS();
loadOverview();
</script></body></html>`)
	return b.String()
}
