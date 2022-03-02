package main

import (
	"log"

	"github.com/Hawkbawk/falcon-proxy/src/syncer"
)

// If we encounter more than MaxRepeatedErrorCount errors repeatedly,
// we should stop syncing, cause we probably are just going to keep having an error.
const MaxRepeatedErrorCount = 10

func main() {
	s, err := syncer.NewSyncer()

	if err != nil {
		log.Fatalf("Unable to start syncing. ERROR: %v", err)
	}

	if err := s.Sync(); err != nil {
		log.Fatalf("Unable to perform an initial sync. Exiting. ERROR: %v", err)
	}
	repeatedErrorCount := 0
	for {
		<-s.EventChannel
		if err := s.Sync(); err != nil {
			log.Printf("Unable to perform a sync. ERROR: %v", err)
			repeatedErrorCount += 1
			if repeatedErrorCount > MaxRepeatedErrorCount {
				log.Fatalf("Stopping syncing as %v or more errors were encountered in a row.", MaxRepeatedErrorCount)
			}
		}
		repeatedErrorCount = 0
	}
}
