package main

import (
	"time"

	"github.com/plusev-terminal/go-plugin-common/plugin"
)

// handleGetTimeframes returns supported timeframes
func (p *OrderlyPlugin) handleGetTimeframes(params map[string]any) plugin.Response {
	timeframes := p.client.GetTimeframes()
	return plugin.SuccessResponse(timeframes, time.Hour*12)
}
