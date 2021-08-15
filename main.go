package main

import (
	"log"
	"time"

	"github.com/Hawkbawk/falcon-proxy/src/syncer"
)

// If we encounter more than maxRepeatedErrorCount errors repeatedly,
// we should stop syncing, cause we probably are just going to keep having an error.
const maxRepeatedErrorCount = 10

func main() {
	s := syncer.NewSyncer()
	repeatedErrorCount := 0
	for {
		<-s.EventChannel
		if err := s.Sync(); err != nil {
			log.Printf("%v ### Unable to perform a sync. ERROR: %v", time.Now(), err)
			repeatedErrorCount++
			if repeatedErrorCount > maxRepeatedErrorCount {
				log.Fatalf("Stopping syncing as %v or more errors were encountered in a row.", maxRepeatedErrorCount)
			}
		}
		repeatedErrorCount = 0
	}
}