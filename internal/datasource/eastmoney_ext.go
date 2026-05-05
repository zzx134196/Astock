package datasource

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"astock/internal/model"
)

// ==================== 龙虎榜 ====================

// FetchLHBList 获取龙虎榜列表(某日所有上榜个股)
func (e *EastMoney) FetchLHBList(date string) ([]model.LHBRecord, error) {
	params := url.Values{}
	params.Set("reportName", "RPT_DAILYBILLBOARD_DETAILSNEW")
	params.Set("columns", "SECURITY_CODE,SECUCODE,SECURITY_NAME_ABBR,TRADE_DATE,CHANGE_RATE,CLOSE_PRICE,BILLBOARD_NET_AMT,BILLBOARD_BUY_AMT,BILLBOARD_SELL_AMT,TURNOVERRATE,EXPLANATION")
	params.Set("filter", fmt.Sprintf("(TRADE_DATE='%s')", formatDateDash(date)))
	params.Set("pageNumber", "1")
	params.Set("pageSize", "500")
	params.Set("sortColumns", "BILLBOARD_NET_AMT")
	params.Set("sortTypes", "-1")
	params.Set("source", "WEB")
	params.Set("client", "WEB")

	apiURL := "https://datacenter-web.eastmoney.com/api/data/v1/get?" + params.Encode()

	body, err := e.doGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取龙虎榜失败: %w", err)
	}

	var result struct {
		Result struct {
			Data []map[string]interface{} `json:"data"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析龙虎榜失败: %w", err)
	}

	d, _ := time.Parse("20060102", date)

	var records []model.LHBRecord
	for _, item := range result.Result.Data {
		records = append(records, model.LHBRecord{
			Code:       jsonStr(item, "SECURITY_CODE"),
			Date:       d,
			Name:       jsonStr(item, "SECURITY_NAME_ABBR"),
			PctChg:     jsonFloat(item, "CHANGE_RATE"),
			Close:      jsonFloat(item, "CLOSE_PRICE"),
			NetAmount:  jsonFloat(item, "BILLBOARD_NET_AMT"),
			BuyAmount:  jsonFloat(item, "BILLBOARD_BUY_AMT"),
			SellAmount: jsonFloat(item, "BILLBOARD_SELL_AMT"),
			Turnover:   jsonFloat(item, "TURNOVERRATE"),
			Reason:     jsonStr(item, "EXPLANATION"),
		})
	}

	return records, nil
}

// FetchLHBDetail 获取龙虎榜某只股票的买卖席位明细
func (e *EastMoney) FetchLHBDetail(code, date string) ([]model.LHBDetail, error) {
	var allDetails []model.LHBDetail

	for _, side := range []string{"buy", "sell"} {
		reportName := "RPT_BILLBOARD_DAILYDETAILSBUY"
		if side == "sell" {
			reportName = "RPT_BILLBOARD_DAILYDETAILSSELL"
		}

		params := url.Values{}
		params.Set("reportName", reportName)
		params.Set("columns", "SECURITY_CODE,TRADE_DATE,OPERATEDEPT_NAME,BUY,SELL,NET")
		params.Set("filter", fmt.Sprintf("(SECURITY_CODE=\"%s\")(TRADE_DATE='%s')", code, formatDateDash(date)))
		params.Set("pageNumber", "1")
		params.Set("pageSize", "50")
		params.Set("source", "WEB")
		params.Set("client", "WEB")

		apiURL := "https://datacenter-web.eastmoney.com/api/data/v1/get?" + params.Encode()

		body, err := e.doGet(apiURL)
		if err != nil {
			continue
		}

		var result struct {
			Result struct {
				Data []map[string]interface{} `json:"data"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		d, _ := time.Parse("20060102", date)
		for i, item := range result.Result.Data {
			allDetails = append(allDetails, model.LHBDetail{
				Code:       code,
				Date:       d,
				DeptName:   jsonStr(item, "OPERATEDEPT_NAME"),
				Side:       side,
				BuyAmount:  jsonFloat(item, "BUY"),
				SellAmount: jsonFloat(item, "SELL"),
				NetAmount:  jsonFloat(item, "NET"),
				Rank:       i + 1,
			})
		}
	}

	return allDetails, nil
}

// ==================== 板块概念 ====================

// FetchSectorList 获取行业板块或概念板块列表（分页）
func (e *EastMoney) FetchSectorList(sectorType string) ([]model.Sector, error) {
	fs := "m:90+t:2"
	if sectorType == "concept" {
		fs = "m:90+t:3"
	}

	var allSectors []model.Sector
	for page := 1; ; page++ {
		apiURL := fmt.Sprintf(
			"https://push2.eastmoney.com/api/qt/clist/get?pn=%d&pz=100&po=1&np=1&ut=bd1d9ddb04089700cf9c27f6f7426281&fltt=2&invt=2&fid=f3&fs=%s&fields=f12,f14",
			page, fs)

		body, err := e.doGet(apiURL)
		if err != nil {
			break
		}

		var result struct {
			Data struct {
				Total int                      `json:"total"`
				Diff  []map[string]interface{} `json:"diff"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			break
		}
		if len(result.Data.Diff) == 0 {
			break
		}

		for _, d := range result.Data.Diff {
			allSectors = append(allSectors, model.Sector{
				Code:       jsonStr(d, "f12"),
				Name:       jsonStr(d, "f14"),
				SectorType: sectorType,
			})
		}

		if len(allSectors) >= result.Data.Total || len(result.Data.Diff) < 100 {
			break
		}
		e.Sleep()
	}

	return allSectors, nil
}

// FetchSectorFlow 获取板块资金流向（分页）
func (e *EastMoney) FetchSectorFlow(sectorType string) ([]model.SectorFlow, error) {
	fs := "m:90+t:2"
	if sectorType == "concept" {
		fs = "m:90+t:3"
	}

	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	var allFlows []model.SectorFlow
	for page := 1; ; page++ {
		apiURL := fmt.Sprintf(
			"https://push2.eastmoney.com/api/qt/clist/get?pn=%d&pz=100&po=1&np=1&ut=bd1d9ddb04089700cf9c27f6f7426281&fltt=2&invt=2&fid=f62&fs=%s&fields=f12,f14,f2,f3,f62,f184,f66,f69,f72,f75,f78,f81,f84,f87",
			page, fs)

		body, err := e.doGet(apiURL)
		if err != nil {
			break
		}

		var result struct {
			Data struct {
				Total int                      `json:"total"`
				Diff  []map[string]interface{} `json:"diff"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			break
		}
		if len(result.Data.Diff) == 0 {
			break
		}

		for _, d := range result.Data.Diff {
			allFlows = append(allFlows, model.SectorFlow{
				SectorCode: jsonStr(d, "f12"),
				Date:       todayDate,
				PctChg:     jsonFloat(d, "f3"),
				MainNet:    jsonFloat(d, "f62"),
				HugeNet:    jsonFloat(d, "f66"),
				BigNet:     jsonFloat(d, "f72"),
				MidNet:     jsonFloat(d, "f78"),
				SmallNet:   jsonFloat(d, "f84"),
			})
		}

		if len(allFlows) >= result.Data.Total || len(result.Data.Diff) < 100 {
			break
		}
		e.Sleep()
	}

	return allFlows, nil
}

// ==================== 个股资金流向 ====================

// FetchStockFlow 获取个股资金流向(当日)
func (e *EastMoney) FetchStockFlow() ([]model.StockFlow, error) {
	baseURL := "https://push2.eastmoney.com/api/qt/clist/get?po=1&np=1&ut=bd1d9ddb04089700cf9c27f6f7426281&fltt=2&invt=2&fid=f62&fs=m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23&fields=f12,f14,f62,f184,f66,f69,f72,f75,f78,f81,f84,f87"

	var allFlows []model.StockFlow
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	for page := 1; ; page++ {
		pageURL := fmt.Sprintf("%s&pn=%d&pz=100", baseURL, page)
		body, err := e.doGet(pageURL)
		if err != nil {
			break
		}

		var result struct {
			Data struct {
				Total int                      `json:"total"`
				Diff  []map[string]interface{} `json:"diff"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			break
		}

		if len(result.Data.Diff) == 0 {
			break
		}

		for _, d := range result.Data.Diff {
			allFlows = append(allFlows, model.StockFlow{
				Code:     jsonStr(d, "f12"),
				Date:     todayDate,
				MainNet:  jsonFloat(d, "f62"),
				HugeNet:  jsonFloat(d, "f66"),
				BigNet:   jsonFloat(d, "f72"),
				MidNet:   jsonFloat(d, "f78"),
				SmallNet: jsonFloat(d, "f84"),
			})
		}

		if len(allFlows) >= result.Data.Total || len(result.Data.Diff) < 100 {
			break
		}
		e.Sleep()
	}

	return allFlows, nil
}

// FetchStockFlowHistory 获取个股历史资金流向（约120个交易日）
func (e *EastMoney) FetchStockFlowHistory(code, market string) ([]model.StockFlow, error) {
	secID := "0." + code
	if market == "SH" {
		secID = "1." + code
	}

	apiURL := fmt.Sprintf(
		"https://push2his.eastmoney.com/api/qt/stock/fflow/daykline/get?secid=%s&lmt=500&klt=101&fields1=f1,f2,f3,f7&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f62,f63,f64,f65&ut=fa5fd1943c7b386f172d6893dbbd4540",
		secID)

	body, err := e.doGet(apiURL)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var flows []model.StockFlow
	for _, line := range result.Data.Klines {
		// 格式: 日期,主力净流入,超大单净流入,散户净流入,超大单净流入,大单净流入,...
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}

		date, err := time.Parse("2006-01-02", parts[0])
		if err != nil {
			continue
		}

		flows = append(flows, model.StockFlow{
			Code:    code,
			Date:    date,
			MainNet: parseFloat(parts[1]),
			HugeNet: parseFloat(parts[4]),
			BigNet:  parseFloat(parts[5]),
			MidNet:  parseFloat(parts[2]),  // 中单(散户)
			SmallNet: parseFloat(parts[3]),
		})
	}

	return flows, nil
}

// FetchSectorFlowHistory 获取板块历史资金流向
func (e *EastMoney) FetchSectorFlowHistory(sectorCode string) ([]model.SectorFlow, error) {
	apiURL := fmt.Sprintf(
		"https://push2his.eastmoney.com/api/qt/stock/fflow/daykline/get?secid=90.%s&lmt=500&klt=101&fields1=f1,f2,f3,f7&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f62,f63,f64,f65&ut=fa5fd1943c7b386f172d6893dbbd4540",
		sectorCode)

	body, err := e.doGet(apiURL)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var flows []model.SectorFlow
	for _, line := range result.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}

		date, err := time.Parse("2006-01-02", parts[0])
		if err != nil {
			continue
		}

		flows = append(flows, model.SectorFlow{
			SectorCode: sectorCode,
			Date:       date,
			MainNet:    parseFloat(parts[1]),
			HugeNet:    parseFloat(parts[4]),
			BigNet:     parseFloat(parts[5]),
			MidNet:     parseFloat(parts[2]),
			SmallNet:   parseFloat(parts[3]),
		})
	}

	return flows, nil
}

// ==================== 异动数据 ====================

// FetchStockChanges 获取盘口异动数据（分页获取全部）
func (e *EastMoney) FetchStockChanges() ([]model.StockChange, error) {
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	changeTypeMap := map[int]string{
		4: "大笔买入", 8: "大笔卖出", 16: "封涨停板", 32: "打开涨停板",
		64: "封跌停板", 128: "打开跌停板",
		8193: "竞价上涨", 8194: "竞价下跌",
		8201: "火箭发射", 8202: "快速反弹", 8203: "高台跳水", 8204: "加速下跌",
		8207: "有大买盘", 8208: "有大卖盘", 8209: "竞价涨停", 8210: "竞价跌停",
		8211: "高开5%", 8212: "低开5%", 8213: "向上缺口", 8214: "向下缺口",
		8215: "60日新高", 8216: "60日新低",
	}

	var allChanges []model.StockChange
	for pageIdx := 0; pageIdx < 50; pageIdx++ {
		apiURL := fmt.Sprintf(
			"https://push2ex.eastmoney.com/getAllStockChanges?ut=7eea3edcaed734bea9cbfc24409ed989&dpt=wzchanges&type=8201,8202,8193,4,32,64,8207,8209,8211,8213,8215,8204,8203,8194,8,16,128,8208,8210,8212,8214,8216&pageindex=%d&pagesize=200",
			pageIdx)

		body, err := e.doGet(apiURL)
		if err != nil {
			break
		}

		var raw struct {
			Data struct {
				Allstock []map[string]interface{} `json:"allstock"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &raw); err != nil {
			break
		}

		if raw.Data.Allstock == nil || len(raw.Data.Allstock) == 0 {
			break
		}

		for _, s := range raw.Data.Allstock {
			dtype := int(jsonFloat(s, "d"))
			ct := ""
			if name, ok := changeTypeMap[dtype]; ok {
				ct = name
			} else {
				ct = fmt.Sprintf("类型%d", dtype)
			}

			allChanges = append(allChanges, model.StockChange{
				Code:       jsonStr(s, "c"),
				Name:       jsonStr(s, "n"),
				Date:       todayDate,
				ChangeTime: jsonStr(s, "t"),
				ChangeType: ct,
				Info:       jsonStr(s, "i"),
			})
		}

		if len(raw.Data.Allstock) < 200 {
			break
		}
		e.Sleep()
	}

	changes := allChanges

	return changes, nil
}

// ==================== 涨停池扩展 ====================

// FetchZTPoolStrong 获取强势股池
func (e *EastMoney) FetchZTPoolStrong(date string) ([]model.ZTPoolExt, error) {
	return e.fetchPoolExt(date, "strong", "getTopicQSPool")
}

// FetchZTPoolFail 获取炸板股池
func (e *EastMoney) FetchZTPoolFail(date string) ([]model.ZTPoolExt, error) {
	return e.fetchPoolExt(date, "fail", "getTopicZBPool")
}

// FetchZTPoolDT 获取跌停股池
func (e *EastMoney) FetchZTPoolDT(date string) ([]model.ZTPoolExt, error) {
	return e.fetchPoolExt(date, "dt", "getTopicDTPool")
}

// FetchZTPoolSubNew 获取次新股池
func (e *EastMoney) FetchZTPoolSubNew(date string) ([]model.ZTPoolExt, error) {
	return e.fetchPoolExt(date, "sub_new", "getTopicCXPool")
}

func (e *EastMoney) fetchPoolExt(date, poolType, endpoint string) ([]model.ZTPoolExt, error) {
	apiURL := fmt.Sprintf(
		"https://push2ex.eastmoney.com/%s?ut=7eea3edcaed734bea9cbfc24409ed989&dpt=wz.ztzt&date=%s",
		endpoint, date)

	body, err := e.doGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取%s池失败: %w", poolType, err)
	}

	var result struct {
		Data struct {
			Pool []map[string]interface{} `json:"pool"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析%s池失败: %w", poolType, err)
	}

	d, _ := time.Parse("20060102", date)

	var records []model.ZTPoolExt
	for _, p := range result.Data.Pool {
		info, _ := json.Marshal(p)
		records = append(records, model.ZTPoolExt{
			Code:      jsonStr(p, "c"),
			Date:      d,
			Name:      jsonStr(p, "n"),
			PoolType:  poolType,
			PctChg:    jsonFloat(p, "zdp"),
			Close:     jsonFloat(p, "p"),
			Amount:    jsonFloat(p, "amount"),
			Turnover:  jsonFloat(p, "hs"),
			ExtraInfo: string(info),
		})
	}

	return records, nil
}

// ==================== 辅助 ====================

func formatDateDash(yyyymmdd string) string {
	if len(yyyymmdd) == 8 {
		return yyyymmdd[:4] + "-" + yyyymmdd[4:6] + "-" + yyyymmdd[6:]
	}
	if strings.Contains(yyyymmdd, "-") {
		return yyyymmdd
	}
	return yyyymmdd
}
