package indexer

type workerSet struct {
	maxWorkers int
	ch         chan struct{}
}

func newWorkerSet(maxNum int) *workerSet {
	var ch = make(chan struct{})
	go func() {
		for i := 0; i < maxNum; i++ {
			ch <- struct{}{}
		}
	}()
	return &workerSet{maxNum, ch}
}

func (ws *workerSet) do(fn func()) {
	<-ws.ch
	go func() {
		fn()
		ws.ch <- struct{}{}
	}()
}
