package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	defer close(ch)
	var i int64 = 1
	for {
		select {
		case <-ctx.Done():
			return
		case ch <- i:
			fn(i)
			i++
		}
	}
}

// Worker читает число из канала in и пишет его в канал out
func Worker(in <-chan int64, out chan<- int64) {
	defer close(out)
	for {
		v, ok := <-in
		if !ok {
			return
		}
		out <- v
		time.Sleep(time.Millisecond)
	}
}

func main() {
	chIn := make(chan int64)

	// Создаем контекст с таймаутом 1 секунда
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Для потокобезопасного подсчета используем atomic
	var inputSum int64
	var inputCount int64

	// Генератор чисел
	go Generator(ctx, chIn, func(i int64) {
		atomic.AddInt64(&inputSum, i)
		atomic.AddInt64(&inputCount, 1)
	})

	const NumOut = 5
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	amounts := make([]int64, NumOut)
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	// Запускаем сборщиков из каналов outs
	for i := 0; i < NumOut; i++ {
		wg.Add(1)
		go func(in <-chan int64, idx int) {
			defer wg.Done()
			for v := range in {
				atomic.AddInt64(&amounts[idx], 1)
				chOut <- v
			}
		}(outs[i], i)
	}

	// Закрываем chOut после завершения всех горутин
	go func() {
		wg.Wait()
		close(chOut)
	}()

	var count int64
	var sum int64

	// Считаем сумму и количество из результирующего канала
	for v := range chOut {
		sum += v
		count++
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	// Проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	
	var total int64
	for _, v := range amounts {
		total += v
	}
	if total != inputCount {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
