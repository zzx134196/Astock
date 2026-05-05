-- 龙虎榜数据表
CREATE TABLE IF NOT EXISTS lhb_records (
    code          VARCHAR(10) NOT NULL,
    date          DATE NOT NULL,
    name          VARCHAR(20) DEFAULT '',
    pct_chg       DOUBLE PRECISION,
    close         DOUBLE PRECISION,
    net_amount    DOUBLE PRECISION DEFAULT 0,  -- 龙虎榜净买入额
    buy_amount    DOUBLE PRECISION DEFAULT 0,  -- 买入总额
    sell_amount   DOUBLE PRECISION DEFAULT 0,  -- 卖出总额
    turnover      DOUBLE PRECISION DEFAULT 0,
    reason        TEXT DEFAULT '',              -- 上榜原因
    PRIMARY KEY (code, date)
);

CREATE INDEX IF NOT EXISTS idx_lhb_date ON lhb_records(date);

-- 龙虎榜席位明细
CREATE TABLE IF NOT EXISTS lhb_detail (
    id            BIGSERIAL PRIMARY KEY,
    code          VARCHAR(10) NOT NULL,
    date          DATE NOT NULL,
    dept_name     VARCHAR(100) DEFAULT '',     -- 营业部名称
    side          VARCHAR(4) NOT NULL,          -- buy / sell
    buy_amount    DOUBLE PRECISION DEFAULT 0,
    sell_amount   DOUBLE PRECISION DEFAULT 0,
    net_amount    DOUBLE PRECISION DEFAULT 0,
    rank          INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_lhb_detail_date ON lhb_detail(date, code);
CREATE INDEX IF NOT EXISTS idx_lhb_detail_dept ON lhb_detail(dept_name);

-- 板块概念表
CREATE TABLE IF NOT EXISTS sectors (
    code        VARCHAR(20) PRIMARY KEY,
    name        VARCHAR(50) NOT NULL,
    sector_type VARCHAR(10) NOT NULL,   -- industry / concept
    updated_at  TIMESTAMP DEFAULT NOW()
);

-- 板块资金流向表
CREATE TABLE IF NOT EXISTS sector_flow (
    sector_code   VARCHAR(20) NOT NULL,
    date          DATE NOT NULL,
    pct_chg       DOUBLE PRECISION DEFAULT 0,
    main_net      DOUBLE PRECISION DEFAULT 0,    -- 主力净流入
    huge_net      DOUBLE PRECISION DEFAULT 0,    -- 超大单净流入
    big_net       DOUBLE PRECISION DEFAULT 0,    -- 大单净流入
    mid_net       DOUBLE PRECISION DEFAULT 0,    -- 中单净流入
    small_net     DOUBLE PRECISION DEFAULT 0,    -- 小单净流入
    PRIMARY KEY (sector_code, date)
);

CREATE INDEX IF NOT EXISTS idx_sector_flow_date ON sector_flow(date);

-- 个股资金流向表
CREATE TABLE IF NOT EXISTS stock_flow (
    code          VARCHAR(10) NOT NULL,
    date          DATE NOT NULL,
    main_net      DOUBLE PRECISION DEFAULT 0,
    huge_net      DOUBLE PRECISION DEFAULT 0,
    big_net       DOUBLE PRECISION DEFAULT 0,
    mid_net       DOUBLE PRECISION DEFAULT 0,
    small_net     DOUBLE PRECISION DEFAULT 0,
    PRIMARY KEY (code, date)
);

CREATE INDEX IF NOT EXISTS idx_stock_flow_date ON stock_flow(date);

-- 异动记录表
CREATE TABLE IF NOT EXISTS stock_changes (
    id            BIGSERIAL PRIMARY KEY,
    code          VARCHAR(10) NOT NULL,
    name          VARCHAR(20) DEFAULT '',
    date          DATE NOT NULL,
    change_time   VARCHAR(10) DEFAULT '',
    change_type   VARCHAR(30) DEFAULT '',      -- 火箭发射/大笔买入/涨停/跌停等
    info          TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_changes_date ON stock_changes(date);
CREATE INDEX IF NOT EXISTS idx_changes_code ON stock_changes(code, date);

-- 涨停池扩展(强势股池/炸板股池/跌停股池)
CREATE TABLE IF NOT EXISTS zt_pool_ext (
    code          VARCHAR(10) NOT NULL,
    date          DATE NOT NULL,
    name          VARCHAR(20) DEFAULT '',
    pool_type     VARCHAR(10) NOT NULL,    -- strong/fail/dt/sub_new
    pct_chg       DOUBLE PRECISION DEFAULT 0,
    close         DOUBLE PRECISION DEFAULT 0,
    amount        DOUBLE PRECISION DEFAULT 0,
    turnover      DOUBLE PRECISION DEFAULT 0,
    extra_info    TEXT DEFAULT '',
    PRIMARY KEY (code, date, pool_type)
);

CREATE INDEX IF NOT EXISTS idx_zt_pool_ext_date ON zt_pool_ext(date, pool_type);
