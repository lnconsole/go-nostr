package nostr

import (
	"context"
	"log"
	"time"
)

type PersistentRelay struct {
	*Relay

	PersistentEvents chan *Event
	filters          []Filters
}

func PersistentRelayConnect(ctx context.Context, url string, opts ...RelayOption) (*PersistentRelay, error) {
	resubscribeFilters := make(chan Filters)
	r := NewRelay(context.Background(), url, resubscribeFilters, opts...)
	err := r.Connect(ctx)
	if err != nil {
		return nil, err
	}

	p := &PersistentRelay{
		Relay:            r,
		PersistentEvents: make(chan *Event),
	}

	go func() {
		// store contents of resubscribeFilters channel
		// so that we can resubscribe to them when the connection breaks
		for {
			select {
			case <-ctx.Done():
				return
			case f := <-resubscribeFilters:
				p.filters = append(p.filters, f)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * 1)

			if p.ConnectionError == nil {
				continue
			}

			ctx := context.Background()
			// try to reconnect by creating a new relay
			recon := NewRelay(ctx, url, resubscribeFilters, opts...)
			err := recon.Connect(ctx)
			if err != nil {
				log.Printf("reconnect failed: %s", err)
				continue
			}

			log.Printf("reconnected to: %s", url)

			// subscribe to the same subscriptions on the old relay
			for _, filters := range p.filters {
				s, err := recon.Subscribe(ctx, filters)
				if err != nil {
					log.Printf("subscribe failed: %s", err)
					continue
				}
				// redirect events to the persistent events channel
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case e := <-s.Events:
							p.PersistentEvents <- e
						}
					}
				}()
			}
			p.filters = []Filters{}
			// swap out the old relay with the new one
			p.Relay = recon
		}
	}()

	return p, nil
}
