// Statping
// Copyright (C) 2018.  Hunter Long and the project contributors
// Written by Hunter Long <info@socialeck.com> and the project contributors
//
// https://github.com/hunterlong/statping
//
// The licenses for most software and other practical works are designed
// to take away your freedom to share and change the works.  By contrast,
// the GNU General Public License is intended to guarantee your freedom to
// share and change all versions of a program--to make sure it remains free
// software for all its users.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"fmt"
	"github.com/hunterlong/statping/core/integrations"
	"github.com/hunterlong/statping/core/notifier"
	"github.com/hunterlong/statping/database"
	"github.com/hunterlong/statping/source"
	"github.com/hunterlong/statping/types"
	"github.com/hunterlong/statping/utils"
	"net"
	"os"
	"time"
)

type PluginJSON types.PluginJSON
type PluginRepos types.PluginRepos

type Core struct {
	*types.Core
	services map[int64]*Service
}

var (
	CoreApp *Core  // CoreApp is a global variable that contains many elements
	VERSION string // VERSION is set on build automatically by setting a -ldflag
	log     = utils.Log.WithField("type", "core")
)

func init() {
	CoreApp = NewCore()
}

// NewCore return a new *core.Core struct
func NewCore() *Core {
	CoreApp = &Core{Core: &types.Core{
		Started: time.Now().UTC(),
	}}
	CoreApp.services = make(map[int64]*Service)
	return CoreApp
}

// ToCore will convert *core.Core to *types.Core
func (c *Core) ToCore() *types.Core {
	return c.Core
}

// InitApp will initialize Statping
func InitApp() error {
	if _, err := SelectCore(); err != nil {
		return err
	}
	if err := InsertNotifierDB(); err != nil {
		return err
	}
	if err := InsertIntegratorDB(); err != nil {
		return err
	}
	if _, err := SelectAllServices(true); err != nil {
		return err
	}
	if err := AttachNotifiers(); err != nil {
		return err
	}
	CoreApp.Notifications = notifier.AllCommunications
	if err := AddIntegrations(); err != nil {
		return err
	}
	CoreApp.Integrations = integrations.Integrations

	go checkServices()

	database.StartMaintenceRoutine()
	CoreApp.Setup = true
	return nil
}

// InsertNotifierDB inject the Statping database instance to the Notifier package
func InsertNotifierDB() error {
	if DbSession == nil {
		err := CoreApp.Connect(false, utils.Directory)
		if err != nil {
			return errors.New("database connection has not been created")
		}
	}
	notifier.SetDB(DbSession)
	return nil
}

// InsertIntegratorDB inject the Statping database instance to the Integrations package
func InsertIntegratorDB() error {
	if DbSession == nil {
		err := CoreApp.Connect(false, utils.Directory)
		if err != nil {
			return errors.New("database connection has not been created")
		}
	}
	integrations.SetDB(DbSession)
	return nil
}

// UpdateCore will update the CoreApp variable inside of the 'core' table in database
func UpdateCore(c *Core) (*Core, error) {
	db := Database(&Core{}).Update(&c)
	return c, db.Error()
}

// CurrentTime will return the current local time
func (c Core) CurrentTime() string {
	t := time.Now().UTC()
	current := utils.Timezoner(t, c.Timezone)
	ansic := "Monday 03:04:05 PM"
	return current.Format(ansic)
}

// Messages will return the current local time
func (c Core) Messages() []*Message {
	var message []*Message
	Database(&Message{}).Where("service = ?", 0).Limit(10).Find(&message)
	return message
}

// UsingAssets will return true if /assets folder is present
func (c Core) UsingAssets() bool {
	return source.UsingAssets(utils.Directory)
}

// SassVars opens the file /assets/scss/variables.scss to be edited in Theme
func (c Core) SassVars() string {
	if !source.UsingAssets(utils.Directory) {
		return ""
	}
	return source.OpenAsset("scss/variables.scss")
}

// BaseSASS is the base design , this opens the file /assets/scss/base.scss to be edited in Theme
func (c Core) BaseSASS() string {
	if !source.UsingAssets(utils.Directory) {
		return ""
	}
	return source.OpenAsset("scss/base.scss")
}

// MobileSASS is the -webkit responsive custom css designs. This opens the
// file /assets/scss/mobile.scss to be edited in Theme
func (c Core) MobileSASS() string {
	if !source.UsingAssets(utils.Directory) {
		return ""
	}
	return source.OpenAsset("scss/mobile.scss")
}

// SelectCore will return the CoreApp global variable and the settings/configs for Statping
func SelectCore() (*Core, error) {
	if DbSession == nil {
		log.Traceln("database has not been initiated yet.")
		return nil, errors.New("database has not been initiated yet.")
	}
	exists := DbSession.HasTable("core")
	if !exists {
		log.Errorf("core database has not been setup yet, does not have the 'core' table")
		return nil, errors.New("core database has not been setup yet.")
	}
	db := Database(&Core{}).First(&CoreApp)
	if db.Error() != nil {
		return nil, db.Error()
	}
	CoreApp.Version = VERSION
	CoreApp.UseCdn = types.NewNullBool(os.Getenv("USE_CDN") == "true")
	return CoreApp, db.Error()
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "http://localhost"
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return fmt.Sprintf("http://%v", ipnet.IP.String())
			}
		}
	}
	return "http://localhost"
}

// ServiceOrder will reorder the services based on 'order_id' (Order)
type ServiceOrder map[int64]*Service

// Sort interface for resroting the Services in order
func (c ServiceOrder) Len() int      { return len(c) }
func (c ServiceOrder) Swap(i, j int) { c[int64(i)], c[int64(j)] = c[int64(j)], c[int64(i)] }
func (c ServiceOrder) Less(i, j int) bool {
	if c[int64(i)] == nil {
		return false
	}
	if c[int64(j)] == nil {
		return false
	}
	return c[int64(i)].Order < c[int64(j)].Order
}
