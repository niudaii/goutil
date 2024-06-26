package progress

import (
	"fmt"
	"github.com/zp857/goutil/mathutil"
	"time"
)

func Calculate(total, finished int, start time.Time) (doneString string, doneFloat, remainingFloat float64) {
	done := float64(finished) / float64(total)
	// 截断到两位小数而不是四舍五入
	doneFloat = float64(int(done*100)) / 100
	doneString = fmt.Sprintf("%.2f", doneFloat)
	if done == 1.00 {
		remainingFloat = 0
		doneString = "1.0"
	} else {
		elapsed := time.Since(start).Seconds()
		// 总估计时间 = 已过时间 / 完成的比例
		totalEstimatedTime := elapsed / done
		// 剩余时间 = 总估计时间 - 已过时间
		remainingTime := totalEstimatedTime - elapsed
		remainingFloat = mathutil.Decimal(remainingTime)
	}
	return
}
