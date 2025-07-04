package formatter

import (
	"context"
	"fmt"
	"net/netip"

	"fli/internal/cache"
	"fli/internal/runner"
)

const (
	fieldInterfaceID = "interface_id"
	fieldSrcAddr     = "srcaddr"
	fieldDstAddr     = "dstaddr"
)

// EnrichResultsWithAnnotations adds ENI and IP annotations to the results.
func EnrichResultsWithAnnotations(results [][]runner.Field, cachePath string) ([][]runner.Field, error) {
	if len(results) == 0 {
		return results, nil
	}

	cache, err := cache.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache for annotations: %w", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			// Log the close error but continue; this is a non-critical error in annotation enrichment
			fmt.Printf("Warning: failed to close cache: %v\n", closeErr)
		}
	}()

	enriched := make([][]runner.Field, len(results))
	for i, row := range results {
		newRow := make([]runner.Field, len(row))
		copy(newRow, row)

		for _, field := range row {
			var anno *runner.Field

			switch field.Name {
			case fieldInterfaceID:
				if tag, _ := cache.LookupEni(context.Background(), field.Value); tag != nil {
					anno = &runner.Field{Name: field.Name + "_annotation", Value: tag.Label}
				}
			case fieldSrcAddr, fieldDstAddr:
				if addr, err := netip.ParseAddr(field.Value); err == nil {
					if annotation, err := cache.LookupIP(addr); err == nil && annotation != "" {
						anno = &runner.Field{Name: field.Name + "_annotation", Value: annotation}
					}
				}
			}

			if anno != nil {
				newRow = append(newRow, *anno)
			}
		}
		enriched[i] = newRow
	}

	return enriched, nil
}
