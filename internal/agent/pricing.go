package agent

import "strings"

type ModelPrices struct {
	InputPerMTok      float64
	OutputPerMTok     float64
	CacheWritePerMTok float64
	CacheReadPerMTok  float64
}

type modelPriceEntry struct {
	match  string
	prices ModelPrices
}

var modelPriceTable = []modelPriceEntry{
	{"opus", ModelPrices{InputPerMTok: 15, OutputPerMTok: 75, CacheWritePerMTok: 18.75, CacheReadPerMTok: 1.50}},
	{"sonnet", ModelPrices{InputPerMTok: 3, OutputPerMTok: 15, CacheWritePerMTok: 3.75, CacheReadPerMTok: 0.30}},
	{"haiku", ModelPrices{InputPerMTok: 0.80, OutputPerMTok: 4, CacheWritePerMTok: 1.00, CacheReadPerMTok: 0.08}},
}

var fallbackModelPrices = ModelPrices{InputPerMTok: 3, OutputPerMTok: 15, CacheWritePerMTok: 3.75, CacheReadPerMTok: 0.30}

func PricesForModel(model string) ModelPrices {
	lowered := strings.ToLower(model)
	for _, entry := range modelPriceTable {
		if strings.Contains(lowered, entry.match) {
			return entry.prices
		}
	}
	return fallbackModelPrices
}

func (p ModelPrices) Estimate(inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens int64) float64 {
	return float64(inputTokens)/1e6*p.InputPerMTok +
		float64(outputTokens)/1e6*p.OutputPerMTok +
		float64(cacheWriteTokens)/1e6*p.CacheWritePerMTok +
		float64(cacheReadTokens)/1e6*p.CacheReadPerMTok
}
