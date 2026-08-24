package review

import (
	"errors"
	"testing"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/store"
)

// newServiceWithSealedBatch 构造一个已完成业务闭环、处于 sealed 状态的批次及其首个峰，
// 用于复核（review）服务的封存守卫回归测试。
func newServiceWithSealedBatch(t *testing.T) (*Service, string) {
	t.Helper()
	db, err := store.Open(t.TempDir() + "/review.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	batches := store.NewBatchStore(db)
	peaks := store.NewPeakStore(db)

	b, err := model.NewBatch("b1", "sealed")
	if err != nil {
		t.Fatal(err)
	}
	if err := batches.Insert(b); err != nil {
		t.Fatal(err)
	}
	// 把批次直接推到 sealed 状态：状态机允许 publishable → sealed。
	b.Status = model.BatchPublishable
	if err := b.Transition(model.BatchSealed); err != nil {
		t.Fatal(err)
	}
	if err := batches.Update(b); err != nil {
		t.Fatal(err)
	}

	p, err := model.NewPeak("p1", "b1", 1, 51, 50, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := peaks.Insert(p); err != nil {
		t.Fatal(err)
	}
	return New(peaks, batches), "p1"
}

// TestSealedBatchRejectsPeakMutation 断言：封存批次上任何修改峰的操作都返回 ErrSealed，
// 而不是被当作普通输入错误放过或误报。
func TestSealedBatchRejectsPeakMutation(t *testing.T) {
	s, peakID := newServiceWithSealedBatch(t)

	cases := []struct {
		name string
		call func() (*model.Peak, error)
	}{
		{"LockReference", func() (*model.Peak, error) { return s.LockReference(peakID) }},
		{"UnlockReference", func() (*model.Peak, error) { return s.UnlockReference(peakID) }},
		{"Exclude", func() (*model.Peak, error) { return s.Exclude(peakID) }},
		{"Restore", func() (*model.Peak, error) { return s.Restore(peakID) }},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			// Restore 需要峰先处于 excluded 态；在 sealed 批次上排除了之后 Restore 才有意义，
			// 但二者都应被 ErrSealed 拦在 assertMutable，所以此处 Restore 用一个未排除峰触发守卫即可。
			_, err := c.call()
			if !errors.Is(err, model.ErrSealed) {
				t.Fatalf("%s on sealed batch: err = %v, want ErrSealed", c.name, err)
			}
		})
	}
}
