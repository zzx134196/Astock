-- 个股所属板块/概念标签
CREATE TABLE IF NOT EXISTS stock_concepts (
    code        VARCHAR(10) NOT NULL,
    board_code  VARCHAR(20) NOT NULL,
    board_name  VARCHAR(50) NOT NULL,
    board_rank  INTEGER DEFAULT 0,
    PRIMARY KEY (code, board_code)
);

CREATE INDEX IF NOT EXISTS idx_stock_concepts_board ON stock_concepts(board_code);
CREATE INDEX IF NOT EXISTS idx_stock_concepts_name ON stock_concepts(board_name);

-- 人气排行榜
CREATE TABLE IF NOT EXISTS hot_rank (
    code        VARCHAR(10) NOT NULL,
    date        DATE NOT NULL,
    rank        INTEGER NOT NULL,
    rank_change INTEGER DEFAULT 0,   -- 排名变动(正=上升)
    PRIMARY KEY (code, date)
);

CREATE INDEX IF NOT EXISTS idx_hot_rank_date ON hot_rank(date);

-- 涨停次日溢价表
CREATE TABLE IF NOT EXISTS zt_premium (
    code             VARCHAR(10) NOT NULL,
    zt_date          DATE NOT NULL,
    board_count      INTEGER DEFAULT 1,
    next_open        DOUBLE PRECISION DEFAULT 0,
    next_close       DOUBLE PRECISION DEFAULT 0,
    next_high        DOUBLE PRECISION DEFAULT 0,
    next_low         DOUBLE PRECISION DEFAULT 0,
    next_pct_chg     DOUBLE PRECISION DEFAULT 0,
    open_premium     DOUBLE PRECISION DEFAULT 0,  -- 开盘溢价率(%)
    close_premium    DOUBLE PRECISION DEFAULT 0,  -- 收盘溢价率(%)
    max_premium      DOUBLE PRECISION DEFAULT 0,  -- 最高溢价率(%)
    is_next_zt       BOOLEAN DEFAULT FALSE,        -- 次日是否继续涨停
    PRIMARY KEY (code, zt_date)
);

CREATE INDEX IF NOT EXISTS idx_zt_premium_date ON zt_premium(zt_date);
CREATE INDEX IF NOT EXISTS idx_zt_premium_board ON zt_premium(board_count);

-- 每日市场情绪明细(天梯图/板块集中度/情绪MA)
CREATE TABLE IF NOT EXISTS daily_sentiment (
    date               DATE PRIMARY KEY,
    zt_count           INTEGER DEFAULT 0,
    dt_count           INTEGER DEFAULT 0,     -- 跌停家数
    fail_count         INTEGER DEFAULT 0,     -- 炸板家数
    up_count           INTEGER DEFAULT 0,     -- 上涨家数
    down_count         INTEGER DEFAULT 0,     -- 下跌家数
    max_board          INTEGER DEFAULT 0,
    -- 连板天梯
    board_1            INTEGER DEFAULT 0,
    board_2            INTEGER DEFAULT 0,
    board_3            INTEGER DEFAULT 0,
    board_4            INTEGER DEFAULT 0,
    board_5plus        INTEGER DEFAULT 0,
    -- 晋级率
    promo_1to2         DOUBLE PRECISION DEFAULT 0,  -- 首板->二板晋级率(%)
    promo_2to3         DOUBLE PRECISION DEFAULT 0,
    -- 情绪均线
    zt_ma5             DOUBLE PRECISION DEFAULT 0,
    zt_ma10            DOUBLE PRECISION DEFAULT 0,
    -- 板块集中度 top3
    top_sector_1       VARCHAR(50) DEFAULT '',
    top_sector_1_count INTEGER DEFAULT 0,
    top_sector_2       VARCHAR(50) DEFAULT '',
    top_sector_2_count INTEGER DEFAULT 0,
    top_sector_3       VARCHAR(50) DEFAULT '',
    top_sector_3_count INTEGER DEFAULT 0,
    -- 赚钱效应
    avg_zt_premium     DOUBLE PRECISION DEFAULT 0   -- 当日涨停票次日平均溢价
);
