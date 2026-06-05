package retention

import (
	"fmt"
	"sort"
	"time"
)

type Policy struct {
	Hourly  int
	Daily   int
	Weekly  int
	Monthly int
}

type Backup struct {
	Name string
	Size int64
	Time time.Time
}

// Apply selects which backups to keep and which to prune using Proxmox-style
// generational retention. Backups are sorted newest-first; one representative
// per slot is kept for each generation bucket.
func Apply(backups []Backup, policy Policy) (keep, prune []Backup) {
	if len(backups) == 0 {
		return nil, nil
	}

	sorted := make([]Backup, len(backups))
	copy(sorted, backups)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Time.After(sorted[j].Time)
	})

	kept := map[string]bool{}

	selectSlots(sorted, kept, policy.Hourly, func(t time.Time) string {
		return t.UTC().Format("2006010215") // YYYYMMDDHH
	})
	selectSlots(sorted, kept, policy.Daily, func(t time.Time) string {
		return t.UTC().Format("20060102") // YYYYMMDD
	})
	selectSlots(sorted, kept, policy.Weekly, func(t time.Time) string {
		y, w := t.UTC().ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	})
	selectSlots(sorted, kept, policy.Monthly, func(t time.Time) string {
		return t.UTC().Format("200601") // YYYYMM
	})

	for _, b := range sorted {
		if kept[b.Name] {
			keep = append(keep, b)
		} else {
			prune = append(prune, b)
		}
	}
	return keep, prune
}

// selectSlots fills up to n slots using the given key function.
// Newest backups win each slot. Already-kept backups still consume a slot.
func selectSlots(sorted []Backup, kept map[string]bool, n int, key func(time.Time) string) {
	if n <= 0 {
		return
	}
	slots := map[string]bool{}
	for _, b := range sorted {
		if len(slots) >= n {
			break
		}
		k := key(b.Time)
		if !slots[k] {
			slots[k] = true
			kept[b.Name] = true
		}
	}
}
