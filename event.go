package selector

import (
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	running = 1
	closed  = 2
)

var (
	ErrEventIsRunning = errors.New("cannot add action when event running")
)

type Event struct {
	status int32 // Event运行状态

	mu            sync.Mutex   // Mutex
	priorityCount map[uint]int // <priority, count>
	actionList    actionList   // Event对应的动作

	deadline  time.Time    // Event截止时间
	notify    chan *action //
	OnTimeout func()       // Event超时后默认执行的Handler
}

func NewEvent(handler func()) *Event {
	return &Event{
		status:        0,
		mu:            sync.Mutex{},
		priorityCount: make(map[uint]int),
		actionList:    make(actionList, 0, 16),
		OnTimeout:     handler,
		notify:        make(chan *action, 16),
	}
}

func (e *Event) AddAction(priority uint) (Action, error) {
	if !e.isInit() {
		return nil, ErrEventIsRunning
	}
	ac := &action{
		priority: priority,
		state:    _statusDefault,
		event:    e,
		handler:  nil,
	}
	e.mu.Lock()
	e.actionList = append(e.actionList, ac)
	e.priorityCount[priority] += 1
	e.mu.Unlock()
	return ac, nil
}

func (e *Event) AddActionWithHandler(priority uint, handler Handler) (Action, error) {
	if !e.isInit() {
		return nil, ErrEventIsRunning
	}
	ac := &action{
		priority: priority,
		state:    _statusDefault,
		event:    e,
		handler:  handler,
	}
	e.mu.Lock()
	e.actionList = append(e.actionList, ac)
	e.priorityCount[priority] += 1
	e.mu.Unlock()
	return ac, nil
}

func (e *Event) Start(timeout time.Duration) {
	if !atomic.CompareAndSwapInt32(&e.status, 0, running) {
		return
	}
	e.deadline = time.Now().Add(timeout)

	e.mu.Lock()
	sort.Sort(e.actionList)
	e.mu.Unlock()

	go func() {
		timer := time.NewTimer(timeout)

		defer func() {
			if timer != nil {
				timer.Stop()
			}
			e.Reset()
		}()

		for {
			select {
			case <-timer.C:
				e.finish(nil)
				return
			case ac, ok := <-e.notify:
				if ok {
					if e.finish(ac) {
						return
					}
				}
			}
		}
	}()
}

func (e *Event) Reset() {
	if e.isClosed() {
		close(e.notify)
		*e = Event{
			status:    0,
			OnTimeout: nil,
			deadline:  time.Time{},
			notify:    make(chan *action, 16),
		}
		e.mu.Lock()
		e.actionList = e.actionList[0:0]
		e.priorityCount = make(map[uint]int)
		e.mu.Unlock()
	}
}

func (e *Event) finish(ac *action) (done bool) {
	defer func() {
		if done {
			atomic.StoreInt32(&e.status, closed)
		}
	}()

	var allAction actionList
	var timeout = ac == nil // 是否超时

	e.mu.Lock()
	if timeout {
		// 超时将尚未决策的action状态置为Timeout
		for _, a := range e.actionList {
			a.timeout()
		}
	}
	allAction = e.actionList
	e.mu.Unlock()

	// Event决策结束的条件:
	// 1. 当前优先级最高的Action做完决策, 立即执行该优先级对应的所有Action
	// 2. 所有Action均做出决策(或Agree或Refuse)
	// 3. 决策超时

	switch {
	case timeout: // 超时
		// 判断是否有Agree的action
		for i, a := range allAction {
			switch a.state {
			case StateTimeout: // 执行Action的超时Handler
				if h := a.handler.get(StateTimeout); h != nil {
					h()
				}
			case StateAgree:
				if !done {
					e.exec(allAction, a.priority, i)
					done = true
				}
			}
		}

	default: // 有action做出决策
		for _, a := range allAction {
			switch a.state {
			case StateAgree: //
				if !done {
					if e.hasMadeDecision(allAction, a.priority) {
						e.exec(allAction, a.priority, -1)
						done = true
						return
					}
					return
				}
			case _statusDefault:
				return
			}
		}
	}

	// 如果没有Agree的action, 需要执行Event的Handler
	if !done {
		done = true
		if e.OnTimeout != nil {
			e.OnTimeout()
		}
	}

	return
}

func (e *Event) exec(list actionList, priority uint, idx int) {
	if idx >= 0 {
		n := e.priorityCount[priority]
		switch {
		case n > 1:
			for _, ac := range list[idx : idx+n] {
				if ac.state != StateAgree {
					continue
				}
				if h := ac.handler.get(StateAgree); h != nil {
					h()
				}
			}
		default:
			if h := list[idx].handler.get(StateAgree); h != nil {
				h()
			}
		}
	} else {
		for _, ac := range list {
			if ac.priority == priority && ac.state == StateAgree {
				if h := ac.handler.get(StateAgree); h != nil {
					h()
				}
			}
		}
	}
}

func (e *Event) hasMadeDecision(list actionList, priority uint) (ok bool) {
	ok = true
	for _, ac := range list {
		if ac.priority > priority {
			continue
		}
		if ac.priority < priority {
			break
		}
		if ac.state == _statusDefault {
			ok = false
			break
		}
	}
	return
}

func (e *Event) makeDecision(state State, ac *action) {
	e.mu.Lock()
	for _, a := range e.actionList {
		if a == ac {
			a.state = state
			if state == StateRefuse {
				if h := a.handler.get(StateRefuse); h != nil {
					h()
				}
			}
			break
		}
	}
	e.mu.Unlock()
}

func (e *Event) isInit() bool {
	return atomic.LoadInt32(&e.status) == 0
}

func (e *Event) isRunning() bool {
	return atomic.LoadInt32(&e.status) == running
}

func (e *Event) isClosed() bool {
	return atomic.LoadInt32(&e.status) == closed
}
