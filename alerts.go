package playwrighttianqing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/viper"
)

// Alert represents the alert data structure
type Alert struct {
	AlertID    string `json:"alertId"`    // 告警ID
	AlertTime  string `json:"alertTime"`  // 告警时间 (改用string接收避免格式解析异常)
	AlertType  string `json:"alertType"`  // 告警类型
	AlertDesc  string `json:"alertDesc"`  // 告警描述
	ClientIP   string `json:"clientIp"`   // 客户端IP
	ClientName string `json:"clientName"` // 客户端名称
	ClientMac  string `json:"clientMac"`  // 客户端MAC地址
	OSVersion  string `json:"osVersion"`  // 操作系统版本
	GroupName  string `json:"groupName"`  // 资产分组名称
	Severity   string `json:"severity"`   // 威胁级别 (可能是数字或者字符串)
	Status     string `json:"status"`     // 状态 (New/Investigating/Resolved)
	AlertName  string `json:"alertName"`  // 告警名称
}

// AlertsResponse is the full response from the script
type AlertsResponse struct {
	Total  int     `json:"total"`  // 总告警数量
	Pages  int     `json:"pages"`  // 总页数
	Alerts []Alert `json:"alerts"` // 告警列表
}

func (pt *PlaywrightTianqing) GetAlerts() (*AlertsResponse, error) {

	alertsUrl := viper.GetString("tianqing.alertspageUrl")
	pt.browser.Goto(pt.ctx, alertsUrl)

	// 等待初始容器和加载动画结束
	pt.browser.WaitForSelector(pt.ctx, "div[class=\"eventbox\"]")

	page, err := pt.browser.GetActivePage(pt.ctx)
	if err == nil {
		// 优化：等待列表 API 请求完成，而不是生硬地等 5 秒
		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})
	}

	js := `// 1. 找到时间下拉框的触发器并点击打开
const selects = document.querySelectorAll('.q-select');
selects.forEach(s => {
  const dropdown = s.querySelector('.q-select-dropdown');
  if (dropdown?.textContent.includes('Last 24 hours') && dropdown.textContent.includes('Specified time')) {
    s.querySelector('.q-select__control, .q-input, [class*="control"]').click();
  }
});

// 2. 等待下拉框出现后点击选项
setTimeout(() => {
  document.querySelectorAll('.q-select-dropdown__item').forEach(item => {
    if (item.textContent.trim() === 'Last 24 hours') item.click();
  });
}, 500);`
	_, err = pt.browser.Evaluate(pt.ctx, js, nil)
	if err != nil {
		fmt.Println("Error evaluating JS:", err)
		return nil, err
	}

	pt.browser.WaitForSelector(pt.ctx, "div[class=\"eventbox\"]")

	pt.browser.Sleep(pt.ctx, 5*time.Second)
	js = `
(async function() {
  const allAlerts = [];
  const pageSize = 20;

  async function getPageData() {
    const tables = document.querySelectorAll('table');
    let alertTableBody = null;

    tables.forEach(table => {
      const text = table.textContent || '';
      if (text.includes('High-risk') && text.includes('New messages')) {
        alertTableBody = table;
      }
    });

    if (!alertTableBody) return null;

    const tableVue = alertTableBody.closest('.q-table')?.__vue__;
    if (!tableVue) return null;

    const data = tableVue.pagination ? tableVue.pagination.data : (tableVue.tableData || tableVue.data || []);
    const pageInfo = {
      page: tableVue.pagination?.page || 1,
      total: tableVue.pagination?.total || tableVue.pagination?.rowsNumber || data.length,
      pageSize: tableVue.pagination?.rowsPerPage || pageSize,
      hasNext: data.length >= pageSize
    };

    return { data, pageInfo };
  }

  async function getNextPage(oldFirstItemStr) {
    const nextBtn = document.querySelector('.q-pagination .btn-next');
    if (nextBtn && !nextBtn.classList.contains('is-disabled')) {
      nextBtn.click();
      
      // 优化：智能等待，不断检查第一条数据是否发生变化，最多等待约 10 秒
      let retries = 20;
      while (retries > 0) {
        await new Promise(r => setTimeout(r, 500));
        const res = await getPageData();
        if (res && res.data.length > 0) {
           const newFirstItemStr = JSON.stringify(res.data[0]);
           if (newFirstItemStr !== oldFirstItemStr) {
              return true; // 表格数据确实刷新了
           }
        }
        retries--;
      }
      return true; // 即使超时也尝试继续，由后续容错处理
    }
    return false;
  }

  // 主循环
  let hasMore = true;
  let page = 1;

  while (hasMore) {
    await new Promise(r => setTimeout(r, 500));

    const result = await getPageData();
    if (!result) {
      console.log('未找到告警表格');
      break;
    }

    const { data, pageInfo } = result;

    // 提取并转换数据
    const pageAlerts = data.map(item => ({
      alertId: item.alert_id || item.id || '',
      alertTime: item.alert_create_time || item.create_time || '',
      alertType: (item.alert_category_name?.find(l => l.key === 'en_US')?.value) ||
                 (item.alert_category_name?.value) ||
                 item.alert_category_name || '',
      alertDesc: item.alert_description || '',
      clientIp: item.client_ip || item.source_ip || '',
      clientName: item.client_name || item.endpoint_name || '',
      clientMac: item.client_mac || '',
      osVersion: item.client_os_version_main || item.os_version || '',
      groupName: item.client_group_name || item.group_name || '',
      severity: String(item.alert_severity || item.severity || ''),
      status: String(item.alert_status || item.status || ''),
      alertName: item.alert_name || item.event_name || ''
    }));

    allAlerts.push(...pageAlerts);
    console.log(` + "`" + `第 ${page} 页: 获取 ${pageAlerts.length} 条，当前总计: ${allAlerts.length}` + "`" + `);

    // 检查是否还有下一页
    hasMore = data.length >= pageSize;

    if (hasMore) {
      const firstItemStr = data.length > 0 ? JSON.stringify(data[0]) : '';
      const moved = await getNextPage(firstItemStr);
      if (!moved) break;
      page++;
    }
  }

  return {
    total: allAlerts.length,
    pages: page,
    alerts: allAlerts
  };
})();
`
	alerts, err := pt.browser.Evaluate(pt.ctx, js, nil)
	if err != nil {
		fmt.Println("Error evaluating JS:", err)
		return nil, err
	}
	jsonData, err := json.Marshal(alerts)
	if err != nil {
		fmt.Println("Error marshaling alerts to JSON:", err)
		return nil, err
	}

	var alertsResp AlertsResponse
	err = json.Unmarshal(jsonData, &alertsResp)
	if err != nil {
		fmt.Println("Error unmarshaling to struct:", err)
		return nil, err
	}

	fmt.Println("alerts retrieved:", len(alertsResp.Alerts))
	return &alertsResp, nil
}

type AlertDetail struct {
	BasicInfo      map[string]string `json:"basicInfo"`
	AlertDetails   map[string]string `json:"alertDetails"`
	ProcessDetails map[string]string `json:"processDetails"`
	ExecuteCommand string            `json:"executeCommand"`
	Error          string            `json:"error,omitempty"`
}

func (pt *PlaywrightTianqing) GetAlertDetails(alertid string) (*AlertDetail, error) {
	alertDetailUrl := viper.GetString("tianqing.alertDetailsUrl") + alertid + viper.GetString("tianqing.alertDetailsUrlEnd")
	pt.browser.Goto(pt.ctx, alertDetailUrl)

	// 等待初始容器出现
	pt.browser.WaitForSelector(pt.ctx, "div[class=\"label-board\"]")

	// 网络请求等待完成
	page, err := pt.browser.GetActivePage(pt.ctx)
	if err == nil {
		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})
	}

	js := `
(async function() {
  // 向上遍历DOM树，寻找挂载了完整后端 JSON 对象的 Vue 实例 
  // 【优化点】：加入重试轮询，因为 Vue 在网络闲置后绑定 computed 属性或挂载 Vue Components 偶尔会有几十毫秒的延迟
  let alertData = null;
  for(let i = 0; i < 20; i++) {
      let el = document.querySelector('.label-board');
      while(el) {
          if(el.__vue__ && el.__vue__.alert) {
             alertData = el.__vue__.alert;
             break;
          }
          el = el.parentElement;
      }
      if(alertData) break;
      await new Promise(r => setTimeout(r, 500)); // 等待 500ms 重查，最多容忍 10 秒
  }
  
  if(!alertData) {
      return { error: "等待 10 秒超时，无法在页面上找到带有 .__vue__.alert 数据的 Vue 实例" };
  }

  // 完美提取底层数据
  const cm = alertData.alertMeta?.clientMeta || alertData.clientMeta || {};
  const am = alertData.alertMeta || {};
  let logContent = {};
  if (am.logContent) {
      try { logContent = JSON.parse(am.logContent); } catch(e) {}
  }
  
  return {
      basicInfo: {
          ipAddress: String(cm.ip || logContent.report_ip || logContent.ip || ''),
          endpointGroup: String(am.groupMeta?.name || logContent.group_name || '-'),
          clientUid: String(alertData.clientId?.id || logContent.client_id || '-'),
          macAddress: String(cm.mac || logContent.mac || '-'),
          user: String(logContent.process_user || cm.loginAccount || '-'),
          osVersion: String((cm.tos?.dist ? (cm.tos.dist + (cm.tos.version?.version ? " ("+cm.tos.version.version+")" : "")) : "") || logContent.env_version || '-')
      },
      alertDetails: {
          alertIncident: String(logContent.alarm_detail || am.description || ''),
          riskLevel: String(am.severity ?? logContent.severity ?? ''),
          alertType: String(alertData.categoryMeta?.name || logContent.event_type || ''),
          // 格式化时间戳
          alertTime: String(logContent.alarm_time ? new Date(logContent.alarm_time).toLocaleString('en-US', {hour12:false}) : ''),
          disposalResult: String(am.defendAction || logContent.defend_action || '')
      },
      processDetails: {
          pid: String(logContent.process_id || am.process_id || ''),
          md5: String(logContent.process_md5 || logContent.file_md5 || am.process_md5 || ''),
          sha1: String(logContent.process_sha1 || logContent.file_sha1 || am.process_sha1 || ''),
          sha256: String(logContent.process_sha256 || logContent.file_sha256 || am.process_sha256 || ''),
          filePath: String(logContent.process_path || logContent.file_path || logContent.target_process_path || am.process_path || ''),
          digitalSignature: String(logContent.process_sign || logContent.file_sign || am.process_sign || ''),
          startTime: String(logContent.process_create_time ? new Date(logContent.process_create_time).toLocaleString('en-US', {hour12:false}) : '-'),
          endTime: "-"
      },
      executeCommand: String(logContent.process_command_line || logContent.target_process_command_line || am.process_command_line || '')
  };
})();
`
	alertDetail, err := pt.browser.Evaluate(pt.ctx, js, nil)
	if err != nil {
		fmt.Println("Error evaluating JS:", err)
		return nil, err
	}
	jsonData, err := json.Marshal(alertDetail)
	if err != nil {
		fmt.Println("Error marshaling alerts to JSON:", err)
		return nil, err
	}

	var detailObj AlertDetail
	err = json.Unmarshal(jsonData, &detailObj)
	if err != nil {
		fmt.Println("Error unmarshaling alert detail to struct:", err)
		return nil, err
	}

	if detailObj.Error != "" {
		return nil, fmt.Errorf("Page script error: %s", detailObj.Error)
	}

	return &detailObj, nil
}
