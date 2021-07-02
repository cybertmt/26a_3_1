package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)
// Размер буфера
var bufferSize = 2

// Врем очистки буфера
var bufferPush = 5 * time.Second

// Кольцевой буфер
type ringBuffer struct {
	array []int
	size  int
	position   int        // текущая позиция в кольце
	mu    sync.Mutex
}

// Конструктор буфера
func NewRingBuffer(size int) *ringBuffer {
	return &ringBuffer{make([]int, size), size, -1, sync.Mutex{}}
}

// Добавление элемента в буфер
func (r *ringBuffer) Push(item int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.position == r.size-1 {
		for i := 1; i <= r.size-1; i++ {
			r.array[i-1] = r.array[i]
		}
		r.array[r.position] = item
	} else {
		r.position++
		r.array[r.position] = item
	}
}

// Получение элементов из буфера
func (r *ringBuffer) Get() []int {
	if r.position < 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var item = r.array[:r.position+1]
	r.position = -1
	return item
}

// Стадия пайплайна
type stage func(<-chan bool, <-chan int) <-chan int

// Пайплайн
type pipeLine struct {
	stages []stage
	done   <-chan bool
}

// Конструктор пайплайна
func NewPipeline(done <-chan bool, stages ...stage) *pipeLine {
	return &pipeLine{done: done, stages: stages}
}

// Запуск пайплайна
func (p *pipeLine) Run(source <-chan int) <-chan int {
	var c = source
	for index := range p.stages {
		c = p.runStage(p.stages[index], c)
	}
	return c
}

// Запуск стадии пайплайна
func (p *pipeLine) runStage(stage stage, source <-chan int) <-chan int {
	return stage(p.done, source)
}

func main() {
	// Ввод данных пользователя
	inputList := func() (<-chan int, <-chan bool) {
		c := make(chan int)
		done := make(chan bool)
		go func() {
			defer close(done)
			scanner := bufio.NewScanner(os.Stdin)
			var text string
			for {
				scanner.Scan()
				text = scanner.Text()
				if strings.EqualFold(text, "quit") {
					fmt.Println("Bye")
					return
				}
				i, err := strconv.Atoi(text)
				if err != nil {
					fmt.Println("Need int")
					continue
				}
				c <- i
			}
		}()
		return c, done
	}
	// Фильтр отрицательных чисел
	filterNegative := func(done <-chan bool, c <-chan int) <-chan int {
		resultChan := make(chan int)
		go func() {
			for {
				select {
				case item := <-c:
					if item > 0 {
						resultChan <- item
					}
				case <-done:
					return
				}
			}
		}()
		return resultChan
	}
	// Фильтр чисел некратных 3
	filterDivision := func(done <-chan bool, c <-chan int) <-chan int {
		resultChan := make(chan int)
		go func() {
			for {
				select {
				case item := <-c:
					if item > 0 && item%3 == 0 {
						resultChan <- item
					}
				case <-done:
					return
				}
			}
		}()
		return resultChan
	}
	// Фильтр чисел больше 900
	filterOneMore := func(done <-chan bool, c <-chan int) <-chan int {
		resultChan := make(chan int)
		go func() {
			for {
				select {
				case item := <-c:
					if item < 900 {
						resultChan <- item
					}
				case <-done:
					return
				}
			}
		}()
		return resultChan
	}
	// Добавление числа в буфер
	buffering := func(done <-chan bool, c <-chan int) <-chan int {
		bufferChan := make(chan int)
		buffer := NewRingBuffer(bufferSize)
		go func() {
			for {
				select {
				case item := <-c:
					buffer.Push(item)
				case <-done:
					return
				}
			}
		}()

		// Чтение буфера
		go func() {
			for {
				select {
				case <-time.After(bufferPush):
					item := buffer.Get()
					if item != nil {
						for _, data := range item {
							bufferChan <- data
						}
					}
				case <-done:
					return
				}
			}
		}()
		return bufferChan
	}
	// Слушатель канала
	subs := func(done <-chan bool, c <-chan int) {
		for {
			select {
			case item := <-c:
				fmt.Printf("Get: %d\n", item)
				log.Printf("Get: %d\n", item)
			case <-done:
				return
			}
		}
	}

	// Запуск источника данных
	source, done := inputList()
	// Создаем пайплайн со стадиями
	pipeline := NewPipeline(done, filterNegative, filterDivision, filterOneMore, buffering)
	// Запуск слушателя
	subs(done, pipeline.Run(source))
}