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
		params.Set("columns", "SECURITY_CODE,TRADE_DATE,OPERATEDEPT_NAME,BUY_AMT,SELL_AMT,NET_AMT,RANK")
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
		for _, item := range result.Result.Data {
			allDetails = append(allDetails, model.LHBDetail{
				Code:       code,
				Date:       d,
				DeptName:   jsonStr(item, "OPERATEDEPT_NAME"),
				Side:       side,
				BuyAmount:  jsonFloat(item, "BUY_AMT"),
				SellAmount: jsonFloat(item, "SELL_AMT"),
				NetAmount:  jsonFloat(item, "NET_AMT"),
				Rank:       int(jsonFloat(item, "RANK")),
			})
		}
	}

	return allDetails, nil
}

// ==================== 板块概念 ====================

// FetchSectorList 获取行业板块或概念板块列表
func (e *EastMoney) FetchSectorList(sectorType string) ([]model.Sector, error) {
	fs := "m:90+t:2" // 行业
	if sectorType == "concept" {
		fs = "m:90+t:3"
	}

	apiURL := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/clist/get?pn=1&pz=500&po=1&np=1&ut=bd1d9ddb04089700cf9c27f6f7426281&fltt=2&invt=2&fid=f3&fs=%s&fields=f12,f14",
		fs)

	body, err := e.doGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取板块列表失败: %w", err)
	}

	var result struct {
		Data struct {
			Diff []map[string]interface{} `json:"diff"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析板块列表失败: %w", err)
	}

	var sectors []model.Sector
	for _, d := range result.Data.Diff {
		sectors = append(sectors, model.Sector{
			Code:       jsonStr(d, "f12"),
			Name:       jsonStr(d, "f14"),
			SectorType: sectorType,
		})
	}

	return sectors, nil
}

// FetchSectorFlow 获取板块资金流向
func (e *EastMoney) FetchSectorFlow(sectorType string) ([]model.SectorFlow, error) {
	fs := "m:90+t:2"
	if sectorType == "concept" {
		fs = "m:90+t:3"
	}

	apiURL := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/clist/get?pn=1&pz=500&po=1&np=1&ut=bd1d9ddb04089700cf9c27f6f7426281&fltt=2&invt=2&fid=f62&fs=%s&fields=f12,f14,f2,f3,f62,f184,f66,f69,f72,f75,f78,f81,f84,f87",
		fs)

	body, err := e.doGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取板块资金流向失败: %w", err)
	}

	var result struct {
		Data struct {
			Diff []map[string]interface{} `json:"diff"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析板块资金流向失败: %w", err)
	}

	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	var flows []model.SectorFlow
	for _, d := range result.Data.Diff {
		flows = append(flows, model.SectorFlow{
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

	return flows, nil
}

// ==================== 个股资金流向 ====================

// FetchStockFlow 获取个股资金流向(当日)
func (e *EastMoney) FetchStockFlow() ([]model.StockFlow, error) {
	apiURL := "https://push2.eastmoney.com/api/qt/clist/get?pn=1&pz=5000&po=1&np=1&ut=bd1d9ddb04089700cf9c27f6f7426281&fltt=2&invt=2&fid=f62&fs=m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23&fields=f12,f14,f62,f184,f66,f69,f72,f75,f78,f81,f84,f87"

	var allFlows []model.StockFlow
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	for page := 1; ; page++ {
		pageURL := fmt.Sprintf("%s&pn=%d&pz=100", apiURL, page)
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

// ==================== 异动数据 ====================

// FetchStockChanges 获取盘口异动数据
func (e *EastMoney) FetchStockChanges() ([]model.StockChange, error) {
	apiURL := "https://push2ex.eastmoney.com/getAllStockChanges?ut=7eea3edcaed734bea9cbfc24409ed989&dpt=wzchanges&type=8201,8202,8193,4,32,64,8207,8209,8211,8213,8215,8204,8203,8194,8,16,128,8208,8210,8212,8214,8216&pageindex=0&pagesize=200"

	body, err := e.doGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取异动数据失败: %w", err)
	}

	var result struct {
		Data struct {
			Allstock []struct {
				C string `json:"c"` // 代码
				N string `json:"n"` // 名称
				T string `json:"t"` // 时间
				D int    `json:"d"` // 类型
				I string `json:"i"` // 信息
			} `json:"allstock"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析异动数据失败: %w", err)
	}

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

	var changes []model.StockChange
	for _, s := range result.Data.Allstock {
		ct := ""
		if name, ok := changeTypeMap[s.D]; ok {
			ct = name
		} else {
			ct = fmt.Sprintf("类型%d", s.D)
		}

		changes = append(changes, model.StockChange{
			Code:       s.C,
			Name:       s.N,
			Date:       todayDate,
			ChangeTime: s.T,
			ChangeType: ct,
			Info:       s.I,
		})
	}

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
