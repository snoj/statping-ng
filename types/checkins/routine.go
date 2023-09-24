package checkins

import (
	"fmt"
	"github.com/statping-ng/statping-ng/types/failures"
	"github.com/statping-ng/statping-ng/utils"
	"time"
)

var log = utils.Log.WithField("type", "checkin")

// checkinRoutine for checking if the last Checkin was within its interval
func (c *Checkin) checkinRoutine(serviceTimeout *int) {
	reCheck := c.Period()

CheckinLoop:
	for {
		select {
		case <-c.Running:
			log.Infoln(fmt.Sprintf("Stopping checkin routine: %s", c.Name))
			c.Failing = false
			break CheckinLoop
		case <-time.After(reCheck):
			lastHit := c.LastHit()
			ago := utils.Now().Sub(lastHit.CreatedAt)

			log.Infoln(fmt.Sprintf("Checkin '%s' expects a request every %s last request was %s ago", c.Name, c.Period(), utils.DurationReadable(ago)))

			if ago.Seconds() > c.Period().Seconds()+float64(*serviceTimeout) {
				issue := fmt.Sprintf("Checkin expects a request every %d minutes", c.Interval)
				log.Warnln(issue)

				fail := &failures.Failure{
					Issue:    issue,
					Method:   "checkin",
					Service:  c.ServiceId,
					PingTime: ago.Milliseconds(),
				}

				if err := c.CreateFailure(fail); err != nil {
					log.Errorln(err)
				}
			}
			reCheck = c.Period() + time.Duration(*serviceTimeout*int(time.Second))
		}
	}
}
