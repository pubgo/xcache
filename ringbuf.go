package xcache

type addCommand struct {
	data []byte
	ret  chan uint16
}

type deleteCommand struct {
	u  uint16
	u2 uint16
}

type ringBuf struct {
	closeChan   chan struct{}
	commandChan chan interface{}
	data        [keyCode]struct {
		data [][]byte
		q    queue
	}
}

func (r *ringBuf) Add(bytes []byte) uint16 {
	ac := addCommand{data: bytes, ret: make(chan uint16)}
	r.commandChan <- ac
	return <-ac.ret
}

func (r *ringBuf) Delete(u uint16, u2 uint16) {
	r.commandChan <- deleteCommand{
		u:  u,
		u2: u2,
	}
}

func (r *ringBuf) Replace(u uint16, u2 uint16, data []byte) uint16 {
	r.commandChan <- deleteCommand{u: u, u2: u2}
	ac := addCommand{data: data, ret: make(chan uint16)}
	r.commandChan <- ac
	return <-ac.ret
}

func (r *ringBuf) Get(u uint16, u2 uint16) []byte {
	if len(r.data[u].data) == 0 {
		return nil
	}
	return r.data[u].data[u2]
}

func (r *ringBuf) Close() {
	r.closeChan <- struct{}{}
}

func (r *ringBuf) run() {
	for {
		select {
		case <-r.closeChan:
			return
		case c := <-r.commandChan:
			switch c := c.(type) {
			case deleteCommand:
				go r.data[c.u].q.Push(int(c.u2))
			case addCommand:
				l := len(c.data) >> 3
				size := r.data[l].q.Pop().(int)
				if size != -1 {
					r.data[l].data[size] = append(r.data[l].data[size][:0], c.data...)
					c.ret <- uint16(size)
				} else {
					r.data[l].data = append(r.data[l].data, c.data)
					c.ret <- uint16(len(r.data[l].data)) - 1
				}
			}
		}
	}
}

func newRingBuf() *ringBuf {
	r := &ringBuf{
		commandChan: make(chan interface{}, 512),
		closeChan:   make(chan struct{}),
	}
	go r.run()
	return r
}
