package handlers

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
	"gorm.io/gorm"
)

// ListSlotDraftEntries godoc
//
//	@Summary	List slot draft entries for a revision
//	@Tags		slot-draft-entries
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Param		revisionId	path		int	true	"Slot revision ID"
//	@Success	200		{array}		models.SlotDraftEntry
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/draft-entries [get]
func ListSlotDraftEntries(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var entries []models.SlotDraftEntry
	if err := database.DB.
		Preload("DepartureAirport").
		Preload("DepRoute").
		Preload("Track").
		Preload("ArrRoute").
		Preload("ArrivalAirport").
		Where("slot_revision_id = ?", revisionID).
		Find(&entries).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(entries)
}

// AddSlotDraftEntries godoc
//
//	@Summary	Add slot draft entries to a revision
//	@Tags		slot-draft-entries
//	@Security	ApiKeyAuth
//	@Accept		json
//	@Produce	json
//	@Param		revisionId	path		int	true	"Slot revision ID"
//	@Param		entries	body		[]SlotDraftEntryInput	true	"Slot draft entries to add"
//	@Success	201		{object}	models.SuccessResponse
//	@Failure	400		{object}	models.ErrorResponse
//	@Failure	404		{object}	models.ErrorResponse
//	@Router		/slot-revisions/{revisionId}/draft-entries [post]
func AddSlotDraftEntries(c fiber.Ctx) error {
	revisionID, err := strconv.ParseUint(c.Params("revisionId"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid revision id")
	}

	var existing models.SlotRevision
	if database.DB.First(&existing, revisionID).Error != nil {
		return fiber.NewError(fiber.StatusNotFound, "slot revision not found")
	}

	var inputs []SlotDraftEntryInput
	if err := c.Bind().JSON(&inputs); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for _, input := range inputs {
			entry := models.SlotDraftEntry{
				SlotRevisionID:     uint(revisionID),
				DepartureAirportID: input.DepartureAirportID,
				DepRouteID:         input.DepRouteID,
				TrackID:            input.TrackID,
				ArrRouteID:         input.ArrRouteID,
				ArrivalAirportID:   input.ArrivalAirportID,
				SlotCount:          input.SlotCount,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true})
}

// MigrateSlotDraftEntries godoc
//
//	@Summary	Migrate slot planner draft commentary JSON to relational SlotDraftEntry records
//	@Tags		slot-draft-entries
//	@Security	ApiKeyAuth
//	@Produce	json
//	@Success	200		{object}	MigrateSlotDraftEntriesResponse
//	@Failure	500		{object}	models.ErrorResponse
//	@Router		/slot-draft-entries/migrate [post]
func MigrateSlotDraftEntries(c fiber.Ctx) error {
	var revisions []models.SlotRevision
	if err := database.DB.
		Where("slot_planner_draft_commentary IS NOT NULL AND slot_planner_draft_commentary != ''").
		Find(&revisions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	migrated := 0
	skipped := 0
	errors := 0
	skippedInvalid := 0

	for _, revision := range revisions {
		var existingCount int64
		database.DB.Model(&models.SlotDraftEntry{}).Where("slot_revision_id = ?", revision.ID).Count(&existingCount)
		if existingCount > 0 {
			log.Printf("[migration] revision %d: already has %d entries, skipping", revision.ID, existingCount)
			skipped++
			continue
		}

		var rawCommentary string
		if err := database.DB.Raw(`
			SELECT slot_planner_draft_commentary FROM slot_revisions WHERE id = ?
		`, revision.ID).Scan(&rawCommentary).Error; err != nil {
			log.Printf("[migration] revision %d: failed to fetch commentary: %v", revision.ID, err)
			errors++
			continue
		}
		if rawCommentary == "" {
			log.Printf("[migration] revision %d: empty commentary, skipping", revision.ID)
			skipped++
			continue
		}

		var commentaryData struct {
			SlotGroups []slotDraftEntryGroupJSON `json:"slotGroups"`
		}
		if err := json.Unmarshal([]byte(rawCommentary), &commentaryData); err != nil {
			log.Printf("[migration] revision %d: failed to parse JSON: %v", revision.ID, err)
			errors++
			continue
		}

		if len(commentaryData.SlotGroups) == 0 {
			log.Printf("[migration] revision %d: no slot groups in commentary, skipping", revision.ID)
			skipped++
			continue
		}

		// Validate all IDs exist before attempting insert
		var validGroups []slotDraftEntryGroup
		for _, group := range commentaryData.SlotGroups {
			if group.DepAirportID == 0 || group.DepRouteID == 0 || group.TrackID == 0 ||
				group.ArrRouteID == 0 || group.ArrAirportID == 0 {
				log.Printf("[migration] revision %d: group has zero ID (dep=%d, depRoute=%d, track=%d, arrRoute=%d, arr=%d), skipping group",
					revision.ID, group.DepAirportID, group.DepRouteID, group.TrackID, group.ArrRouteID, group.ArrAirportID)
				skippedInvalid++
				continue
			}

			// Verify referenced routes/airports actually exist
			var depAirportCount, arrAirportCount, depRouteCount, trackCount, arrRouteCount int64
			database.DB.Model(&models.Airport{}).Where("id = ?", group.DepAirportID).Count(&depAirportCount)
			database.DB.Model(&models.Airport{}).Where("id = ?", group.ArrAirportID).Count(&arrAirportCount)
			database.DB.Model(&models.RouteSegment{}).Where("id = ?", group.DepRouteID).Count(&depRouteCount)
			database.DB.Model(&models.RouteSegment{}).Where("id = ?", group.TrackID).Count(&trackCount)
			database.DB.Model(&models.RouteSegment{}).Where("id = ?", group.ArrRouteID).Count(&arrRouteCount)

			if depAirportCount == 0 || arrAirportCount == 0 || depRouteCount == 0 || trackCount == 0 || arrRouteCount == 0 {
				log.Printf("[migration] revision %d: referenced entity no longer exists (depAirport=%d, arrAirport=%d, depRoute=%d, track=%d, arrRoute=%d), skipping group",
					revision.ID, depAirportCount, arrAirportCount, depRouteCount, trackCount, arrRouteCount)
				skippedInvalid++
				continue
			}

			validGroups = append(validGroups, slotDraftEntryGroup{
				DepAirportID: group.DepAirportID,
				DepRouteID:   group.DepRouteID,
				TrackID:      group.TrackID,
				ArrRouteID:   group.ArrRouteID,
				ArrAirportID: group.ArrAirportID,
				SlotCount:    group.Value,
			})
		}

		if len(validGroups) == 0 {
			log.Printf("[migration] revision %d: no valid groups after filtering, skipping", revision.ID)
			skippedInvalid++
			continue
		}

		err := database.DB.Transaction(func(tx *gorm.DB) error {
			for _, group := range validGroups {
				entry := models.SlotDraftEntry{
					SlotRevisionID:     revision.ID,
					DepartureAirportID: group.DepAirportID,
					DepRouteID:         group.DepRouteID,
					TrackID:            group.TrackID,
					ArrRouteID:         group.ArrRouteID,
					ArrivalAirportID:   group.ArrAirportID,
					SlotCount:          group.SlotCount,
				}
				if err := tx.Create(&entry).Error; err != nil {
					log.Printf("[migration] revision %d: FK violation inserting group (dep=%d, arr=%d): %v",
						revision.ID, group.DepAirportID, group.ArrAirportID, err)
					return err
				}
			}
			return nil
		})
		if err != nil {
			errors++
			continue
		}
		log.Printf("[migration] revision %d: successfully migrated %d groups", revision.ID, len(validGroups))
		migrated++
	}

	log.Printf("[migration] complete: migrated=%d, skipped(existing)=%d, skipped(invalid/zero-IDs)=%d, errors=%d",
		migrated, skipped, skippedInvalid, errors)

	return c.JSON(MigrateSlotDraftEntriesResponse{
		Migrated: migrated,
		Skipped:  skipped + skippedInvalid,
		Errors:   errors,
	})
}
