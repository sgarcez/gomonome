package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sgarcez/gomonome/monome"
)

func gridDemo(port int32) *monome.Grid {
	g, err := monome.StartGrid(port)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			e := g.Read()
			if e == nil {
				break
			}
			go func(e monome.DeviceEvent) {
				switch e.(type) {
				case monome.ReadyEvent:
					fmt.Println("connected, testing /led/all")
					g.LedAll(true)
					time.Sleep(time.Second / 2)
					g.LedAll(false)
				case monome.KeyEvent:
					fmt.Println(e)
					ke := e.(monome.KeyEvent)
					if ke.S == int32(1) {
						g.LedSet(ke.X, ke.Y, ke.S)
					}
				case monome.TiltEvent:
					te := e.(monome.TiltEvent)
					fmt.Println(te.N)
					fmt.Println(te.X)
					fmt.Println(te.Y)
					fmt.Println(te.Z)
				}
			}(e)
		}
	}()

	return g
}

func arcDemo(port int32) *monome.Arc {
	a, err := monome.StartArc(port)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			e := a.Read()
			if e == nil {
				break
			}
			go func(e monome.DeviceEvent) {
				switch e.(type) {
				case monome.ReadyEvent:
					fmt.Println("connected, testing /ring/all")
					a.RingAll(0, 15)
					a.RingAll(1, 15)
					time.Sleep(time.Second / 2)
					a.RingAll(0, 0)
					a.RingAll(1, 0)
				case monome.RingPressEvent:
					fmt.Println(e)
					if e.(monome.RingPressEvent).S == 1 {
						a.RingAll(e.(monome.RingPressEvent).E, 15)
					} else {
						a.RingAll(e.(monome.RingPressEvent).E, 0)
					}
				}
			}(e)
		}
	}()

	return a

}

func main() {
	s, err := monome.StartSerialOSC()
	if err != nil {
		panic(err)
	}

	// Read 2 list events and close control plane
	for i := 0; i < 2; i++ {
		e, err := s.Read()
		if err != nil {
			panic(err)
		}
		fmt.Println(e, e.DeviceKind())

		switch e.DeviceKind() {
		case "grid":
			defer gridDemo(e.Port).Close()
		case "arc":
			defer arcDemo(e.Port).Close()
		}
	}
	s.Close()
	waitOnSignal()
}

func waitOnSignal() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	<-done
	fmt.Println("exiting")
}
