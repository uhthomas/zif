package util

import "time"

type Limiter struct {
	Throttle chan time.Time
	Ticker   *time.Ticker
}

// Return a new rate limiter. This is used to make sure that something like a
// requrest for instance does not run too many times. However, it does allow
// bursting. For example, it may refill at a rate of 3 tokens per minute, and
// have a burst of three. This means that if it has been running for more than
// a minute without being used then, it will be able to be used 3 times in
// rapid succession - no limiting will apply.
func NewLimiter(rate time.Duration, burst int, fill bool) *Limiter {
	tick := time.NewTicker(rate)
	throttle := make(chan time.Time, burst)

	if fill {
		for i := 0; i < burst; i++ {
			throttle <- time.Now()
		}
	}

	go func() {
		for t := range tick.C {
			select {
			case throttle <- t:
			default:
			}
		}
	}()

	return &Limiter{throttle, tick}
}

// Block until the given time has elapsed. Or just use a token from the bucket.
func (l *Limiter) Wait() {
	_, _ = <-l.Throttle
}

// Finish running.
func (l *Limiter) Stop() {
	l.Ticker.Stop()
	close(l.Throttle)
}

// Limits requests from peers
type PeerLimiter struct {
	queryLimiter    *Limiter
	announceLimiter *Limiter
}

func (pl *PeerLimiter) Setup() {
	// Allow an announce every 10 minutes, bursting to allow three.
	// The burst is there as people may make "mistakes" with titles or descriptions
	pl.announceLimiter = NewLimiter(time.Minute*10, 3, true)

	pl.queryLimiter = NewLimiter(time.Second/3, 3, true)
}
