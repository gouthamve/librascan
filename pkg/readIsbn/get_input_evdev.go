package readIsbn

import (
	"fmt"
	"time"

	"github.com/holoplot/go-evdev"
)

var (
// devicePath = "/dev/input/by-id/usb-Barcode_AFANDA_BARCODE_AFANDBARCODE-event-kbd"
)

type deviceInput struct {
	devicePath string
	device     *evdev.InputDevice

	bufferedCodes chan string
}

func grabAndSetupDevice(inputDevicePath string) (*deviceInput, error) {
	di := &deviceInput{
		devicePath:    inputDevicePath,
		bufferedCodes: make(chan string, 100),
	}

	if err := di.open(); err != nil {
		return nil, err
	}

	go di.readToBuffer()

	return di, nil
}

func (d *deviceInput) open() error {
	for {
		time.Sleep(500 * time.Millisecond)
		inputDevice, err := evdev.Open(d.devicePath)
		if err != nil {
			continue
		}
		if err := inputDevice.NonBlock(); err != nil {
			fmt.Println("device nonblock error", err)
			inputDevice.Close()
			continue
		}

		if err := inputDevice.Grab(); err != nil {
			fmt.Println("device grab error", err)
			inputDevice.Close()
			continue
		}

		fmt.Println("device found and opened.")
		d.device = inputDevice

		return nil
	}
}

func (d *deviceInput) readToBuffer() {
	var barcode string

	for {
		// If device is not open, open it.
		if _, err := d.device.Name(); err != nil {
			fmt.Println("device name error", err)
			// It's likely sleeping or not yet connected. This will wait for it to open.
			for {
				if err := d.open(); err == nil {
					break
				}
			}
		}

		for {
			ev, err := d.device.ReadOne()
			if err != nil {
				fmt.Println("device read error", err)

				d.bufferedCodes <- barcode
				barcode = ""
				break
			}

			if ev.Type == evdev.EV_KEY && ev.Value == 1 {
				if ev.Code == evdev.KEY_ENTER {
					d.bufferedCodes <- barcode
					barcode = ""
					break
				}
			}

			if ev.Type == evdev.EV_KEY && ev.Value == 1 {
				barcode += eventToString(*ev)
			}
		}
	}
}

func (d *deviceInput) read() string {
	return <-d.bufferedCodes
}

func eventToString(ev evdev.InputEvent) string {
	switch ev.Code {
	case evdev.KEY_0:
		return "0"
	case evdev.KEY_1:
		return "1"
	case evdev.KEY_2:
		return "2"
	case evdev.KEY_3:
		return "3"
	case evdev.KEY_4:
		return "4"
	case evdev.KEY_5:
		return "5"
	case evdev.KEY_6:
		return "6"
	case evdev.KEY_7:
		return "7"
	case evdev.KEY_8:
		return "8"
	case evdev.KEY_9:
		return "9"
	default:
		return ""
	}
}
