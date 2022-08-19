package selector

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ModeIndependent = 1 // 每个Event执行自己的Handler
	ModeUnited      = 2 // 所有Event执行统一的Handler
)

type Event struct {
	mode       uint8
	mu         sync.Mutex
	running    int32
	actionList actionList
	handler    handler
	deadline   time.Time
}

func NewEvent(mode uint8) *Event {
	return &Event{
		mode:       mode,
		mu:         sync.Mutex{},
		running:    0,
		actionList: make(actionList, 0, 16),
		handler:    nil,
		deadline:   time.Time{},
	}
}

func (e *Event) AddHandler(status Status, fn func()) {
	if e.isRunning() {
		return
	}
	if status.valid() == false {
		return
	}
	if e.mode != ModeUnited {
		return
	}
	e.mu.Lock()
	if e.handler == nil {
		e.handler = make(handler)
	}
	e.handler[status] = fn
	e.mu.Unlock()
}

func (e *Event) AddAction(priority uint) *Action {
	if e.isRunning() {
		return nil
	}
	ac := &Action{
		priority: priority,
		status:   _statusDefault,
		event:    e,
		handler:  nil,
	}
	e.mu.Lock()
	e.actionList = append(e.actionList, ac)
	e.mu.Unlock()
	return ac
}

func (e *Event) Start(timeout time.Duration) {
	if e.isRunning() {
		return
	}
	atomic.StoreInt32(&e.running, 1)
	e.deadline = time.Now().Add(timeout)

	time.AfterFunc(timeout, func() {
		e.setTimeoutStatus()
		e.finish(true)
	})
}

func (e *Event) finish(timeout bool) {
	allDone := e.isActionDone()

	if !timeout && !allDone {
		return
	}

	agrees, _, overs := e.classify()

	switch {
	case timeout:
		switch e.mode {
		case ModeIndependent:
			// 如果有做出决策的action, 选择优先级最高的执行
			if len(agrees) > 0 {
				sort.Sort(agrees)
				for _, ac := range agrees {
					if ac.priority == agrees[0].priority {
						if h := ac.handler.get(StatusAgree); h != nil {
							h()
						}
					}
				}
			}

			// 如果没有action做出决策，则执行每个超时action对应的超时handler
			if len(overs) > 0 {
				for _, ac := range overs {
					if h := ac.handler.get(StatusTimeout); h != nil {
						h()
					}
				}
			}
		case ModeUnited:
			agreeHandler := e.handler.get(StatusAgree)
			overHandler := e.handler.get(StatusTimeout)
			if len(agrees) > 0 {
				sort.Sort(agrees)
				for _, ac := range agrees {
					if ac.priority == agrees[0].priority {
						if agreeHandler != nil {
							agreeHandler()
						}
					}
				}
			}
			if len(overs) > 0 {
				for _, ac := range overs {
					if ac.status == StatusTimeout {
						if overHandler != nil {
							overHandler()
						}
					}
				}
			}
		}
	default: // 没有超时，证明所有action都做出了决策，此时选择优先级最高的执行即可
		sort.Sort(agrees)
		h := e.handler.get(StatusAgree)
		switch e.mode {
		case ModeIndependent:
			for _, ac := range agrees {
				if ac.priority == agrees[0].priority {
					if ah := ac.handler.get(StatusAgree); ah != nil {
						ah()
					}
				}
			}
		case ModeUnited:
			if h != nil {
				for _, ac := range agrees {
					if ac.priority == agrees[0].priority {
						h()
					}
				}
			}
		}

	}
	return
}

func (e *Event) isRunning() bool {
	return atomic.LoadInt32(&e.running) == 1
}

func (e *Event) setTimeoutStatus() {
	e.mu.Lock()
	actions := e.actionList
	e.mu.Unlock()
	for _, ac := range actions {
		if ac.status == _statusDefault {
			ac.status = StatusTimeout
		}
	}
}

func (e *Event) isActionDone() (ok bool) {
	e.mu.Lock()
	actions := e.actionList
	e.mu.Unlock()

	ok = true
	for _, ac := range actions {
		if ac.status == _statusDefault {
			ok = false
			break
		}
	}
	return
}

func (e *Event) classify() (agrees actionList, refuses actionList, timeouts actionList) {
	e.mu.Lock()
	actions := e.actionList
	e.mu.Unlock()

	for _, ac := range actions {
		ac := ac
		switch ac.status {
		case StatusAgree:
			agrees = append(agrees, ac)
		case StatusRefuse:
			refuses = append(refuses, ac)
		case StatusTimeout:
			timeouts = append(timeouts, ac)
		}
	}
	return
}
