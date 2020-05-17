package routes

import (
	"errors"
	"strings"
)

type Route int8

const (
	//отмена прохода
	TelegramCallbackUpdate Route = iota + 1
	TelegramCommandUpdate
	TelegramPromptUpdate
)

type Queue struct {
	ID             Route
	name           string
	workerPoolSize int
	qos            int
	globalQos      bool
}

func (q *Queue) Name() string {
	return q.name
}

func (q *Queue) Route() string {
	return strings.ReplaceAll(q.name, "_", ".")
}

func (q *Queue) RouteWildcard() string {
	return q.Route() + ".*"
}

func (q *Queue) NameByParam(p string) string {
	return q.name + "_" + p
}

func (q *Queue) RouteByParam(p string) string {
	return q.Route() + "." + p
}

func (q *Queue) WorkerPoolSize() int {
	if q.workerPoolSize == 0 {
		q.workerPoolSize = 20
	}
	return q.workerPoolSize
}

func (q *Queue) Qos() int {
	return q.qos
}

func (q *Queue) GlobalQos() bool {
	return q.globalQos
}

var routes = []*Queue{
	{
		ID:   TelegramCallbackUpdate,
		name: "tg_callback_update",
	},
	{
		ID:   TelegramPromptUpdate,
		name: "tg_prompt_update",
	},
	{
		ID:   TelegramCommandUpdate,
		name: "tg_command_update",
	},
}

var queuesByID = make(map[Route]*Queue)

func init() {
	for _, v := range routes {
		queuesByID[v.ID] = v
	}
}

func QueueByID(id Route) (*Queue, error) {
	v, ok := queuesByID[id]
	if !ok {
		return nil, errors.New("not found route")
	}

	return v, nil
}

func MustQueueByID(id Route) *Queue {
	v, err := QueueByID(id)
	if err != nil {
		panic(err)
	}

	return v
}
