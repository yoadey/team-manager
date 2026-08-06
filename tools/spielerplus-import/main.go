// Command spielerplus-import migrates a club's members, events, attendance,
// and planned absences from SpielerPlus into Teamverwaltung. See README.md
// for setup and openspec/changes/spielerplus-data-migration/ (in the main
// repo) for the full design.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/yoadey/team-manager/tools/spielerplus-import/db"
	"github.com/yoadey/team-manager/tools/spielerplus-import/importrun"
	"github.com/yoadey/team-manager/tools/spielerplus-import/mapping"
	"github.com/yoadey/team-manager/tools/spielerplus-import/spielerplus"
	"github.com/yoadey/team-manager/tools/spielerplus-import/storage"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "scrape and validate everything, but write nothing to the database")
	flag.Parse()

	if err := run(*dryRun); err != nil {
		log.Fatalf("spielerplus-import: %v", err)
	}
}

func run(dryRun bool) error {
	cfg, err := loadConfig(dryRun)
	if err != nil {
		return err
	}

	roleMapping, err := mapping.LoadRoleMapping(cfg.RoleMappingPath)
	if err != nil {
		return err
	}
	state, err := mapping.LoadState(cfg.StatePath)
	if err != nil {
		return err
	}

	sp, err := spielerplus.NewClient(cfg.SpielerPlusSessionCookie, spielerplus.WithRequestDelay(cfg.RequestDelay))
	if err != nil {
		return err
	}

	activeTeam, err := sp.FetchActiveTeamName()
	if err != nil {
		return err
	}
	if err := confirmTeam(activeTeam, cfg.ExpectedTeamName, os.Stdin, os.Stdout); err != nil {
		return err
	}

	ctx := context.Background()
	store, err := db.Open(ctx, cfg.DatabaseURL, cfg.DryRun)
	if err != nil {
		return err
	}
	defer store.Close()

	if cfg.DryRun {
		log.Println("dry run: no database writes will be made")
	}
	log.Printf("throttling requests to SpielerPlus to at least %s apart", cfg.RequestDelay)

	var photoStore importrun.PhotoUploader
	if cfg.PhotoImportEnabled() {
		s3, err := storage.New(storage.Config{
			Endpoint:        cfg.S3Endpoint,
			Region:          cfg.S3Region,
			Bucket:          cfg.S3Bucket,
			AccessKeyID:     cfg.S3AccessKeyID,
			SecretAccessKey: cfg.S3SecretAccessKey,
			UsePathStyle:    cfg.S3UsePathStyle,
		})
		if err != nil {
			return fmt.Errorf("connect to object store for photo import: %w", err)
		}
		photoStore = s3
	} else {
		log.Println("S3_ENDPOINT/S3_BUCKET not set: skipping member photo import")
	}

	summary, err := importrun.Run(ctx, sp, store, importrun.Options{
		TeamID:      cfg.TeamID,
		RoleMapping: roleMapping,
		State:       state,
		Now:         time.Now(),
		PhotoStore:  photoStore,
	})
	printSummary(os.Stdout, summary, cfg.DryRun)
	if err != nil {
		return err
	}
	return nil
}

func printSummary(w *os.File, s *importrun.Summary, dryRun bool) {
	if s == nil {
		return
	}
	verb := "created"
	if dryRun {
		verb = "would create"
	}
	fmt.Fprintf(w, "\n--- spielerplus-import summary (dry-run=%v) ---\n", dryRun)
	fmt.Fprintf(w, "members: %d %s, %d already existed\n", s.MembersCreated, verb, s.MembersExisting)
	fmt.Fprintf(w, "events: %d %s, %d already imported (skipped)\n", s.EventsCreated, verb, s.EventsSkipped)
	attendanceVerb := "written"
	if dryRun {
		attendanceVerb = "would be written"
	}
	fmt.Fprintf(w, "attendance: %d %s, %d skipped\n", s.AttendanceWritten, attendanceVerb, s.AttendanceSkipped)
	fmt.Fprintf(w, "absences: %d %s, %d skipped\n", s.AbsencesCreated, verb, s.AbsencesSkipped)
	fmt.Fprintf(w, "transactions: %d %s, %d skipped\n", s.TransactionsCreated, verb, s.TransactionsSkipped)
	fmt.Fprintf(w, "dues: %d %s, %d skipped\n", s.DuesCreated, verb, s.DuesSkipped)
	fmt.Fprintf(w, "penalties: %d %s, %d skipped\n", s.PenaltiesCreated, verb, s.PenaltiesSkipped)
	uploadVerb := "uploaded"
	if dryRun {
		uploadVerb = "would be uploaded"
	}
	fmt.Fprintf(w, "photos: %d %s, %d skipped\n", s.PhotosUploaded, uploadVerb, s.PhotosSkipped)
	if len(s.SkipReasons) > 0 {
		fmt.Fprintln(w, "skip reasons:")
		for _, r := range s.SkipReasons {
			fmt.Fprintf(w, "  - %s\n", r)
		}
	}
}
