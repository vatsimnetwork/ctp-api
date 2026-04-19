package handlers

import (
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"github.com/vatsimnetwork/ctp-api/database"
	"github.com/vatsimnetwork/ctp-api/models"
)

var (
	lockMu    sync.RWMutex
	slotLock  bool
	routeLock bool
)

// LoadLockState reads lock state from the database into memory. Call on startup.
func LoadLockState() {
	var settings models.LockSetting
	if err := database.DB.First(&settings, 1).Error; err != nil {
		log.Warn().Err(err).Msg("locks: could not load lock state from db, defaulting to unlocked")
		return
	}
	lockMu.Lock()
	slotLock = settings.SlotLock
	routeLock = settings.RouteLock
	lockMu.Unlock()
	log.Info().Bool("slotLock", settings.SlotLock).Bool("routeLock", settings.RouteLock).Msg("locks: loaded from database")
}

func IsSlotLocked() bool {
	lockMu.RLock()
	defer lockMu.RUnlock()
	return slotLock
}

func IsRouteLocked() bool {
	lockMu.RLock()
	defer lockMu.RUnlock()
	return routeLock
}

// UpdateLocks receives lock state, persists to DB, and updates the in-memory cache.
//
//	@Summary		Update planner lock state
//	@Description	Set slot lock and route lock state (called by auth-sso)
//	@Tags			locks
//	@Accept			json
//	@Produce		json
//	@Param			body	body	object	true	"Lock state"
//	@Success		200
//	@Router			/locks [put]
func UpdateLocks(c fiber.Ctx) error {
	var body struct {
		SlotLock  bool `json:"slotLock"`
		RouteLock bool `json:"routeLock"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "bad_request",
			"message": "invalid request body",
		})
	}

	if err := database.DB.Model(&models.LockSetting{}).Where("id = 1").Updates(map[string]interface{}{
		"slot_lock":  body.SlotLock,
		"route_lock": body.RouteLock,
	}).Error; err != nil {
		log.Error().Err(err).Msg("locks: failed to persist lock state")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "failed to save lock state",
		})
	}

	lockMu.Lock()
	slotLock = body.SlotLock
	routeLock = body.RouteLock
	lockMu.Unlock()

	log.Info().Bool("slotLock", body.SlotLock).Bool("routeLock", body.RouteLock).Msg("lock state updated")

	return c.JSON(fiber.Map{
		"slotLock":  body.SlotLock,
		"routeLock": body.RouteLock,
	})
}

// GetLocks returns the current lock state.
//
//	@Summary		Get planner lock state
//	@Description	Returns current slot lock and route lock state
//	@Tags			locks
//	@Produce		json
//	@Success		200
//	@Router			/locks [get]
func GetLocks(c fiber.Ctx) error {
	lockMu.RLock()
	s := slotLock
	r := routeLock
	lockMu.RUnlock()

	return c.JSON(fiber.Map{
		"slotLock":  s,
		"routeLock": r,
	})
}

// SlotLockGuard returns 423 Locked if the slot lock is active.
func SlotLockGuard() fiber.Handler {
	return func(c fiber.Ctx) error {
		if IsSlotLocked() {
			return c.Status(fiber.StatusLocked).JSON(fiber.Map{
				"error":   "locked",
				"message": "slot modifications are currently locked",
			})
		}
		return c.Next()
	}
}

// RouteLockGuard returns 423 Locked if the route lock is active.
func RouteLockGuard() fiber.Handler {
	return func(c fiber.Ctx) error {
		if IsRouteLocked() {
			return c.Status(fiber.StatusLocked).JSON(fiber.Map{
				"error":   "locked",
				"message": "route modifications are currently locked",
			})
		}
		return c.Next()
	}
}
