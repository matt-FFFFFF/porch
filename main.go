package main

import (
	"context"
	"os"
	"os/signal"
	"time"
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer close(signalChan)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		// Context done, exit
		return
	case <-signalChan:
		signal.Stop(signalChan)
	}

}
