package server

import "sync"

// ChannelWaitGroup works as sync.WaitGroup but its Wait() func returns a channel that will flag that all goroutines
// have finished
type ChannelWaitGroup struct {
	sync.WaitGroup
	doneChan chan struct{}
	once     sync.Once
}

func (wg *ChannelWaitGroup) Wait() chan struct{} {
	wg.once.Do(func() {
		wg.doneChan = make(chan struct{})
		go func() {
			wg.WaitGroup.Wait()
			wg.doneChan <- struct{}{}
			close(wg.doneChan)
		}()
	})

	return wg.doneChan
}
