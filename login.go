package playwrighttianqing

import (
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/viper"
)

func (pt *PlaywrightTianqing) Login() {

	url := viper.GetString("tianqing.url")
	fmt.Println("url", url)
	pt.browser.Goto(pt.ctx, url)

	// wait username field
	usernameSelector := viper.GetString("tianqing.usernameSelector")
	fmt.Println("WaitForSelector Username")
	pt.browser.WaitForSelector(pt.ctx, usernameSelector)
	// fill username
	username := viper.GetString("tianqing.username")
	fmt.Println("Fill username")
	pt.browser.Fill(pt.ctx, usernameSelector, username)
	// check
	Url, err := pt.browser.GetUrl(pt.ctx)
	if err != nil {
		fmt.Println("Error getting URL:", err)
		return
	}
	if Url != url {
		fmt.Println("URL mismatch. Expected:", url, "Got:", Url)
		return
	}
	// select auth type
	authTypeSelector := viper.GetString("tianqing.authTypeSelector")
	fmt.Println("WaitForSelector AuthType")
	pt.browser.WaitForSelector(pt.ctx, authTypeSelector)
	// click to open dropdown
	pt.browser.Click(pt.ctx, authTypeSelector)
	// select option
	authType := viper.GetString("tianqing.authType")
	fmt.Println("Select auth type", authType)
	pt.browser.Click(pt.ctx, authType)
	// click next
	nextSelector := viper.GetString("tianqing.nextSelector")
	fmt.Println("WaitForSelector Next")
	pt.browser.WaitForSelector(pt.ctx, nextSelector)
	fmt.Println("Click Next")
	pt.browser.Click(pt.ctx, nextSelector)
	// wait password field
	passwordSelector := viper.GetString("tianqing.passwordSelector")
	fmt.Println("WaitForSelector Password")
	pt.browser.WaitForSelector(pt.ctx, passwordSelector)
	// fill password
	password := viper.GetString("tianqing.password")
	fmt.Println("Fill password")
	pt.browser.Fill(pt.ctx, passwordSelector, password)
	// click login
	page, err := pt.browser.GetActivePage(pt.ctx)
	if err != nil {
		fmt.Println("Error getting active page:", err)
		return
	}
	err = page.GetByRole("button", playwright.PageGetByRoleOptions{
		Name:  "Login",
		Exact: playwright.Bool(true),
	}).Click()
	if err != nil {
		fmt.Println("Error clicking login button:", err)
		return
	}

	// 优化：把硬等待 (Sleep) 替换为等待页面的网络请求变得空闲，说明登录请求和后续加载完成
	fmt.Println("Waiting for main page to load after login...")
	pt.browser.Sleep(pt.ctx, 1*time.Second)
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	fmt.Println("Login script executed successfully.")
}
