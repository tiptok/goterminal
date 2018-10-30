package main

func main() {
	var exit chan int
	exit = make(chan int)
	t := NewTerminal("218.5.10.82", 38922)
	//t := NewTerminal("127.0.0.1", 38922)
	if t != nil {
		go t.KeepConn()
	}
	<-exit
}
