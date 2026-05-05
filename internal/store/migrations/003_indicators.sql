-- 技术指标表
CREATE TABLE IF NOT EXISTS stock_indicators (
    code       VARCHAR(10) NOT NULL,
    date       DATE NOT NULL,
    -- 均线
    ma5        DOUBLE PRECISION DEFAULT 0,
    ma10       DOUBLE PRECISION DEFAULT 0,
    ma20       DOUBLE PRECISION DEFAULT 0,
    ma60       DOUBLE PRECISION DEFAULT 0,
    -- 成交量均线
    vma5       DOUBLE PRECISION DEFAULT 0,
    vma10      DOUBLE PRECISION DEFAULT 0,
    -- MACD
    dif        DOUBLE PRECISION DEFAULT 0,
    dea        DOUBLE PRECISION DEFAULT 0,
    macd       DOUBLE PRECISION DEFAULT 0,
    -- KDJ
    k_val      DOUBLE PRECISION DEFAULT 0,
    d_val      DOUBLE PRECISION DEFAULT 0,
    j_val      DOUBLE PRECISION DEFAULT 0,
    -- RSI
    rsi6       DOUBLE PRECISION DEFAULT 0,
    rsi12      DOUBLE PRECISION DEFAULT 0,
    -- BOLL
    boll_upper DOUBLE PRECISION DEFAULT 0,
    boll_mid   DOUBLE PRECISION DEFAULT 0,
    boll_lower DOUBLE PRECISION DEFAULT 0,
    -- 额外特征
    vol_ratio  DOUBLE PRECISION DEFAULT 0,  -- 量比(今日量/5日均量)
    is_break_ma20 BOOLEAN DEFAULT FALSE,     -- 是否突破20日均线
    consecutive_up INTEGER DEFAULT 0,        -- 连涨天数
    PRIMARY KEY (code, date)
);

CREATE INDEX IF NOT EXISTS idx_indicators_date ON stock_indicators(date);
