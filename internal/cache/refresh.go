package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"fli/internal/aws"
)

// ENITagProvider defines an interface for fetching ENI tag information.
type ENITagProvider interface {
	GetENITag(ctx context.Context, eniID string) (aws.ENITag, error)
}

// RefreshENIs fetches tags for a list of ENIs from a provider and updates the cache.
func (c *Cache) RefreshENIs(ctx context.Context, eniProvider ENITagProvider, enis []string) error {
	for i, eni := range enis {
		log.Printf("Refreshing ENI %d/%d: %s", i+1, len(enis), eni)
		awsTag, err := eniProvider.GetENITag(ctx, eni)
		if err != nil {
			c.handleENIError(eni, err)
			continue
		}

		// Skip if the ENI tag is empty (ENI not found)
		if awsTag.ENI == "" {
			log.Printf("ENI %s not found, skipping", eni)
			continue
		}

		// Convert aws.ENITag to cache.ENITag
		cacheTag := ENITag{
			ENI:        awsTag.ENI,
			Label:      awsTag.Label,
			SGNames:    awsTag.SGNames,
			PrivateIPs: awsTag.PrivateIPs,
			FirstSeen:  time.Now().Unix(),
		}

		if err := c.UpsertEni(cacheTag); err != nil {
			log.Printf("Warning: failed to upsert ENI %s: %v", eni, err)
			continue
		}
		log.Printf("Tagged ENI %s: %s", eni, cacheTag.Label)
	}
	return nil
}

// handleENIError handles errors that occur when fetching ENI tags.
func (c *Cache) handleENIError(eni string, err error) {
	// Check if the ENI no longer exists
	if aws.IsENINotFoundError(err) {
		log.Printf("ENI %s no longer exists, removing from cache", eni)
		if deleteErr := c.DeleteENI(eni); deleteErr != nil {
			log.Printf("Warning: failed to remove ENI %s from cache: %v", eni, deleteErr)
		} else {
			log.Printf("Removed ENI %s from cache", eni)
		}
	} else {
		log.Printf("Warning: failed to tag ENI %s: %v", eni, err)
	}
}

// RefreshAllENIs fetches tags for all ENIs currently in the cache.
func (c *Cache) RefreshAllENIs(ctx context.Context, eniProvider ENITagProvider) error {
	enis, err := c.ListENIs()
	if err != nil {
		return fmt.Errorf("failed to list ENIs in cache: %w", err)
	}
	if len(enis) == 0 {
		log.Println("No ENIs found in cache to refresh.")
		return nil
	}
	return c.RefreshENIs(ctx, eniProvider, enis)
}
