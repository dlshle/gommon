package dbutils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/log"
	"github.com/dlshle/gommon/slices"
	"github.com/dlshle/gommon/utils"
)

type Migration struct {
	Version   string    `db:"version"`
	Hash      string    `db:"hash"`
	CreatedAt time.Time `db:"created_at"`
}

type MigrationScript struct {
	Version string
	SQL     string
	hash    string
}

func LoadAndExecuteMigrations(ctx context.Context, tx SQLTransactional, path string) error {
	scripts, err := loadMigrationScripts(ctx, path)
	if err != nil {
		return err
	}
	return execMigration(ctx, tx, scripts)
}

func loadMigrationScripts(ctx context.Context, path string) ([]*MigrationScript, error) {
	var (
		files        []string
		scripts      []*MigrationScript
		hasDuplicate bool
	)
	// open all *.sql files under path
	files, err := utils.DiscoverFiles(path + "/*.sql")
	if err != nil {
		return nil, errors.WrapWithStackTrace(err)
	}
	files = slices.Map(files, func(file string) string {
		return strings.ToLower(file)
	})
	files, hasDuplicate = utils.Deduplicate(files)
	if hasDuplicate {
		log.Errorf(ctx, "duplicate migration files detected")
		return nil, errors.Error("duplicate migration files detected")
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.Compare(files[i], files[j]) >= 0
	})
	for _, file := range files {
		fd, err := os.OpenFile(path+"/"+file, os.O_RDONLY, 0644)
		if err != nil {
			log.Errorf(ctx, "failed to open file %s due to %s", file, err.Error())
			return nil, errors.WrapWithStackTrace(err)
		}
		version := extractFileVersionByName(fd.Name())
		// read text from fd
		text, err := io.ReadAll(fd)
		if err != nil {
			log.Errorf(ctx, "failed to read file %s due to %s", file, err.Error())
			return nil, errors.WrapWithStackTrace(err)
		}
		scripts = append(scripts, &MigrationScript{
			SQL:     string(text),
			Version: version,
		})
	}
	return scripts, nil
}

func execMigration(ctx context.Context, tx SQLTransactional, scripts []*MigrationScript) error {
	// sort scripts by name
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Version < scripts[j].Version
	})

	// compute hash for scripts
	scripts = slices.Map(scripts, func(script *MigrationScript) *MigrationScript {
		script.hash = computeHash(script.SQL)
		return script
	})

	if err := upsertTable(tx); err != nil {
		log.Errorf(ctx, "failed to create migrations table due to: %s", err.Error())
		return err
	}

	migrations, err := getMigrations(tx)
	if err != nil {
		log.Errorf(ctx, "failed to get migrations due to: %s", err.Error())
		return err
	}

	// execute scripts
	for _, script := range scripts {
		if mig, ok := migrations[script.Version]; ok {
			if mig.Hash != script.hash {
				log.Errorf(ctx, "migration hash mismatch for version %s with db hash %s and script hash %s", script.Version, mig.Hash, script.hash)
				return errors.Error("migration script with version " + mig.Version + " has been modified")
			}
			continue
		}
		log.Infof(ctx, "executing migration script with version %s", script.Version)
		_, err := tx.Exec(script.SQL)
		if err != nil {
			log.Errorf(ctx, "failed to execute migration script with version %s due to %s", script.Version, err.Error())
			return err
		}
	}
	return nil
}

func upsertTable(tx SQLTransactional) error {
	_, err := tx.Exec("CREATE TABLE IF NOT EXISTS migrations (created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, hash VARCHAR(255) NOT NULL, version VARCHAR(255) NOT NULL)")
	return err
}

func getMigrations(tx SQLTransactional) (map[string]*Migration, error) {
	var migrations []*Migration
	err := tx.Select(&migrations, "SELECT * FROM migrations")
	if err != nil {
		return nil, err
	}
	return slices.ToMap(migrations, func(m *Migration) (string, *Migration) {
		return m.Version, m
	}), nil
}

func computeHash(script string) string {
	hash := sha256.Sum256([]byte(script))
	return hex.EncodeToString(hash[:])
}

func extractFileVersionByName(fileName string) string {
	pureVersion := strings.Split(fileName, ".")[0]
	return pureVersion
}
