package backup

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// RunDaily starts a daily backup loop: at the configured local time, copy sourcePath
// to backupDir/YYYY-MM-DD.csv and maintain retentionDays worth of backups.
func RunDaily(ctx context.Context, sourcePath string, backupDir string, timeOfDay string, tz string, retentionDays int, logger *log.Logger) {
	if logger == nil {
		logger = log.Default()
	}

	loc := time.Local
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		} else {
			logger.Printf("backup: failed to load timezone %q, using local: %v", tz, err)
		}
	}

	h, m, err := parseHHMM(timeOfDay)
	if err != nil {
		logger.Printf("backup: invalid BACKUP_TIME %q, defaulting 03:00: %v", timeOfDay, err)
		h, m = 3, 0
	}

	ensureDir(backupDir, logger)

	// Run immediately on start to ensure at least one backup exists
	doBackup(sourcePath, backupDir, retentionDays, loc, logger)

	for {
		next := nextAtTime(time.Now().In(loc), h, m)
		d := time.Until(next)
		timer := time.NewTimer(d)
		logger.Printf("backup: next run at %s (%s)", next.Format(time.RFC3339), loc.String())

		select {
		case <-ctx.Done():
			timer.Stop()
			logger.Printf("backup: stopping: %v", ctx.Err())
			return
		case <-timer.C:
			doBackup(sourcePath, backupDir, retentionDays, loc, logger)
		}
	}
}

func parseHHMM(s string) (int, int, error) {
	if s == "" {
		return 3, 0, nil
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, err
	}
	return t.Hour(), t.Minute(), nil
}

func nextAtTime(now time.Time, hour, minute int) time.Time {
	n := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !n.After(now) {
		n = n.AddDate(0, 0, 1)
	}
	return n
}

func ensureDir(dir string, logger *log.Logger) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logger.Printf("backup: failed to create dir %s: %v", dir, err)
	}
}

func doBackup(sourcePath, backupDir string, retentionDays int, loc *time.Location, logger *log.Logger) {
	// Use date in the chosen timezone
	today := time.Now().In(loc).Format("2006-01-02")
	dst := filepath.Join(backupDir, fmt.Sprintf("%s.csv", today))
	tmp := dst + ".tmp"

	if err := copyFileAtomic(sourcePath, tmp, dst); err != nil {
		logger.Printf("backup: failed to copy %s -> %s: %v", sourcePath, dst, err)
		return
	}
	logger.Printf("backup: wrote %s", dst)

	// Also update latest.csv symlink or copy
	latest := filepath.Join(backupDir, "latest.csv")
	_ = os.Remove(latest)
	// symlink might not be supported on all FS; fallback to copy
	if err := os.Symlink(dst, latest); err != nil {
		// fallback: copy
		_ = copyFile(dst, latest)
	}

	if retentionDays > 0 {
		enforceRetention(backupDir, retentionDays, logger)
	}
}

func copyFileAtomic(src, tmp, final string) error {
	if err := copyFile(src, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, final)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

var dateFileRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\.csv$`)

func enforceRetention(backupDir string, retentionDays int, logger *log.Logger) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		logger.Printf("backup: retention list failed: %v", err)
		return
	}

	type item struct {
		name string
		date time.Time
	}
	var files []item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := dateFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		if d, err := time.Parse("2006-01-02", m[1]); err == nil {
			files = append(files, item{name: e.Name(), date: d})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].date.Before(files[j].date) })

	// Keep last N days; delete older
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for _, f := range files {
		if f.date.Before(cutoff) {
			path := filepath.Join(backupDir, f.name)
			if err := os.Remove(path); err != nil {
				logger.Printf("backup: failed to remove old %s: %v", path, err)
			} else {
				logger.Printf("backup: removed old %s", path)
			}
		}
	}
}
