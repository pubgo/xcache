package xcache

import (
	"fmt"
	"runtime"
)

var defaultTask = newTasks()

func newTasks() *tasks {
	tsk := &tasks{tsk: make(chan func(), 1000), closed: make(chan struct{}), err: make(chan error)}
	go tsk.run()
	return tsk
}

type tasks struct {
	tsk    chan func()
	closed chan struct{}
	err    chan error
}

func (t *tasks) run() {
	for {
		select {
		case k := <-t.tsk:
			go k()
		case <-t.closed:
			return
		}
	}
}

func (t *tasks) wait() error {
	for {
		select {
		case err := <-t.err:
			t.closed <- struct{}{}
			return err
		default:
			if len(t.tsk) == 0 {
				return nil
			}
			runtime.Gosched()
		}
	}

}

func (t *tasks) submit(fn func() error) {
	t.tsk <- func() {
		defer func() {
			if err1 := recover(); err1 != nil {
				t.err <- fmt.Errorf("%+v", err1)
			}
		}()

		if err := fn(); err != nil {
			t.err <- err
		}
	}
}
