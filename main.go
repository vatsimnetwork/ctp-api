// @title			CTP API
// @version		1.0.0
// @description	Central data API for CTP (Cross the Pond) event management. All /api/* endpoints require an X-API-Key header.
// @BasePath		/api
// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						X-API-Key
package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vatsimnetwork/ctp-api/config"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/docs"
	"github.com/vatsimnetwork/ctp-api/handlers"
	"github.com/vatsimnetwork/ctp-api/middleware"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()

	config.Load()
	docs.SwaggerInfo.Host = ""
	database.Connect()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			if code >= 500 {
				log.Error().Err(err).Int("status", code).Msg("server error")
				return c.Status(code).JSON(fiber.Map{
					"error":   "internal_error",
					"message": "an internal error occurred",
				})
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   "error",
				"message": err.Error(),
			})
		},
	})

	app.Hooks().OnPreStartupMessage(func(sm *fiber.PreStartupMessageData) error {
		sm.BannerHeader = `
      ______________      ___   ___  ____
     / ___/_  __/ _ \____/ _ | / _ \/  _/
    / /__  / / / ___/___/ __ |/ ___// /
    \___/ /_/ /_/      /_/ |_/_/  /___/
		`
		return nil
	})

	app.Use(recover.New())

	app.Get("/api/docs/*", swaggo.HandlerDefault)

	api := app.Group("/api", middleware.APIKeyAuth())

	api.Get("/events", handlers.ListEvents)
	api.Post("/events", handlers.CreateEvent)
	api.Get("/events/:id", handlers.GetEvent)
	api.Put("/events/:id", handlers.UpdateEvent)
	api.Delete("/events/:id", handlers.DeleteEvent)
	api.Get("/events/:id/simulator-data", handlers.GetSimulatorData)
	api.Get("/events/:id/simulator-data/latest-with-slots", handlers.GetSimulatorDataLatestWithSlots)
	api.Get("/events/:id/charts/departure-airports", handlers.ChartsDepartureAirports)
	api.Get("/events/:id/charts/sectors",            handlers.ChartsSectors)
	api.Get("/events/:id/charts/arrival-airports",   handlers.ChartsArrivalAirports)
	api.Get("/events/:id/calculate-slots/preview", handlers.PreviewCalculatePayload)
	api.Get("/events/:id/simulate-slots/preview", handlers.PreviewSimulatePayload)
	api.Post("/events/:id/calculate-slots", handlers.CalculateSlots)
	api.Post("/events/:id/simulate-slots", handlers.SimulateSlots)
	api.Get("/events/:id/simulate-status", handlers.GetSimulateStatus)
	api.Get("/events/:id/latest-simulator-response", handlers.GetLatestSimulatorResponse)

	api.Get("/events/:eventId/airports", handlers.ListAirports)
	api.Post("/events/:eventId/airports", handlers.CreateAirport)
	api.Put("/airports/:id", handlers.UpdateAirport)
	api.Patch("/airports/:id/capacity", handlers.PatchAirportCapacity)
	api.Patch("/airports/:id/departure-time-window-start", handlers.PatchAirportDepartureTimeWindowStart)
	api.Delete("/airports/:id", handlers.DeleteAirport)

	api.Get("/waypoints", handlers.ListWaypoints)
	api.Put("/waypoints/:id", handlers.UpdateWaypoint)

	api.Get("/events/:eventId/sectors", handlers.ListSectors)
	api.Post("/events/:eventId/sectors", handlers.CreateSector)
	api.Put("/sectors/:id", handlers.UpdateSector)
	api.Patch("/sectors/:id/capacity", handlers.PatchSectorCapacity)
	api.Delete("/sectors/:id", handlers.DeleteSector)

	api.Get("/events/:eventId/tag-limits", handlers.ListEventTagLimits)
	api.Patch("/events/:eventId/tag-limits", handlers.UpsertEventTagLimits)

	api.Get("/events/:eventId/slot-revisions", handlers.ListSlotRevisions)
	api.Post("/events/:eventId/slot-revisions", handlers.CreateSlotRevision)
	api.Get("/events/:eventId/slot-revisions/latest", handlers.GetLatestSlotRevision)
	api.Get("/events/:eventId/slot-revisions/:number", handlers.GetSlotRevision)
	api.Put("/slot-revisions/:revisionId", handlers.UpdateSlotRevision)
	api.Delete("/slot-revisions/:revisionId", handlers.DeleteSlotRevision)
	api.Post("/slot-revisions/:revisionId/slots", handlers.AddSlotsToRevision)
	api.Post("/slot-revisions/:revisionId/throughput-states", handlers.AddThroughputStatesToRevision)
	api.Post("/slot-revisions/:revisionId/throughput-snapshots", handlers.AddThroughputSnapshotsToRevision)

	api.Get("/route-segments", handlers.ListAllRouteSegments)
	api.Get("/events/:eventId/route-segments", handlers.ListEventRouteSegments)
	api.Post("/route-segments", handlers.CreateRouteSegment)
	api.Post("/route-segments/save", handlers.BatchSaveRouteSegments)
	api.Post("/route-segments/reparse-facilities", handlers.ReparseAllFacilities)
	api.Put("/route-segments/:id", handlers.UpdateRouteSegment)
	api.Patch("/route-segments/:id/capacity", handlers.PatchRouteSegmentCapacity)
	api.Delete("/route-segments/:id", handlers.DeleteRouteSegment)

	api.Get("/airways", handlers.ListAirways)
	api.Post("/airways", handlers.CreateAirway)
	api.Post("/airways/bulk", handlers.BulkCreateAirways)
	api.Delete("/airways/:identifier", handlers.DeleteAirway)
	api.Delete("/airways", handlers.DeleteAllAirways)

	api.Get("/custom-fixes", handlers.ListCustomFixes)
	api.Post("/custom-fixes", handlers.UpsertCustomFix)
	api.Delete("/custom-fixes/:identifier", handlers.DeleteCustomFix)

	api.Get("/highlighted-waypoints", handlers.ListHighlightedWaypoints)
	api.Post("/highlighted-waypoints", handlers.UpsertHighlightedWaypoint)
	api.Delete("/highlighted-waypoints/:identifier", handlers.DeleteHighlightedWaypoint)

	api.Get("/route-revisions", handlers.ListRouteRevisions)
	api.Post("/route-revisions", handlers.CreateRouteRevision)
	api.Get("/route-revisions/:number", handlers.GetRouteRevision)

	go func() {
		if err := app.Listen(":" + config.C.AppPort); err != nil {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	log.Info().Str("port", config.C.AppPort).Msg("ctp-api started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down...")
	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("shutdown error")
	}
}
