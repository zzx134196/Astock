-- 股票基本信息表
CREATE TABLE IF NOT EXISTS stocks (
    code       VARCHAR(10) PRIMARY KEY,
    name       VARCHAR(20) NOT NULL,
    market     VARCHAR(4)  NOT NULL, -- SH / SZ
    industry   VARCHAR(50) DEFAULT '',
    list_date  DATE,
    is_st      BOOLEAN DEFAULT FALSE,
    total_share DOUBLE PRECISION DEFAULT 0,
    float_share DOUBLE PRECISION DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stocks_market ON stocks(market);
CREATE INDEX IF NOT EXISTS idx_stocks_industry ON stocks(industry);

-- 日K线数据表
CREATE TABLE IF NOT EXISTS daily_quotes (
    code      VARCHAR(10) NOT NULL,
    date      DATE NOT NULL,
    open      DOUBLE PRECISION,
    close     DOUBLE PRECISION,
    high      DOUBLE PRECISION,
    low       DOUBLE PRECISION,
    volume    DOUBLE PRECISION, -- 成交量(手)
    amount    DOUBLE PRECISION, -- 成交额(元)
    pct_chg   DOUBLE PRECISION, -- 涨跌幅(%)
    change    DOUBLE PRECISION, -- 涨跌额
    amplitude DOUBLE PRECISION, -- 振幅(%)
    turnover  DOUBLE PRECISION, -- 换手率(%)
    pre_close DOUBLE PRECISION,
    PRIMARY KEY (code, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_quotes_date ON daily_quotes(date);
CREATE INDEX IF NOT EXISTS idx_daily_quotes_pct ON daily_quotes(date, pct_chg);

-- 涨停记录表
CREATE TABLE IF NOT EXISTS zt_records (
    code            VARCHAR(10) NOT NULL,
    date            DATE NOT NULL,
    name            VARCHAR(20) DEFAULT '',
    pct_chg         DOUBLE PRECISION,
    close           DOUBLE PRECISION,
    amount          DOUBLE PRECISION,
    float_mv        DOUBLE PRECISION,
    total_mv        DOUBLE PRECISION,
    turnover        DOUBLE PRECISION,
    seal_amount     DOUBLE PRECISION DEFAULT 0,
    first_seal_time VARCHAR(10) DEFAULT '',
    last_seal_time  VARCHAR(10) DEFAULT '',
    fail_count      INTEGER DEFAULT 0,
    board_count     INTEGER DEFAULT 1,
    industry        VARCHAR(50) DEFAULT '',
    is_calculated   BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (code, date)
);

CREATE INDEX IF NOT EXISTS idx_zt_records_date ON zt_records(date);
CREATE INDEX IF NOT EXISTS idx_zt_records_board ON zt_records(date, board_count);

-- 涨停分析汇总表(每日一条)
CREATE TABLE IF NOT EXISTS zt_analysis (
    date               DATE PRIMARY KEY,
    total_zt_count     INTEGER DEFAULT 0,
    max_board_height   INTEGER DEFAULT 0,
    first_board_count  INTEGER DEFAULT 0,
    second_board_count INTEGER DEFAULT 0,
    high_board_count   INTEGER DEFAULT 0,
    fail_zt_count      INTEGER DEFAULT 0,
    sentiment_score    DOUBLE PRECISION DEFAULT 0,
    sentiment_phase    VARCHAR(20) DEFAULT '',
    top_sectors        TEXT DEFAULT '[]',
    board_distribution TEXT DEFAULT '{}'
);

-- 竞价数据表
CREATE TABLE IF NOT EXISTS bid_data (
    code       VARCHAR(10) NOT NULL,
    date       DATE NOT NULL,
    bid_price  DOUBLE PRECISION,
    bid_volume DOUBLE PRECISION,
    bid_amount DOUBLE PRECISION,
    bid_pct_chg DOUBLE PRECISION,
    pre_close  DOUBLE PRECISION,
    PRIMARY KEY (code, date)
);

CREATE INDEX IF NOT EXISTS idx_bid_data_date ON bid_data(date);

-- 选股信号表
CREATE TABLE IF NOT EXISTS strategy_signals (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(10) NOT NULL,
    name        VARCHAR(20) DEFAULT '',
    date        DATE NOT NULL,
    signal_type VARCHAR(10) NOT NULL, -- close / bid
    score       DOUBLE PRECISION DEFAULT 0,
    buy_price   DOUBLE PRECISION DEFAULT 0,
    stop_loss   DOUBLE PRECISION DEFAULT 0,
    reason      TEXT DEFAULT '',
    board_count INTEGER DEFAULT 0,
    industry    VARCHAR(50) DEFAULT '',
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_signals_date ON strategy_signals(date);
CREATE INDEX IF NOT EXISTS idx_signals_type ON strategy_signals(signal_type, date);

-- 交易记录表
CREATE TABLE IF NOT EXISTS trade_records (
    id         BIGSERIAL PRIMARY KEY,
    signal_id  BIGINT REFERENCES strategy_signals(id),
    code       VARCHAR(10) NOT NULL,
    name       VARCHAR(20) DEFAULT '',
    buy_date   DATE NOT NULL,
    buy_price  DOUBLE PRECISION,
    sell_date  DATE,
    sell_price DOUBLE PRECISION DEFAULT 0,
    pnl        DOUBLE PRECISION DEFAULT 0,
    pnl_pct    DOUBLE PRECISION DEFAULT 0,
    is_backtest BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trades_date ON trade_records(buy_date);
