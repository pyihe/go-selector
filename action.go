package selector

import (
	"sync"
	"time"
)

type Action interface {
	Agree()
	Refuse()
	Deadline() time.Time
	AddHandler(State, func())
}

type State uint8 // 动作状态

const (
	_statusDefault State = iota // 默认状态
	StateAgree                  // 同意
	StateRefuse                 // 拒绝
	StateTimeout                // 超时
	_end
)

func (s State) valid() bool {
	return s < _end && s > _statusDefault
}

type Handler map[State]func()

func (h Handler) add(status State, fn func()) {
	h[status] = fn
}

func (h Handler) get(status State) func() {
	return h[status]
}

type action struct {
	priority uint   // 优先级
	state    State  // 状态
	event    *Event // Event

	mu      sync.Mutex
	handler Handler // action对应的handler
}

func (ac *action) AddHandler(s State, fn func()) {
	if !s.valid() {
		return
	}
	ac.mu.Lock()
	if ac.handler == nil {
		ac.handler = make(Handler)
	}
	ac.handler.add(s, fn)
	ac.mu.Unlock()
}

func (ac *action) Deadline() time.Time {
	return ac.event.deadline
}

func (ac *action) Agree() {
	if ac.state != _statusDefault || !ac.event.isRunning() {
		return
	}
	ac.event.makeDecision(StateAgree, ac)
	ac.event.notify <- ac
}

func (ac *action) Refuse() {
	if ac.state != _statusDefault || !ac.event.isRunning() {
		return
	}
	ac.event.makeDecision(StateRefuse, ac)
	ac.event.notify <- ac
}

func (ac *action) timeout() {
	if ac.state != _statusDefault {
		return
	}
	ac.state = StateTimeout
}

type actionList []*action

func (al actionList) Len() int {
	return len(al)
}

func (al actionList) Swap(i, j int) {
	al[i], al[j] = al[j], al[i]
}

func (al actionList) Less(i, j int) bool {
	return al[i].priority > al[j].priority
}
