package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	RPS    int
	seq    []time.Time
	mtxSeq sync.Mutex
	ix     int
}

//func NewRT() *RateLimiter {
//}

// Вариант решения на массиве (слайсе) с len = RPC.
// Можно обрабатывать очередной запрос, если с момента выполнения RPC-го запроса назад от текущего прошло более 1 сек.
func (rl *RateLimiter) checkLimit() bool {
	rl.mtxSeq.Lock()
	defer rl.mtxSeq.Unlock()

	ts := time.Now()
	// пока len слайса меньше RPC просто добавляем записи и разрешаем выполнение запроса
	if len(rl.seq) < rl.RPS {
		rl.seq = append(rl.seq, ts)
		rl.ix++
		return true
	}

	// при достижении конца слайса с len = RPC возвращаемся к началу (перезаписываем значения)
	if rl.ix > (rl.RPS - 1) {
		rl.ix = 0
	}

	duration := ts.Sub(rl.seq[rl.ix])
	// если RPC запросов выполнены за < 1 сек - отклоняем текущий запрос
	if duration < time.Second*1 {
		return false
	}

	// разрешаем выполнение запроса, фиксируем время его поступления
	rl.seq[rl.ix] = ts
	rl.ix++
	return true
}

func (rl *RateLimiter) Middleware(h func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	// do request here
	return func(w http.ResponseWriter, r *http.Request) {
		// если не превышен RPC - разрешаем обработку запроса
		if rl.checkLimit() {
			handler(w, r)
			return
		}
		// иначе возвращаем ошибку 429
		http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		fmt.Println("REQUEST NOT PROCESSED")
	}
}

// rps 1 - 1
// 0 ... 1 ... 2
//   0.9..1.1

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("REQUEST COMPLETED")
}

func main() {
	rl := &RateLimiter{RPS: 100}

	http.HandleFunc("/", rl.Middleware(handler))

	go func() {
		for i := 0; i < 1000; i++ {
			time.Sleep(10 * time.Millisecond)
			if _, err := http.Get("http://localhost:8080/"); err != nil {
				log.Fatal(err)
			}
		}
	}()
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}
