package cache

import (
	"bytes"
	"context"
	"fmt"
	"net/netip"
	"strings"
)

// List returns a formatted string of all items in the cache.
func (c *Cache) List(ctx context.Context) (string, error) {
	var buf bytes.Buffer

	// List ENIs
	enis, err := c.ListENIs()
	if err != nil {
		return "", fmt.Errorf("failed to list ENIs: %w", err)
	}
	buf.WriteString("ENIs:\n")
	for _, eni := range enis {
		tag, err := c.LookupEni(ctx, eni)
		if err != nil {
			// Log or handle error, but continue for now
			continue
		}
		buf.WriteString(fmt.Sprintf("  %s%s\n", eni, formatENITag(tag)))
	}
	buf.WriteString("\n")

	// List IPs
	ips, err := c.ListIPs()
	if err != nil {
		return "", fmt.Errorf("failed to list IPs: %w", err)
	}
	buf.WriteString("IPs:\n")
	for _, ip := range ips {
		addr, err := netip.ParseAddr(ip)
		if err != nil {
			// Should not happen if cache is valid
			continue
		}
		annotation, err := c.LookupIP(addr)
		if err != nil {
			// Log or handle error, but continue for now
			continue
		}
		buf.WriteString(fmt.Sprintf("  %s %s\n", ip, formatIPAnnotation(annotation)))
	}
	buf.WriteString("\n")

	// List Prefixes
	prefixes, err := c.ListPrefixes()
	if err != nil {
		return "", fmt.Errorf("failed to list prefixes: %w", err)
	}
	buf.WriteString("Prefixes:\n")
	for _, prefix := range prefixes {
		prefixAddr, err := netip.ParsePrefix(prefix)
		if err != nil {
			buf.WriteString(fmt.Sprintf("  %s (invalid prefix)\n", prefix))
			continue
		}
		// Use the first IP in the prefix for lookup.
		annotation, err := c.LookupIP(prefixAddr.Addr())
		if err != nil {
			// Log or handle error, but continue for now
			continue
		}
		buf.WriteString(fmt.Sprintf("  %s %s\n", prefix, formatIPAnnotation(annotation)))
	}

	return buf.String(), nil
}

// formatIPAnnotation wraps the raw annotation string in parentheses if it's not empty.
func formatIPAnnotation(annotation string) string {
	if annotation == "" {
		return ""
	}
	return fmt.Sprintf("(%s)", annotation)
}

// formatENITag formats an ENITag into a readable string for display.
func formatENITag(tag *ENITag) string {
	if tag == nil {
		return ""
	}
	var parts []string
	if tag.Label != "" {
		parts = append(parts, tag.Label)
	}
	if len(tag.SGNames) > 0 {
		parts = append(parts, fmt.Sprintf("SGs: %v", tag.SGNames))
	}
	if len(tag.PrivateIPs) > 0 {
		parts = append(parts, fmt.Sprintf("IPs: %v", tag.PrivateIPs))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
}
