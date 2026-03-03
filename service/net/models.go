package net

import (
	"time"
)

type DeviceInfo struct {
	// uuid
	DeviceID   string `json:"device_id"`
	Hostname   string `json:"hostname"`
	OS         string `json:"os"`
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
	Timezone   string `json:"timezone"`
}

type UpdateCheckReq struct {
	DeviceID       string `json:"device_id"`
	CurrentVersion string `json:"current_version"`
	UseBeta        bool   `json:"use_beta"`
}

type ReleaseInfo struct {
	ID         string    `json:"id"`
	AppVersion string    `json:"app_version"`
	IsBeta     bool      `json:"is_beta"`
	Changelog  string    `json:"changelog"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
