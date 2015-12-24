package api

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
	"time"
)

const (
	// How often does the monitor check the DB for an update?
	dbPollPeriod = 1 * time.Second
)

type ApiHandlers struct {
	db      *sql.DB
	monitor *dbMonitor
}

func NewApiHandlers(d *sql.DB) *ApiHandlers {
	a := &ApiHandlers{
		db: d,
		monitor: &dbMonitor{
			lastFifteenSecResTime: time.Unix(0, 0),
			lastTenMinResTime:     time.Unix(0, 0),
			observers:             make(([]chan WSMessage), 0),
		},
	}
	go a.runMonitor()

	return a
}

type dbMonitor struct {
	lastFifteenSecResTime time.Time
	lastTenMinResTime     time.Time
	latestFifteenSecRes   WSMessage
	latestTenMinRes       WSMessage
	observers             []chan WSMessage
	sync.RWMutex
}

// getDBObserver gets a new channel that can be used to listen for database updates
func (a *ApiHandlers) getDBObserver() chan WSMessage {
	n := make(chan WSMessage)
	a.monitor.Lock()
	a.monitor.observers = append(a.monitor.observers, n)
	a.monitor.Unlock()
	defer a.notifyOfLatest(&n)
	return n
}

// notifyOfLatest adds the most recent values the monitor has collected to a (usually newly-created)
// observer's channel, without blocking and preventing the observer from picking them up.
func (a *ApiHandlers) notifyOfLatest(c *(chan WSMessage)) {
	go func(ia *ApiHandlers, ic *(chan WSMessage)) {
		*ic <- ia.monitor.latestFifteenSecRes
		*ic <- ia.monitor.latestTenMinRes
	}(a, c)
}

// runMonitor starts the monitor for this ApiHandlers objects
func (a *ApiHandlers) runMonitor() {
	dbTicker := time.NewTicker(dbPollPeriod)
	defer func() {
		dbTicker.Stop()
	}()

	for {
		select {
		case <-dbTicker.C:
			results, err := pollDB(a)

			if err != nil {
				// Errors have been logged upstream in polldb.
				// TODO: Can we return an error state to the client? Do they need to retry?
				return
			}

			// Notify all observers of update
			for _, o := range a.monitor.observers {
				for _, r := range results {
					o <- r
				}
			}
		}
	}
}

// pollDB is a helper function to runMonitor() and is being called every dbTicker seconds
// pollDB only outputs results in the case that the latest information from the database
// is newer than the most recent data cached in the monitor.
func pollDB(a *ApiHandlers) ([]WSMessage, error) {
	res := make([]WSMessage, 0)

	rows, err := a.db.Query("SELECT * FROM housestation_15sec_wind ORDER BY ID DESC LIMIT 1")
	if err != nil {
		jww.ERROR.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		f := FifteenSecWindMsg{}
		err := rows.Scan(&f.ID, &f.DateTime, &f.WindDirCur, &f.WindDirCurEng, &f.WindSpeedCur)
		if err != nil {
			jww.ERROR.Println(err)
		}

		if f.DateTime.After(a.monitor.lastFifteenSecResTime) {
			r1 := WSMessage{
				MsgType: FifteenSecWind,
				Payload: f,
			}
			res = append(res, r1)
			a.monitor.lastFifteenSecResTime = f.DateTime
			a.monitor.latestFifteenSecRes = r1
		}

	}
	err = rows.Err()
	if err != nil {
		jww.ERROR.Println(err)
	}

	// See if there's an updated 10 minute result
	trrows, err := a.db.Query("SELECT * FROM housestation_10min_all ORDER BY ID DESC LIMIT 1")
	if err != nil {
		jww.ERROR.Println(err)
	}
	defer trrows.Close()
	for trrows.Next() {
		t := TenMinAllRow{}
		err := trrows.Scan(&t.ID, &t.DateTime, &t.TempOutCur, &t.HumOutCur, &t.PressCur, &t.DewCur, &t.HeatIdxCur, &t.WindChillCur, &t.TempInCur,
			&t.HumInCur, &t.WindSpeedCur, &t.WindAvgSpeedCur, &t.WindDirCur, &t.WindDirCurEng, &t.WindGust10, &t.WindDirAvg10, &t.WindDirAvg10Eng,
			&t.UVAvg10, &t.UVMax10, &t.SolarRadAvg10, &t.SolarRadMax10, &t.RainRateCur, &t.RainDay, &t.RainYest, &t.RainMonth, &t.RainYear)
		if err != nil {
			jww.ERROR.Println(err)
		}

		if t.DateTime.After(a.monitor.lastTenMinResTime) {
			r2 := WSMessage{MsgType: TenMinute, Payload: t}
			res = append(res, r2)
			a.monitor.lastTenMinResTime = t.DateTime
			a.monitor.latestTenMinRes = r2
		}
	}

	if len(res) > 0 {
		return res, nil
	} else {
		return nil, nil
	}
}
