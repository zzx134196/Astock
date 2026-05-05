package web

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>A股涨停板量化系统</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, 'Microsoft YaHei', sans-serif; background: #0f1923; color: #e0e0e0; }
.header { background: linear-gradient(135deg, #1a2332 0%, #2d3748 100%); padding: 20px 30px; border-bottom: 2px solid #e53e3e; }
.header h1 { font-size: 24px; color: #fff; }
.header .subtitle { color: #a0aec0; font-size: 13px; margin-top: 4px; }
.container { max-width: 1400px; margin: 0 auto; padding: 20px; }
.tabs { display: flex; gap: 8px; margin-bottom: 20px; flex-wrap: wrap; }
.tab { padding: 8px 20px; background: #1a2332; border: 1px solid #2d3748; border-radius: 6px; cursor: pointer; font-size: 14px; color: #a0aec0; }
.tab.active { background: #e53e3e; color: #fff; border-color: #e53e3e; }
.tab:hover { border-color: #e53e3e; }
.panel { display: none; }
.panel.active { display: block; }
.cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
.card { background: #1a2332; border-radius: 8px; padding: 20px; border: 1px solid #2d3748; }
.card .label { font-size: 13px; color: #718096; margin-bottom: 6px; }
.card .value { font-size: 28px; font-weight: bold; }
.card .value.up { color: #e53e3e; }
.card .value.down { color: #48bb78; }
.card .value.neutral { color: #ecc94b; }
.chart-box { background: #1a2332; border-radius: 8px; padding: 20px; border: 1px solid #2d3748; margin-bottom: 20px; }
.chart-box h3 { font-size: 16px; margin-bottom: 12px; color: #e2e8f0; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { background: #2d3748; padding: 10px 12px; text-align: left; color: #a0aec0; position: sticky; top: 0; }
td { padding: 8px 12px; border-bottom: 1px solid #2d3748; }
tr:hover { background: #1e2d3d; }
.up { color: #e53e3e; }
.down { color: #48bb78; }
.score-bar { display: inline-block; height: 8px; background: #e53e3e; border-radius: 4px; margin-right: 8px; }
.table-wrap { max-height: 500px; overflow-y: auto; border-radius: 8px; border: 1px solid #2d3748; }
.premium-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; margin-bottom: 20px; }
.premium-card { background: #1a2332; border-radius: 8px; padding: 16px; text-align: center; border: 1px solid #2d3748; }
.premium-card .board { font-size: 20px; font-weight: bold; color: #e53e3e; }
.premium-card .stat { font-size: 12px; color: #a0aec0; margin-top: 4px; }
</style>
</head>
<body>

<div class="header">
  <h1>A股涨停板量化系统</h1>
  <div class="subtitle" id="headerDate">加载中...</div>
</div>

<div class="container">
  <div class="tabs">
    <div class="tab active" onclick="showPanel('overview')">市场概览</div>
    <div class="tab" onclick="showPanel('sentiment')">情绪周期</div>
    <div class="tab" onclick="showPanel('ztlist')">涨停列表</div>
    <div class="tab" onclick="showPanel('signals')">选股信号</div>
    <div class="tab" onclick="showPanel('premium')">溢价统计</div>
    <div class="tab" onclick="showPanel('backtest')">回测报告</div>
    <div class="tab" onclick="showPanel('hotrank')">人气排行</div>
  </div>

  <div id="overview" class="panel active">
    <div class="cards" id="overviewCards"></div>
    <div class="chart-box"><h3>涨停家数趋势(近60日)</h3><canvas id="ztTrendChart"></canvas></div>
    <div class="chart-box"><h3>连板天梯(近60日)</h3><canvas id="ladderChart"></canvas></div>
  </div>

  <div id="sentiment" class="panel">
    <div class="chart-box"><h3>情绪周期(涨停数MA)</h3><canvas id="sentimentChart"></canvas></div>
    <div class="chart-box"><h3>晋级率趋势</h3><canvas id="promoChart"></canvas></div>
    <div class="chart-box"><h3>热门板块分布</h3><div id="sectorTable"></div></div>
  </div>

  <div id="ztlist" class="panel">
    <div class="table-wrap" id="ztTableWrap"></div>
  </div>

  <div id="signals" class="panel">
    <div class="table-wrap" id="signalTableWrap"></div>
  </div>

  <div id="premium" class="panel">
    <div class="premium-grid" id="premiumGrid"></div>
    <div class="chart-box"><h3>各连板高度次日开盘溢价</h3><canvas id="premiumChart"></canvas></div>
  </div>

  <div id="backtest" class="panel">
    <div class="cards" id="btCards"></div>
    <div class="chart-box"><h3>累计收益曲线</h3><canvas id="btCurveChart"></canvas></div>
  </div>

  <div id="hotrank" class="panel">
    <div class="table-wrap" id="hotRankTable"></div>
  </div>
</div>

<script>
const API = '';

function showPanel(id) {
  document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.getElementById(id).classList.add('active');
  event.target.classList.add('active');
}

async function fetchJSON(url) {
  const res = await fetch(API + url);
  return res.json();
}

function fmtNum(n, d=1) { return n == null ? '-' : Number(n).toFixed(d); }

async function loadOverview() {
  const data = await fetchJSON('/api/overview');
  document.getElementById('headerDate').textContent = '数据日期: ' + (data.date || '-');

  const a = data.analysis || {};
  const cards = [
    {label: '涨停家数', value: data.zt_count || 0, cls: 'up'},
    {label: '最高连板', value: a.max_board_height || '-', cls: 'up'},
    {label: '首板数', value: a.first_board_count || '-', cls: 'neutral'},
    {label: '二板数', value: a.second_board_count || '-', cls: 'neutral'},
    {label: '高位板(>=3)', value: a.high_board_count || '-', cls: 'up'},
    {label: '情绪阶段', value: a.sentiment_phase || '-', cls: 'neutral'},
  ];

  document.getElementById('overviewCards').innerHTML = cards.map(c =>
    '<div class="card"><div class="label">'+c.label+'</div><div class="value '+c.cls+'">'+c.value+'</div></div>'
  ).join('');
}

async function loadSentiment() {
  const data = await fetchJSON('/api/sentiment');
  if (!data || !data.length) return;

  const labels = data.map(d => d.date.slice(5));
  const ztCounts = data.map(d => d.zt_count);
  const ma5 = data.map(d => d.zt_ma5);
  const ma10 = data.map(d => d.zt_ma10);

  new Chart(document.getElementById('ztTrendChart'), {
    type: 'bar', data: {
      labels, datasets: [
        {label: '涨停数', data: ztCounts, backgroundColor: 'rgba(229,62,62,0.6)', order: 2},
        {label: 'MA5', data: ma5, type: 'line', borderColor: '#ecc94b', borderWidth: 2, pointRadius: 0, order: 1},
        {label: 'MA10', data: ma10, type: 'line', borderColor: '#4299e1', borderWidth: 2, pointRadius: 0, order: 1},
      ]
    }, options: {responsive: true, plugins: {legend: {labels: {color: '#a0aec0'}}}, scales: {x: {ticks: {color: '#718096'}}, y: {ticks: {color: '#718096'}}}}
  });

  new Chart(document.getElementById('ladderChart'), {
    type: 'bar', data: {
      labels, datasets: [
        {label: '1板', data: data.map(d=>d.board_1), backgroundColor: '#4299e1'},
        {label: '2板', data: data.map(d=>d.board_2), backgroundColor: '#ecc94b'},
        {label: '3板', data: data.map(d=>d.board_3), backgroundColor: '#ed8936'},
        {label: '4板', data: data.map(d=>d.board_4), backgroundColor: '#e53e3e'},
        {label: '5板+', data: data.map(d=>d.board_5plus), backgroundColor: '#9f7aea'},
      ]
    }, options: {responsive: true, scales: {x: {stacked: true, ticks:{color:'#718096'}}, y: {stacked: true, ticks:{color:'#718096'}}}, plugins: {legend: {labels: {color: '#a0aec0'}}}}
  });

  new Chart(document.getElementById('sentimentChart'), {
    type: 'line', data: {
      labels, datasets: [
        {label: '涨停数', data: ztCounts, borderColor: '#e53e3e', borderWidth: 1, pointRadius: 0, fill: false},
        {label: 'MA5', data: ma5, borderColor: '#ecc94b', borderWidth: 2, pointRadius: 0},
        {label: 'MA10', data: ma10, borderColor: '#4299e1', borderWidth: 2, pointRadius: 0},
      ]
    }, options: {responsive: true, plugins: {legend: {labels: {color: '#a0aec0'}}}, scales: {x: {ticks: {color: '#718096'}}, y: {ticks: {color: '#718096'}}}}
  });

  new Chart(document.getElementById('promoChart'), {
    type: 'line', data: {
      labels, datasets: [
        {label: '首板→二板(%)', data: data.map(d=>d.promo_1to2), borderColor: '#48bb78', borderWidth: 2, pointRadius: 1},
        {label: '二板→三板(%)', data: data.map(d=>d.promo_2to3), borderColor: '#ed8936', borderWidth: 2, pointRadius: 1},
      ]
    }, options: {responsive: true, plugins: {legend: {labels: {color: '#a0aec0'}}}, scales: {x: {ticks: {color: '#718096'}}, y: {ticks: {color: '#718096'}}}}
  });

  let sectorHTML = '<table><tr><th>日期</th><th>#1板块</th><th>数量</th><th>#2板块</th><th>数量</th><th>#3板块</th><th>数量</th></tr>';
  data.slice(-20).reverse().forEach(d => {
    sectorHTML += '<tr><td>'+d.date.slice(5)+'</td><td>'+d.top_sector_1+'</td><td class="up">'+d.top_sector_1_count+'</td><td>'+d.top_sector_2+'</td><td class="up">'+d.top_sector_2_count+'</td><td>'+d.top_sector_3+'</td><td class="up">'+d.top_sector_3_count+'</td></tr>';
  });
  sectorHTML += '</table>';
  document.getElementById('sectorTable').innerHTML = sectorHTML;
}

async function loadZTList() {
  const data = await fetchJSON('/api/zt/today');
  let html = '<table><tr><th>代码</th><th>名称</th><th>涨幅</th><th>收盘价</th><th>连板</th><th>封板时间</th><th>炸板</th><th>换手率</th><th>成交额(亿)</th><th>行业</th></tr>';
  (data.records||[]).forEach(r => {
    html += '<tr><td>'+r.code+'</td><td>'+r.name+'</td><td class="up">'+fmtNum(r.pct_chg,2)+'%</td><td>'+fmtNum(r.close,2)+'</td><td class="up">'+r.board_count+'</td><td>'+(r.first_seal_time||'-')+'</td><td>'+r.fail_count+'</td><td>'+fmtNum(r.turnover)+'%</td><td>'+fmtNum(r.amount/1e8,2)+'</td><td>'+r.industry+'</td></tr>';
  });
  html += '</table>';
  document.getElementById('ztTableWrap').innerHTML = html;
}

async function loadSignals() {
  const data = await fetchJSON('/api/signals');
  let html = '<table><tr><th>#</th><th>代码</th><th>名称</th><th>评分</th><th>连板</th><th>买入价</th><th>止损</th><th>行业</th><th>原因</th></tr>';
  (data.signals||[]).forEach((s,i) => {
    const barW = Math.min(s.score, 100);
    html += '<tr><td>'+(i+1)+'</td><td>'+s.code+'</td><td>'+s.name+'</td><td><span class="score-bar" style="width:'+barW+'px"></span>'+fmtNum(s.score)+'</td><td class="up">'+s.board_count+'</td><td>'+fmtNum(s.buy_price,2)+'</td><td>'+fmtNum(s.stop_loss,2)+'</td><td>'+s.industry+'</td><td>'+s.reason+'</td></tr>';
  });
  html += '</table>';
  document.getElementById('signalTableWrap').innerHTML = html;
}

async function loadPremium() {
  const data = await fetchJSON('/api/premium');
  if (!data || !data.length) return;

  document.getElementById('premiumGrid').innerHTML = data.filter(d=>d.board_count<=8).map(d =>
    '<div class="premium-card"><div class="board">'+d.board_count+'板</div><div class="stat">样本: '+d.sample_count+'</div><div class="stat">开盘溢价: <span class="'+(d.avg_open_premium>0?'up':'down')+'">'+fmtNum(d.avg_open_premium,2)+'%</span></div><div class="stat">正溢价率: '+fmtNum(d.win_rate)+'%</div><div class="stat">次日连板: '+fmtNum(d.next_zt_rate)+'%</div></div>'
  ).join('');

  new Chart(document.getElementById('premiumChart'), {
    type: 'bar', data: {
      labels: data.filter(d=>d.board_count<=8).map(d=>d.board_count+'板'),
      datasets: [
        {label: '开盘溢价(%)', data: data.filter(d=>d.board_count<=8).map(d=>d.avg_open_premium), backgroundColor: 'rgba(229,62,62,0.7)'},
        {label: '收盘溢价(%)', data: data.filter(d=>d.board_count<=8).map(d=>d.avg_close_premium), backgroundColor: 'rgba(66,153,225,0.7)'},
      ]
    }, options: {responsive: true, plugins: {legend: {labels: {color: '#a0aec0'}}}, scales: {x: {ticks: {color: '#718096'}}, y: {ticks: {color: '#718096'}}}}
  });
}

async function loadBacktest() {
  const data = await fetchJSON('/api/backtest');
  const cards = [
    {label: '总交易', value: data.total_trades, cls: 'neutral'},
    {label: '胜率', value: fmtNum(data.win_rate)+'%', cls: data.win_rate>50?'up':'down'},
    {label: '总收益', value: fmtNum(data.total_pnl,2)+'%', cls: data.total_pnl>0?'up':'down'},
    {label: '平均每笔', value: fmtNum(data.avg_pnl,2)+'%', cls: data.avg_pnl>0?'up':'down'},
  ];
  document.getElementById('btCards').innerHTML = cards.map(c =>
    '<div class="card"><div class="label">'+c.label+'</div><div class="value '+c.cls+'">'+c.value+'</div></div>'
  ).join('');

  if (data.curve && data.curve.length) {
    const labels = data.curve.map(c=>c.date.slice(5));
    const cumPnls = data.curve.map(c=>c.cum_pnl);
    new Chart(document.getElementById('btCurveChart'), {
      type: 'line', data: {
        labels, datasets: [{label: '累计收益(%)', data: cumPnls, borderColor: '#e53e3e', borderWidth: 1.5, pointRadius: 0, fill: {target: 'origin', above: 'rgba(229,62,62,0.1)', below: 'rgba(72,187,120,0.1)'}}]
      }, options: {responsive: true, plugins: {legend: {labels: {color: '#a0aec0'}}}, scales: {x: {ticks: {color: '#718096', maxTicksLimit: 20}}, y: {ticks: {color: '#718096'}}}}
    });
  }
}

async function loadHotRank() {
  const data = await fetchJSON('/api/hot');
  let html = '<table><tr><th>排名</th><th>代码</th><th>名称</th><th>排名变动</th></tr>';
  (data||[]).forEach(r => {
    const chg = r.rank_change > 0 ? '<span class="up">↑'+r.rank_change+'</span>' : r.rank_change < 0 ? '<span class="down">↓'+(-r.rank_change)+'</span>' : '-';
    html += '<tr><td>'+r.rank+'</td><td>'+r.code+'</td><td>'+r.name+'</td><td>'+chg+'</td></tr>';
  });
  html += '</table>';
  document.getElementById('hotRankTable').innerHTML = html;
}

(async function() {
  await loadOverview();
  await loadSentiment();
  loadZTList();
  loadSignals();
  loadPremium();
  loadBacktest();
  loadHotRank();
})();
</script>
</body>
</html>`
