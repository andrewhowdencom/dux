package enrich

import (
	"fmt"

	"github.com/andrewhowdencom/dux/internal/config"
)

// NewFromConfig builds an array of enrichers from raw agent configuration.
func NewFromConfig(cfgs []config.Enricher) ([]Enricher, error) {
	var results []Enricher

	for _, c := range cfgs {
		switch c.Type {
		case "time":
			results = append(results, &timeEnricher{})
		case "os":
			results = append(results, &osEnricher{})
		case "prompt":
			results = append(results, &promptEnricher{text: c.Text})
		default:
			// For unrecognized types, we could either error out, or just log/skip.
			// Returning an error ensures configuration typos are caught.
			return nil, fmt.Errorf("unknown enricher type: %s", c.Type)
		}
	}

	return results, nil
}
