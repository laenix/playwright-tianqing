package playwrighttianqing

import (
	"context"
	"fmt"

	playwrightbase "github.com/laenix/playwright-base"
)

type PlaywrightTianqing struct {
	browser *playwrightbase.Browser
	ctx     context.Context
}

func NewPlaywrightTianqing() (*PlaywrightTianqing, error) {
	browser := playwrightbase.Browser{}
	ctx := context.Background()
	err := browser.OpenBrowser(ctx, map[string]interface{}{
		"headless": false,
	})
	if err != nil {
		fmt.Println("Error opening browser:", err)
		return nil, err
	}
	return &PlaywrightTianqing{
		browser: &browser,
		ctx:     ctx,
	}, nil
}
