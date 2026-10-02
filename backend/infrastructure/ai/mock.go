package ai

import (
	"context"
	"fmt"
)

type MockAISummaryProvider struct{}

func (MockAISummaryProvider) Reasons(ctx context.Context, code string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch code {
	case "VALUE":
		return []string{"ราคามีความคุ้มค่าตามเกณฑ์ EV ของโมเดลจำลอง", "ใช้ Poisson จากประตูคาดการณ์จำลอง ไม่ใช่ข้อมูลการแข่งขันจริง"}, nil
	case "SMALL_VALUE":
		return []string{"EV เป็นบวก แต่ยังไม่ถึงเกณฑ์ PLAY", "ติดตามราคาที่เหมาะสมก่อนตัดสินใจ"}, nil
	case "NO_VALUE":
		return []string{"ยังไม่มีราคาที่คุ้มตามโมเดลจำลอง"}, nil
	case "STALE_ODDS":
		return []string{"ราคาหมดอายุ กรุณารัน worker เพื่ออัปเดตข้อมูลจำลอง"}, nil
	case "MATCH_STARTED":
		return []string{"การแข่งขันเริ่มแล้วหรือจบแล้ว"}, nil
	case "MISSING_INPUTS":
		return []string{"ข้อมูลสำหรับวิเคราะห์ยังไม่ครบ"}, nil
	default:
		return nil, fmt.Errorf("unknown reason code %s", code)
	}
}
