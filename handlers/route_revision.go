package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

// ListRouteRevisions godoc
//
//	@Summary	List all route revisions
//	@Tags		route-revisions
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200	{array}		models.RouteRevisionSet
//	@Failure	500	{object}	models.ErrorResponse
//	@Router		/route-revisions [get]
func ListRouteRevisions(c fiber.Ctx) error {
	var revisions []models.RouteRevisionSet
	if err := database.DB.Order("number DESC").Find(&revisions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(revisions)
}

// CreateRouteRevision godoc
//
//	@Summary	Snapshot current route segments as a new revision
//	@Tags		route-revisions
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		eventId	query		int	false	"Event ID — snapshots segments for this event; omit for unassigned segments"
//	@Success	201		{object}	models.RouteRevisionSet
//	@Failure	500		{object}	models.ErrorResponse
//	@Router		/route-revisions [post]
func CreateRouteRevision(c fiber.Ctx) error {
	var revision models.RouteRevisionSet

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var latest models.RouteRevisionSet
		tx.Order("number DESC").First(&latest)
		nextNumber := latest.Number + 1

		revision = models.RouteRevisionSet{Number: nextNumber}
		if err := tx.Create(&revision).Error; err != nil {
			return err
		}

		var segments []models.RouteSegment
		segQuery := tx.Preload("Tags")
		if eventIDStr := c.Query("eventId"); eventIDStr != "" {
			if eid, err := strconv.ParseUint(eventIDStr, 10, 64); err == nil {
				segQuery = segQuery.Where("event_id = ?", eid)
			}
		} else {
			segQuery = segQuery.Where("event_id IS NULL")
		}
		if err := segQuery.Find(&segments).Error; err != nil {
			return err
		}

		entries := make([]models.RouteRevisionEntry, 0, len(segments))
		for _, seg := range segments {
			var tagParts []string
			for _, t := range seg.Tags {
				tagParts = append(tagParts, t.Tag)
			}
			entries = append(entries, models.RouteRevisionEntry{
				RevisionID:  revision.ID,
				Identifier:  seg.Identifier,
				Group:       seg.RouteSegmentGroup,
				RouteString: seg.RouteString,
				Facilities:  seg.Facilities,
				Tags:        strings.Join(tagParts, " "),
				Color:       seg.Color,
				Enabled:     seg.Enabled,
			})
		}

		if len(entries) > 0 {
			if err := tx.Create(&entries).Error; err != nil {
				return err
			}
		}

		revision.Entries = entries

		if eventIDStr := c.Query("eventId"); eventIDStr != "" {
			if eid, err := strconv.ParseUint(eventIDStr, 10, 64); err == nil {
				tx.Model(&models.VATSIMEvent{}).Where("id = ?", eid).Update("route_revision", nextNumber)
			}
		}

		return nil
	})

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(revision)
}

// GetRouteRevision godoc
//
//	@Summary	Get route revision by number
//	@Tags		route-revisions
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		number	path		int	true	"Revision number"
//	@Success	200		{object}	models.RouteRevisionSet
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/route-revisions/{number} [get]
func GetRouteRevision(c fiber.Ctx) error {
	number, err := strconv.ParseUint(c.Params("number"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision number")
	}

	var revision models.RouteRevisionSet
	result := database.DB.
		Preload("Entries").
		Where("number = ?", number).
		First(&revision)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "revision not found")
	}

	return c.JSON(revision)
}
