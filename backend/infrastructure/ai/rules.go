package ai

import "context"

// RuleSummaryProvider supplies deterministic explanations; no external AI call.
type RuleSummaryProvider struct{}

func (RuleSummaryProvider) Reasons(ctx context.Context, code string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch code {
	case "VALUE":
		return []string{"ราคาผ่านเกณฑ์ EV ของโมเดล Poisson จากผลการแข่งขันย้อนหลัง", "โมเดลพื้นฐานนี้ยังไม่มีการยืนยันความแม่นยำ และยังไม่ปรับตามรายชื่อผู้เล่น"}, nil
	case "NO_VALUE":
		return []string{"ยังไม่มีราคาที่คุ้มตามโมเดลพื้นฐานจากผลย้อนหลัง"}, nil
	case "STALE_ODDS":
		return []string{"ราคาต้นทางเก่าเกินเกณฑ์ แม้เพิ่งซิงก์ก็ยังเลือกไม่ได้"}, nil
	case "MISSING_INPUTS":
		return []string{"ยังไม่มีราคาที่รองรับ หรือผลย้อนหลังเหย้า/เยือนอย่างน้อยฝั่งละ 3 นัดสำหรับประเมิน EV"}, nil
	default:
		return (MockAISummaryProvider{}).Reasons(ctx, code)
	}
}
