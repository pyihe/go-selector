package main

import (
	"fmt"
	"time"

	"github.com/pyihe/go-selector"
)

func main() {
	event := selector.NewEvent(selector.ModeUnited)
	event.AddHandler(selector.StatusAgree, func() {
		fmt.Println("agree")
	})
	event.AddHandler(selector.StatusRefuse, func() {
		fmt.Println("refuse")
	})
	event.AddHandler(selector.StatusTimeout, func() {
		fmt.Println("timeout")
	})

	a1 := event.AddAction(1)
	a2 := event.AddAction(2)
	a3 := event.AddAction(3)
	event.Start(3 * time.Second)
	go func() {
		a1.Agree()
	}()
	go func() {
		_ = a2
	}()
	go func() {
		_ = a3
	}()

}
