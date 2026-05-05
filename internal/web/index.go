package web

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>A股涨停板量化系统</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<style>
:root {
  --bg: #f0f2f5; --card: #ffffff; --border: #e4e8ee; --text: #1a1a2e;
  --text2: #5a6078; --text3: #a0a8bf; --accent: #4361ee; --accent2: #e63946;
  --green: #2ec4b6; --red: #e63946; --blue: #4361ee; --purple: #7209b7;
  --orange: #f77f00; --yellow: #fcbf49; --shadow: 0 1px 8px rgba(0,0,0,0.06);
  --radius: 10px;
}
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family: -apple-system, 'PingFang SC', 'Segoe UI', sans-serif; background:var(--bg); color:var(--text); font-size:13px; }
a { color:var(--accent); text-decoration:none; cursor:pointer; }
a:hover { text-decoration:underline; }

.header { background:linear-gradient(135deg,#1a1a2e 0%,#16213e 100%); padding:14px 24px; display:flex; align-items:center; justify-content:space-between; color:#fff; }
.header h1 { font-size:18px; font-weight:600; letter-spacing:1px; }
.header .info { color:rgba(255,255,255,0.7); font-size:12px; }

.search-box { position:relative; margin-left:16px; }
.search-box input { width:220px; padding:6px 12px 6px 32px; border:none; border-radius:20px; background:rgba(255,255,255,0.15); color:#fff; font-size:12px; outline:none; }
.search-box input::placeholder { color:rgba(255,255,255,0.5); }
.search-box input:focus { background:rgba(255,255,255,0.25); }
.search-box svg { position:absolute; left:10px; top:50%; transform:translateY(-50%); width:14px; height:14px; fill:rgba(255,255,255,0.5); }
.search-dropdown { position:absolute; top:100%; left:0; width:320px; background:var(--card); border-radius:8px; box-shadow:0 4px 20px rgba(0,0,0,0.15); z-index:1000; display:none; max-height:400px; overflow-y:auto; margin-top:4px; }
.search-dropdown.show { display:block; }
.search-item { padding:10px 14px; display:flex; align-items:center; gap:8px; cursor:pointer; border-bottom:1px solid var(--border); color:var(--text); }
.search-item:hover { background:#f5f7ff; }
.search-item .code { font-weight:600; color:var(--accent); min-width:60px; }
.search-item .name { flex:1; }
.search-item .ind { font-size:11px; color:var(--text3); }

.container { max-width:1480px; margin:0 auto; padding:12px 16px; }
.nav { display:flex; gap:2px; margin-bottom:12px; background:var(--card); padding:4px; border-radius:var(--radius); box-shadow:var(--shadow); overflow-x:auto; }
.nav-item { padding:7px 16px; border-radius:7px; cursor:pointer; font-size:12px; color:var(--text2); white-space:nowrap; transition:all 0.15s; font-weight:500; }
.nav-item:hover { background:#eef0f8; color:var(--text); }
.nav-item.active { background:var(--accent); color:#fff; }

.panel { display:none; }
.panel.active { display:block; }

.grid2 { display:grid; grid-template-columns:1fr 1fr; gap:12px; margin-bottom:12px; }
.grid3 { display:grid; grid-template-columns:1fr 1fr 1fr; gap:12px; margin-bottom:12px; }
.grid4 { display:grid; grid-template-columns:repeat(auto-fit,minmax(160px,1fr)); gap:10px; margin-bottom:12px; }
@media(max-width:768px) { .grid2,.grid3{grid-template-columns:1fr;} }

.card { background:var(--card); border-radius:var(--radius); padding:16px; box-shadow:var(--shadow); }
.card h3 { font-size:13px; font-weight:600; margin-bottom:12px; color:var(--text); display:flex; align-items:center; gap:6px; }
.card h3::before { content:''; width:3px; height:14px; background:var(--accent); border-radius:2px; }

.stat-card { background:var(--card); border-radius:var(--radius); padding:14px 16px; box-shadow:var(--shadow); }
.stat-card .label { font-size:11px; color:var(--text3); margin-bottom:4px; }
.stat-card .value { font-size:22px; font-weight:700; line-height:1.3; }
.stat-card .sub { font-size:11px; color:var(--text2); margin-top:2px; }
.up { color:var(--red); }
.down { color:var(--green); }
.neutral { color:var(--blue); }

table { width:100%; border-collapse:collapse; font-size:12px; }
thead th { background:#f8f9fc; padding:8px 10px; text-align:left; color:var(--text2); font-weight:600; border-bottom:2px solid var(--border); position:sticky; top:0; z-index:1; }
tbody td { padding:7px 10px; border-bottom:1px solid var(--border); }
tbody tr:hover { background:#f7f8fd; }
tbody tr { cursor:pointer; }
.table-wrap { max-height:500px; overflow-y:auto; border-radius:var(--radius); }

.tag { display:inline-block; padding:2px 7px; border-radius:4px; font-size:10px; font-weight:600; }
.tag-red { background:#ffeaea; color:var(--red); }
.tag-green { background:#e6fff5; color:var(--green); }
.tag-blue { background:#e8f0ff; color:var(--blue); }
.tag-purple { background:#f3eeff; color:var(--purple); }
.tag-orange { background:#fff3e0; color:var(--orange); }

.score-bar { display:inline-block; height:5px; border-radius:3px; background:linear-gradient(90deg,var(--accent),var(--red)); vertical-align:middle; margin-right:5px; }
.empty { text-align:center; padding:50px 20px; color:var(--text3); }
.badge { display:inline-block; min-width:18px; text-align:center; padding:1px 5px; border-radius:10px; font-size:10px; font-weight:600; }
.badge-up { background:#ffeaea; color:var(--red); }
.badge-down { background:#e6fff5; color:var(--green); }

.date-picker { display:flex; align-items:center; gap:8px; margin-bottom:12px; }
.date-picker select, .date-picker input { padding:6px 10px; border:1px solid var(--border); border-radius:6px; font-size:12px; background:var(--card); color:var(--text); }
.date-picker label { font-size:12px; color:var(--text2); font-weight:500; }

.stock-header { display:flex; align-items:center; gap:16px; margin-bottom:12px; flex-wrap:wrap; }
.stock-header .stock-name { font-size:20px; font-weight:700; }
.stock-header .stock-code { font-size:14px; color:var(--text2); }
.stock-header .stock-price { font-size:24px; font-weight:700; }
.stock-header .stock-chg { font-size:14px; font-weight:600; }
.concept-tags { display:flex; flex-wrap:wrap; gap:4px; margin-bottom:12px; }

.back-btn { display:inline-flex; align-items:center; gap:4px; padding:6px 14px; border-radius:6px; background:var(--card); border:1px solid var(--border); cursor:pointer; font-size:12px; color:var(--text2); margin-bottom:12px; }
.back-btn:hover { background:#f0f2f8; }

.tab-bar { display:flex; gap:2px; margin-bottom:12px; }
.tab-btn { padding:6px 14px; border-radius:6px; cursor:pointer; font-size:12px; color:var(--text2); background:var(--card); border:1px solid var(--border); font-weight:500; }
.tab-btn.active { background:var(--accent); color:#fff; border-color:var(--accent); }
.tab-content { display:none; }
.tab-content.active { display:block; }

.mini-table { font-size:12px; }
.mini-table td { padding:4px 8px; }
.mini-table .lbl { color:var(--text3); width:80px; }
</style>
</head>
<body>

<div class="header">
  <div style="display:flex;align-items:center;gap:12px">
    <h1>A股涨停板量化系统</h1>
    <div class="search-box">
      <svg viewBox="0 0 24 24"><path d="M15.5 14h-.79l-.28-.27A6.47 6.47 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 5L20.49 19l-5-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/></svg>
      <input id="searchInput" placeholder="搜索股票代码/名称..." autocomplete="off"/>
      <div class="search-dropdown" id="searchDropdown"></div>
    </div>
  </div>
  <div class="info" id="headerInfo">加载中...</div>
</div>

<div class="container">
  <div class="nav" id="mainNav">
    <div class="nav-item active" onclick="switchTab('overview',this)">市场概览</div>
    <div class="nav-item" onclick="switchTab('ztlist',this)">涨停列表</div>
    <div class="nav-item" onclick="switchTab('sentiment',this)">情绪周期</div>
    <div class="nav-item" onclick="switchTab('premium',this)">溢价统计</div>
    <div class="nav-item" onclick="switchTab('lhb',this)">龙虎榜</div>
    <div class="nav-item" onclick="switchTab('flow',this)">资金流向</div>
    <div class="nav-item" onclick="switchTab('signals',this)">选股信号</div>
    <div class="nav-item" onclick="switchTab('hotrank',this)">人气排行</div>
    <div class="nav-item" onclick="switchTab('backtest',this)">回测报告</div>
    <div class="nav-item" onclick="switchTab('dbstats',this)">数据统计</div>
  </div>

  <div id="overview" class="panel active">
    <div class="grid4" id="overviewCards"></div>
    <div class="grid2">
      <div class="card"><h3>涨停家数趋势</h3><canvas id="ztTrendChart"></canvas></div>
      <div class="card"><h3>连板天梯图</h3><canvas id="ladderChart"></canvas></div>
    </div>
    <div class="grid2">
      <div class="card"><h3>晋级率走势</h3><canvas id="promoChart"></canvas></div>
      <div class="card"><h3>近期热门板块</h3><div id="sectorHeatTable"></div></div>
    </div>
  </div>

  <div id="ztlist" class="panel">
    <div class="card"><h3>涨停个股明细 <span style="font-weight:400;color:var(--text3);font-size:11px" id="ztDateLabel"></span></h3><div class="table-wrap" id="ztTableWrap"></div></div>
  </div>

  <div id="sentiment" class="panel">
    <div class="date-picker"><label>显示天数</label><select id="sentDaysSelect" onchange="loadSentiment()"><option value="30">30天</option><option value="60" selected>60天</option><option value="120">120天</option><option value="250">一年</option></select></div>
    <div class="grid4" id="sentimentCards"></div>
    <div class="card" style="margin-bottom:12px"><h3>情绪周期(涨停MA)</h3><canvas id="sentimentChart"></canvas></div>
    <div class="grid2">
      <div class="card"><h3>最高连板走势</h3><canvas id="maxBoardChart"></canvas></div>
      <div class="card"><h3>炸板数走势</h3><canvas id="failRateChart"></canvas></div>
    </div>
  </div>

  <div id="premium" class="panel">
    <div class="grid4" id="premiumCards"></div>
    <div class="grid2">
      <div class="card"><h3>各连板高度次日溢价</h3><canvas id="premiumChart"></canvas></div>
      <div class="card"><h3>次日连板率</h3><canvas id="nextZTChart"></canvas></div>
    </div>
  </div>

  <div id="lhb" class="panel">
    <div class="card"><h3>龙虎榜上榜个股 <span style="font-weight:400;color:var(--text3);font-size:11px" id="lhbDateLabel"></span></h3><div class="table-wrap" id="lhbTableWrap"></div></div>
  </div>

  <div id="flow" class="panel">
    <div class="date-picker">
      <label>选择日期</label>
      <select id="flowDateSelect" onchange="loadFlowByDate()"></select>
    </div>
    <div class="grid2">
      <div class="card"><h3>主力净流入 TOP30</h3><div class="table-wrap" id="inflowTable"></div></div>
      <div class="card"><h3>主力净流出 TOP30</h3><div class="table-wrap" id="outflowTable"></div></div>
    </div>
  </div>

  <div id="signals" class="panel">
    <div class="card"><h3>选股信号</h3><div class="table-wrap" id="signalTableWrap"></div></div>
  </div>

  <div id="hotrank" class="panel">
    <div class="card"><h3>人气排行 TOP100</h3><div class="table-wrap" id="hotRankTableWrap"></div></div>
  </div>

  <div id="backtest" class="panel">
    <div class="grid4" id="btCards"></div>
    <div class="card"><h3>累计收益曲线</h3><canvas id="btCurveChart"></canvas></div>
  </div>

  <div id="dbstats" class="panel">
    <div class="card"><h3>数据库统计</h3><div class="table-wrap" id="dbStatsTable"></div></div>
  </div>

  <!-- 个股详情 -->
  <div id="stockDetail" class="panel">
    <button class="back-btn" onclick="closeStockDetail()">← 返回</button>
    <div id="stockHeaderArea"></div>
    <div class="concept-tags" id="conceptTags"></div>
    <div class="tab-bar" id="stockTabBar">
      <div class="tab-btn active" onclick="switchStockTab('kline',this)">K线走势</div>
      <div class="tab-btn" onclick="switchStockTab('flow',this)">资金流向</div>
      <div class="tab-btn" onclick="switchStockTab('zthistory',this)">涨停历史</div>
      <div class="tab-btn" onclick="switchStockTab('lhbhistory',this)">龙虎榜</div>
      <div class="tab-btn" onclick="switchStockTab('techind',this)">技术指标</div>
    </div>
    <div id="stockKline" class="tab-content active">
      <div class="grid2">
        <div class="card"><h3>日K线 & 均线</h3><canvas id="klineChart"></canvas></div>
        <div class="card"><h3>成交量</h3><canvas id="volChart"></canvas></div>
      </div>
      <div class="grid2">
        <div class="card"><h3>MACD</h3><canvas id="macdChart"></canvas></div>
        <div class="card"><h3>KDJ</h3><canvas id="kdjChart"></canvas></div>
      </div>
    </div>
    <div id="stockFlow" class="tab-content">
      <div class="card" style="margin-bottom:12px"><h3>主力资金净流入趋势</h3><canvas id="stockFlowChart"></canvas></div>
      <div class="card"><h3>资金流向明细</h3><div class="table-wrap" id="stockFlowTable"></div></div>
    </div>
    <div id="stockZthistory" class="tab-content">
      <div class="card"><h3>涨停记录</h3><div class="table-wrap" id="stockZTTable"></div></div>
    </div>
    <div id="stockLhbhistory" class="tab-content">
      <div class="card"><h3>龙虎榜记录</h3><div class="table-wrap" id="stockLHBTable"></div></div>
    </div>
    <div id="stockTechind" class="tab-content">
      <div class="grid2">
        <div class="card"><h3>RSI(6/12)</h3><canvas id="rsiChart"></canvas></div>
        <div class="card"><h3>BOLL通道</h3><canvas id="bollChart"></canvas></div>
      </div>
    </div>
  </div>
</div>

<script>
const f=(n,d=1)=>n==null?'-':Number(n).toFixed(d);
const fW=n=>{const v=Math.abs(n);if(v>=1e8)return f(n/1e8,2)+'亿';if(v>=1e4)return f(n/1e4,1)+'万';return f(n,0);};
const cls=n=>n>0?'up':n<0?'down':'';
let charts={};

function destroyChart(id){if(charts[id]){charts[id].destroy();delete charts[id];}}
function makeChart(id,cfg){destroyChart(id);charts[id]=new Chart(document.getElementById(id),cfg);return charts[id];}

function switchTab(id,el){
  document.getElementById('stockDetail').classList.remove('active');
  document.querySelectorAll('.panel').forEach(p=>{if(p.id!=='stockDetail')p.classList.remove('active');});
  document.querySelectorAll('#mainNav .nav-item').forEach(t=>t.classList.remove('active'));
  document.getElementById(id).classList.add('active');
  if(el)el.classList.add('active');
}

function switchStockTab(id,el){
  document.querySelectorAll('#stockDetail .tab-content').forEach(c=>c.classList.remove('active'));
  document.querySelectorAll('#stockTabBar .tab-btn').forEach(b=>b.classList.remove('active'));
  const map={kline:'stockKline',flow:'stockFlow',zthistory:'stockZthistory',lhbhistory:'stockLhbhistory',techind:'stockTechind'};
  document.getElementById(map[id]).classList.add('active');
  el.classList.add('active');
}

async function api(url){return(await fetch(url)).json();}

function statCard(label,value,sub,cc){
  return '<div class="stat-card"><div class="label">'+label+'</div><div class="value '+(cc||'')+'">'+value+'</div>'+(sub?'<div class="sub">'+sub+'</div>':'')+'</div>';
}

function openStock(code){
  document.querySelectorAll('.panel').forEach(p=>p.classList.remove('active'));
  document.querySelectorAll('#mainNav .nav-item').forEach(t=>t.classList.remove('active'));
  document.getElementById('stockDetail').classList.add('active');
  loadStockDetail(code);
}

function closeStockDetail(){
  document.getElementById('stockDetail').classList.remove('active');
  document.getElementById('overview').classList.add('active');
  document.querySelector('#mainNav .nav-item').classList.add('active');
}

// ========== 搜索 ==========
let searchTimer;
document.getElementById('searchInput').addEventListener('input',function(){
  clearTimeout(searchTimer);
  const q=this.value.trim();
  if(!q){document.getElementById('searchDropdown').classList.remove('show');return;}
  searchTimer=setTimeout(async()=>{
    const data=await api('/api/stock/search?q='+encodeURIComponent(q));
    const dd=document.getElementById('searchDropdown');
    if(!data||!data.length){dd.classList.remove('show');return;}
    dd.innerHTML=data.map(s=>'<div class="search-item" onclick="openStock(\''+s.code+'\')"><span class="code">'+s.code+'</span><span class="name">'+s.name+'</span><span class="ind">'+s.industry+'</span></div>').join('');
    dd.classList.add('show');
  },300);
});
document.addEventListener('click',e=>{if(!e.target.closest('.search-box'))document.getElementById('searchDropdown').classList.remove('show');});

// ========== Overview ==========
async function loadOverview(){
  const[ov,sent]=await Promise.all([api('/api/overview'),api('/api/sentiment')]);
  const latest=sent&&sent.length?sent[sent.length-1]:{};
  document.getElementById('headerInfo').textContent='数据日期: '+(ov.date||'-')+' | 涨停: '+(latest.zt_count||0)+'家 | 炸板: '+(latest.fail_count||0)+'家 | 最高: '+(latest.max_board||0)+'板';

  document.getElementById('overviewCards').innerHTML=[
    statCard('涨停家数',latest.zt_count||0,'今日涨停','up'),
    statCard('炸板',latest.fail_count||0,'','down'),
    statCard('最高连板',(latest.max_board||0)+'板','','up'),
    statCard('首板→二板',f(latest.promo_1to2)+'%','晋级率','neutral'),
    statCard('涨停MA5',f(latest.zt_ma5,1),'5日均值',''),
    statCard('热门板块',latest.top_sector_1||'-',(latest.top_sector_1_count||0)+'只涨停','neutral'),
  ].join('');

  if(!sent||!sent.length)return;
  const labels=sent.map(d=>d.date.slice(5));

  makeChart('ztTrendChart',{
    type:'bar',data:{labels,datasets:[
      {label:'涨停数',data:sent.map(d=>d.zt_count),backgroundColor:'rgba(227,57,70,0.4)',borderColor:'rgba(227,57,70,0.8)',borderWidth:1,borderRadius:2,order:2},
      {label:'MA5',data:sent.map(d=>d.zt_ma5),type:'line',borderColor:'#4361ee',borderWidth:2,pointRadius:0,tension:0.3,order:1},
      {label:'MA10',data:sent.map(d=>d.zt_ma10),type:'line',borderColor:'#7209b7',borderWidth:2,pointRadius:0,tension:0.3,order:1},
    ]},options:chartOpts()
  });

  makeChart('ladderChart',{
    type:'bar',data:{labels,datasets:[
      {label:'1板',data:sent.map(d=>d.board_1),backgroundColor:'#74b9ff'},
      {label:'2板',data:sent.map(d=>d.board_2),backgroundColor:'#fdcb6e'},
      {label:'3板',data:sent.map(d=>d.board_3),backgroundColor:'#e17055'},
      {label:'4板',data:sent.map(d=>d.board_4),backgroundColor:'#d63031'},
      {label:'5+板',data:sent.map(d=>d.board_5plus),backgroundColor:'#6c5ce7'},
    ]},options:{...chartOpts(),scales:{x:{stacked:true,...axisOpts()},y:{stacked:true,...axisOpts()}}}
  });

  makeChart('promoChart',{
    type:'line',data:{labels,datasets:[
      {label:'首板→二板%',data:sent.map(d=>d.promo_1to2),borderColor:'#2ec4b6',borderWidth:2,pointRadius:1,tension:0.3},
      {label:'二板→三板%',data:sent.map(d=>d.promo_2to3),borderColor:'#e17055',borderWidth:2,pointRadius:1,tension:0.3},
    ]},options:chartOpts()
  });

  let ht='<table><thead><tr><th>日期</th><th>TOP1</th><th>数量</th><th>TOP2</th><th>数量</th><th>TOP3</th><th>数量</th></tr></thead><tbody>';
  sent.slice(-15).reverse().forEach(d=>{
    ht+='<tr><td>'+d.date.slice(5)+'</td><td>'+d.top_sector_1+'</td><td class="up">'+d.top_sector_1_count+'</td><td>'+d.top_sector_2+'</td><td class="up">'+d.top_sector_2_count+'</td><td>'+d.top_sector_3+'</td><td class="up">'+d.top_sector_3_count+'</td></tr>';
  });
  ht+='</tbody></table>';
  document.getElementById('sectorHeatTable').innerHTML=ht;
}

// ========== Sentiment ==========
async function loadSentiment(){
  const days=document.getElementById('sentDaysSelect').value;
  const sent=await api('/api/sentiment?days='+days);
  if(!sent||!sent.length)return;
  const latest=sent[sent.length-1];
  document.getElementById('sentimentCards').innerHTML=[
    statCard('涨停家数',latest.zt_count,'','up'),
    statCard('炸板',latest.fail_count,'','down'),
    statCard('跌停',latest.dt_count||0,'','down'),
    statCard('最高板',latest.max_board+'板','','up'),
    statCard('天梯',latest.board_1+'/'+latest.board_2+'/'+latest.board_3+'/'+latest.board_4+'/'+latest.board_5plus,'1板/2板/3板/4板/5+',''),
  ].join('');

  const labels=sent.map(d=>d.date.slice(5));
  makeChart('sentimentChart',{type:'line',data:{labels,datasets:[
    {label:'涨停数',data:sent.map(d=>d.zt_count),borderColor:'#e63946',borderWidth:1,pointRadius:0,fill:{target:'origin',above:'rgba(230,57,70,0.06)'}},
    {label:'MA5',data:sent.map(d=>d.zt_ma5),borderColor:'#4361ee',borderWidth:2,pointRadius:0,tension:0.3},
    {label:'MA10',data:sent.map(d=>d.zt_ma10),borderColor:'#7209b7',borderWidth:2,pointRadius:0,tension:0.3},
  ]},options:chartOpts()});

  makeChart('maxBoardChart',{type:'line',data:{labels,datasets:[
    {label:'最高连板',data:sent.map(d=>d.max_board),borderColor:'#e63946',borderWidth:2,pointRadius:2,tension:0.3,fill:{target:'origin',above:'rgba(230,57,70,0.05)'}},
  ]},options:chartOpts()});

  makeChart('failRateChart',{type:'bar',data:{labels,datasets:[
    {label:'炸板数',data:sent.map(d=>d.fail_count),backgroundColor:'rgba(46,196,182,0.5)',borderColor:'#2ec4b6',borderWidth:1,borderRadius:2},
    {label:'跌停数',data:sent.map(d=>d.dt_count||0),backgroundColor:'rgba(114,9,183,0.3)',borderColor:'#7209b7',borderWidth:1,borderRadius:2},
  ]},options:chartOpts()});
}

// ========== ZT List ==========
async function loadZTList(){
  const data=await api('/api/zt/today');
  document.getElementById('ztDateLabel').textContent=data.date+' ('+data.count+'只)';
  if(!data.records||!data.records.length){document.getElementById('ztTableWrap').innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>代码</th><th>名称</th><th>涨幅</th><th>收盘</th><th>连板</th><th>封板时间</th><th>炸板</th><th>换手</th><th>成交额</th><th>行业</th></tr></thead><tbody>';
  data.records.forEach(r=>{
    const bt=r.board_count>=3?'tag-red':r.board_count==2?'tag-orange':'tag-blue';
    h+='<tr onclick="openStock(\''+r.code+'\')"><td>'+r.code+'</td><td><b>'+r.name+'</b></td><td class="up">'+f(r.pct_chg,2)+'%</td><td>'+f(r.close,2)+'</td><td><span class="tag '+bt+'">'+r.board_count+'板</span></td><td>'+(r.first_seal_time||'-')+'</td><td>'+(r.fail_count||0)+'</td><td>'+f(r.turnover)+'%</td><td>'+fW(r.amount)+'</td><td><span class="tag tag-purple">'+(r.industry||'')+'</span></td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('ztTableWrap').innerHTML=h;
}

// ========== Premium ==========
async function loadPremium(){
  const data=await api('/api/premium');
  if(!data||!data.length)return;
  const items=data.filter(d=>d.board_count>=1&&d.board_count<=8);
  document.getElementById('premiumCards').innerHTML=items.map(d=>{
    return '<div class="stat-card" style="text-align:center;padding:14px"><div style="font-size:24px;font-weight:700;color:var(--red)">'+d.board_count+'</div><div style="font-size:11px;color:var(--text3)">板</div><div style="font-size:14px;font-weight:600;margin-top:4px" class="'+cls(d.avg_open_premium)+'">'+f(d.avg_open_premium,2)+'%</div><div style="font-size:10px;color:var(--text2)">开盘溢价 | '+d.sample_count+'样本</div><div style="font-size:10px;color:var(--text2)">正溢价率 '+f(d.win_rate)+'%</div><div style="font-size:10px;color:var(--text2)">次日连板 '+f(d.next_zt_rate)+'%</div></div>';
  }).join('');

  makeChart('premiumChart',{type:'bar',data:{
    labels:items.map(d=>d.board_count+'板'),
    datasets:[
      {label:'开盘溢价%',data:items.map(d=>d.avg_open_premium),backgroundColor:'rgba(230,57,70,0.5)',borderRadius:3},
      {label:'收盘溢价%',data:items.map(d=>d.avg_close_premium),backgroundColor:'rgba(67,97,238,0.5)',borderRadius:3},
      {label:'最高溢价%',data:items.map(d=>d.avg_max_premium),backgroundColor:'rgba(114,9,183,0.3)',borderRadius:3},
    ]
  },options:chartOpts()});

  makeChart('nextZTChart',{type:'bar',data:{
    labels:items.map(d=>d.board_count+'板'),
    datasets:[{label:'次日连板率%',data:items.map(d=>d.next_zt_rate),backgroundColor:'rgba(230,57,70,0.5)',borderRadius:3}]
  },options:chartOpts()});
}

// ========== LHB ==========
async function loadLHB(){
  const data=await api('/api/lhb');
  document.getElementById('lhbDateLabel').textContent=(data.date||'')+' ('+((data.records||[]).length)+'只)';
  if(!data.records||!data.records.length){document.getElementById('lhbTableWrap').innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>代码</th><th>名称</th><th>涨幅</th><th>净买入</th><th>买入</th><th>卖出</th><th>原因</th></tr></thead><tbody>';
  data.records.forEach(r=>{
    h+='<tr onclick="openStock(\''+r.code+'\')"><td>'+r.code+'</td><td><b>'+r.name+'</b></td><td class="'+cls(r.pct_chg)+'">'+f(r.pct_chg,2)+'%</td><td class="'+cls(r.net_amount)+'"><b>'+fW(r.net_amount)+'</b></td><td class="up">'+fW(r.buy_amount)+'</td><td class="down">'+fW(r.sell_amount)+'</td><td style="max-width:260px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+r.reason+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('lhbTableWrap').innerHTML=h;
}

// ========== Flow with date ==========
async function initFlowDates(){
  const dates=await api('/api/flow/dates');
  const sel=document.getElementById('flowDateSelect');
  sel.innerHTML='';
  if(dates&&dates.length){
    dates.forEach(d=>{sel.innerHTML+='<option value="'+d+'">'+d+'</option>';});
    loadFlowByDate();
  }
}
async function loadFlowByDate(){
  const date=document.getElementById('flowDateSelect').value;
  const data=await api('/api/flow/top?date='+date);
  flowTable(data.inflows,'inflowTable');
  flowTable(data.outflows,'outflowTable');
}
function flowTable(items,el){
  if(!items||!items.length){document.getElementById(el).innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>代码</th><th>名称</th><th>主力净流入</th><th>超大单</th><th>大单</th></tr></thead><tbody>';
  items.forEach(r=>{
    h+='<tr onclick="openStock(\''+r.code+'\')"><td>'+r.code+'</td><td><b>'+r.name+'</b></td><td class="'+cls(r.main_net)+'"><b>'+fW(r.main_net)+'</b></td><td class="'+cls(r.huge_net)+'">'+fW(r.huge_net)+'</td><td class="'+cls(r.big_net)+'">'+fW(r.big_net)+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById(el).innerHTML=h;
}

// ========== Signals ==========
async function loadSignals(){
  const data=await api('/api/signals');
  if(!data.signals||!data.signals.length){document.getElementById('signalTableWrap').innerHTML='<div class="empty">暂无选股信号</div>';return;}
  let h='<table><thead><tr><th>#</th><th>代码</th><th>名称</th><th>评分</th><th>连板</th><th>买入价</th><th>止损</th><th>行业</th><th>原因</th></tr></thead><tbody>';
  data.signals.forEach((s,i)=>{
    const w=Math.min(s.score,100);
    h+='<tr onclick="openStock(\''+s.code+'\')"><td>'+(i+1)+'</td><td>'+s.code+'</td><td><b>'+s.name+'</b></td><td><span class="score-bar" style="width:'+w+'px"></span>'+f(s.score)+'</td><td><span class="tag tag-red">'+s.board_count+'板</span></td><td>'+f(s.buy_price,2)+'</td><td class="down">'+f(s.stop_loss,2)+'</td><td><span class="tag tag-purple">'+s.industry+'</span></td><td style="max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+s.reason+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('signalTableWrap').innerHTML=h;
}

// ========== Hot ==========
async function loadHotRank(){
  const data=await api('/api/hot');
  if(!data||!data.length){document.getElementById('hotRankTableWrap').innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>排名</th><th>代码</th><th>名称</th><th>变动</th></tr></thead><tbody>';
  data.forEach(r=>{
    const chg=r.rank_change>0?'<span class="badge badge-up">↑'+r.rank_change+'</span>':r.rank_change<0?'<span class="badge badge-down">↓'+(-r.rank_change)+'</span>':'<span style="color:var(--text3)">-</span>';
    h+='<tr onclick="openStock(\''+r.code+'\')"><td><b>'+r.rank+'</b></td><td>'+r.code+'</td><td><b>'+r.name+'</b></td><td>'+chg+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('hotRankTableWrap').innerHTML=h;
}

// ========== Backtest ==========
async function loadBacktest(){
  const data=await api('/api/backtest');
  document.getElementById('btCards').innerHTML=[
    statCard('总交易',data.total_trades||0,'',''),
    statCard('胜率',f(data.win_rate)+'%','',data.win_rate>50?'up':'down'),
    statCard('总收益',f(data.total_pnl,2)+'%','',data.total_pnl>0?'up':'down'),
    statCard('平均每笔',f(data.avg_pnl,2)+'%','',data.avg_pnl>0?'up':'down'),
  ].join('');

  if(data.curve&&data.curve.length){
    const labels=data.curve.filter((_,i)=>i%5===0).map(c=>c.date.slice(5));
    const vals=data.curve.filter((_,i)=>i%5===0).map(c=>c.cum_pnl);
    makeChart('btCurveChart',{type:'line',data:{labels,datasets:[
      {label:'累计收益%',data:vals,borderColor:'#e63946',borderWidth:1.5,pointRadius:0,
       fill:{target:'origin',above:'rgba(230,57,70,0.06)',below:'rgba(46,196,182,0.06)'}},
    ]},options:chartOpts()});
  }
}

// ========== DB Stats ==========
async function loadDBStats(){
  const data=await api('/api/stats');
  let h='<table><thead><tr><th>数据表</th><th>记录数</th></tr></thead><tbody>';
  let total=0;
  (data||[]).forEach(r=>{
    total+=r.count;
    h+='<tr><td>'+r.table+'</td><td><b>'+Number(r.count).toLocaleString()+'</b></td></tr>';
  });
  h+='<tr style="background:#f8f9fc"><td><b>总计</b></td><td><b style="color:var(--accent)">'+total.toLocaleString()+'</b></td></tr>';
  h+='</tbody></table>';
  document.getElementById('dbStatsTable').innerHTML=h;
}

// ========== Stock Detail ==========
async function loadStockDetail(code){
  const data=await api('/api/stock?code='+code+'&months=6');
  if(!data)return;

  // Header
  const quotes=data.quotes||[];
  const last=quotes.length?quotes[quotes.length-1]:null;
  let hdr='<div class="stock-header">';
  hdr+='<span class="stock-name">'+(data.name||code)+'</span>';
  hdr+='<span class="stock-code">'+code+' · '+(data.market||'')+' · '+(data.industry||'')+'</span>';
  if(last){
    hdr+='<span class="stock-price '+cls(last.pct_chg)+'">'+f(last.close,2)+'</span>';
    hdr+='<span class="stock-chg '+cls(last.pct_chg)+'">'+(last.pct_chg>0?'+':'')+f(last.pct_chg,2)+'%</span>';
  }
  hdr+='</div>';
  document.getElementById('stockHeaderArea').innerHTML=hdr;

  // Concepts
  const concepts=data.concepts||[];
  document.getElementById('conceptTags').innerHTML=concepts.slice(0,15).map(c=>'<span class="tag tag-blue">'+c.board_name+'</span>').join('');

  // Reset to first tab
  document.querySelectorAll('#stockDetail .tab-content').forEach(c=>c.classList.remove('active'));
  document.querySelectorAll('#stockTabBar .tab-btn').forEach(b=>b.classList.remove('active'));
  document.getElementById('stockKline').classList.add('active');
  document.querySelector('#stockTabBar .tab-btn').classList.add('active');

  renderKline(data);
  renderStockFlow(data);
  renderZTHistory(data);
  renderLHBHistory(data);
  renderTechIndicators(data);
}

function renderKline(data){
  const quotes=data.quotes||[];
  if(!quotes.length)return;
  const indicators=data.indicators||[];
  const labels=quotes.map(q=>q.date.slice(5,10));

  // Price + MA
  const datasets=[
    {label:'收盘价',data:quotes.map(q=>q.close),borderColor:'#e63946',borderWidth:1.5,pointRadius:0,tension:0.1},
  ];
  if(indicators.length===quotes.length){
    datasets.push({label:'MA5',data:indicators.map(d=>d.ma5||null),borderColor:'#4361ee',borderWidth:1,pointRadius:0,tension:0.3});
    datasets.push({label:'MA10',data:indicators.map(d=>d.ma10||null),borderColor:'#f77f00',borderWidth:1,pointRadius:0,tension:0.3});
    datasets.push({label:'MA20',data:indicators.map(d=>d.ma20||null),borderColor:'#7209b7',borderWidth:1,pointRadius:0,tension:0.3});
  }
  makeChart('klineChart',{type:'line',data:{labels,datasets},options:chartOpts()});

  // Volume
  const vColors=quotes.map(q=>q.pct_chg>=0?'rgba(230,57,70,0.5)':'rgba(46,196,182,0.5)');
  makeChart('volChart',{type:'bar',data:{labels,datasets:[
    {label:'成交量',data:quotes.map(q=>q.volume),backgroundColor:vColors,borderRadius:1}
  ]},options:chartOpts()});

  // MACD
  if(indicators.length===quotes.length){
    const mLabels=quotes.map(q=>q.date.slice(5,10));
    const macdColors=indicators.map(d=>(d.macd||0)>=0?'rgba(230,57,70,0.5)':'rgba(46,196,182,0.5)');
    makeChart('macdChart',{type:'bar',data:{labels:mLabels,datasets:[
      {label:'MACD',data:indicators.map(d=>d.macd),backgroundColor:macdColors,borderRadius:1,order:2},
      {label:'DIF',data:indicators.map(d=>d.dif),type:'line',borderColor:'#4361ee',borderWidth:1.5,pointRadius:0,tension:0.3,order:1},
      {label:'DEA',data:indicators.map(d=>d.dea),type:'line',borderColor:'#f77f00',borderWidth:1.5,pointRadius:0,tension:0.3,order:1},
    ]},options:chartOpts()});

    makeChart('kdjChart',{type:'line',data:{labels:mLabels,datasets:[
      {label:'K',data:indicators.map(d=>d.k),borderColor:'#4361ee',borderWidth:1.5,pointRadius:0,tension:0.3},
      {label:'D',data:indicators.map(d=>d.d),borderColor:'#f77f00',borderWidth:1.5,pointRadius:0,tension:0.3},
      {label:'J',data:indicators.map(d=>d.j),borderColor:'#7209b7',borderWidth:1.5,pointRadius:0,tension:0.3},
    ]},options:chartOpts()});
  }
}

function renderStockFlow(data){
  const flows=data.flows||[];
  if(!flows.length){document.getElementById('stockFlowTable').innerHTML='<div class="empty">暂无资金流向数据</div>';return;}

  const labels=flows.map(f=>f.date.slice(5));
  makeChart('stockFlowChart',{type:'bar',data:{labels,datasets:[
    {label:'主力净流入',data:flows.map(f=>f.main_net),backgroundColor:flows.map(f=>f.main_net>=0?'rgba(230,57,70,0.5)':'rgba(46,196,182,0.5)'),borderRadius:1},
  ]},options:chartOpts()});

  let h='<table><thead><tr><th>日期</th><th>主力净流入</th><th>超大单</th><th>大单</th><th>中单</th><th>小单</th></tr></thead><tbody>';
  flows.slice().reverse().forEach(r=>{
    h+='<tr><td>'+r.date+'</td><td class="'+cls(r.main_net)+'"><b>'+fW(r.main_net)+'</b></td><td class="'+cls(r.huge_net)+'">'+fW(r.huge_net)+'</td><td class="'+cls(r.big_net)+'">'+fW(r.big_net)+'</td><td class="'+cls(r.mid_net)+'">'+fW(r.mid_net)+'</td><td class="'+cls(r.small_net)+'">'+fW(r.small_net)+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('stockFlowTable').innerHTML=h;
}

function renderZTHistory(data){
  const zt=data.zt_records||[];
  if(!zt.length){document.getElementById('stockZTTable').innerHTML='<div class="empty">无涨停记录</div>';return;}
  let h='<table><thead><tr><th>日期</th><th>连板</th><th>涨幅</th><th>收盘</th><th>封板时间</th><th>炸板</th><th>换手</th><th>成交额</th></tr></thead><tbody>';
  zt.slice().reverse().forEach(r=>{
    const bt=r.board_count>=3?'tag-red':r.board_count===2?'tag-orange':'tag-blue';
    h+='<tr><td>'+r.date.slice(0,10)+'</td><td><span class="tag '+bt+'">'+r.board_count+'板</span></td><td class="up">'+f(r.pct_chg,2)+'%</td><td>'+f(r.close,2)+'</td><td>'+(r.first_seal_time||'-')+'</td><td>'+(r.fail_count||0)+'</td><td>'+f(r.turnover)+'%</td><td>'+fW(r.amount)+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('stockZTTable').innerHTML=h;
}

function renderLHBHistory(data){
  const lhb=data.lhb||[];
  if(!lhb.length){document.getElementById('stockLHBTable').innerHTML='<div class="empty">无龙虎榜记录</div>';return;}
  let h='<table><thead><tr><th>日期</th><th>净买入</th><th>买入</th><th>卖出</th><th>原因</th></tr></thead><tbody>';
  lhb.forEach(r=>{
    h+='<tr><td>'+r.date+'</td><td class="'+cls(r.net_amount)+'"><b>'+fW(r.net_amount)+'</b></td><td class="up">'+fW(r.buy_amount)+'</td><td class="down">'+fW(r.sell_amount)+'</td><td>'+r.reason+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('stockLHBTable').innerHTML=h;
}

function renderTechIndicators(data){
  const quotes=data.quotes||[];
  const indicators=data.indicators||[];
  if(!indicators.length||indicators.length!==quotes.length)return;
  const labels=quotes.map(q=>q.date.slice(5,10));

  makeChart('rsiChart',{type:'line',data:{labels,datasets:[
    {label:'RSI6',data:indicators.map(d=>d.rsi6),borderColor:'#4361ee',borderWidth:1.5,pointRadius:0,tension:0.3},
    {label:'RSI12',data:indicators.map(d=>d.rsi12),borderColor:'#f77f00',borderWidth:1.5,pointRadius:0,tension:0.3},
  ]},options:{...chartOpts(),scales:{x:axisOpts(),y:{...axisOpts(),min:0,max:100}}}});

  makeChart('bollChart',{type:'line',data:{labels,datasets:[
    {label:'收盘价',data:quotes.map(q=>q.close),borderColor:'#e63946',borderWidth:1.5,pointRadius:0,tension:0.1},
    {label:'上轨',data:indicators.map(d=>d.boll_upper),borderColor:'#7209b7',borderWidth:1,borderDash:[4,2],pointRadius:0,tension:0.3},
    {label:'中轨',data:indicators.map(d=>d.boll_mid),borderColor:'#4361ee',borderWidth:1,pointRadius:0,tension:0.3},
    {label:'下轨',data:indicators.map(d=>d.boll_lower),borderColor:'#7209b7',borderWidth:1,borderDash:[4,2],pointRadius:0,tension:0.3},
  ]},options:chartOpts()});
}

function chartOpts(){
  return{responsive:true,plugins:{legend:{labels:{color:'#5a6078',font:{size:10},boxWidth:12}},tooltip:{mode:'index',intersect:false}},
    scales:{x:axisOpts(),y:axisOpts()},interaction:{mode:'index',intersect:false}};
}
function axisOpts(){
  return{ticks:{color:'#a0a8bf',font:{size:10},maxRotation:0},grid:{color:'rgba(0,0,0,0.03)'}};
}

(async()=>{
  await loadOverview();
  loadSentiment();loadZTList();loadPremium();loadLHB();initFlowDates();
  loadSignals();loadHotRank();loadBacktest();loadDBStats();
})();
</script>
</body>
</html>` + ""
