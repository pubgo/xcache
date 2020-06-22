package xcache

/*
// 定时清理过期数据
func (x *xcache) initJanitor(interval time.Duration) error {
	if interval > 0 {
		if interval < consts.DefaultMinExpiration {
			return xerror.WrapF(ErrClearTime, "过期时间(%s)小于最小过期时间(%s)", interval, consts.DefaultMinExpiration)
		}

		x.mutex.Lock()
		defer x.mutex.Unlock()
		if x.janitor == nil {
			runtime.SetFinalizer(x, stopJanitor)
		} else {
			stopJanitor(x)
		}
		runJanitor(x, interval)
	}
	return nil
}

func stopJanitor(c *xcache) {
	c.janitor.stop <- true
}

func runJanitor(c *xcache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

// Run ...
func (j *janitor) Run(c *xcache) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			_ = c.DeleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}
*/
