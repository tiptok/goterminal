package main

import (
	"fmt"
	"sync"
)

func main() {
	var exit chan int
	exit = make(chan int)
	ConfigInit()
	//t := NewTerminal("127.0.0.1", 38922)
	if Param.InPressureTestMode {
		var wg sync.WaitGroup
		for i := 0; i < Param.TermNum; i++ {
			wg.Add(1)
			tmpSimNum := fmt.Sprintf("%d", (Param.OriginSimNum + int64(i)))
			t := NewTerminal(Param.RemoteIP, Param.RemotePort, tmpSimNum)
			if t != nil {
				go t.KeepConn()
				go t.PosInterval(Param.PosInterval)
				wg.Done()
				_DefaultTermUpdater.Add(t)
			}
		}
		fmt.Println("当前终端数量:", len(_DefaultTermUpdater.List))
		wg.Wait()
	} else {
		t := NewTerminal(Param.RemoteIP, Param.RemotePort, Param.SimNum)
		if t != nil {
			go t.KeepConn()
		}
	}
	<-exit
}
