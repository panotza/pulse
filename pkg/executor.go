package pkg

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type executor struct {
	wg            *sync.WaitGroup
	builder       *builder
	runner        *runner
	waitCancel    chan struct{}
	cancelRunning context.CancelFunc
}

func NewExecutor(wg *sync.WaitGroup, builder *builder, runner *runner) *executor {
	return &executor{
		wg:      wg,
		builder: builder,
		runner:  runner,
	}
}

func (e *executor) Do(ctx context.Context, _ string) {
	if e.cancelRunning != nil {
		e.cancelRunning()
		<-e.waitCancel
	}

	e.wg.Add(1)
	e.waitCancel = make(chan struct{})
	ctx, e.cancelRunning = context.WithCancel(ctx)

	go func() {
		defer func() {
			close(e.waitCancel)
			e.wg.Done()
		}()

		log.Print("Building...")
		err := e.builder.Build(ctx)
		if err != nil {
			log.Print("Build failed")
			fmt.Println(err)
			return
		}
		log.Print("Build finished")

		err = e.runner.Run(ctx)
		if err != nil {
			log.Print(err)
		}
	}()
}

func NewPrintStdoutExecutor() *printStdoutExecutor {
	return &printStdoutExecutor{}
}

type printStdoutExecutor struct{}

func (e *printStdoutExecutor) Do(ctx context.Context, file string) {
	fmt.Printf("[event]: %s\n", file)
}
