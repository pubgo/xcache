package xcache

import (
	"errors"
	"testing"
)

func TestTasks(t *testing.T) {
	for i := 0; i < 1000; i++ {
		_i := i
		defaultTask.submit(func() error {
			if _i == 100 {
				return errors.New("100")
			}
			//fmt.Println(_i)
			return nil
		})
	}
	if err := defaultTask.wait(); err != nil {
		t.Log(err)
	}
}
