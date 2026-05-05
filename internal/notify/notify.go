package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Notifier struct {
	enabled    bool
	webhookURL string
	client     *http.Client
}

func New(enabled bool, webhookURL string) *Notifier {
	return &Notifier{
		enabled:    enabled,
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type SignalMessage struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Score      float64 `json:"score"`
	BuyPrice   float64 `json:"buy_price"`
	StopLoss   float64 `json:"stop_loss"`
	Reason     string  `json:"reason"`
	BoardCount int     `json:"board_count"`
}

// SendSignals 推送选股信号
func (n *Notifier) SendSignals(signalType string, signals []SignalMessage) error {
	if !n.enabled || n.webhookURL == "" {
		log.Println("[通知] 未启用通知推送")
		return nil
	}

	title := "收盘选股信号"
	if signalType == "bid" {
		title = "竞价选股信号"
	}

	content := fmt.Sprintf("### %s (%s)\n\n", title, time.Now().Format("2006-01-02"))

	for i, s := range signals {
		content += fmt.Sprintf("%d. **%s(%s)** | 评分:%.1f | %d板 | 买入:%.2f 止损:%.2f\n   > %s\n\n",
			i+1, s.Name, s.Code, s.Score, s.BoardCount, s.BuyPrice, s.StopLoss, s.Reason)
	}

	if len(signals) == 0 {
		content += "今日无选股信号\n"
	}

	return n.sendWebhook(title, content)
}

func (n *Notifier) sendWebhook(title, content string) error {
	// 支持企业微信/钉钉/飞书等Webhook格式
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title":   title,
			"text":    content,
			"content": content,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送通知失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("通知返回状态码: %d", resp.StatusCode)
	}

	log.Printf("[通知] 推送成功: %s", title)
	return nil
}
