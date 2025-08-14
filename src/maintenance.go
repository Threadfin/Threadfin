package src

import (
	"fmt"
	"math/rand"
	"time"
)

// InitMaintenance : Wartungsprozess initialisieren
func InitMaintenance() (err error) {

	rand.Seed(time.Now().Unix())
	System.TimeForAutoUpdate = fmt.Sprintf("0%d%d", randomTime(0, 2), randomTime(10, 59))

	go maintenance()

	return
}

func maintenance() {

	for {

		var t = time.Now()

		// Aktualisierung der Playlist und XMLTV Dateien
		systemMutex.Lock()
		if System.ScanInProgress == 0 {
			systemMutex.Unlock()
			for _, schedule := range Settings.Update {

				if schedule == t.Format("1504") {

					showInfo("Update:" + schedule)

					// Backup erstellen
					err := ThreadfinAutoBackup()
					if err != nil {
						ShowError(err, 000)
					}

					// Playlist und XMLTV Dateien aktualisieren
					getProviderData("m3u", "")
					getProviderData("hdhr", "")

					if Settings.EpgSource == "XEPG" {
						getProviderData("xmltv", "")
					}

					// Datenbank für DVR erstellen
					err = buildDatabaseDVR()
					if err != nil {
						ShowError(err, 000)
					}

					systemMutex.Lock()
					if !Settings.CacheImages && System.ImageCachingInProgress == 0 {
						systemMutex.Unlock()
						removeChildItems(System.Folder.ImagesCache)
					} else {
						systemMutex.Unlock()
					}

					// XEPG Dateien erstellen
					buildXEPG(false)

				}

			}

			// Update Threadfin (Binary)
			systemMutex.Lock()
			if System.TimeForAutoUpdate == t.Format("1504") {
				systemMutex.Unlock()
				BinaryUpdate()
			} else {
				systemMutex.Unlock()
			}

		} else {
			systemMutex.Unlock()
		}

		time.Sleep(60 * time.Second)

	}

	return
}

func randomTime(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
