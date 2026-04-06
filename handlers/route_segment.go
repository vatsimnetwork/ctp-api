package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type locationInput struct {
	Identifier string  `json:"identifier"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	WaypointID int64   `json:"waypointId"`
	SortOrder  uint    `json:"sortOrder"`
}

type tagInput struct {
	Tag string `json:"tag"`
}

type routeSegmentInput struct {
	ID                          uint            `json:"id"`
	Identifier                  string          `json:"identifier"`
	MaximumAircraftPerHour      *uint16         `json:"maximumAircraftPerHour,omitempty"`
	RouteString                 string          `json:"routeString"`
	RouteSegmentGroup           string          `json:"routeSegmentGroup"`
	Color                       string          `json:"color"`
	Enabled                     bool            `json:"enabled"`
	Facilities                  string          `json:"facilities"`
	Tags                        []tagInput      `json:"tags"`
	ProvidedFacilityProgression []models.Sector `json:"providedFacilityProgression"`
	Locations                   []locationInput `json:"locations"`
	RouteRevision               uint            `json:"routeRevision"`
	EventID                     *uint           `json:"eventId,omitempty"`
}


// ListAllRouteSegments godoc
//
//	@Summary	List all route segments
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		routeSegmentGroup	query		string	false	"Route Segment Group"
//	@Success	200	{array}		models.RouteSegment
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/route-segments [get]
func ListAllRouteSegments(c fiber.Ctx) error {
	group := c.Query("routeSegmentGroup")

	var segments []models.RouteSegment
	query := database.DB.Preload("Tags.TagRef").Preload("Locations.Waypoint")
	
	if group != "" {
		query = query.Where("route_segment_group = ?", group)
	}
	
	if err := query.Find(&segments).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(segments)
}

// ListEventRouteSegments godoc
//
//	@Summary	List route segments for an event
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	path		int	true	"Event ID"
//	@Param		routeSegmentGroup	query		string	false	"Route Segment Group"
//	@Success	200		{array}		models.RouteSegment
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/events/{eventId}/route-segments [get]
func ListEventRouteSegments(c fiber.Ctx) error {
	eventID, err := strconv.ParseUint(c.Params("eventId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	group := c.Query("routeSegmentGroup")

	var segments []models.RouteSegment
	query := database.DB.
		Preload("Tags.TagRef").
		Preload("Locations.Waypoint").
		Preload("ProvidedFacilityProgression").
		Where("event_id = ?", eventID)

	if group != "" {
		query = query.Where("route_segment_group = ?", group)
	}

	if err := query.Find(&segments).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(segments)
}

// CreateRouteSegment godoc
//
//	@Summary	Create route segment
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		segment	body		models.RouteSegment	true	"Route segment"
//	@Success	201		{object}	models.RouteSegment
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/route-segments [post]
func CreateRouteSegment(c fiber.Ctx) error {
	var segment models.RouteSegment
	if err := c.Bind().JSON(&segment); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := database.DB.Create(&segment).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(segment)
}

// BatchSaveRouteSegments godoc
//
//	@Summary	Batch create/update/delete route segments in a single transaction
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		models.BatchSaveRequest	true	"Batch payload"
//	@Success	200		{object}	models.SuccessResponse
//	@Failure	400		{object}	models.ErrorResponse
//	@Router		/route-segments/save [post]
func BatchSaveRouteSegments(c fiber.Ctx) error {
	var payload struct {
		Updates []routeSegmentInput `json:"updates"`
		Deletes []uint              `json:"deletes"`
	}
	if err := c.Bind().JSON(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, del := range payload.Deletes {
			seg := models.RouteSegment{}
			seg.ID = del
			if err := tx.Model(&seg).Association("ProvidedFacilityProgression").Clear(); err != nil {
				return err
			}
			if err := tx.Model(&seg).Association("Slots").Clear(); err != nil {
				return err
			}
			if err := tx.Delete(&models.RouteSegment{}, del).Error; err != nil {
				return err
			}
		}

		waypointMap := map[int64]models.Waypoint{}
		for _, seg := range payload.Updates {
			for _, l := range seg.Locations {
				if l.WaypointID != 0 {
					waypointMap[l.WaypointID] = models.Waypoint{
						ID:         l.WaypointID,
						Identifier: l.Identifier,
						Latitude:   l.Latitude,
						Longitude:  l.Longitude,
					}
				}
			}
		}
		if len(waypointMap) > 0 {
			waypoints := make([]models.Waypoint, 0, len(waypointMap))
			for _, w := range waypointMap {
				waypoints = append(waypoints, w)
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"identifier", "latitude", "longitude"}),
			}).Create(&waypoints).Error; err != nil {
				return err
			}
		}

		// Set waypoint capacity defaults based on role:
		// - OCA exit points (last fix of an OCA segment) → 20/hr
		// - All other waypoints → 65535/hr (effectively unlimited)
		ocaExitIDs := map[int64]struct{}{}
		for _, seg := range payload.Updates {
			if strings.EqualFold(seg.RouteSegmentGroup, "OCA") {
				// find the location with the highest sortOrder
				var maxOrder int = -1
				var exitID int64
				for _, l := range seg.Locations {
					if l.WaypointID != 0 && int(l.SortOrder) > maxOrder {
						maxOrder = int(l.SortOrder)
						exitID = l.WaypointID
					}
				}
				if exitID != 0 {
					ocaExitIDs[exitID] = struct{}{}
				}
			}
		}
		for id := range waypointMap {
			if _, isExit := ocaExitIDs[id]; isExit {
				tx.Model(&models.Waypoint{}).Where("id = ? AND maximum_aircraft_per_hour != 20", id).
					Update("maximum_aircraft_per_hour", 20)
			} else {
				tx.Model(&models.Waypoint{}).Where("id = ? AND maximum_aircraft_per_hour < 65535", id).
					Update("maximum_aircraft_per_hour", 65535)
			}
		}

		// Upsert global sectors from all facilities strings.
		allIdents := map[string]struct{}{}
		for _, seg := range payload.Updates {
			if seg.Facilities == "" {
				continue
			}
			for _, ident := range strings.Fields(seg.Facilities) {
				allIdents[ident] = struct{}{}
			}
		}
		for ident := range allIdents {
			var count int64
			tx.Model(&models.Sector{}).Where("identifier = ? AND event_id IS NULL", ident).Count(&count)
			if count == 0 {
				s := models.Sector{}
				s.Identifier = ident
				s.MaximumAircraftPerHour = 20
				tx.Create(&s)
			}
		}

		var globalSectors []models.Sector
		tx.Where("event_id IS NULL").Find(&globalSectors)
		sectorByIdent := make(map[string]models.Sector, len(globalSectors))
		for _, s := range globalSectors {
			sectorByIdent[s.Identifier] = s
		}

		for _, input := range payload.Updates {
			// Resolve facilities string ("EISN EGGX") to Sector records.
			// An empty Facilities string explicitly clears PFP — no fallback.
			var resolvedSectors []models.Sector
			if input.Facilities != "" {
				for _, ident := range strings.Fields(input.Facilities) {
					if s, found := sectorByIdent[ident]; found {
						resolvedSectors = append(resolvedSectors, s)
					}
				}
				// Fall back to any explicitly provided objects only when the
				// facilities string is non-empty but none of its identifiers
				// resolved (e.g. all are event-specific sectors not in sectorByIdent).
				if len(resolvedSectors) == 0 {
					resolvedSectors = input.ProvidedFacilityProgression
				}
			}

			seg := models.RouteSegment{
				RouteString:                 input.RouteString,
				RouteSegmentGroup:           input.RouteSegmentGroup,
				Color:                       input.Color,
				Enabled:                     input.Enabled,
				Facilities:                  input.Facilities,
				ProvidedFacilityProgression: resolvedSectors,
				RouteRevision:               input.RouteRevision,
				EventID:                     input.EventID,
			}
			seg.Identifier = input.Identifier
			if input.MaximumAircraftPerHour != nil {
				seg.MaximumAircraftPerHour = *input.MaximumAircraftPerHour
			} else if input.ID != 0 {
				// Preserve existing value — don't overwrite with 0 when field is omitted.
				var existing models.RouteSegment
				tx.Select("maximum_aircraft_per_hour").First(&existing, input.ID)
				seg.MaximumAircraftPerHour = existing.MaximumAircraftPerHour
			} else {
				seg.MaximumAircraftPerHour = 20 // default for new segments
			}

			if input.ID == 0 {
				if err := tx.Session(&gorm.Session{FullSaveAssociations: true}).Create(&seg).Error; err != nil {
					return err
				}
			} else {
				seg.ID = input.ID
				tx.Where("route_segment_id = ?", input.ID).Delete(&models.RouteSegmentTag{})
				tx.Where("route_segment_id = ?", input.ID).Delete(&models.Location{})
				if err := tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(&seg).Error; err != nil {
					return err
				}
			}

			tags := make([]models.RouteSegmentTag, 0, len(input.Tags))
			for _, t := range input.Tags {
				tagName := strings.TrimSpace(t.Tag)
				if tagName == "" {
					continue
				}
				// Upsert the EventTag (deduped by event+name) and get its ID.
				var eventTag models.EventTag
				if input.EventID != nil {
					tx.Where(models.EventTag{EventID: *input.EventID, Name: tagName}).
						FirstOrCreate(&eventTag)
				}
				rst := models.RouteSegmentTag{RouteSegmentID: seg.ID}
				if eventTag.ID != 0 {
					id := eventTag.ID
					rst.TagID = &id
				}
				tags = append(tags, rst)
			}
			if len(tags) > 0 {
				if err := tx.Create(&tags).Error; err != nil {
					return err
				}
			}

			for _, l := range input.Locations {
				if l.WaypointID == 0 {
					continue
				}
				if err := tx.Create(&models.Location{
					RouteSegmentID: seg.ID,
					WaypointID:     l.WaypointID,
					SortOrder:      l.SortOrder,
				}).Error; err != nil {
					return err
				}
			}
		}

		// Clean up orphaned EventTags: delete any EventTag for each affected event
		// that is no longer referenced by any RouteSegmentTag.
		affectedEventIDs := map[uint]struct{}{}
		for _, input := range payload.Updates {
			if input.EventID != nil {
				affectedEventIDs[*input.EventID] = struct{}{}
			}
		}
		for eid := range affectedEventIDs {
			if err := tx.Exec(`
				DELETE FROM event_tags
				WHERE event_id = ?
				AND id NOT IN (
					SELECT DISTINCT rst.tag_id
					FROM route_segment_tags rst
					INNER JOIN route_segments rs ON rs.id = rst.route_segment_id
					WHERE rs.event_id = ? AND rst.tag_id IS NOT NULL
				)`, eid, eid).Error; err != nil {
				return err
			}
		}

		// Clean up orphaned Sectors: delete global sectors (event_id IS NULL) and
		// event-specific sectors that are no longer referenced by any route segment.
		if err := tx.Exec(`
			DELETE FROM sectors
			WHERE event_id IS NULL
			AND id NOT IN (
				SELECT DISTINCT sector_id FROM route_segment_sectors WHERE sector_id IS NOT NULL
			)`).Error; err != nil {
			return err
		}
		for eid := range affectedEventIDs {
			if err := tx.Exec(`
				DELETE FROM sectors
				WHERE event_id = ?
				AND id NOT IN (
					SELECT DISTINCT rss.sector_id
					FROM route_segment_sectors rss
					INNER JOIN route_segments rs ON rs.id = rss.route_segment_id
					WHERE rs.event_id = ? AND rss.sector_id IS NOT NULL
				)`, eid, eid).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}

// UpdateRouteSegment godoc
//
//	@Summary	Update route segment
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int					true	"Route segment ID"
//	@Param		segment	body		models.RouteSegment	true	"Route segment fields to update"
//	@Success	200		{object}	models.RouteSegment
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/route-segments/{id} [put]
func UpdateRouteSegment(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid route segment id")
	}

	var existing models.RouteSegment
	if database.DB.First(&existing, id).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "route segment not found")
	}

	var updates models.RouteSegment
	if err := c.Bind().JSON(&updates); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	updates.ID = uint(id)
	if err := database.DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	database.DB.Preload("Tags.TagRef").Preload("Locations.Waypoint").Preload("ProvidedFacilityProgression").First(&existing, id)
	return c.JSON(existing)
}

// DeleteRouteSegment godoc
//
//	@Summary	Delete route segment
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		id	path		int			true	"Route segment ID"
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	400	{object}	models.ErrorResponse
//	@Failure	404	{object}	models.ErrorResponse
//	@Router		/route-segments/{id} [delete]
func DeleteRouteSegment(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid route segment id")
	}

	seg := models.RouteSegment{}
	seg.ID = uint(id)
	if err := database.DB.Model(&seg).Association("ProvidedFacilityProgression").Clear(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := database.DB.Model(&seg).Association("Slots").Clear(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := database.DB.Delete(&models.RouteSegment{}, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "route segment not found")
	}

	// Clean up any sectors that are now unreferenced (global and event-specific).
	database.DB.Exec(`
		DELETE FROM sectors
		WHERE id NOT IN (
			SELECT DISTINCT sector_id FROM route_segment_sectors WHERE sector_id IS NOT NULL
		)`)

	return c.JSON(fiber.Map{"success": true})
}

// ReparseAllFacilities godoc
//
//	@Summary	Re-parse facilities strings and rebuild route_segment_sectors for all route segments
//	@Tags		route-segments
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{object}	models.SuccessResponse
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/route-segments/reparse-facilities [post]
func ReparseAllFacilities(c fiber.Ctx) error {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var segments []models.RouteSegment
		if err := tx.Find(&segments).Error; err != nil {
			return err
		}

		// Upsert any missing global sectors from all facilities strings.
		allIdents := map[string]struct{}{}
		for _, seg := range segments {
			for _, ident := range strings.Fields(seg.Facilities) {
				allIdents[ident] = struct{}{}
			}
		}
		for ident := range allIdents {
			var count int64
			tx.Model(&models.Sector{}).Where("identifier = ? AND event_id IS NULL", ident).Count(&count)
			if count == 0 {
				s := models.Sector{}
				s.Identifier = ident
				s.MaximumAircraftPerHour = 20
				tx.Create(&s)
			}
		}

		var globalSectors []models.Sector
		tx.Where("event_id IS NULL").Find(&globalSectors)
		sectorByIdent := make(map[string]models.Sector, len(globalSectors))
		for _, s := range globalSectors {
			sectorByIdent[s.Identifier] = s
		}

		for i := range segments {
			seg := &segments[i]
			var resolvedSectors []models.Sector
			for _, ident := range strings.Fields(seg.Facilities) {
				if s, found := sectorByIdent[ident]; found {
					resolvedSectors = append(resolvedSectors, s)
				}
			}
			if err := tx.Model(seg).Association("ProvidedFacilityProgression").Replace(resolvedSectors); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}

// PatchRouteSegmentCapacity godoc
//
//@SummaryPatch route segment maximum_aircraft_per_hour
//@Tagsroute-segments
//@SecurityApiKeyAuth
//@Acceptjson
//@Producejson
//@Paramidpathinttrue"Route Segment ID"
//@Success200{object}models.RouteSegment
//@Failure400{object}models.ErrorResponse
//@Failure404{object}models.ErrorResponse
//@Router/route-segments/{id}/capacity [patch]
func PatchRouteSegmentCapacity(c fiber.Ctx) error {
id, err := strconv.ParseUint(c.Params("id"), 10, 64)
if err != nil {
return fiber.NewError(fiber.StatusBadRequest, "invalid route segment id")
}

var existing models.RouteSegment
if database.DB.First(&existing, id).Error != nil {
return fiber.NewError(fiber.StatusNotFound, "route segment not found")
}

var input struct {
MaximumAircraftPerHour *uint16 `json:"maximumAircraftPerHour"`
}
if err := c.Bind().JSON(&input); err != nil {
return fiber.NewError(fiber.StatusBadRequest, err.Error())
}

if input.MaximumAircraftPerHour != nil {
if err := database.DB.Model(&existing).UpdateColumn("maximum_aircraft_per_hour", *input.MaximumAircraftPerHour).Error; err != nil {
return fiber.NewError(fiber.StatusInternalServerError, err.Error())
}
}

database.DB.Preload("Tags.TagRef").Preload("Locations.Waypoint").First(&existing, id)
return c.JSON(existing)
}
