// server/internal/agent/tools/weather.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"ai-curton/server/internal/agent"

	"github.com/go-resty/resty/v2"
)

const qweatherBaseURL = "https://devapi.qweather.com/v7"

// qweatherNowResponse 和风天气实时天气响应
type qweatherNowResponse struct {
	Code string `json:"code"` // "200" 表示成功
	Now  struct {
		Temp      string `json:"temp"`      // 温度 ℃
		FeelsLike string `json:"feelsLike"` // 体感温度
		Text      string `json:"text"`      // 天气状况文字，如"晴"
		WindDir   string `json:"windDir"`   // 风向
		WindScale string `json:"windScale"` // 风力等级
		Humidity  string `json:"humidity"`  // 相对湿度 %
		Vis       string `json:"vis"`       // 能见度 km
	} `json:"now"`
}

// qweatherCityResponse 和风天气城市查询响应
type qweatherCityResponse struct {
	Code     string `json:"code"`
	Location []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Adm1 string `json:"adm1"` // 省份
		Adm2 string `json:"adm2"` // 城市
	} `json:"location"`
}

// NewWeatherTool 创建天气查询工具
func NewWeatherTool(apiKey string) *agent.Tool {
	client := resty.New()

	return &agent.Tool{
		Name:        "get_weather",
		Description: "查询指定城市的当前实时天气信息，包括温度、体感温度、天气状况、风向、风力、湿度等",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{
					"type":        "string",
					"description": "城市名称，如：北京、上海、广州",
				},
			},
			"required": []string{"city"},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			city, _ := args["city"].(string)
			if city == "" {
				return "", fmt.Errorf("city is required")
			}

			// 1. 先查询城市 ID
			cityResp, err := client.R().
				SetContext(ctx).
				SetQueryParams(map[string]string{
					"location": city,
					"key":      apiKey,
					"number":   "1",
				}).
				Get("https://geoapi.qweather.com/v2/city/lookup")
			if err != nil {
				return "", fmt.Errorf("city lookup failed: %w", err)
			}

			var cityResult qweatherCityResponse
			if err := json.Unmarshal(cityResp.Body(), &cityResult); err != nil {
				return "", fmt.Errorf("city lookup parse failed: %w", err)
			}
			if cityResult.Code != "200" || len(cityResult.Location) == 0 {
				return fmt.Sprintf("未找到城市：%s", city), nil
			}

			locationID := cityResult.Location[0].ID
			cityName := cityResult.Location[0].Name
			province := cityResult.Location[0].Adm1

			// 2. 查询实时天气
			weatherResp, err := client.R().
				SetContext(ctx).
				SetQueryParams(map[string]string{
					"location": locationID,
					"key":      apiKey,
				}).
				Get(qweatherBaseURL + "/weather/now")
			if err != nil {
				return "", fmt.Errorf("weather query failed: %w", err)
			}

			var weatherResult qweatherNowResponse
			if err := json.Unmarshal(weatherResp.Body(), &weatherResult); err != nil {
				return "", fmt.Errorf("weather parse failed: %w", err)
			}
			if weatherResult.Code != "200" {
				return fmt.Sprintf("天气查询失败，错误码：%s", weatherResult.Code), nil
			}

			now := weatherResult.Now
			return fmt.Sprintf(
				"%s（%s）当前天气：%s，温度 %s℃，体感温度 %s℃，%s %s级，湿度 %s%%，能见度 %skm",
				cityName, province,
				now.Text, now.Temp, now.FeelsLike,
				now.WindDir, now.WindScale,
				now.Humidity, now.Vis,
			), nil
		},
	}
}
