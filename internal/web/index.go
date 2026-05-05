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
  --bg: #f5f6fa; --card: #ffffff; --border: #e8ecf1; --text: #2d3436;
  --text2: #636e72; --text3: #b2bec3; --accent: #e17055; --accent2: #d63031;
  --green: #00b894; --blue: #0984e3; --purple: #6c5ce7; --orange: #f39c12;
  --yellow: #fdcb6e; --shadow: 0 2px 12px rgba(0,0,0,0.06);
  --radius: 12px;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, 'PingFang SC', 'Microsoft YaHei', sans-serif; background: var(--bg); color: var(--text); font-size: 14px; }
.header { background: var(--card); padding: 16px 28px; border-bottom: 1px solid var(--border); display: flex; align-items: center; justify-content: space-between; }
.header h1 { font-size: 20px; font-weight: 600; color: var(--accent2); }
.header .info { color: var(--text2); font-size: 13px; }
.container { max-width: 1440px; margin: 0 auto; padding: 16px 20px; }

.nav { display: flex; gap: 4px; margin-bottom: 16px; background: var(--card); padding: 6px; border-radius: var(--radius); box-shadow: var(--shadow); overflow-x: auto; }
.nav-item { padding: 8px 18px; border-radius: 8px; cursor: pointer; font-size: 13px; color: var(--text2); white-space: nowrap; transition: all 0.2s; font-weight: 500; }
.nav-item:hover { background: #f0f0f5; color: var(--text); }
.nav-item.active { background: var(--accent); color: #fff; }

.panel { display: none; }
.panel.active { display: block; }

.grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
.grid3 { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 16px; margin-bottom: 16px; }
.grid4 { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-bottom: 16px; }
@media (max-width: 768px) { .grid2, .grid3 { grid-template-columns: 1fr; } }

.card { background: var(--card); border-radius: var(--radius); padding: 20px; box-shadow: var(--shadow); border: 1px solid var(--border); }
.card h3 { font-size: 15px; font-weight: 600; margin-bottom: 14px; color: var(--text); display: flex; align-items: center; gap: 6px; }
.card h3::before { content: ''; width: 3px; height: 16px; background: var(--accent); border-radius: 2px; }

.stat-card { background: var(--card); border-radius: var(--radius); padding: 16px 20px; box-shadow: var(--shadow); border: 1px solid var(--border); }
.stat-card .label { font-size: 12px; color: var(--text3); margin-bottom: 6px; font-weight: 500; }
.stat-card .value { font-size: 26px; font-weight: 700; line-height: 1.2; }
.stat-card .sub { font-size: 12px; color: var(--text2); margin-top: 4px; }
.up { color: var(--accent2); }
.down { color: var(--green); }
.neutral { color: var(--blue); }

table { width: 100%; border-collapse: collapse; font-size: 13px; }
thead th { background: #f8f9fb; padding: 10px 12px; text-align: left; color: var(--text2); font-weight: 600; border-bottom: 2px solid var(--border); position: sticky; top: 0; z-index: 1; }
tbody td { padding: 9px 12px; border-bottom: 1px solid var(--border); }
tbody tr:hover { background: #f5f7fa; }
.table-wrap { max-height: 520px; overflow-y: auto; border-radius: var(--radius); }

.tag { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
.tag-red { background: #ffeaea; color: var(--accent2); }
.tag-green { background: #e6fff5; color: var(--green); }
.tag-blue { background: #e8f4fd; color: var(--blue); }
.tag-purple { background: #f3f0ff; color: var(--purple); }
.tag-orange { background: #fff5e6; color: var(--orange); }

.score-bar { display: inline-block; height: 6px; border-radius: 3px; background: linear-gradient(90deg, var(--accent), var(--accent2)); vertical-align: middle; margin-right: 6px; }

.premium-card { text-align: center; padding: 16px; }
.premium-card .board-num { font-size: 28px; font-weight: 700; color: var(--accent2); }
.premium-card .board-label { font-size: 12px; color: var(--text3); }
.premium-card .prem-val { font-size: 16px; font-weight: 600; margin-top: 6px; }
.premium-card .prem-sub { font-size: 11px; color: var(--text2); margin-top: 2px; }

.empty { text-align: center; padding: 60px 20px; color: var(--text3); }
.badge { display: inline-block; min-width: 20px; text-align: center; padding: 1px 6px; border-radius: 10px; font-size: 11px; font-weight: 600; }
.badge-up { background: #ffeaea; color: var(--accent2); }
.badge-down { background: #e6fff5; color: var(--green); }
</style>
</head>
<body>

<div class="header">
  <h1>A股涨停板量化系统</h1>
  <div class="info" id="headerInfo">加载中...</div>
</div>

<div class="container">
  <div class="nav">
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

  <!-- 市场概览 -->
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

  <!-- 涨停列表 -->
  <div id="ztlist" class="panel">
    <div class="card"><h3>涨停个股明细 <span style="font-weight:400;color:var(--text3);font-size:12px" id="ztDateLabel"></span></h3><div class="table-wrap" id="ztTableWrap"></div></div>
  </div>

  <!-- 情绪周期 -->
  <div id="sentiment" class="panel">
    <div class="grid4" id="sentimentCards"></div>
    <div class="card" style="margin-bottom:16px"><h3>情绪周期(涨停MA)</h3><canvas id="sentimentChart"></canvas></div>
    <div class="grid2">
      <div class="card"><h3>最高连板走势</h3><canvas id="maxBoardChart"></canvas></div>
      <div class="card"><h3>炸板率走势</h3><canvas id="failRateChart"></canvas></div>
    </div>
  </div>

  <!-- 溢价统计 -->
  <div id="premium" class="panel">
    <div class="grid4" id="premiumCards"></div>
    <div class="grid2">
      <div class="card"><h3>各连板高度次日溢价</h3><canvas id="premiumChart"></canvas></div>
      <div class="card"><h3>次日连板率</h3><canvas id="nextZTChart"></canvas></div>
    </div>
  </div>

  <!-- 龙虎榜 -->
  <div id="lhb" class="panel">
    <div class="card"><h3>龙虎榜上榜个股 <span style="font-weight:400;color:var(--text3);font-size:12px" id="lhbDateLabel"></span></h3><div class="table-wrap" id="lhbTableWrap"></div></div>
  </div>

  <!-- 资金流向 -->
  <div id="flow" class="panel">
    <div class="grid2">
      <div class="card"><h3>主力净流入 TOP30</h3><div class="table-wrap" id="inflowTable"></div></div>
      <div class="card"><h3>主力净流出 TOP30</h3><div class="table-wrap" id="outflowTable"></div></div>
    </div>
  </div>

  <!-- 选股信号 -->
  <div id="signals" class="panel">
    <div class="card"><h3>选股信号</h3><div class="table-wrap" id="signalTableWrap"></div></div>
  </div>

  <!-- 人气排行 -->
  <div id="hotrank" class="panel">
    <div class="card"><h3>人气排行 TOP100</h3><div class="table-wrap" id="hotRankTableWrap"></div></div>
  </div>

  <!-- 回测报告 -->
  <div id="backtest" class="panel">
    <div class="grid4" id="btCards"></div>
    <div class="card"><h3>累计收益曲线</h3><canvas id="btCurveChart"></canvas></div>
  </div>

  <!-- 数据统计 -->
  <div id="dbstats" class="panel">
    <div class="card"><h3>数据库统计</h3><div class="table-wrap" id="dbStatsTable"></div></div>
  </div>
</div>

<script>
const f = (n,d=1) => n==null?'-':Number(n).toFixed(d);
const fW = n => { const v=Math.abs(n); if(v>=1e8) return f(n/1e8,2)+'亿'; if(v>=1e4) return f(n/1e4,1)+'万'; return f(n,0); };
const cls = n => n>0?'up':n<0?'down':'';

function switchTab(id, el) {
  document.querySelectorAll('.panel').forEach(p=>p.classList.remove('active'));
  document.querySelectorAll('.nav-item').forEach(t=>t.classList.remove('active'));
  document.getElementById(id).classList.add('active');
  el.classList.add('active');
}

async function api(url) { return (await fetch(url)).json(); }

function statCard(label, value, sub, colorCls) {
  return '<div class="stat-card"><div class="label">'+label+'</div><div class="value '+(colorCls||'')+'">'+value+'</div>'+(sub?'<div class="sub">'+sub+'</div>':'')+'</div>';
}

async function loadOverview() {
  const [ov, sent] = await Promise.all([api('/api/overview'), api('/api/sentiment')]);
  const a = ov.analysis || {};
  const latest = sent && sent.length ? sent[sent.length-1] : {};
  document.getElementById('headerInfo').textContent = '数据日期: '+(ov.date||'-')+'  |  涨停: '+(latest.zt_count||0)+'家  |  最高: '+(latest.max_board||0)+'板';

  document.getElementById('overviewCards').innerHTML = [
    statCard('涨停家数', latest.zt_count||0, '今日涨停', 'up'),
    statCard('最高连板', (latest.max_board||0)+'板', '', 'up'),
    statCard('首板→二板', f(latest.promo_1to2)+'%', '晋级率', 'neutral'),
    statCard('二板→三板', f(latest.promo_2to3)+'%', '晋级率', 'neutral'),
    statCard('涨停MA5', f(latest.zt_ma5,1), '5日均值', ''),
    statCard('热门板块', latest.top_sector_1||'-', (latest.top_sector_1_count||0)+'只涨停', 'neutral'),
  ].join('');

  if (!sent || !sent.length) return;
  const labels = sent.map(d=>d.date.slice(5));
  const ztArr = sent.map(d=>d.zt_count);

  new Chart(document.getElementById('ztTrendChart'), {
    type:'bar', data:{labels, datasets:[
      {label:'涨停数', data:ztArr, backgroundColor:'rgba(225,112,85,0.5)', borderColor:'rgba(225,112,85,0.8)', borderWidth:1, borderRadius:3, order:2},
      {label:'MA5', data:sent.map(d=>d.zt_ma5), type:'line', borderColor:'#0984e3', borderWidth:2, pointRadius:0, tension:0.3, order:1},
      {label:'MA10', data:sent.map(d=>d.zt_ma10), type:'line', borderColor:'#6c5ce7', borderWidth:2, pointRadius:0, tension:0.3, order:1},
    ]}, options:chartOpts()
  });

  new Chart(document.getElementById('ladderChart'), {
    type:'bar', data:{labels, datasets:[
      {label:'1板', data:sent.map(d=>d.board_1), backgroundColor:'#74b9ff'},
      {label:'2板', data:sent.map(d=>d.board_2), backgroundColor:'#fdcb6e'},
      {label:'3板', data:sent.map(d=>d.board_3), backgroundColor:'#e17055'},
      {label:'4板', data:sent.map(d=>d.board_4), backgroundColor:'#d63031'},
      {label:'5+板', data:sent.map(d=>d.board_5plus), backgroundColor:'#6c5ce7'},
    ]}, options:{...chartOpts(), scales:{x:{stacked:true,...axisOpts()}, y:{stacked:true,...axisOpts()}}}
  });

  new Chart(document.getElementById('promoChart'), {
    type:'line', data:{labels, datasets:[
      {label:'首板→二板%', data:sent.map(d=>d.promo_1to2), borderColor:'#00b894', borderWidth:2, pointRadius:1, tension:0.3},
      {label:'二板→三板%', data:sent.map(d=>d.promo_2to3), borderColor:'#e17055', borderWidth:2, pointRadius:1, tension:0.3},
    ]}, options:chartOpts()
  });

  let ht='<table><thead><tr><th>日期</th><th>TOP1板块</th><th>数量</th><th>TOP2板块</th><th>数量</th><th>TOP3板块</th><th>数量</th></tr></thead><tbody>';
  sent.slice(-15).reverse().forEach(d=>{
    ht+='<tr><td>'+d.date.slice(5)+'</td><td>'+d.top_sector_1+'</td><td class="up">'+d.top_sector_1_count+'</td><td>'+d.top_sector_2+'</td><td class="up">'+d.top_sector_2_count+'</td><td>'+d.top_sector_3+'</td><td class="up">'+d.top_sector_3_count+'</td></tr>';
  });
  ht+='</tbody></table>';
  document.getElementById('sectorHeatTable').innerHTML=ht;
}

async function loadSentiment() {
  const sent = await api('/api/sentiment');
  if (!sent||!sent.length) return;
  const latest=sent[sent.length-1];
  document.getElementById('sentimentCards').innerHTML=[
    statCard('涨停家数', latest.zt_count, '', 'up'),
    statCard('炸板', latest.fail_count, '', 'down'),
    statCard('最高板', latest.max_board+'板', '', 'up'),
    statCard('1板/2板/3板/4板/5+', latest.board_1+'/'+latest.board_2+'/'+latest.board_3+'/'+latest.board_4+'/'+latest.board_5plus, '天梯分布', ''),
  ].join('');

  const labels=sent.map(d=>d.date.slice(5));
  new Chart(document.getElementById('sentimentChart'),{type:'line',data:{labels,datasets:[
    {label:'涨停数',data:sent.map(d=>d.zt_count),borderColor:'#e17055',borderWidth:1,pointRadius:0,fill:{target:'origin',above:'rgba(225,112,85,0.08)'}},
    {label:'MA5',data:sent.map(d=>d.zt_ma5),borderColor:'#0984e3',borderWidth:2,pointRadius:0,tension:0.3},
    {label:'MA10',data:sent.map(d=>d.zt_ma10),borderColor:'#6c5ce7',borderWidth:2,pointRadius:0,tension:0.3},
  ]},options:chartOpts()});

  new Chart(document.getElementById('maxBoardChart'),{type:'line',data:{labels,datasets:[
    {label:'最高连板',data:sent.map(d=>d.max_board),borderColor:'#d63031',borderWidth:2,pointRadius:2,tension:0.3,fill:{target:'origin',above:'rgba(214,48,49,0.06)'}},
  ]},options:chartOpts()});

  new Chart(document.getElementById('failRateChart'),{type:'line',data:{labels,datasets:[
    {label:'炸板数',data:sent.map(d=>d.fail_count),borderColor:'#00b894',borderWidth:2,pointRadius:1,tension:0.3},
  ]},options:chartOpts()});
}

async function loadZTList() {
  const data = await api('/api/zt/today');
  document.getElementById('ztDateLabel').textContent=data.date+' ('+data.count+'只)';
  if(!data.records||!data.records.length){document.getElementById('ztTableWrap').innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>代码</th><th>名称</th><th>涨幅</th><th>收盘</th><th>连板</th><th>封板时间</th><th>炸板</th><th>换手</th><th>成交额</th><th>行业</th></tr></thead><tbody>';
  data.records.forEach(r=>{
    const boardTag = r.board_count>=3?'tag-red':r.board_count==2?'tag-orange':'tag-blue';
    h+='<tr><td>'+r.code+'</td><td>'+r.name+'</td><td class="up">'+f(r.pct_chg,2)+'%</td><td>'+f(r.close,2)+'</td><td><span class="tag '+boardTag+'">'+r.board_count+'板</span></td><td>'+(r.first_seal_time||'-')+'</td><td>'+(r.fail_count||0)+'</td><td>'+f(r.turnover)+'%</td><td>'+fW(r.amount)+'</td><td><span class="tag tag-purple">'+r.industry+'</span></td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('ztTableWrap').innerHTML=h;
}

async function loadPremium() {
  const data = await api('/api/premium');
  if(!data||!data.length) return;
  const items=data.filter(d=>d.board_count>=1&&d.board_count<=8);

  document.getElementById('premiumCards').innerHTML=items.map(d=>{
    return '<div class="stat-card premium-card"><div class="board-num">'+d.board_count+'</div><div class="board-label">板</div><div class="prem-val '+cls(d.avg_open_premium)+'">'+f(d.avg_open_premium,2)+'%</div><div class="prem-sub">开盘溢价 | 样本'+d.sample_count+'</div><div class="prem-sub">正溢价率 '+f(d.win_rate)+'%</div><div class="prem-sub">次日连板 '+f(d.next_zt_rate)+'%</div></div>';
  }).join('');

  new Chart(document.getElementById('premiumChart'),{type:'bar',data:{
    labels:items.map(d=>d.board_count+'板'),
    datasets:[
      {label:'开盘溢价%',data:items.map(d=>d.avg_open_premium),backgroundColor:'rgba(225,112,85,0.6)',borderRadius:4},
      {label:'收盘溢价%',data:items.map(d=>d.avg_close_premium),backgroundColor:'rgba(9,132,227,0.6)',borderRadius:4},
      {label:'最高溢价%',data:items.map(d=>d.avg_max_premium),backgroundColor:'rgba(108,92,231,0.3)',borderRadius:4},
    ]
  },options:chartOpts()});

  new Chart(document.getElementById('nextZTChart'),{type:'bar',data:{
    labels:items.map(d=>d.board_count+'板'),
    datasets:[{label:'次日连板率%',data:items.map(d=>d.next_zt_rate),backgroundColor:'rgba(214,48,49,0.6)',borderRadius:4}]
  },options:chartOpts()});
}

async function loadLHB() {
  const data = await api('/api/lhb');
  document.getElementById('lhbDateLabel').textContent=(data.date||'')+' ('+((data.records||[]).length)+'只)';
  if(!data.records||!data.records.length){document.getElementById('lhbTableWrap').innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>代码</th><th>名称</th><th>涨幅</th><th>净买入</th><th>买入</th><th>卖出</th><th>上榜原因</th></tr></thead><tbody>';
  data.records.forEach(r=>{
    h+='<tr><td>'+r.code+'</td><td>'+r.name+'</td><td class="'+cls(r.pct_chg)+'">'+f(r.pct_chg,2)+'%</td><td class="'+cls(r.net_amount)+'">'+fW(r.net_amount)+'</td><td class="up">'+fW(r.buy_amount)+'</td><td class="down">'+fW(r.sell_amount)+'</td><td style="max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+r.reason+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('lhbTableWrap').innerHTML=h;
}

async function loadFlow() {
  const data = await api('/api/flow/top');
  function flowTable(items, el) {
    if(!items||!items.length){document.getElementById(el).innerHTML='<div class="empty">暂无数据</div>';return;}
    let h='<table><thead><tr><th>代码</th><th>名称</th><th>主力净流入</th><th>超大单</th><th>大单</th></tr></thead><tbody>';
    items.forEach(r=>{
      h+='<tr><td>'+r.code+'</td><td>'+r.name+'</td><td class="'+cls(r.main_net)+'"><b>'+fW(r.main_net)+'</b></td><td class="'+cls(r.huge_net)+'">'+fW(r.huge_net)+'</td><td class="'+cls(r.big_net)+'">'+fW(r.big_net)+'</td></tr>';
    });
    h+='</tbody></table>';
    document.getElementById(el).innerHTML=h;
  }
  flowTable(data.inflows,'inflowTable');
  flowTable(data.outflows,'outflowTable');
}

async function loadSignals() {
  const data = await api('/api/signals');
  if(!data.signals||!data.signals.length){document.getElementById('signalTableWrap').innerHTML='<div class="empty">暂无选股信号</div>';return;}
  let h='<table><thead><tr><th>#</th><th>代码</th><th>名称</th><th>评分</th><th>连板</th><th>买入价</th><th>止损价</th><th>行业</th><th>选股原因</th></tr></thead><tbody>';
  data.signals.forEach((s,i)=>{
    const w=Math.min(s.score,100);
    h+='<tr><td>'+(i+1)+'</td><td>'+s.code+'</td><td><b>'+s.name+'</b></td><td><span class="score-bar" style="width:'+w+'px"></span>'+f(s.score)+'</td><td><span class="tag tag-red">'+s.board_count+'板</span></td><td>'+f(s.buy_price,2)+'</td><td class="down">'+f(s.stop_loss,2)+'</td><td><span class="tag tag-purple">'+s.industry+'</span></td><td style="max-width:260px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+s.reason+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('signalTableWrap').innerHTML=h;
}

async function loadHotRank() {
  const data = await api('/api/hot');
  if(!data||!data.length){document.getElementById('hotRankTableWrap').innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>排名</th><th>代码</th><th>名称</th><th>排名变动</th></tr></thead><tbody>';
  data.forEach(r=>{
    const chg = r.rank_change>0?'<span class="badge badge-up">↑'+r.rank_change+'</span>':r.rank_change<0?'<span class="badge badge-down">↓'+(-r.rank_change)+'</span>':'<span style="color:var(--text3)">-</span>';
    h+='<tr><td><b>'+r.rank+'</b></td><td>'+r.code+'</td><td>'+r.name+'</td><td>'+chg+'</td></tr>';
  });
  h+='</tbody></table>';
  document.getElementById('hotRankTableWrap').innerHTML=h;
}

async function loadBacktest() {
  const data = await api('/api/backtest');
  document.getElementById('btCards').innerHTML=[
    statCard('总交易', data.total_trades||0, '', ''),
    statCard('胜率', f(data.win_rate)+'%', '', data.win_rate>50?'up':'down'),
    statCard('总收益', f(data.total_pnl,2)+'%', '', data.total_pnl>0?'up':'down'),
    statCard('平均每笔', f(data.avg_pnl,2)+'%', '', data.avg_pnl>0?'up':'down'),
  ].join('');

  if(data.curve&&data.curve.length){
    const labels=data.curve.filter((_,i)=>i%5===0).map(c=>c.date.slice(5));
    const vals=data.curve.filter((_,i)=>i%5===0).map(c=>c.cum_pnl);
    new Chart(document.getElementById('btCurveChart'),{type:'line',data:{labels,datasets:[
      {label:'累计收益%',data:vals,borderColor:'#e17055',borderWidth:1.5,pointRadius:0,
       fill:{target:'origin',above:'rgba(225,112,85,0.08)',below:'rgba(0,184,148,0.08)'}},
    ]},options:chartOpts()});
  }
}

async function loadDBStats() {
  const data = await api('/api/stats');
  let h='<table><thead><tr><th>数据表</th><th>记录数</th></tr></thead><tbody>';
  let total=0;
  (data||[]).forEach(r=>{
    total+=r.count;
    h+='<tr><td>'+r.table+'</td><td><b>'+Number(r.count).toLocaleString()+'</b></td></tr>';
  });
  h+='<tr style="background:#f8f9fb"><td><b>总计</b></td><td><b>'+total.toLocaleString()+'</b></td></tr>';
  h+='</tbody></table>';
  document.getElementById('dbStatsTable').innerHTML=h;
}

function chartOpts() {
  return {responsive:true, plugins:{legend:{labels:{color:'#636e72',font:{size:11}}}},
    scales:{x:axisOpts(), y:axisOpts()}};
}
function axisOpts() {
  return {ticks:{color:'#b2bec3',font:{size:10}}, grid:{color:'rgba(0,0,0,0.04)'}};
}

(async()=>{
  await loadOverview();
  loadSentiment(); loadZTList(); loadPremium(); loadLHB(); loadFlow();
  loadSignals(); loadHotRank(); loadBacktest(); loadDBStats();
})();
</script>
</body>
</html>`
