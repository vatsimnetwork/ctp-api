package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type SimulatorDataResponse struct {
	Event    VATSIMEvent   `json:"event"`
	Revision *SlotRevision `json:"revision"`
}

type BatchSaveRequest struct {
	Updates []RouteSegment `json:"updates"`
	Deletes []uint         `json:"deletes"`
}

type Duration time.Duration

func (d Duration) Value() (driver.Value, error) {
	return int64(d), nil
}

func (d *Duration) Scan(value any) error {
	if value == nil {
		*d = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*d = Duration(v)
	default:
		return fmt.Errorf("cannot scan %T into Duration", value)
	}
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		var n int64
		if err2 := json.Unmarshal(b, &n); err2 != nil {
			return err
		}
		*d = Duration(n)
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}
