package datasource

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"astock/internal/model"
)

type EastMoney struct {
	client    *http.Client
	userAgent string
	interval  time.Duration
}

func NewEastMoney(userAgent string, intervalMs int) *EastMoney {
	return &EastMoney{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: userAgent,
		interval:  time.Duration(intervalMs) * time.Millisecond,
	}
}

func (e *EastMoney) doGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", e.userAgent)
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (e *EastMoney) Sleep() {
	time.Sleep(e.interval)
}

// FetchStockList 获取沪深A股列表（分页获取全部）
func (e *EastMoney) FetchStockList() ([]model.Stock, error) {
	var allStocks []model.Stock
	pageSize := 100

	for page := 1; ; page++ {
		url := fmt.Sprintf(
			"https://82.push2.eastmoney.com/api/qt/clist/get?pn=%d&pz=%d&po=1&np=1&ut=bd1d9ddb04089700cf9c27f6f7426281&fltt=2&invt=2&fid=f3&fs=m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23&fields=f2,f3,f12,f14,f20,f21,f100,f26",
			page, pageSize)

		body, err := e.doGet(url)
		if err != nil {
			return nil, fmt.Errorf("获取股票列表第%d页失败: %w", page, err)
		}

		var raw struct {
			Data struct {
				Total int                      `json:"total"`
				Diff  []map[string]interface{} `json:"diff"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, fmt.Errorf("解析股票列表第%d页失败: %w", page, err)
		}

		if len(raw.Data.Diff) == 0 {
			break
		}

		for _, d := range raw.Data.Diff {
			code := jsonStr(d, "f12")
			if len(code) != 6 {
				continue
			}

			market := "SZ"
			if strings.HasPrefix(code, "6") {
				market = "SH"
			}

			name := jsonStr(d, "f14")
			isST := strings.Contains(name, "ST")

			var listDate time.Time
			f26 := jsonStr(d, "f26")
			if f26 != "" && f26 != "-" {
				listDate, _ = time.Parse("20060102", f26)
			}

			allStocks = append(allStocks, model.Stock{
				Code:       code,
				Name:       name,
				Market:     market,
				Industry:   jsonStr(d, "f100"),
				ListDate:   listDate,
				IsST:       isST,
				TotalShare: jsonFloat(d, "f20"),
				FloatShare: jsonFloat(d, "f21"),
			})
		}

		if len(allStocks) >= raw.Data.Total || len(raw.Data.Diff) < pageSize {
			break
		}

		e.Sleep()
	}

	return allStocks, nil
}

// FetchDailyKline 获取个股日K线历史数据
func (e *EastMoney) FetchDailyKline(code, market, startDate, endDate string) ([]model.DailyQuote, error) {
	secID := "0." + code
	if market == "SH" {
		secID = "1." + code
	}

	url := fmt.Sprintf(
		"https://push2his.eastmoney.com/api/qt/stock/kline/get?secid=%s&klt=101&fqt=1&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61&beg=%s&end=%s&ut=fa5fd1943c7b386f172d6893dbbd4540",
		secID, startDate, endDate)

	body, err := e.doGet(url)
	if err != nil {
		return nil, fmt.Errorf("获取K线失败 %s: %w", code, err)
	}

	var result struct {
		Data struct {
			Code   string   `json:"code"`
			Klines []string `json:"klines"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析K线失败 %s: %w", code, err)
	}

	if result.Data.Klines == nil {
		return nil, nil
	}

	var quotes []model.DailyQuote
	var prevClose float64

	for _, line := range result.Data.Klines {
		// 格式: 日期,开盘,收盘,最高,最低,成交量,成交额,振幅,涨跌幅,涨跌额,换手率
		parts := strings.Split(line, ",")
		if len(parts) < 11 {
			continue
		}

		date, err := time.Parse("2006-01-02", parts[0])
		if err != nil {
			continue
		}

		open := parseFloat(parts[1])
		close := parseFloat(parts[2])
		high := parseFloat(parts[3])
		low := parseFloat(parts[4])
		volume := parseFloat(parts[5])
		amount := parseFloat(parts[6])
		amplitude := parseFloat(parts[7])
		pctChg := parseFloat(parts[8])
		change := parseFloat(parts[9])
		turnover := parseFloat(parts[10])

		q := model.DailyQuote{
			Code:      code,
			Date:      date,
			Open:      open,
			Close:     close,
			High:      high,
			Low:       low,
			Volume:    volume,
			Amount:    amount,
			PctChg:    pctChg,
			Change:    change,
			Amplitude: amplitude,
			Turnover:  turnover,
			PreClose:  prevClose,
		}
		if prevClose == 0 && close != 0 && pctChg != 0 {
			q.PreClose = close / (1 + pctChg/100)
		}
		prevClose = close

		quotes = append(quotes, q)
	}

	return quotes, nil
}

// FetchZTPool 获取当日涨停板池(仅近期数据可用)
func (e *EastMoney) FetchZTPool(date string) ([]model.ZTRecord, error) {
	url := fmt.Sprintf(
		"https://push2ex.eastmoney.com/getTopicZTPool?ut=7eea3edcaed734bea9cbfc24409ed989&dpt=wz.ztzt&Ession=0&date=%s",
		date)

	body, err := e.doGet(url)
	if err != nil {
		return nil, fmt.Errorf("获取涨停池失败: %w", err)
	}

	var result struct {
		Data struct {
			Pool []struct {
				C  string  `json:"c"`  // 代码
				N  string  `json:"n"`  // 名称
				Zdp float64 `json:"zdp"` // 涨跌幅
				P  float64 `json:"p"`  // 最新价(需/100)
				Amount float64 `json:"amount"` // 成交额
				Ltsz float64 `json:"ltsz"` // 流通市值
				Tshare float64 `json:"tshare"` // 总市值
				Hs float64 `json:"hs"` // 换手率
				Fba float64 `json:"fund"` // 封板资金
				Fbt string  `json:"fbt"` // 首次封板时间
				Lbt string  `json:"lbt"` // 最后封板时间
				Zbc int     `json:"zbc"` // 炸板次数
				Lbc int     `json:"lbc"` // 连板数
				Hybk string `json:"hybk"` // 行业
			} `json:"pool"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析涨停池失败: %w", err)
	}

	d, _ := time.Parse("20060102", date)

	var records []model.ZTRecord
	for _, p := range result.Data.Pool {
		records = append(records, model.ZTRecord{
			Code:          p.C,
			Date:          d,
			Name:          p.N,
			PctChg:        p.Zdp,
			Close:         p.P,
			Amount:        p.Amount,
			FloatMV:       p.Ltsz,
			TotalMV:       p.Tshare,
			Turnover:      p.Hs,
			SealAmount:    p.Fba,
			FirstSealTime: p.Fbt,
			LastSealTime:  p.Lbt,
			FailCount:     p.Zbc,
			BoardCount:    p.Lbc,
			Industry:      p.Hybk,
			IsCalculated:  false,
		})
	}

	return records, nil
}

// FetchRealtimeQuote 获取实时行情(用于竞价数据)
func (e *EastMoney) FetchRealtimeQuote(secID string) (map[string]interface{}, error) {
	url := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/stock/get?secid=%s&ut=fa5fd1943c7b386f172d6893dbbd4540&fields=f43,f44,f45,f46,f47,f48,f50,f51,f52,f55,f57,f58,f60,f116,f117,f162,f168,f170",
		secID)

	body, err := e.doGet(url)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func parseFloat(s string) float64 {
	if s == "" || s == "-" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func jsonStr(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		if val == "-" {
			return ""
		}
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func jsonFloat(m map[string]interface{}, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}
