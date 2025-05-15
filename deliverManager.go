package main

import (
	"context"
	"errors"
)

type DeliverManager struct {
	TasksChan      chan *Task
	DeliverTimeout int // if task chan is full, wait for deliver timeout before discarding task
}

var (
	ErrNilTaskFunction = errors.New("nil task function")
	ErrTaskQueueFull   = errors.New("task queue full")
)

func NewDeliverManager(buffer int, deliverThreads int, ctx context.Context) *DeliverManager {
	dm := &DeliverManager{
		TasksChan: make(chan *Task, buffer),
	}
	// run deliver threads
	for i := 0; i < deliverThreads; i++ {
		go func(dm *DeliverManager) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					task := <-dm.TasksChan
					task.Run()
				}
			}
		}(dm)
	}

	return dm
}

func (dm *DeliverManager) Shutdown() {
	close(dm.TasksChan)
}

func (dm *DeliverManager) AddTask(taskFunction func()) error {
	if taskFunction == nil {
		return ErrNilTaskFunction
	}

	task := &Task{
		TaskFunction: taskFunction,
	}
	// dm.TasksChan <- task
	select {
	case dm.TasksChan <- task:
		return nil
	default:
		return ErrTaskQueueFull
	}
}

// Task
type Task struct {
	TaskFunction func()
}

func (t *Task) Run() {
	t.TaskFunction()
}
