package settings

import (
	"os"
	"runtime"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

type Device struct {
	// machine id or uuid
	ID       string
	Hostname string
	OS       string
	Platform string
}

func (g *GeneralSettings) GetDeviceInfo() Device {
	hostname, _ := os.Hostname()
	return Device{
		ID:       g.DeviceID,
		Hostname: hostname,
		OS:       runtime.GOOS,
		Platform: runtime.GOARCH,
	}
}

func genDeviceID() string {
	id, err := machineid.ProtectedID("Typstify")
	if err != nil {
		// in what cases should call to ProtectedID fail?
		id = uuid.NewString()
		return uuid.NewString()
	}

	return id
}
