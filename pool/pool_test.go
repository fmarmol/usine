package pool

import (
	"sync"
	"testing"
)

func TestPool(t *testing.T) {
	p := NewPool(5, 10)
	if p == nil {
		t.Error("could not init pool")
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	p.Init()
	go func() {
		p.Run()
		wg.Done()
	}()
	jm := NewJobManager()
	jm.Run(p)
	wg.Wait()
}
