package retention_test

import (
	"testing"
	"time"

	"github.com/gustavogebhardt/mc-backup/internal/retention"
)

// hour returns a time at a specific hour on 2026-01-01.
func hour(h int) time.Time {
	return time.Date(2026, 1, 1, h, 0, 0, 0, time.UTC)
}

// day returns noon on a specific day of January 2026.
func day(d int) time.Time {
	return time.Date(2026, 1, d, 12, 0, 0, 0, time.UTC)
}

// week returns noon on a specific day across multiple weeks.
func week(w, d int) time.Time {
	return time.Date(2026, 1, w*7+d, 12, 0, 0, 0, time.UTC)
}

func TestApply_EmptyList(t *testing.T) {
	policy := retention.Policy{Hourly: 24, Daily: 7, Weekly: 4, Monthly: 12}
	keep, prune := retention.Apply(nil, policy)
	if len(keep) != 0 {
		t.Errorf("expected 0 kept, got %d", len(keep))
	}
	if len(prune) != 0 {
		t.Errorf("expected 0 pruned, got %d", len(prune))
	}
}

func TestApply_FewerBackupsThanSlots(t *testing.T) {
	backups := []retention.Backup{
		{Name: "mc_backup_20260101_120000.tar.gz", Time: day(1)},
		{Name: "mc_backup_20260102_120000.tar.gz", Time: day(2)},
	}
	policy := retention.Policy{Hourly: 24, Daily: 7, Weekly: 4, Monthly: 12}
	keep, prune := retention.Apply(backups, policy)

	if len(keep) != 2 {
		t.Errorf("expected 2 kept, got %d", len(keep))
	}
	if len(prune) != 0 {
		t.Errorf("expected 0 pruned, got %d", len(prune))
	}
}

func TestApply_HourlyRetention(t *testing.T) {
	// 30 backups each one hour apart (unique hourly slots), policy keeps only 24
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var backups []retention.Backup
	for i := 0; i < 30; i++ {
		t_ := base.Add(time.Duration(i) * time.Hour)
		backups = append(backups, retention.Backup{
			Name: t_.Format("mc_backup_20060102_150405.tar.gz"),
			Time: t_,
		})
	}
	policy := retention.Policy{Hourly: 24, Daily: 0, Weekly: 0, Monthly: 0}
	keep, prune := retention.Apply(backups, policy)

	if len(keep)+len(prune) != 30 {
		t.Errorf("total should equal input: got keep=%d prune=%d", len(keep), len(prune))
	}
	if len(keep) > 24 {
		t.Errorf("should keep at most 24, got %d", len(keep))
	}
	if len(prune) < 6 {
		t.Errorf("should prune at least 6, got %d", len(prune))
	}
}

func TestApply_MostRecentIsAlwaysKept(t *testing.T) {
	backups := []retention.Backup{
		{Name: "old.tar.gz", Time: day(1)},
		{Name: "newest.tar.gz", Time: day(30)},
	}
	policy := retention.Policy{Hourly: 1, Daily: 0, Weekly: 0, Monthly: 0}
	keep, _ := retention.Apply(backups, policy)

	kept := map[string]bool{}
	for _, b := range keep {
		kept[b.Name] = true
	}
	if !kept["newest.tar.gz"] {
		t.Error("most recent backup should always be kept")
	}
}

func TestApply_GenerationalRetention(t *testing.T) {
	// Simulate 60 days of hourly backups
	var backups []retention.Backup
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60*24; i++ {
		t_ := base.Add(time.Duration(i) * time.Hour)
		backups = append(backups, retention.Backup{
			Name: t_.Format("mc_backup_20060102_150405.tar.gz"),
			Time: t_,
		})
	}

	policy := retention.Policy{Hourly: 24, Daily: 7, Weekly: 4, Monthly: 2}
	keep, prune := retention.Apply(backups, policy)

	total := len(keep) + len(prune)
	if total != len(backups) {
		t.Errorf("keep+prune (%d) != input (%d)", total, len(backups))
	}

	// Every kept backup must come from the input
	inputNames := map[string]bool{}
	for _, b := range backups {
		inputNames[b.Name] = true
	}
	for _, b := range keep {
		if !inputNames[b.Name] {
			t.Errorf("kept backup %q not in input", b.Name)
		}
	}

	// Kept count should not exceed theoretical max slots
	maxSlots := 24 + 7 + 4 + 2
	if len(keep) > maxSlots {
		t.Errorf("kept %d, expected at most %d slots", len(keep), maxSlots)
	}
}

func TestApply_NoDuplicatesInKeepAndPrune(t *testing.T) {
	var backups []retention.Backup
	for i := 0; i < 100; i++ {
		t_ := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Hour)
		backups = append(backups, retention.Backup{
			Name: t_.Format("mc_backup_20060102_150405.tar.gz"),
			Time: t_,
		})
	}
	policy := retention.Policy{Hourly: 24, Daily: 7, Weekly: 4, Monthly: 12}
	keep, prune := retention.Apply(backups, policy)

	seen := map[string]bool{}
	for _, b := range keep {
		if seen[b.Name] {
			t.Errorf("duplicate in keep: %q", b.Name)
		}
		seen[b.Name] = true
	}
	for _, b := range prune {
		if seen[b.Name] {
			t.Errorf("%q appears in both keep and prune", b.Name)
		}
		seen[b.Name] = true
	}
}

// Silence unused import warnings for helper functions.
var _ = hour
var _ = week
