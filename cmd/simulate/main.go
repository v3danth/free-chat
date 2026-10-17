// Command simulate fills a local Drift server with scripted guests, so a
// person testing in a browser meets every chat state from both sides:
// knocks in and out, replies, declines, the reveal ladder, reloads and
// people leaving and coming back. Every refusal the server sends a bot is
// logged; an unexpected one is a bug worth reading.
//
//	go run ./cmd/simulate -bots 6 -target "Your Name"
package main

import (
	"context"
	"flag"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {
	server := flag.String("url", "http://localhost:8080", "server to fill")
	bots := flag.Int("bots", 6, "how many guests to run (the server allows 30 sign-ups per 10 minutes per IP)")
	target := flag.String("target", "", "name of the person testing; bots knock on them")
	runFor := flag.Duration("for", 15*time.Minute, "how long to run")
	pace := flag.Duration("pace", 6*time.Second, "average gap between a bot's own moves")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if *bots < 1 || *bots > len(personas) {
		slog.Error("bots out of range", "min", 1, "max", len(personas))
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *runFor)
	defer cancel()

	cfg := config{server: *server, target: *target, pace: *pace}
	tally := newTally()
	picks := rand.Perm(len(personas))[:*bots]
	slog.Info("simulation started", "bots", *bots, "target", *target, "for", *runFor)

	var wg sync.WaitGroup
	for i, p := range picks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Staggered arrivals look like a room filling up, and keep the
			// sign-up limiter from seeing a burst.
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(i) * 1500 * time.Millisecond):
			}
			b, err := newBot(ctx, cfg, personas[p], tally, uint64(i))
			if err != nil {
				slog.Error("bot could not start", "bot", personas[p].Name, "err", err)
				return
			}
			b.run(ctx)
		}()
	}
	wg.Wait()
	tally.log()
}
