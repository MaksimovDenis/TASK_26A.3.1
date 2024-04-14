package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	bufferSize          int           = 5
	bufferDrainInterval time.Duration = 5 * time.Second
)

type IntRing struct {
	array []int
	pos   int
	size  int
	m     sync.Mutex
}

func NewIntRing(size int) *IntRing {
	return &IntRing{make([]int, size), -1, size, sync.Mutex{}}
}

func (ir *IntRing) Push(el int) {
	ir.m.Lock()
	defer ir.m.Unlock()
	if ir.pos == ir.size-1 {
		for i := 1; i <= ir.size-1; i++ {
			ir.array[i-1] = ir.array[i]
		}
		ir.array[ir.pos] = el
	} else {
		ir.pos++
		ir.array[ir.pos] = el
	}
}

func (ir *IntRing) Get() []int {
	if ir.pos < 0 {
		return nil
	}
	ir.m.Lock()
	defer ir.m.Unlock()
	var output []int = ir.array[:ir.pos+1]
	ir.pos = -1
	return output
}

type StageInt func(<-chan bool, <-chan int) <-chan int

type PipeLineInt struct {
	stages []StageInt
	done   <-chan bool
}

func NewPipelineInt(done <-chan bool, stages ...StageInt) *PipeLineInt {
	return &PipeLineInt{done: done, stages: stages}
}

func (p *PipeLineInt) Run(source <-chan int) <-chan int {
	var c <-chan int = source
	for index := range p.stages {
		c = p.runStageInt(p.stages[index], c)
	}
	return c
}

func (p *PipeLineInt) runStageInt(stage StageInt, sourceChan <-chan int) <-chan int {
	return stage(p.done, sourceChan)
}

func main() {

	dataSource := func() (<-chan int, <-chan bool) {
		c := make(chan int)
		done := make(chan bool)
		go func() {
			defer close(done)
			scanner := bufio.NewScanner(os.Stdin)
			var data string
			for {
				scanner.Scan()
				data = scanner.Text()
				if strings.EqualFold(data, "exit") {
					fmt.Println("Программа завершила работу")
					return
				}
				i, err := strconv.Atoi(data)
				if err != nil {
					fmt.Println("Пожалуйста, введите целое число")
					continue
				}
				c <- i
			}
		}()
		return c, done
	}

	stageFilterFirst := func(done <-chan bool, c <-chan int) <-chan int {
		numberStreamFirst := make(chan int)
		go func() {
			for {
				select {
				case data := <-c:
					if data > 0 {
						select {
						case numberStreamFirst <- data:
						case <-done:
							return
						}
					}
				case <-done:
					return
				}
			}
		}()
		return numberStreamFirst
	}

	stageFilterSecond := func(done <-chan bool, c <-chan int) <-chan int {
		numberStreamSecond := make(chan int)
		go func() {
			for {
				select {
				case data := <-c:
					if data != 0 && data%3 == 0 {
						select {
						case numberStreamSecond <- data:
						case <-done:
							return
						}
					}
				case <-done:
					return
				}
			}
		}()
		return numberStreamSecond
	}

	bufferStageInt := func(done <-chan bool, c <-chan int) <-chan int {
		bufferedIntChan := make(chan int)
		buffer := NewIntRing(bufferSize)
		go func() {
			for {
				select {
				case data := <-c:
					buffer.Push(data)
				case <-done:
					return
				}
			}
		}()

		go func() {
			for {
				select {
				case <-time.After(bufferDrainInterval):
					bufferData := buffer.Get()
					if bufferData != nil {
						for _, data := range bufferData {
							select {
							case bufferedIntChan <- data:
							case <-done:
								return
							}
						}
					}
				case <-done:
					return
				}
			}
		}()
		return bufferedIntChan
	}

	consumer := func(done <-chan bool, c <-chan int) {
		for {
			select {
			case data := <-c:
				fmt.Printf("Получены данные: %d\n", data)
			case <-done:
				return
			}
		}
	}

	source, done := dataSource()

	pipeline := NewPipelineInt(done, stageFilterFirst, stageFilterSecond, bufferStageInt)
	consumer(done, pipeline.Run(source))

}
