package playwrighttianqing

import (
	"encoding/json"
	"fmt"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/viper"
)

type AssetsResponse struct {
	Total  int                      `json:"total"`
	Pages  int                      `json:"pages"`
	Assets []map[string]interface{} `json:"assets"`
}

func (pt *PlaywrightTianqing) GetAssets() (*AssetsResponse, error) {
	assetsUrl := viper.GetString("tianqing.assetsUrl")
	fmt.Println("Navigating to assets page:", assetsUrl)
	pt.browser.Goto(pt.ctx, assetsUrl)

	// 等待页面容器出现
	pt.browser.WaitForSelector(pt.ctx, "div[class*=\"p-table\"]")

	page, err := pt.browser.GetActivePage(pt.ctx)
	if err == nil {
		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})
	}

	js := `
(async function() {
  const allAssets = [];
  const targetPageSize = 100;

  // ========== 1. 定义数据获取逻辑 ==========
  async function getPageData() {
    const tables = document.querySelectorAll('table');
    let targetTableVue = null;
    for (let table of tables) {
        let el = table;
        while(el) {
            if (el.__vue__ && (el.__vue__.data || el.__vue__.tableData || (el.__vue__.pagination && el.__vue__.pagination.data))) {
                targetTableVue = el.__vue__;
                break;
            }
            el = el.parentElement;
        }
        if (targetTableVue) break;
    }
    if (!targetTableVue) return null;

    let rawData = [];
    if (targetTableVue.pagination && targetTableVue.pagination.data) {
        rawData = targetTableVue.pagination.data;
    } else if (targetTableVue.tableData) {
        rawData = targetTableVue.tableData;
    } else if (targetTableVue.data) {
        rawData = targetTableVue.data;
    }
    const data = JSON.parse(JSON.stringify(rawData));
    return { data };
  }

  // ========== 2. 等待页面首批表格数据和分页组件出现 ==========
  let initialDataLoaded = false;
  for(let i=0; i<20; i++) {
      let res = await getPageData();
      if (res && res.data && res.data.length > 0) {
          initialDataLoaded = true;
          break;
      }
      await new Promise(r => setTimeout(r, 500));
  }

  // ========== 3. 将分页调整为每页100条 ==========
  try {
      const selects = document.querySelectorAll('.q-pagination .q-select, .el-pagination .el-select, .biz-skylar-pagination-table .q-select');
      for (let s of selects) {
          const trigger = s.querySelector('.q-select__control, .q-field__control, .q-input, .el-input__inner, [class*="control"]');
          if (trigger) {
              const currentVal = (trigger.textContent || trigger.value || "").trim();
              if (currentVal.includes("100")) {
                  // 已经是100条了，跳过
                  continue;
              }

              // 触发点击展开下拉菜单
              const mdEvent = new MouseEvent('mousedown', {bubbles: true, cancelable: true});
              trigger.dispatchEvent(mdEvent);
              trigger.click();

              await new Promise(r => setTimeout(r, 1500));
              
              const oldDataInfo = await getPageData();
              const oldLen = oldDataInfo && oldDataInfo.data ? oldDataInfo.data.length : 0;
              
              const items = document.querySelectorAll('.q-select-dropdown__item, .el-select-dropdown__item, .el-select-dropdown__list li, .q-item');
              for (let item of items) {
                  if (item.textContent.trim().includes('100')) {
                      const clickEv = new MouseEvent('click', {bubbles: true, cancelable: true});
                      item.dispatchEvent(clickEv);
                      item.click();
                      
                      // 动态等待100条数据加载出来
                      let loaded = false;
                      for (let wait = 0; wait < 30; wait++) { // 最多等15秒
                          await new Promise(r => setTimeout(r, 500));
                          const check = await getPageData();
                          if (check && check.data && check.data.length > oldLen) {
                              loaded = true;
                              break;
                          }
                          // 另外：如果总条数不足20条，改100条长度没变化，此时我们可以检查下第一个元素的数据指纹是否变化
                          // 但大部分情况总数大于20，长度会直接变化。
                      }
                      
                      // 如果还是没触发跳出（比如总数真就只有不到20个或者网络极慢），再保底等2秒
                      if (!loaded) {
                          await new Promise(r => setTimeout(r, 2000));
                      }
                      break;
                  }
              }
          }
      }
  } catch (e) {
      console.log('分页选择器调整失败, 继续使用默认', e);
  }

  // ========== 4. 安全翻页逻辑 ==========
  async function getNextPage(oldFirstItemStr, pageNum) {
    const nextBtn = document.querySelector('.q-pagination .btn-next, .el-pagination .btn-next, .btn-next');
    if (nextBtn && !nextBtn.disabled && !nextBtn.classList.contains('is-disabled') && !nextBtn.classList.contains('disabled')) {
      console.log("-> 准备点击翻页，目标页码: " + (pageNum + 1));
      nextBtn.click();
      
      let retries = 40; // 放宽到 20 秒
      while (retries > 0) {
        await new Promise(r => setTimeout(r, 500));
        
        // 尝试捕获由于网络请求正在加载时，天擎前端表格呈现的 loading 遮罩层。
        const loadingMask = document.querySelector('.el-loading-mask, .q-inner-loading, .el-loading-spinner');
        if (loadingMask && window.getComputedStyle(loadingMask).display !== 'none') {
            console.log("-> 页面正在加载态，继续等待...");
            continue;
        }

        const res = await getPageData();
        if (res && res.data && res.data.length > 0) {
           const newFirstItemStr = JSON.stringify(res.data[0]);
           if (newFirstItemStr !== oldFirstItemStr) {
              console.log("-> 新一页数据已侦测到！新数据首页与前页不同。");
              await new Promise(r => setTimeout(r, 1500));
              return true;
           }
        }
        retries--;
      }
      console.log("-> 警告：翻页等待超时，未能侦测到新数据。网络卡顿？");
      return false; // 超时就不翻了，避免疯狂乱点跳页
    }
    return false;
  }

  // ========== 5. 主提取循环 ==========
  let hasMore = true;
  let page = 1;

  while (hasMore) {
    await new Promise(r => setTimeout(r, 1000));

    const result = await getPageData();
    if (!result || !result.data) {
      console.log('未找到数据或 Vue 实例');
      break;
    }

    const curLen = result.data.length;
    allAssets.push(...result.data);
    console.log("Assets extraction: 已攫取第 " + page + " 页, 当前本页条数: " + curLen + ", 总累计: " + allAssets.length);
    
    // 如果返回条数太少，且小于我们请求的pageSize（可能尾页）
    const theNextBtn = document.querySelector('.q-pagination .btn-next, .el-pagination .btn-next');
    const isNextBtnDisabled = !theNextBtn || theNextBtn.disabled || theNextBtn.classList.contains('is-disabled') || theNextBtn.classList.contains('disabled');
    
    if (isNextBtnDisabled || curLen === 0) {
        break; // 没有下一页了
    }

    const firstItemStr = curLen > 0 ? JSON.stringify(result.data[0]) : '';
    const moved = await getNextPage(firstItemStr, page);
    if (!moved) break;
    
    page++;
  }

  return {
    total: allAssets.length,
    pages: page,
    assets: allAssets
  };
})();
`

	assetsData, err := pt.browser.Evaluate(pt.ctx, js, nil)
	if err != nil {
		fmt.Println("Error evaluating JS on assets page:", err)
		return nil, err
	}

	jsonData, err := json.Marshal(assetsData)
	if err != nil {
		fmt.Println("Error marshaling assets result to JSON:", err)
		return nil, err
	}

	var assetsResp AssetsResponse
	err = json.Unmarshal(jsonData, &assetsResp)
	if err != nil {
		fmt.Println("Error unmarshaling assets result to struct:", err)
		return nil, err
	}

	fmt.Println("============ Assets 提取完成 ============")
	fmt.Println("共提取页数:", assetsResp.Pages)
	fmt.Println("总提取资产数:", len(assetsResp.Assets))
	return &assetsResp, nil
}
