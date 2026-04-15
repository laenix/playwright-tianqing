package main

import (
	"encoding/json"
	"fmt"
	"os"

	playwrighttianqing "github.com/laenix/playwright-tianqing"
	"github.com/laenix/playwright-tianqing/config"
)

func main() {
	config.InitConfig()
	PlaywrightTianqingInstance, err := playwrighttianqing.NewPlaywrightTianqing()
	if err != nil {
		fmt.Println("Error initializing PlaywrightTianqing:", err)
		return
	}
	// login
	PlaywrightTianqingInstance.Login()

	// // get alerts
	// alertsResp, err := PlaywrightTianqingInstance.GetAlerts()
	// if err != nil {
	// 	fmt.Println("Error getting alerts:", err)
	// 	return
	// }

	// // 最终需要导出的全部明细数据
	// var exportedData []map[string]interface{}

	// for i, alert := range alertsResp.Alerts {
	// 	fmt.Printf("[%d/%d] Fetching details for alert ID: %s\n", i+1, len(alertsResp.Alerts), alert.AlertID)
	// 	detailObj, err := PlaywrightTianqingInstance.GetAlertDetails(alert.AlertID)
	// 	if err != nil {
	// 		fmt.Printf(" [!] Error getting details for %s: %v\n", alert.AlertID, err)
	// 		continue
	// 	}

	// 	fmt.Printf("   > IP: %s | User: %s | Command length: %d\n",
	// 		detailObj.BasicInfo["ipAddress"],
	// 		detailObj.BasicInfo["user"],
	// 		len(detailObj.ExecuteCommand),
	// 	)

	// 	// 将该条告警简报和详情组合放入大数组
	// 	exportedData = append(exportedData, map[string]interface{}{
	// 		"summary": alert,
	// 		"details": detailObj,
	// 	})
	// }

	// // 将获取到的所有详细内容导出成 JSON，以便分析排查或者二次对接
	// if len(exportedData) > 0 {
	// 	outBytes, marshalErr := json.MarshalIndent(exportedData, "", "  ")
	// 	if marshalErr == nil {
	// 		os.WriteFile("tianqing_alerts_export.json", outBytes, 0644)
	// 		fmt.Printf("\nPerfect! 成功合并导出 %d 条详细告警数据至当前目录 tianqing_alerts_export.json\n", len(exportedData))
	// 	}
	// }

	// =========================== 抓取资产 (Assets) ===========================
	fmt.Println("\n====== 开始抓取终端资产 ======")
	assetsResp, err := PlaywrightTianqingInstance.GetAssets()
	if err != nil {
		fmt.Println("Error getting assets:", err)
		return
	}

	if len(assetsResp.Assets) > 0 {
		outBytes, marshalErr := json.MarshalIndent(assetsResp.Assets, "", "  ")
		if marshalErr == nil {
			os.WriteFile("tianqing_assets_export.json", outBytes, 0644)
			fmt.Printf("Perfect! 成功导出 %d 条资产数据至当前目录 tianqing_assets_export.json\n", len(assetsResp.Assets))
		}
	}
}
