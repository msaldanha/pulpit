package server

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChannelWaitGroup", func() {

	BeforeEach(func() {
	})

	It("Should block until all finish", func() {
		wg := new(ChannelWaitGroup)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(time.Millisecond * 300)
			}()
		}
		select {
		case <-wg.Wait():
		// ok
		case <-time.After(time.Second):
			Fail("timed out waiting for channel to finish")
		}
	})
	It("Should allow Wait() be called many times", func() {
		wg := new(ChannelWaitGroup)
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond * 300)
		}()
		timeout := time.After(time.Second)
		heartBeat := 0
		for i := 0; i < 5; i++ {
			select {
			case <-wg.Wait():
				return
			case <-time.After(time.Millisecond * 100):
				// heart beat
				heartBeat++
			case <-timeout:
				Fail("timed out waiting for channel to finish")
			}
		}
		Expect(heartBeat > 1).To(BeTrue())
	})
})
