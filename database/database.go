package database

import (
	"context"
	"time"

	zl "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vatsimnetwork/ctp-api/config"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	var err error

	log.Info().Msg("connecting to database...")

	DB, err = gorm.Open(postgres.Open(config.C.DatabaseURL), &gorm.Config{
		Logger: newZerologAdapter(),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	log.Info().Msg("running migrations...")

	if err := DB.AutoMigrate(
		&models.VATSIMEvent{},
		&models.Waypoint{},
		&models.Airport{},
		&models.Location{},
		&models.RouteSegment{},
		&models.EventTag{},
		&models.RouteSegmentTag{},
		&models.Sector{},
		&models.SectorBoundary{},
		&models.SectorBoundaryCoordinate{},
		&models.SlotRevision{},
		&models.Slot{},
		&models.SlotRouteSegment{},
		&models.SlotDraftEntry{},
		&models.ThroughputState{},
		&models.ThroughputSnapshot{},
		&models.Airway{},
		&models.AirwayWaypoint{},
		&models.CustomFix{},
		&models.HighlightedWaypoint{},
		&models.RouteRevisionSet{},
		&models.RouteRevisionEntry{},
		&models.GeoJsonOverlay{},
		&models.SlotPosition{},
		&models.DeparturePairPreference{},
	); err != nil {
		log.Fatal().Err(err).Msg("automigrate failed")
	}

	log.Info().Msg("database ready")
}

type zerologAdapter struct {
	SlowThreshold time.Duration
	level         gormlogger.LogLevel
}

func newZerologAdapter() gormlogger.Interface {
	return &zerologAdapter{
		SlowThreshold: 200 * time.Millisecond,
		level:         gormlogger.Warn,
	}
}

func (z *zerologAdapter) LogMode(lvl gormlogger.LogLevel) gormlogger.Interface {
	copy := *z
	copy.level = lvl
	return &copy
}

func (z *zerologAdapter) Info(_ context.Context, msg string, args ...any) {
	if z.level >= gormlogger.Info {
		log.Info().Msgf(msg, args...)
	}
}

func (z *zerologAdapter) Warn(_ context.Context, msg string, args ...any) {
	if z.level >= gormlogger.Warn {
		log.Warn().Msgf(msg, args...)
	}
}

func (z *zerologAdapter) Error(_ context.Context, msg string, args ...any) {
	if z.level >= gormlogger.Error {
		log.Error().Msgf(msg, args...)
	}
}

func (z *zerologAdapter) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if z.level <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	var ev *zl.Event
	switch {
	case err != nil && z.level >= gormlogger.Error:
		ev = log.Error().Err(err)
	case elapsed > z.SlowThreshold && z.level >= gormlogger.Warn:
		ev = log.Warn().Dur("elapsed", elapsed).Str("slow_query", "true")
	case z.level >= gormlogger.Info:
		ev = log.Debug().Dur("elapsed", elapsed)
	default:
		return
	}

	ev.Str("sql", sql).Int64("rows", rows).Msg("gorm")
}
