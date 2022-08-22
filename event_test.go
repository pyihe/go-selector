package selector

import (
	"testing"
	"time"
)

const (
	Chi = iota + 1
	Peng
	Gang
	Hu
)

func TestNewEvent(t *testing.T) {
	var event = NewEvent(func() {
		t.Logf("没有玩家选择操作, 继续摸牌...\n")
	})

	Timeout(event, t)
	time.Sleep(5 * time.Second)

	Work(event, t)
	time.Sleep(5 * time.Second)

	LowAgreeHighRefuse(event, t)
	time.Sleep(5 * time.Second)

	MultipleHighest(event, t)
	time.Sleep(5 * time.Second)

	MultipleLower(event, t)
	time.Sleep(5 * time.Second)
}

func Timeout(event *Event, t *testing.T) {
	chi, _ := event.AddActionWithHandler(uint(Chi), Handler{
		StateAgree: func() {
			t.Logf("吃牌...\n")
		},
		StateRefuse: func() {
			t.Logf("拒绝吃牌...\n")
		},
		StateTimeout: func() {
			t.Logf("吃牌超时...\n")
		},
	})
	event.Start(3 * time.Second)
	time.Sleep(1 * time.Second)

	t.Logf("remain: %v\n", chi.Deadline().Sub(time.Now()).Seconds())

	time.Sleep(3 * time.Second)
	t.Logf("\n")
}

func Work(event *Event, t *testing.T) {
	chi, _ := event.AddActionWithHandler(uint(Chi), Handler{
		StateAgree: func() {
			t.Logf("吃牌...\n")
		},
		StateRefuse: func() {
			t.Logf("拒绝吃牌...\n")
		},
		StateTimeout: func() {
			t.Logf("吃牌超时...\n")
		},
	})

	peng, _ := event.AddAction(uint(Peng))
	peng.AddHandler(StateAgree, func() {
		t.Logf("碰牌...\n")
	})
	peng.AddHandler(StateRefuse, func() {
		t.Logf("拒绝碰牌...\n")
	})
	peng.AddHandler(StateTimeout, func() {
		t.Logf("碰牌超时...\n")
	})
	peng.AddHandler(State(10), func() {
		t.Logf("无效Handler\n")
	})

	gang, _ := event.AddActionWithHandler(uint(Hu), Handler{
		StateAgree: func() {
			t.Logf("杠牌...\n")
		},
		StateRefuse: func() {
			t.Logf("拒绝杠牌...\n")
		},
		StateTimeout: func() {
			t.Logf("杠牌超时...\n")
		},
	})

	hu, _ := event.AddActionWithHandler(uint(Hu), Handler{
		StateAgree: func() {
			t.Logf("胡牌...\n")
		},
		StateRefuse: func() {
			t.Logf("拒绝胡牌...\n")
		},
		StateTimeout: func() {
			t.Logf("胡牌超时...\n")
		},
	})
	event.Start(3 * time.Second)
	time.Sleep(5 * time.Millisecond)

	chi.Refuse()
	peng.Agree()
	hu.Refuse()
	gang.Agree()
	time.Sleep(3 * time.Second)
	t.Logf("\n")
}

func LowAgreeHighRefuse(event *Event, t *testing.T) {
	chi, _ := event.AddActionWithHandler(uint(Chi), Handler{
		StateAgree: func() {
			t.Logf("吃牌...\n")
		},
		StateRefuse: func() {
			t.Logf("拒绝吃牌...\n")
		},
		StateTimeout: func() {
			t.Logf("吃牌超时...\n")
		},
	})

	peng, _ := event.AddAction(uint(Peng))
	peng.AddHandler(StateAgree, func() {
		t.Logf("碰牌...\n")
	})
	peng.AddHandler(StateRefuse, func() {
		t.Logf("拒绝碰牌...\n")
	})
	peng.AddHandler(StateTimeout, func() {
		t.Logf("碰牌超时...\n")
	})

	event.Start(3 * time.Second)
	// 无效的Start
	event.Start(1 * time.Second)
	time.Sleep(5 * time.Millisecond)

	if _, err := event.AddAction(10); err != nil {
		t.Logf("Event启动后添加Action失败: %v\n", err)
	}
	if _, err := event.AddActionWithHandler(11, Handler{
		StateAgree: func() {
			t.Logf("test\n")
		},
	}); err != nil {
		t.Logf("Event启动后添加Action失败: %v\n", err)
	}

	chi.Agree()
	time.Sleep(3 * time.Second)
	t.Logf("\n")
}

func MultipleHighest(event *Event, t *testing.T) {
	chi, _ := event.AddActionWithHandler(uint(Chi), Handler{
		StateAgree: func() {
			t.Logf("吃牌...\n")
		},
	})

	peng1, _ := event.AddAction(uint(Peng))
	peng1.AddHandler(StateAgree, func() {
		t.Logf("碰牌1...\n")
	})

	peng2, _ := event.AddAction(uint(Peng))
	peng2.AddHandler(StateAgree, func() {
		t.Logf("碰牌2...\n")
	})

	event.Start(3 * time.Second)
	time.Sleep(5 * time.Millisecond)

	chi.Agree()
	peng2.Agree()
	time.Sleep(1 * time.Second)
	peng1.Refuse()

	time.Sleep(3 * time.Second)
	t.Logf("\n")
}

func MultipleLower(event *Event, t *testing.T) {
	chi, _ := event.AddActionWithHandler(uint(Chi), Handler{
		StateAgree: func() {
			t.Logf("吃牌...\n")
		},
	})

	chi2, _ := event.AddAction(uint(Chi))
	chi2.AddHandler(StateAgree, func() {
		t.Logf("吃牌2...\n")
	})
	// 无效
	chi2.Refuse()
	chi2.Agree()

	chi3, _ := event.AddActionWithHandler(uint(Chi), Handler{
		StateAgree: func() {
			t.Logf("吃牌3...\n")
		},
	})

	peng, _ := event.AddAction(uint(Peng))
	peng.AddHandler(StateAgree, func() {
		t.Logf("碰牌2...\n")
	})
	peng.Agree()

	event.Start(3 * time.Second)
	time.Sleep(5 * time.Millisecond)

	chi.Agree()
	chi2.Agree()
	peng.Refuse()
	_ = chi3

	time.Sleep(3 * time.Second)
	t.Logf("\n")
}
