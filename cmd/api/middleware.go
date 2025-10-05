package main 

import (
	"fmt"
	// "net"
	"net/http"
	"sync"
	"time"

	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		defer func () {
			pv := recover()
			if pv != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w,r, fmt.Errorf("%v", pv))
			}
		}()
		
		next.ServeHTTP(w,r)
	})
}


func (app *application) rateLimit(next http.Handler) http.Handler { 
	if !app.config.limiter.enabled {
		return next
	}
	
	type client struct {
		limiter 	*rate.Limiter
		lastSeen	time.Time
	}
	
	var (
		mtx		sync.Mutex
		clients = make(map[string]*client)
	)

	go func () {
		for {
			time.Sleep(time.Minute)

			mtx.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3 * time.Minute {
					delete(clients, ip)
				}
			}

			mtx.Unlock()
		}
	}()


	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realip.FromRequest(r)

		mtx.Lock()

		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
		}

		if !clients[ip].limiter.Allow() {
			mtx.Unlock()
			app.rateLimitExceededResponse(w,r)
			return
		}

		mtx.Unlock()

		next.ServeHTTP(w, r)
	})
}