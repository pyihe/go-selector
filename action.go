package selector

import "time"

type Status uint8 // 动作状态

const (
	_statusDefault Status = iota // 默认状态
	StatusAgree                  // 同意
	StatusRefuse                 // 拒绝
	StatusTimeout                // 超时
	_end
)

func (s Status) valid() bool {
	return s < _end && s > _statusDefault
}

type handler map[Status]func()

func (h handler) add(status Status, fn func()) {
	h[status] = fn
}

func (h handler) get(status Status) func() {
	return h[status]
}

type Action struct {
	priority uint   // 优先级
	status   Status // 状态
	event    *Event
	handler  handler // action对应的handler
}

func (ac *Action) Deadline() time.Time {
	return ac.event.deadline
}

func (ac *Action) Agree() {
	if ac.status != _statusDefault || !ac.event.isRunning() {
		return
	}
	ac.status = StatusAgree
	ac.event.finish(false)
}

func (ac *Action) Refuse() {
	if ac.status != _statusDefault || !ac.event.isRunning() {
		return
	}
	ac.status = StatusRefuse
	if ac.event.mode == ModeUnited {
		h := ac.event.handler.get(StatusRefuse)
		if h != nil {
			h()
		}
	}
	ac.event.finish(false)
}

type actionList []*Action

func (al actionList) Len() int {
	return len(al)
}

func (al actionList) Swap(i, j int) {
	al[i], al[j] = al[j], al[i]
}

func (al actionList) Less(i, j int) bool {
	return al[i].priority > al[j].priority
}
