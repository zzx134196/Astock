package web

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>A股涨停板量化系统</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&display=swap');
:root {
  --bg: #f5f0e8;
  --card: rgba(255,255,255,0.82);
  --card-border: rgba(0,0,0,0.07);
  --card-shadow: 0 1px 3px rgba(0,0,0,0.04);
  --card-hover-shadow: 0 4px 16px rgba(0,0,0,0.07);
  --text: #1e293b;
  --text2: #64748b;
  --text3: #94a3b8;
  --green: #10b981;
  --red: #ef4444;
  --blue: #3b82f6;
  --purple: #8b5cf6;
  --amber: #f59e0b;
  --emerald: #059669;
  --radius: 16px;
}
*{margin:0;padding:0;box-sizing:border-box;}
body{font-family:'Inter',system-ui,-apple-system,'PingFang SC',sans-serif;background:var(--bg);color:var(--text);font-size:13px;min-height:100vh;}
::-webkit-scrollbar{width:5px;height:5px;}
::-webkit-scrollbar-track{background:transparent;}
::-webkit-scrollbar-thumb{background:#c8c0b4;border-radius:999px;}
::-webkit-scrollbar-thumb:hover{background:#a89e92;}
a{color:var(--blue);text-decoration:none;cursor:pointer;}

.topbar{height:48px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;border-bottom:1px solid rgba(0,0,0,0.08);background:rgba(253,250,246,0.9);backdrop-filter:blur(12px);position:sticky;top:0;z-index:100;}
.topbar h1{font-size:14px;font-weight:700;letter-spacing:0.5px;color:#1e293b;}
.topbar .logo-icon{width:22px;height:22px;border-radius:8px;display:flex;align-items:center;justify-content:center;background:linear-gradient(135deg,rgba(139,92,246,0.2),rgba(139,92,246,0.08));margin-right:8px;font-size:12px;}
.topbar .info{color:var(--text3);font-size:11px;display:flex;align-items:center;gap:8px;}
.topbar .status-dot{width:6px;height:6px;border-radius:50%;display:inline-block;}

.search-box{position:relative;margin-left:12px;}
.search-box input{width:200px;padding:6px 12px 6px 30px;border:1px solid rgba(0,0,0,0.08);border-radius:10px;background:rgba(0,0,0,0.03);color:var(--text);font-size:11px;outline:none;transition:all 0.2s;}
.search-box input:focus{background:white;border-color:rgba(0,0,0,0.12);box-shadow:0 2px 8px rgba(0,0,0,0.06);}
.search-box input::placeholder{color:var(--text3);}
.search-box svg{position:absolute;left:9px;top:50%;transform:translateY(-50%);width:13px;height:13px;fill:var(--text3);}
.search-dropdown{position:absolute;top:100%;left:0;width:300px;background:white;border-radius:12px;box-shadow:0 8px 30px rgba(0,0,0,0.12);z-index:1000;display:none;max-height:360px;overflow-y:auto;margin-top:4px;border:1px solid rgba(0,0,0,0.08);}
.search-dropdown.show{display:block;}
.search-item{padding:10px 14px;display:flex;align-items:center;gap:8px;cursor:pointer;border-bottom:1px solid rgba(0,0,0,0.04);transition:background 0.15s;}
.search-item:hover{background:rgba(0,0,0,0.03);}
.search-item .code{font-weight:700;color:var(--blue);min-width:56px;font-size:12px;}
.search-item .name{flex:1;font-size:12px;}
.search-item .ind{font-size:10px;color:var(--text3);}

.layout{display:flex;min-height:calc(100vh - 48px);}
.sidebar{width:60px;background:linear-gradient(180deg,rgba(253,250,246,1) 0%,var(--bg) 100%);border-right:1px solid rgba(0,0,0,0.08);display:flex;flex-direction:column;align-items:center;padding:12px 0;gap:2px;position:sticky;top:48px;height:calc(100vh - 48px);}
.nav-btn{width:48px;height:48px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:2px;border-radius:12px;cursor:pointer;transition:all 0.2s;border:none;background:none;color:var(--text3);font-size:9px;font-weight:500;position:relative;}
.nav-btn:hover{background:rgba(0,0,0,0.04);color:var(--text2);}
.nav-btn.active{background:rgba(16,185,129,0.12);color:var(--emerald);}
.nav-btn.active::before{content:'';position:absolute;left:0;top:50%;transform:translateY(-50%);width:3px;height:18px;background:linear-gradient(180deg,#10b981,#059669);border-radius:0 4px 4px 0;}
.nav-btn svg{width:17px;height:17px;stroke:currentColor;stroke-width:1.8;fill:none;}
.sidebar .sep{width:28px;height:1px;background:rgba(0,0,0,0.06);margin:6px 0;}

.main{flex:1;overflow-y:auto;padding:20px;background:linear-gradient(135deg,var(--bg) 0%,#f8f4ed 50%,var(--bg) 100%);}
.panel{display:none;}
.panel.active{display:block;animation:fadeIn 0.3s ease;}
@keyframes fadeIn{from{opacity:0;transform:translateY(6px);}to{opacity:1;transform:translateY(0);}}
@media(max-width:768px){.sidebar{display:none;}.main{padding:12px;}}

.grid2{display:grid;grid-template-columns:1fr 1fr;gap:12px;margin-bottom:12px;}
.grid3{display:grid;grid-template-columns:1fr 1fr 1fr;gap:12px;margin-bottom:12px;}
.grid4{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:10px;margin-bottom:12px;}
@media(max-width:768px){.grid2,.grid3{grid-template-columns:1fr;}}

.glass{background:var(--card);backdrop-filter:blur(12px);border:1px solid var(--card-border);box-shadow:var(--card-shadow);border-radius:var(--radius);padding:16px;transition:all 0.3s cubic-bezier(0.4,0,0.2,1);}
.glass:hover{border-color:rgba(0,0,0,0.12);box-shadow:var(--card-hover-shadow);}
.glass h3{font-size:12px;font-weight:700;margin-bottom:12px;color:var(--text);display:flex;align-items:center;gap:6px;}
.glass h3 .dot{width:6px;height:6px;border-radius:50%;background:var(--emerald);}

.stat{background:var(--card);backdrop-filter:blur(12px);border:1px solid var(--card-border);box-shadow:var(--card-shadow);border-radius:14px;padding:14px 16px;transition:all 0.3s;}
.stat:hover{box-shadow:var(--card-hover-shadow);}
.stat .label{font-size:10px;color:var(--text3);margin-bottom:3px;font-weight:500;text-transform:uppercase;letter-spacing:0.3px;}
.stat .value{font-size:20px;font-weight:800;line-height:1.3;letter-spacing:-0.5px;}
.stat .sub{font-size:10px;color:var(--text2);margin-top:2px;}
.up{color:var(--red);}
.down{color:var(--green);}

table{width:100%;border-collapse:separate;border-spacing:0;font-size:12px;}
thead th{padding:8px 10px;text-align:left;color:var(--text3);font-weight:600;font-size:10px;text-transform:uppercase;letter-spacing:0.3px;border-bottom:1px solid rgba(0,0,0,0.06);position:sticky;top:0;background:rgba(255,255,255,0.9);backdrop-filter:blur(8px);z-index:1;}
tbody td{padding:8px 10px;border-bottom:1px solid rgba(0,0,0,0.04);}
tbody tr{transition:background 0.15s;cursor:pointer;}
tbody tr:hover{background:rgba(0,0,0,0.02);}
.table-wrap{max-height:480px;overflow-y:auto;border-radius:12px;}

.tag{display:inline-block;padding:2px 8px;border-radius:6px;font-size:10px;font-weight:600;}
.tag-red{background:rgba(239,68,68,0.08);color:var(--red);}
.tag-green{background:rgba(16,185,129,0.08);color:var(--green);}
.tag-blue{background:rgba(59,130,246,0.08);color:var(--blue);}
.tag-purple{background:rgba(139,92,246,0.08);color:var(--purple);}
.tag-amber{background:rgba(245,158,11,0.08);color:var(--amber);}
.score-bar{display:inline-block;height:4px;border-radius:2px;background:linear-gradient(90deg,var(--emerald),var(--blue));vertical-align:middle;margin-right:4px;}
.empty{text-align:center;padding:48px 20px;color:var(--text3);}
.badge{display:inline-block;min-width:16px;text-align:center;padding:1px 5px;border-radius:8px;font-size:10px;font-weight:600;}
.badge-up{background:rgba(239,68,68,0.08);color:var(--red);}
.badge-down{background:rgba(16,185,129,0.08);color:var(--green);}

.sub-tabs{display:flex;gap:3px;margin-bottom:14px;flex-wrap:wrap;}
.sub-tab{padding:5px 14px;border-radius:10px;cursor:pointer;font-size:11px;color:var(--text2);background:var(--card);border:1px solid var(--card-border);font-weight:600;transition:all 0.2s;}
.sub-tab:hover{background:white;}
.sub-tab.active{background:var(--emerald);color:white;border-color:var(--emerald);box-shadow:0 2px 8px rgba(16,185,129,0.2);}
.sub-content{display:none;}
.sub-content.active{display:block;animation:fadeIn 0.25s ease;}

.date-picker{display:flex;align-items:center;gap:8px;margin-bottom:12px;}
.date-picker select,.date-picker input{padding:6px 10px;border:1px solid rgba(0,0,0,0.08);border-radius:10px;font-size:11px;background:white;color:var(--text);outline:none;}
.date-picker label{font-size:11px;color:var(--text2);font-weight:600;}

.page-title{display:flex;align-items:center;gap:10px;margin-bottom:16px;}
.page-title .icon-wrap{width:32px;height:32px;border-radius:10px;display:flex;align-items:center;justify-content:center;background:linear-gradient(135deg,white,var(--bg));border:1px solid var(--card-border);box-shadow:0 1px 3px rgba(0,0,0,0.04);}
.page-title .icon-wrap svg{width:15px;height:15px;stroke:var(--purple);stroke-width:1.8;fill:none;}
.page-title h2{font-size:16px;font-weight:800;letter-spacing:-0.3px;color:var(--text);}

.stock-header{display:flex;align-items:center;gap:16px;margin-bottom:12px;flex-wrap:wrap;}
.stock-header .sname{font-size:20px;font-weight:800;letter-spacing:-0.5px;}
.stock-header .scode{font-size:13px;color:var(--text2);}
.stock-header .sprice{font-size:22px;font-weight:800;font-variant-numeric:tabular-nums;}
.stock-header .schg{font-size:13px;font-weight:700;}
.concept-tags{display:flex;flex-wrap:wrap;gap:4px;margin-bottom:12px;}
.back-btn{display:inline-flex;align-items:center;gap:4px;padding:6px 14px;border-radius:10px;background:var(--card);border:1px solid var(--card-border);cursor:pointer;font-size:11px;color:var(--text2);margin-bottom:12px;transition:all 0.2s;}
.back-btn:hover{background:white;box-shadow:var(--card-hover-shadow);}
.tab-bar{display:flex;gap:3px;margin-bottom:12px;}
.tab-btn{padding:6px 14px;border-radius:10px;cursor:pointer;font-size:11px;color:var(--text2);background:var(--card);border:1px solid var(--card-border);font-weight:600;transition:all 0.2s;}
.tab-btn:hover{background:white;}
.tab-btn.active{background:var(--emerald);color:white;border-color:var(--emerald);box-shadow:0 2px 8px rgba(16,185,129,0.2);}
.tab-content{display:none;}
.tab-content.active{display:block;}

.trade-empty{text-align:center;padding:80px 20px;color:var(--text3);}
.trade-empty .icon{font-size:48px;margin-bottom:16px;opacity:0.3;}
.trade-empty h3{font-size:16px;font-weight:700;color:var(--text2);margin-bottom:8px;}
.trade-empty p{font-size:12px;line-height:1.8;max-width:400px;margin:0 auto;}
</style>
</head>
<body>

<div class="topbar">
  <div style="display:flex;align-items:center">
    <div class="logo-icon">⚡</div>
    <h1>A股量化</h1>
    <div class="search-box">
      <svg viewBox="0 0 24 24"><path d="M15.5 14h-.79l-.28-.27A6.47 6.47 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 5L20.49 19l-5-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/></svg>
      <input id="searchInput" placeholder="搜索代码/名称..." autocomplete="off"/>
      <div class="search-dropdown" id="searchDropdown"></div>
    </div>
  </div>
  <div class="info" id="headerInfo"><span class="status-dot" style="background:var(--green)"></span>加载中...</div>
</div>

<div class="layout">
  <div class="sidebar" id="sideNav">
    <button class="nav-btn active" onclick="navTo('data',this)" title="数据中心">
      <svg viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/></svg><span>数据</span>
    </button>
    <button class="nav-btn" onclick="navTo('strategy',this)" title="策略回测">
      <svg viewBox="0 0 24 24"><path d="M10 2v7.527a2 2 0 01-.211.896L4.72 20.55a1 1 0 00.9 1.45h12.76a1 1 0 00.9-1.45l-5.069-10.127A2 2 0 0114 9.527V2"/><path d="M8.5 2h7"/></svg><span>策略</span>
    </button>
    <button class="nav-btn" onclick="navTo('trade',this)" title="实盘交易">
      <svg viewBox="0 0 24 24"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg><span>交易</span>
    </button>
    <div class="sep"></div>
    <button class="nav-btn" onclick="navTo('system',this)" title="系统">
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg><span>系统</span>
    </button>
  </div>

  <div class="main">
    <!-- ============ 数据中心 ============ -->
    <div id="data" class="panel active">
      <div class="page-title">
        <div class="icon-wrap"><svg viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/></svg></div>
        <h2>数据中心</h2>
      </div>
      <div class="sub-tabs" id="dataSubTabs">
        <div class="sub-tab active" onclick="dataTab('overview',this)">市场概览</div>
        <div class="sub-tab" onclick="dataTab('ztlist',this)">涨停列表</div>
        <div class="sub-tab" onclick="dataTab('sentiment',this)">情绪周期</div>
        <div class="sub-tab" onclick="dataTab('premium',this)">溢价统计</div>
        <div class="sub-tab" onclick="dataTab('lhb',this)">龙虎榜</div>
        <div class="sub-tab" onclick="dataTab('flow',this)">资金流向</div>
        <div class="sub-tab" onclick="dataTab('hotrank',this)">人气排行</div>
      </div>

      <div id="d_overview" class="sub-content active">
        <div class="grid4" id="overviewCards"></div>
        <div class="grid2">
          <div class="glass"><h3><span class="dot"></span>涨停家数趋势</h3><canvas id="ztTrendChart"></canvas></div>
          <div class="glass"><h3><span class="dot"></span>连板天梯图</h3><canvas id="ladderChart"></canvas></div>
        </div>
        <div class="grid2">
          <div class="glass"><h3><span class="dot"></span>晋级率走势</h3><canvas id="promoChart"></canvas></div>
          <div class="glass"><h3><span class="dot"></span>近期热门板块</h3><div id="sectorHeatTable"></div></div>
        </div>
      </div>

      <div id="d_ztlist" class="sub-content">
        <div class="date-picker"><label>日期</label><select id="ztDateSelect" onchange="loadZTByDate()"></select></div>
        <div class="glass"><h3><span class="dot"></span>涨停个股 <span style="font-weight:400;color:var(--text3);font-size:10px;margin-left:4px" id="ztDateLabel"></span></h3><div class="table-wrap" id="ztTableWrap"></div></div>
      </div>

      <div id="d_sentiment" class="sub-content">
        <div class="date-picker"><label>周期</label><select id="sentDaysSelect" onchange="loadSentiment()"><option value="30">30天</option><option value="60" selected>60天</option><option value="120">120天</option><option value="250">全年</option></select></div>
        <div class="grid4" id="sentimentCards"></div>
        <div class="glass" style="margin-bottom:12px"><h3><span class="dot"></span>情绪周期(涨停MA)</h3><canvas id="sentimentChart"></canvas></div>
        <div class="grid2">
          <div class="glass"><h3><span class="dot"></span>最高连板</h3><canvas id="maxBoardChart"></canvas></div>
          <div class="glass"><h3><span class="dot"></span>炸板 & 跌停</h3><canvas id="failRateChart"></canvas></div>
        </div>
      </div>

      <div id="d_premium" class="sub-content">
        <div class="grid4" id="premiumCards"></div>
        <div class="grid2">
          <div class="glass"><h3><span class="dot"></span>各板次日溢价</h3><canvas id="premiumChart"></canvas></div>
          <div class="glass"><h3><span class="dot"></span>次日连板率</h3><canvas id="nextZTChart"></canvas></div>
        </div>
      </div>

      <div id="d_lhb" class="sub-content">
        <div class="date-picker"><label>日期</label><select id="lhbDateSelect" onchange="loadLHBByDate()"></select></div>
        <div class="glass"><h3><span class="dot"></span>龙虎榜 <span style="font-weight:400;color:var(--text3);font-size:10px;margin-left:4px" id="lhbDateLabel"></span></h3><div class="table-wrap" id="lhbTableWrap"></div></div>
      </div>

      <div id="d_flow" class="sub-content">
        <div class="date-picker"><label>日期</label><select id="flowDateSelect" onchange="loadFlowByDate()"></select></div>
        <div class="grid2">
          <div class="glass"><h3><span class="dot" style="background:var(--red)"></span>主力净流入 TOP30</h3><div class="table-wrap" id="inflowTable"></div></div>
          <div class="glass"><h3><span class="dot" style="background:var(--green)"></span>主力净流出 TOP30</h3><div class="table-wrap" id="outflowTable"></div></div>
        </div>
      </div>

      <div id="d_hotrank" class="sub-content">
        <div class="glass"><h3><span class="dot"></span>人气排行 TOP100</h3><div class="table-wrap" id="hotRankTableWrap"></div></div>
      </div>
    </div>

    <!-- ============ 策略回测 ============ -->
    <div id="strategy" class="panel">
      <div class="page-title">
        <div class="icon-wrap"><svg viewBox="0 0 24 24"><path d="M10 2v7.527a2 2 0 01-.211.896L4.72 20.55a1 1 0 00.9 1.45h12.76a1 1 0 00.9-1.45l-5.069-10.127A2 2 0 0114 9.527V2"/><path d="M8.5 2h7"/></svg></div>
        <h2>策略研发</h2>
      </div>
      <div class="sub-tabs" id="stratSubTabs">
        <div class="sub-tab active" onclick="stratTab('signals',this)">选股信号</div>
        <div class="sub-tab" onclick="stratTab('backtest',this)">回测报告</div>
        <div class="sub-tab" onclick="stratTab('params',this)">策略参数</div>
      </div>

      <div id="s_signals" class="sub-content active">
        <div class="glass"><h3><span class="dot"></span>最新选股信号</h3><div class="table-wrap" id="signalTableWrap"></div></div>
      </div>

      <div id="s_backtest" class="sub-content">
        <div class="grid4" id="btCards"></div>
        <div class="glass" style="margin-bottom:12px"><h3><span class="dot"></span>累计收益曲线</h3><canvas id="btCurveChart"></canvas></div>
        <div class="glass"><h3><span class="dot"></span>交易明细</h3><div class="table-wrap" id="btTradesTable"></div></div>
      </div>

      <div id="s_params" class="sub-content">
        <div class="glass">
          <h3><span class="dot"></span>策略参数 <span style="font-weight:400;color:var(--text3);font-size:10px;margin-left:4px">修改后需重新回测</span></h3>
          <div style="color:var(--text2);font-size:12px;line-height:2;">
            <p>当前策略类型: <b>涨停板打板</b></p>
            <p>选股模式: <b>收盘选股 + 竞价选股</b></p>
            <p style="margin-top:12px;color:var(--text3);font-size:11px;">策略参数调优功能正在开发中，后续将支持：</p>
            <ul style="margin-left:20px;color:var(--text3);font-size:11px;line-height:2;">
              <li>各维度评分权重调整</li>
              <li>过滤条件自定义（换手率、封板时间等）</li>
              <li>止损止盈比例设置</li>
              <li>回测参数（资金量、手续费、滑点等）</li>
              <li>策略对比分析</li>
            </ul>
          </div>
        </div>
      </div>
    </div>

    <!-- ============ 实盘交易 ============ -->
    <div id="trade" class="panel">
      <div class="page-title">
        <div class="icon-wrap"><svg viewBox="0 0 24 24"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg></div>
        <h2>实盘交易</h2>
      </div>
      <div class="trade-empty">
        <div class="icon">📡</div>
        <h3>即将上线</h3>
        <p>实盘交易模块正在规划中，后续将支持：<br><br>
        <b>1. 券商接入</b> — 对接同花顺/通达信/QMT等交易接口<br>
        <b>2. 自动下单</b> — 策略信号触发自动委托<br>
        <b>3. 持仓管理</b> — 实时持仓、盈亏跟踪<br>
        <b>4. 风控系统</b> — 仓位控制、止损保护<br>
        <b>5. 交易日志</b> — 完整的委托和成交记录</p>
      </div>
    </div>

    <!-- ============ 系统 ============ -->
    <div id="system" class="panel">
      <div class="page-title">
        <div class="icon-wrap"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg></div>
        <h2>系统管理</h2>
      </div>
      <div class="glass"><h3><span class="dot"></span>数据库统计</h3><div class="table-wrap" id="dbStatsTable"></div></div>
    </div>

    <!-- ============ 个股详情 ============ -->
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
        <div class="glass" style="margin-bottom:12px"><h3><span class="dot"></span>日K线 & 均线</h3><div style="position:relative"><canvas id="klineCanvas" style="width:100%;height:340px"></canvas></div></div>
        <div class="glass" style="margin-bottom:12px"><h3><span class="dot"></span>成交量</h3><canvas id="volChart"></canvas></div>
        <div class="grid2">
          <div class="glass"><h3><span class="dot"></span>MACD</h3><canvas id="macdChart"></canvas></div>
          <div class="glass"><h3><span class="dot"></span>KDJ</h3><canvas id="kdjChart"></canvas></div>
        </div>
      </div>
      <div id="stockFlow" class="tab-content">
        <div class="glass" style="margin-bottom:12px"><h3><span class="dot"></span>主力资金净流入</h3><canvas id="stockFlowChart"></canvas></div>
        <div class="glass"><h3><span class="dot"></span>资金明细</h3><div class="table-wrap" id="stockFlowTable"></div></div>
      </div>
      <div id="stockZthistory" class="tab-content">
        <div class="glass"><h3><span class="dot"></span>涨停记录</h3><div class="table-wrap" id="stockZTTable"></div></div>
      </div>
      <div id="stockLhbhistory" class="tab-content">
        <div class="glass"><h3><span class="dot"></span>龙虎榜记录</h3><div class="table-wrap" id="stockLHBTable"></div></div>
      </div>
      <div id="stockTechind" class="tab-content">
        <div class="grid2">
          <div class="glass"><h3><span class="dot"></span>RSI(6/12)</h3><canvas id="rsiChart"></canvas></div>
          <div class="glass"><h3><span class="dot"></span>BOLL通道</h3><canvas id="bollChart"></canvas></div>
        </div>
      </div>
    </div>
  </div>
</div>

<script>
const f=(n,d=1)=>n==null?'-':Number(n).toFixed(d);
const fW=n=>{if(n==null)return'-';const v=Math.abs(n);if(v>=1e8)return f(n/1e8,2)+'亿';if(v>=1e4)return f(n/1e4,1)+'万';return f(n,0);};
const cls=n=>n>0?'up':n<0?'down':'';
let charts={};
function destroyChart(id){if(charts[id]){charts[id].destroy();delete charts[id];}}
function makeChart(id,cfg){destroyChart(id);const el=document.getElementById(id);if(!el)return null;charts[id]=new Chart(el,cfg);return charts[id];}

let lastNav='data';
function navTo(id,el){
  document.getElementById('stockDetail').classList.remove('active');
  document.querySelectorAll('.panel').forEach(p=>p.classList.remove('active'));
  document.querySelectorAll('#sideNav .nav-btn').forEach(b=>b.classList.remove('active'));
  document.getElementById(id).classList.add('active');
  if(el)el.classList.add('active');
  lastNav=id;
}
function dataTab(id,el){
  document.querySelectorAll('#data .sub-content').forEach(c=>c.classList.remove('active'));
  document.querySelectorAll('#dataSubTabs .sub-tab').forEach(b=>b.classList.remove('active'));
  document.getElementById('d_'+id).classList.add('active');el.classList.add('active');
}
function stratTab(id,el){
  document.querySelectorAll('#strategy .sub-content').forEach(c=>c.classList.remove('active'));
  document.querySelectorAll('#stratSubTabs .sub-tab').forEach(b=>b.classList.remove('active'));
  document.getElementById('s_'+id).classList.add('active');el.classList.add('active');
}
function switchStockTab(id,el){
  document.querySelectorAll('#stockDetail .tab-content').forEach(c=>c.classList.remove('active'));
  document.querySelectorAll('#stockTabBar .tab-btn').forEach(b=>b.classList.remove('active'));
  const map={kline:'stockKline',flow:'stockFlow',zthistory:'stockZthistory',lhbhistory:'stockLhbhistory',techind:'stockTechind'};
  document.getElementById(map[id]).classList.add('active');el.classList.add('active');
}
async function api(url){return(await fetch(url)).json();}
function statCard(label,value,sub,cc){return '<div class="stat"><div class="label">'+label+'</div><div class="value '+(cc||'')+'">'+value+'</div>'+(sub?'<div class="sub">'+sub+'</div>':'')+'</div>';}
function openStock(code){
  document.querySelectorAll('.panel').forEach(p=>p.classList.remove('active'));
  document.querySelectorAll('#sideNav .nav-btn').forEach(b=>b.classList.remove('active'));
  document.getElementById('stockDetail').classList.add('active');
  loadStockDetail(code);
}
function closeStockDetail(){
  document.getElementById('stockDetail').classList.remove('active');
  document.getElementById(lastNav).classList.add('active');
  document.querySelectorAll('#sideNav .nav-btn').forEach(b=>{if(b.title&&b.title.indexOf(lastNav==='data'?'数据':lastNav==='strategy'?'策略':'')>=0)b.classList.add('active');});
  document.querySelector('#sideNav .nav-btn').classList.add('active');
}

let searchTimer;
document.getElementById('searchInput').addEventListener('input',function(){
  clearTimeout(searchTimer);const q=this.value.trim();
  if(!q){document.getElementById('searchDropdown').classList.remove('show');return;}
  searchTimer=setTimeout(async()=>{
    const data=await api('/api/stock/search?q='+encodeURIComponent(q));const dd=document.getElementById('searchDropdown');
    if(!data||!data.length){dd.classList.remove('show');return;}
    dd.innerHTML=data.map(s=>'<div class="search-item" onclick="openStock(\''+s.code+'\')"><span class="code">'+s.code+'</span><span class="name">'+s.name+'</span><span class="ind">'+(s.industry||'')+'</span></div>').join('');dd.classList.add('show');
  },300);
});
document.addEventListener('click',e=>{if(!e.target.closest('.search-box'))document.getElementById('searchDropdown').classList.remove('show');});

// === Data loaders ===
async function loadOverview(){
  const[ov,sent]=await Promise.all([api('/api/overview'),api('/api/sentiment')]);
  const latest=sent&&sent.length?sent[sent.length-1]:{};
  document.getElementById('headerInfo').innerHTML='<span class="status-dot" style="background:var(--green)"></span> '+(ov.date||'')+' · 涨停 '+(latest.zt_count||0)+' · 炸板 '+(latest.fail_count||0)+' · 最高 '+(latest.max_board||0)+'板';
  document.getElementById('overviewCards').innerHTML=[
    statCard('涨停家数',latest.zt_count||0,'今日涨停','up'),statCard('炸板',latest.fail_count||0,'',''),
    statCard('最高连板',(latest.max_board||0)+'板','','up'),statCard('首板→二板',f(latest.promo_1to2)+'%','晋级率',''),
    statCard('MA5',f(latest.zt_ma5,1),'5日均值',''),statCard('热门板块',latest.top_sector_1||'-',(latest.top_sector_1_count||0)+'只涨停',''),
  ].join('');
  if(!sent||!sent.length)return;const labels=sent.map(d=>d.date.slice(5));
  makeChart('ztTrendChart',{type:'bar',data:{labels,datasets:[
    {label:'涨停',data:sent.map(d=>d.zt_count),backgroundColor:'rgba(239,68,68,0.25)',borderColor:'rgba(239,68,68,0.6)',borderWidth:1,borderRadius:2,order:2},
    {label:'MA5',data:sent.map(d=>d.zt_ma5),type:'line',borderColor:'#3b82f6',borderWidth:1.5,pointRadius:0,tension:0.3,order:1},
    {label:'MA10',data:sent.map(d=>d.zt_ma10),type:'line',borderColor:'#8b5cf6',borderWidth:1.5,pointRadius:0,tension:0.3,order:1},
  ]},options:co()});
  makeChart('ladderChart',{type:'bar',data:{labels,datasets:[
    {label:'1板',data:sent.map(d=>d.board_1),backgroundColor:'#93c5fd'},{label:'2板',data:sent.map(d=>d.board_2),backgroundColor:'#fcd34d'},
    {label:'3板',data:sent.map(d=>d.board_3),backgroundColor:'#f97316'},{label:'4板',data:sent.map(d=>d.board_4),backgroundColor:'#ef4444'},
    {label:'5+',data:sent.map(d=>d.board_5plus),backgroundColor:'#8b5cf6'},
  ]},options:{...co(),scales:{x:{stacked:true,...ao()},y:{stacked:true,...ao()}}}});
  makeChart('promoChart',{type:'line',data:{labels,datasets:[
    {label:'1→2板',data:sent.map(d=>d.promo_1to2),borderColor:'#10b981',borderWidth:1.5,pointRadius:1,tension:0.3},
    {label:'2→3板',data:sent.map(d=>d.promo_2to3),borderColor:'#f97316',borderWidth:1.5,pointRadius:1,tension:0.3},
  ]},options:co()});
  let ht='<table><thead><tr><th>日期</th><th>TOP1</th><th>#</th><th>TOP2</th><th>#</th><th>TOP3</th><th>#</th></tr></thead><tbody>';
  sent.slice(-12).reverse().forEach(d=>{ht+='<tr><td>'+d.date.slice(5)+'</td><td>'+d.top_sector_1+'</td><td class="up">'+d.top_sector_1_count+'</td><td>'+d.top_sector_2+'</td><td class="up">'+d.top_sector_2_count+'</td><td>'+d.top_sector_3+'</td><td class="up">'+d.top_sector_3_count+'</td></tr>';});
  ht+='</tbody></table>';document.getElementById('sectorHeatTable').innerHTML=ht;
}
async function loadSentiment(){
  const days=document.getElementById('sentDaysSelect').value;const sent=await api('/api/sentiment?days='+days);if(!sent||!sent.length)return;const latest=sent[sent.length-1];
  document.getElementById('sentimentCards').innerHTML=[statCard('涨停',latest.zt_count,'','up'),statCard('炸板',latest.fail_count,'',''),statCard('跌停',latest.dt_count||0,'','down'),statCard('最高板',latest.max_board+'板','','up'),statCard('天梯',latest.board_1+'/'+latest.board_2+'/'+latest.board_3+'/'+latest.board_4+'/'+latest.board_5plus,'1/2/3/4/5+板','')].join('');
  const labels=sent.map(d=>d.date.slice(5));
  makeChart('sentimentChart',{type:'line',data:{labels,datasets:[{label:'涨停',data:sent.map(d=>d.zt_count),borderColor:'#ef4444',borderWidth:1,pointRadius:0,fill:{target:'origin',above:'rgba(239,68,68,0.04)'}},{label:'MA5',data:sent.map(d=>d.zt_ma5),borderColor:'#3b82f6',borderWidth:1.5,pointRadius:0,tension:0.3},{label:'MA10',data:sent.map(d=>d.zt_ma10),borderColor:'#8b5cf6',borderWidth:1.5,pointRadius:0,tension:0.3}]},options:co()});
  makeChart('maxBoardChart',{type:'line',data:{labels,datasets:[{label:'最高连板',data:sent.map(d=>d.max_board),borderColor:'#ef4444',borderWidth:2,pointRadius:1.5,tension:0.3,fill:{target:'origin',above:'rgba(239,68,68,0.04)'}}]},options:co()});
  makeChart('failRateChart',{type:'bar',data:{labels,datasets:[{label:'炸板',data:sent.map(d=>d.fail_count),backgroundColor:'rgba(245,158,11,0.3)',borderColor:'#f59e0b',borderWidth:1,borderRadius:2},{label:'跌停',data:sent.map(d=>d.dt_count||0),backgroundColor:'rgba(16,185,129,0.2)',borderColor:'#10b981',borderWidth:1,borderRadius:2}]},options:co()});
}
async function initZTDates(){const dates=await api('/api/zt/dates');const sel=document.getElementById('ztDateSelect');sel.innerHTML='';if(dates&&dates.length){dates.forEach(d=>{sel.innerHTML+='<option value="'+d+'">'+d+'</option>';});loadZTByDate();}}
async function loadZTByDate(){const date=document.getElementById('ztDateSelect').value;const data=await api('/api/zt/today?date='+date);document.getElementById('ztDateLabel').textContent=data.date+' · '+data.count+'只';if(!data.records||!data.records.length){document.getElementById('ztTableWrap').innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>代码</th><th>名称</th><th>涨幅</th><th>收盘</th><th>连板</th><th>封板</th><th>炸板</th><th>换手</th><th>成交额</th><th>行业</th></tr></thead><tbody>';data.records.forEach(r=>{const bt=r.board_count>=3?'tag-red':r.board_count==2?'tag-amber':'tag-blue';h+='<tr onclick="openStock(\''+r.code+'\')"><td><b>'+r.code+'</b></td><td>'+r.name+'</td><td class="up">'+f(r.pct_chg,2)+'%</td><td>'+f(r.close,2)+'</td><td><span class="tag '+bt+'">'+r.board_count+'板</span></td><td>'+(r.first_seal_time||'-')+'</td><td>'+(r.fail_count||0)+'</td><td>'+f(r.turnover)+'%</td><td>'+fW(r.amount)+'</td><td><span class="tag tag-purple">'+(r.industry||'')+'</span></td></tr>';});
  h+='</tbody></table>';document.getElementById('ztTableWrap').innerHTML=h;}
async function loadPremium(){const data=await api('/api/premium');if(!data||!data.length)return;const items=data.filter(d=>d.board_count>=1&&d.board_count<=8);
  document.getElementById('premiumCards').innerHTML=items.map(d=>'<div class="stat" style="text-align:center"><div style="font-size:22px;font-weight:800;color:var(--red)">'+d.board_count+'</div><div style="font-size:10px;color:var(--text3)">板</div><div style="font-size:14px;font-weight:700;margin-top:4px" class="'+cls(d.avg_open_premium)+'">'+f(d.avg_open_premium,2)+'%</div><div style="font-size:9px;color:var(--text2)">溢价 · '+d.sample_count+'样本</div><div style="font-size:9px;color:var(--text2)">正溢价 '+f(d.win_rate)+'%</div></div>').join('');
  makeChart('premiumChart',{type:'bar',data:{labels:items.map(d=>d.board_count+'板'),datasets:[{label:'开盘',data:items.map(d=>d.avg_open_premium),backgroundColor:'rgba(239,68,68,0.3)',borderRadius:4},{label:'收盘',data:items.map(d=>d.avg_close_premium),backgroundColor:'rgba(59,130,246,0.3)',borderRadius:4},{label:'最高',data:items.map(d=>d.avg_max_premium),backgroundColor:'rgba(139,92,246,0.2)',borderRadius:4}]},options:co()});
  makeChart('nextZTChart',{type:'bar',data:{labels:items.map(d=>d.board_count+'板'),datasets:[{label:'连板率%',data:items.map(d=>d.next_zt_rate),backgroundColor:'rgba(239,68,68,0.3)',borderRadius:4}]},options:co()});}
async function initLHBDates(){const dates=await api('/api/lhb/dates');const sel=document.getElementById('lhbDateSelect');sel.innerHTML='';if(dates&&dates.length){dates.forEach(d=>{sel.innerHTML+='<option value="'+d+'">'+d+'</option>';});loadLHBByDate();}}
async function loadLHBByDate(){const date=document.getElementById('lhbDateSelect').value;const data=await api('/api/lhb?date='+date);document.getElementById('lhbDateLabel').textContent=(data.date||'')+' · '+((data.records||[]).length)+'只';if(!data.records||!data.records.length){document.getElementById('lhbTableWrap').innerHTML='<div class="empty">暂无数据</div>';return;}
  let h='<table><thead><tr><th>代码</th><th>名称</th><th>涨幅</th><th>净买入</th><th>买入</th><th>卖出</th><th>原因</th></tr></thead><tbody>';data.records.forEach(r=>{h+='<tr onclick="openStock(\''+r.code+'\')"><td><b>'+r.code+'</b></td><td>'+r.name+'</td><td class="'+cls(r.pct_chg)+'">'+f(r.pct_chg,2)+'%</td><td class="'+cls(r.net_amount)+'"><b>'+fW(r.net_amount)+'</b></td><td class="up">'+fW(r.buy_amount)+'</td><td class="down">'+fW(r.sell_amount)+'</td><td style="max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+r.reason+'</td></tr>';});
  h+='</tbody></table>';document.getElementById('lhbTableWrap').innerHTML=h;}
async function initFlowDates(){const dates=await api('/api/flow/dates');const sel=document.getElementById('flowDateSelect');sel.innerHTML='';if(dates&&dates.length){dates.forEach(d=>{sel.innerHTML+='<option value="'+d+'">'+d+'</option>';});loadFlowByDate();}}
async function loadFlowByDate(){const date=document.getElementById('flowDateSelect').value;const data=await api('/api/flow/top?date='+date);flowTbl(data.inflows,'inflowTable');flowTbl(data.outflows,'outflowTable');}
function flowTbl(items,el){if(!items||!items.length){document.getElementById(el).innerHTML='<div class="empty">暂无</div>';return;}let h='<table><thead><tr><th>代码</th><th>名称</th><th>主力净流入</th><th>超大单</th><th>大单</th></tr></thead><tbody>';items.forEach(r=>{h+='<tr onclick="openStock(\''+r.code+'\')"><td><b>'+r.code+'</b></td><td>'+r.name+'</td><td class="'+cls(r.main_net)+'"><b>'+fW(r.main_net)+'</b></td><td class="'+cls(r.huge_net)+'">'+fW(r.huge_net)+'</td><td class="'+cls(r.big_net)+'">'+fW(r.big_net)+'</td></tr>';});h+='</tbody></table>';document.getElementById(el).innerHTML=h;}
async function loadSignals(){const data=await api('/api/signals');if(!data.signals||!data.signals.length){document.getElementById('signalTableWrap').innerHTML='<div class="empty">暂无选股信号</div>';return;}
  let h='<table><thead><tr><th>#</th><th>代码</th><th>名称</th><th>评分</th><th>连板</th><th>买入</th><th>止损</th><th>行业</th><th>原因</th></tr></thead><tbody>';data.signals.forEach((s,i)=>{const w=Math.min(s.score,100);h+='<tr onclick="openStock(\''+s.code+'\')"><td>'+(i+1)+'</td><td><b>'+s.code+'</b></td><td>'+s.name+'</td><td><span class="score-bar" style="width:'+w+'px"></span>'+f(s.score)+'</td><td><span class="tag tag-red">'+s.board_count+'板</span></td><td>'+f(s.buy_price,2)+'</td><td class="down">'+f(s.stop_loss,2)+'</td><td><span class="tag tag-purple">'+s.industry+'</span></td><td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+s.reason+'</td></tr>';});
  h+='</tbody></table>';document.getElementById('signalTableWrap').innerHTML=h;}
async function loadHotRank(){const data=await api('/api/hot');if(!data||!data.length){document.getElementById('hotRankTableWrap').innerHTML='<div class="empty">暂无</div>';return;}
  let h='<table><thead><tr><th>排名</th><th>代码</th><th>名称</th><th>变动</th></tr></thead><tbody>';data.forEach(r=>{const chg=r.rank_change>0?'<span class="badge badge-up">↑'+r.rank_change+'</span>':r.rank_change<0?'<span class="badge badge-down">↓'+(-r.rank_change)+'</span>':'<span style="color:var(--text3)">-</span>';h+='<tr onclick="openStock(\''+r.code+'\')"><td><b>'+r.rank+'</b></td><td>'+r.code+'</td><td><b>'+r.name+'</b></td><td>'+chg+'</td></tr>';});
  h+='</tbody></table>';document.getElementById('hotRankTableWrap').innerHTML=h;}
async function loadBacktest(){const data=await api('/api/backtest');
  document.getElementById('btCards').innerHTML=[statCard('总交易',data.total_trades||0,'',''),statCard('胜率',f(data.win_rate)+'%','',data.win_rate>50?'up':'down'),statCard('总收益',f(data.total_pnl,2)+'%','',data.total_pnl>0?'up':'down'),statCard('平均',f(data.avg_pnl,2)+'%/笔','',data.avg_pnl>0?'up':'down'),statCard('最大回撤',f(data.max_drawdown,2)+'%','','down'),statCard('盈亏比',f(data.profit_ratio,2)+'','')].join('');
  if(data.curve&&data.curve.length){const labels=data.curve.filter((_,i)=>i%5===0).map(c=>c.date.slice(5));const vals=data.curve.filter((_,i)=>i%5===0).map(c=>c.cum_pnl);
    makeChart('btCurveChart',{type:'line',data:{labels,datasets:[{label:'累计%',data:vals,borderColor:'#10b981',borderWidth:1.5,pointRadius:0,fill:{target:'origin',above:'rgba(16,185,129,0.06)',below:'rgba(239,68,68,0.06)'}}]},options:co()});}
  if(data.trades&&data.trades.length){let h='<table><thead><tr><th>日期</th><th>代码</th><th>名称</th><th>买入</th><th>卖出</th><th>盈亏%</th></tr></thead><tbody>';data.trades.slice(-50).reverse().forEach(t=>{h+='<tr onclick="openStock(\''+t.code+'\')"><td>'+t.buy_date+'</td><td><b>'+t.code+'</b></td><td>'+t.name+'</td><td>'+f(t.buy_price,2)+'</td><td>'+f(t.sell_price,2)+'</td><td class="'+cls(t.pnl_pct)+'"><b>'+f(t.pnl_pct,2)+'%</b></td></tr>';});
    h+='</tbody></table>';document.getElementById('btTradesTable').innerHTML=h;}else{document.getElementById('btTradesTable').innerHTML='<div class="empty">暂无回测交易记录</div>';}}
async function loadDBStats(){const data=await api('/api/stats');let h='<table><thead><tr><th>数据表</th><th>记录数</th></tr></thead><tbody>';let total=0;(data||[]).forEach(r=>{total+=r.count;h+='<tr><td>'+r.table+'</td><td><b>'+Number(r.count).toLocaleString()+'</b></td></tr>';});h+='<tr style="background:rgba(0,0,0,0.02)"><td><b>总计</b></td><td><b style="color:var(--emerald)">'+total.toLocaleString()+'</b></td></tr></tbody></table>';document.getElementById('dbStatsTable').innerHTML=h;}

// === Stock Detail ===
async function loadStockDetail(code){
  const data=await api('/api/stock?code='+code+'&months=6');if(!data)return;const quotes=data.quotes||[];const last=quotes.length?quotes[quotes.length-1]:null;
  let hdr='<div class="stock-header"><span class="sname">'+(data.name||code)+'</span><span class="scode">'+code+' · '+(data.market||'')+' · '+(data.industry||'')+'</span>';
  if(last){hdr+='<span class="sprice '+cls(last.pct_chg)+'">'+f(last.close,2)+'</span><span class="schg '+cls(last.pct_chg)+'">'+(last.pct_chg>0?'+':'')+f(last.pct_chg,2)+'%</span>';}
  hdr+='</div>';document.getElementById('stockHeaderArea').innerHTML=hdr;
  document.getElementById('conceptTags').innerHTML=(data.concepts||[]).slice(0,15).map(c=>'<span class="tag tag-blue">'+c.board_name+'</span>').join('');
  document.querySelectorAll('#stockDetail .tab-content').forEach(c=>c.classList.remove('active'));document.querySelectorAll('#stockTabBar .tab-btn').forEach(b=>b.classList.remove('active'));
  document.getElementById('stockKline').classList.add('active');document.querySelector('#stockTabBar .tab-btn').classList.add('active');
  renderKline(data);renderStockFlow(data);renderZTHistory(data);renderLHBHistory(data);renderTechIndicators(data);
}
function renderKline(data){const quotes=data.quotes||[];if(!quotes.length)return;const indicators=data.indicators||[];const labels=quotes.map(q=>q.date.slice(5,10));
  drawCandlestick('klineCanvas',quotes,indicators);
  const vColors=quotes.map(q=>q.pct_chg>=0?'rgba(239,68,68,0.35)':'rgba(16,185,129,0.35)');
  makeChart('volChart',{type:'bar',data:{labels,datasets:[{label:'成交量',data:quotes.map(q=>q.volume),backgroundColor:vColors,borderRadius:1}]},options:co()});
  if(indicators.length===quotes.length){const mL=labels;const mc=indicators.map(d=>(d.macd||0)>=0?'rgba(239,68,68,0.35)':'rgba(16,185,129,0.35)');
    makeChart('macdChart',{type:'bar',data:{labels:mL,datasets:[{label:'MACD',data:indicators.map(d=>d.macd),backgroundColor:mc,borderRadius:1,order:2},{label:'DIF',data:indicators.map(d=>d.dif),type:'line',borderColor:'#3b82f6',borderWidth:1.5,pointRadius:0,tension:0.3,order:1},{label:'DEA',data:indicators.map(d=>d.dea),type:'line',borderColor:'#f59e0b',borderWidth:1.5,pointRadius:0,tension:0.3,order:1}]},options:co()});
    makeChart('kdjChart',{type:'line',data:{labels:mL,datasets:[{label:'K',data:indicators.map(d=>d.k),borderColor:'#3b82f6',borderWidth:1.5,pointRadius:0,tension:0.3},{label:'D',data:indicators.map(d=>d.d),borderColor:'#f59e0b',borderWidth:1.5,pointRadius:0,tension:0.3},{label:'J',data:indicators.map(d=>d.j),borderColor:'#8b5cf6',borderWidth:1.5,pointRadius:0,tension:0.3}]},options:co()});}}
function renderStockFlow(data){const flows=data.flows||[];if(!flows.length){document.getElementById('stockFlowTable').innerHTML='<div class="empty">暂无</div>';return;}
  const labels=flows.map(f=>f.date.slice(5));makeChart('stockFlowChart',{type:'bar',data:{labels,datasets:[{label:'主力净流入',data:flows.map(f=>f.main_net),backgroundColor:flows.map(f=>f.main_net>=0?'rgba(239,68,68,0.35)':'rgba(16,185,129,0.35)'),borderRadius:1}]},options:co()});
  let h='<table><thead><tr><th>日期</th><th>主力净流入</th><th>超大单</th><th>大单</th><th>中单</th><th>小单</th></tr></thead><tbody>';flows.slice().reverse().forEach(r=>{h+='<tr><td>'+r.date+'</td><td class="'+cls(r.main_net)+'"><b>'+fW(r.main_net)+'</b></td><td class="'+cls(r.huge_net)+'">'+fW(r.huge_net)+'</td><td class="'+cls(r.big_net)+'">'+fW(r.big_net)+'</td><td class="'+cls(r.mid_net)+'">'+fW(r.mid_net)+'</td><td class="'+cls(r.small_net)+'">'+fW(r.small_net)+'</td></tr>';});h+='</tbody></table>';document.getElementById('stockFlowTable').innerHTML=h;}
function renderZTHistory(data){const zt=data.zt_records||[];if(!zt.length){document.getElementById('stockZTTable').innerHTML='<div class="empty">无涨停记录</div>';return;}
  let h='<table><thead><tr><th>日期</th><th>连板</th><th>涨幅</th><th>收盘</th><th>封板</th><th>炸板</th><th>换手</th><th>成交额</th></tr></thead><tbody>';zt.slice().reverse().forEach(r=>{const bt=r.board_count>=3?'tag-red':r.board_count===2?'tag-amber':'tag-blue';h+='<tr><td>'+r.date.slice(0,10)+'</td><td><span class="tag '+bt+'">'+r.board_count+'板</span></td><td class="up">'+f(r.pct_chg,2)+'%</td><td>'+f(r.close,2)+'</td><td>'+(r.first_seal_time||'-')+'</td><td>'+(r.fail_count||0)+'</td><td>'+f(r.turnover)+'%</td><td>'+fW(r.amount)+'</td></tr>';});h+='</tbody></table>';document.getElementById('stockZTTable').innerHTML=h;}
function renderLHBHistory(data){const lhb=data.lhb||[];if(!lhb.length){document.getElementById('stockLHBTable').innerHTML='<div class="empty">无龙虎榜记录</div>';return;}
  let h='<table><thead><tr><th>日期</th><th>净买入</th><th>买入</th><th>卖出</th><th>原因</th></tr></thead><tbody>';lhb.forEach(r=>{h+='<tr><td>'+r.date+'</td><td class="'+cls(r.net_amount)+'"><b>'+fW(r.net_amount)+'</b></td><td class="up">'+fW(r.buy_amount)+'</td><td class="down">'+fW(r.sell_amount)+'</td><td>'+r.reason+'</td></tr>';});h+='</tbody></table>';document.getElementById('stockLHBTable').innerHTML=h;}
function renderTechIndicators(data){const quotes=data.quotes||[];const indicators=data.indicators||[];if(!indicators.length||indicators.length!==quotes.length)return;const labels=quotes.map(q=>q.date.slice(5,10));
  makeChart('rsiChart',{type:'line',data:{labels,datasets:[{label:'RSI6',data:indicators.map(d=>d.rsi6),borderColor:'#3b82f6',borderWidth:1.5,pointRadius:0,tension:0.3},{label:'RSI12',data:indicators.map(d=>d.rsi12),borderColor:'#f59e0b',borderWidth:1.5,pointRadius:0,tension:0.3}]},options:{...co(),scales:{x:ao(),y:{...ao(),min:0,max:100}}}});
  makeChart('bollChart',{type:'line',data:{labels,datasets:[{label:'收盘',data:quotes.map(q=>q.close),borderColor:'#ef4444',borderWidth:1.5,pointRadius:0,tension:0.1},{label:'上轨',data:indicators.map(d=>d.boll_upper),borderColor:'#8b5cf6',borderWidth:1,borderDash:[4,2],pointRadius:0,tension:0.3},{label:'中轨',data:indicators.map(d=>d.boll_mid),borderColor:'#3b82f6',borderWidth:1,pointRadius:0,tension:0.3},{label:'下轨',data:indicators.map(d=>d.boll_lower),borderColor:'#8b5cf6',borderWidth:1,borderDash:[4,2],pointRadius:0,tension:0.3}]},options:co()});}

function drawCandlestick(cid,quotes,indicators){
  const canvas=document.getElementById(cid);const dpr=window.devicePixelRatio||1;const rect=canvas.parentElement.getBoundingClientRect();const W=rect.width;const H=340;
  canvas.width=W*dpr;canvas.height=H*dpr;canvas.style.width=W+'px';canvas.style.height=H+'px';const ctx=canvas.getContext('2d');ctx.scale(dpr,dpr);
  const n=quotes.length;if(!n)return;const pad={top:20,right:55,bottom:28,left:10};const cw=W-pad.left-pad.right;const ch=H-pad.top-pad.bottom;
  let minP=Infinity,maxP=-Infinity;quotes.forEach(q=>{if(q.low<minP)minP=q.low;if(q.high>maxP)maxP=q.high;});
  if(indicators.length===n)indicators.forEach(d=>{[d.ma5,d.ma10,d.ma20].forEach(v=>{if(v&&v>0){if(v<minP)minP=v;if(v>maxP)maxP=v;}});});
  const pr=maxP-minP||1;const pp=pr*0.05;minP-=pp;maxP+=pp;
  const toY=p=>pad.top+ch*(1-(p-minP)/(maxP-minP));const gap=cw/n;const barW=Math.max(1,gap*0.65);
  ctx.strokeStyle='rgba(0,0,0,0.04)';ctx.lineWidth=1;
  for(let i=0;i<=4;i++){const p=minP+(maxP-minP)*i/4;const y=toY(p);ctx.beginPath();ctx.moveTo(pad.left,y);ctx.lineTo(W-pad.right,y);ctx.stroke();ctx.fillStyle='#94a3b8';ctx.font='10px Inter,sans-serif';ctx.textAlign='right';ctx.fillText(p.toFixed(2),W-pad.right+42,y+3);}
  ctx.fillStyle='#94a3b8';ctx.font='10px Inter,sans-serif';ctx.textAlign='center';const ls=Math.max(1,Math.floor(n/10));
  for(let i=0;i<n;i+=ls){ctx.fillText(quotes[i].date.slice(5,10),pad.left+i*gap+gap/2,H-8);}
  for(let i=0;i<n;i++){const q=quotes[i];const x=pad.left+i*gap+gap/2;const isUp=q.close>=q.open;const color=isUp?'#ef4444':'#10b981';
    ctx.strokeStyle=color;ctx.lineWidth=1;ctx.beginPath();ctx.moveTo(x,toY(q.high));ctx.lineTo(x,toY(q.low));ctx.stroke();
    const oY=toY(q.open);const cY=toY(q.close);const bH=Math.max(1,Math.abs(cY-oY));const bT=Math.min(oY,cY);
    if(isUp){ctx.fillStyle='rgba(255,255,255,0.95)';ctx.fillRect(x-barW/2,bT,barW,bH);ctx.strokeStyle=color;ctx.lineWidth=1;ctx.strokeRect(x-barW/2,bT,barW,bH);}
    else{ctx.fillStyle=color;ctx.fillRect(x-barW/2,bT,barW,bH);}}
  const maL=[{key:'ma5',color:'#3b82f6',label:'MA5'},{key:'ma10',color:'#f59e0b',label:'MA10'},{key:'ma20',color:'#8b5cf6',label:'MA20'}];
  if(indicators.length===n){maL.forEach(ma=>{ctx.strokeStyle=ma.color;ctx.lineWidth=1.2;ctx.beginPath();let s=false;for(let i=0;i<n;i++){const v=indicators[i][ma.key];if(!v||v<=0){s=false;continue;}const x=pad.left+i*gap+gap/2;const y=toY(v);if(!s){ctx.moveTo(x,y);s=true;}else ctx.lineTo(x,y);}ctx.stroke();});
    let lx=pad.left+5;maL.forEach(ma=>{ctx.fillStyle=ma.color;ctx.font='bold 10px Inter,sans-serif';ctx.fillRect(lx,6,12,3);lx+=14;ctx.fillText(ma.label,lx,10);lx+=34;});}
  canvas.onmousemove=function(e){const r=canvas.getBoundingClientRect();const mx=e.clientX-r.left;const idx=Math.floor((mx-pad.left)/gap);if(idx<0||idx>=n){canvas.title='';return;}const q=quotes[idx];
    canvas.title=q.date.slice(0,10)+'\\n开:'+q.open.toFixed(2)+' 高:'+q.high.toFixed(2)+' 低:'+q.low.toFixed(2)+' 收:'+q.close.toFixed(2)+'\\n涨幅:'+q.pct_chg.toFixed(2)+'%';};
}

function co(){return{responsive:true,plugins:{legend:{labels:{color:'#94a3b8',font:{size:10,family:'Inter'},boxWidth:10}},tooltip:{mode:'index',intersect:false,backgroundColor:'rgba(30,41,59,0.9)',titleFont:{size:11},bodyFont:{size:11},padding:8,cornerRadius:8}},scales:{x:ao(),y:ao()},interaction:{mode:'index',intersect:false}};}
function ao(){return{ticks:{color:'#94a3b8',font:{size:9,family:'Inter'},maxRotation:0},grid:{color:'rgba(0,0,0,0.03)'}};}

(async()=>{await loadOverview();loadSentiment();initZTDates();loadPremium();initLHBDates();initFlowDates();loadSignals();loadHotRank();loadBacktest();loadDBStats();})();
</script>
</body>
</html>` + ""
