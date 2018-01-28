package pool

import (
	"log"
	"testing"
	"time"

	"github.com/fmarmol/usine/status"
)

func TestPool(t *testing.T) {
	p := NewPool(5, 10)
	if p == nil {
		t.Error("could not init pool")
	}
	p.Init()
	go func() {
		p.Run()
	}()
	time.Sleep(time.Second)
	log.Println("Send status")
	p.ChanCli <- status.PW_STATUS
	log.Println("Send status end")
	time.Sleep(10 * time.Second)
}
