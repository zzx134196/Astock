package datasource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"astock/internal/model"
)

// FetchStockConcepts 获取个股所属板块/概念标签
func (e *EastMoney) FetchStockConcepts(code, market string) ([]model.StockConcept, error) {
	prefix := "SZ"
	if market == "SH" {
		prefix = "SH"
	}

	apiURL := fmt.Sprintf(
		"http://emweb.securities.eastmoney.com/PC_HSF10/CoreConception/PageAjax?code=%s%s",
		prefix, code)

	body, err := e.doGet(apiURL)
	if err != nil {
		return nil, err
	}

	var result struct {
		SSBK []struct {
			BoardCode string `json:"BOARD_CODE"`
			BoardName string `json:"BOARD_NAME"`
			BoardRank int    `json:"BOARD_RANK"`
		} `json:"ssbk"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var concepts []model.StockConcept
	for _, b := range result.SSBK {
		concepts = append(concepts, model.StockConcept{
			Code:      code,
			BoardCode: b.BoardCode,
			BoardName: b.BoardName,
			BoardRank: b.BoardRank,
		})
	}

	return concepts, nil
}

// FetchHotRank 获取人气排行榜TOP100
func (e *EastMoney) FetchHotRank() ([]model.HotRank, error) {
	apiURL := "https://emappdata.eastmoney.com/stockrank/getAllCurrentList"

	payload := `{"appId":"appId01","pageNo":1,"pageSize":100}`

	req, err := newPostRequest(apiURL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", e.userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取人气排行失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			SC    string `json:"sc"`    // SH600000 / SZ000001
			RK    int    `json:"rk"`    // 排名
			RC    int    `json:"rc"`    // 排名变动
			HisRC int    `json:"hisRc"` // 历史排名变动
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析人气排行失败: %w", err)
	}

	var ranks []model.HotRank
	for _, d := range result.Data {
		code := d.SC
		if len(code) > 2 {
			code = code[2:] // SH600000 -> 600000
		}
		ranks = append(ranks, model.HotRank{
			Code:       code,
			Rank:       d.RK,
			RankChange: d.HisRC,
		})
	}

	return ranks, nil
}

func newPostRequest(url, body string) (*http.Request, error) {
	return http.NewRequest("POST", url, strings.NewReader(body))
}
