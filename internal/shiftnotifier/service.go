package shiftnotifier

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Cubicroots-Playground/trollinfo/internal/angelapi"
	"github.com/Cubicroots-Playground/trollinfo/internal/matrixmessenger"

	_ "time/tzdata"
)

type service struct {
	angelAPI  angelapi.Service
	messenger matrixmessenger.Messenger
	config    *Config
}

// Config holds the configuration for the shift notifier.
type Config struct {
	LocationNames          []string
	NotifyBeforeShiftStart time.Duration
	MatrixRoomID           string
}

// ParseFromEnvironment parses the config from the environment.
func (c *Config) ParseFromEnvironment() {
	c.LocationNames = strings.Split(os.Getenv("TROLLINFO_LOCATIONS"), ",")
	c.MatrixRoomID = os.Getenv("TROLLINFO_MATRIX_ROOM_ID")
	c.NotifyBeforeShiftStart = time.Minute * 15
}

// New assembles a new shift notifier.
func New(config *Config, angelAPI angelapi.Service, messenger matrixmessenger.Messenger) Service {
	return &service{
		angelAPI:  angelAPI,
		messenger: messenger,
		config:    config,
	}
}

func (service *service) Start() error {
	// TODO make loop
	service.getNextShifts()
	return nil
}

func (service *service) Stop() error {
	return nil
}

type shiftUser struct {
	Nickname  string
	AngelType string
	ShiftName string
}

type shiftDiff struct {
	UsersLeaving  []shiftUser
	UsersWorking  []shiftUser
	UsersArriving []shiftUser
	ExpectedUsers int64
	OpenUsers     map[string]int64
}

type shiftDiffs struct {
	DiffsInLocations map[string]shiftDiff
	ReferenceTime    time.Time
}

func (service *service) getNextShifts() {
	locations, err := service.getLocationIDs()
	if err != nil {
		slog.Error("failed to list locations", "error", err.Error())
		return
	}

	diffs := map[string]shiftDiff{}

	// TODO change to time.Now()
	refTime := time.Date(2024, 5, 31, 19, 54, 0, 0, time.UTC)

	for locationID, locationName := range locations {
		diff := shiftDiff{
			UsersLeaving:  []shiftUser{},
			UsersWorking:  []shiftUser{},
			UsersArriving: []shiftUser{},
			OpenUsers:     map[string]int64{},
		}

		shifts, err := service.angelAPI.ListShiftsInLocation(locationID, nil)
		if err != nil {
			slog.Error("failed to list shifts", "location_id", locationID, "error", err.Error())
			continue
		}

		for _, shift := range shifts {
			timeUntilShiftStart := shift.StartsAt.Sub(refTime)
			timeUntilShiftEnd := shift.EndsAt.Sub(refTime)

			// Next shift, users should arrive.
			if timeUntilShiftStart > 0 && timeUntilShiftStart < time.Minute*15 {
				for _, shiftEntry := range shift.Entries {
					diff.ExpectedUsers += shiftEntry.Needs

					for _, user := range shiftEntry.Users {
						diff.UsersArriving = append(diff.UsersArriving, shiftUser{
							Nickname:  user.NickName,
							AngelType: shiftEntry.Type.Name,
							ShiftName: shift.Title,
						})
					}

					diff.OpenUsers[shiftEntry.Type.Name] += shiftEntry.Needs - int64(len(diff.UsersArriving))
				}
				continue
			}

			// Previous shift, users should leave.
			if timeUntilShiftStart < 0 &&
				timeUntilShiftEnd > 0 &&
				timeUntilShiftEnd <= (service.config.NotifyBeforeShiftStart+time.Minute) {
				for _, shiftEntry := range shift.Entries {
					for _, user := range shiftEntry.Users {
						diff.UsersLeaving = append(diff.UsersArriving, shiftUser{
							Nickname:  user.NickName,
							AngelType: shiftEntry.Type.Name,
							ShiftName: shift.Title,
						})
					}
				}
				continue
			}

			// Overlapping shift, users should stay.
			if timeUntilShiftStart < 0 && timeUntilShiftEnd > 0 {
				for _, shiftEntry := range shift.Entries {
					for _, user := range shiftEntry.Users {
						diff.UsersWorking = append(diff.UsersArriving, shiftUser{
							Nickname:  user.NickName,
							AngelType: shiftEntry.Type.Name,
							ShiftName: shift.Title,
						})
					}
				}
				continue
			}
		}

		diffs[locationName] = diff
	}

	msg, msgFormatted := service.diffToMessage(shiftDiffs{
		DiffsInLocations: service.cleanUpDiffs(diffs),
		ReferenceTime:    refTime,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = service.messenger.SendMessage(ctx, matrixmessenger.HTMLMessage(
		msg, msgFormatted, service.config.MatrixRoomID,
	))
	if err != nil {
		slog.Error("failed to send matrix message", "error", err.Error())
	}
}

func (service *service) getLocationIDs() (map[int64]string, error) {
	locations, err := service.angelAPI.ListLocations(nil)
	if err != nil {
		return nil, err
	}

	locationMap := make(map[int64]string)

	for _, location := range locations {
		isKnown := false
		for _, loc := range service.config.LocationNames {
			if loc == location.Name {
				isKnown = true
				break
			}
		}

		if !isKnown {
			continue
		}

		locationMap[location.ID] = location.Name
	}

	return locationMap, nil
}

func (service *service) cleanUpDiffs(diffs map[string]shiftDiff) map[string]shiftDiff {
	// Ugly af, needs refactoring. Users that are leaving & arriving should be moved to the
	// "working" list.
	newDiffs := make(map[string]shiftDiff)

	skipLeavingUser := []shiftUser{}
	for location, diff := range diffs {
		newDiff := shiftDiff{
			UsersLeaving:  []shiftUser{},
			UsersWorking:  []shiftUser{},
			UsersArriving: []shiftUser{},
			OpenUsers:     diffs[location].OpenUsers,
			ExpectedUsers: diffs[location].ExpectedUsers,
		}

		for _, user := range diff.UsersArriving {
			isUserStaying := false
			for i := range diff.UsersLeaving {
				if user.Nickname == diff.UsersLeaving[i].Nickname {
					isUserStaying = true
					skipLeavingUser = append(skipLeavingUser, diff.UsersLeaving[i])
					newDiff.UsersWorking = append(newDiff.UsersWorking, user)
					break
				}
			}
			if isUserStaying {
				continue
			}

			newDiff.UsersArriving = append(newDiff.UsersArriving, user)
		}

		for _, user := range diff.UsersLeaving {
			isUserStaying := false
			for _, skipUser := range skipLeavingUser {
				if user.Nickname == skipUser.Nickname {
					isUserStaying = true
					break
				}
			}

			if isUserStaying {
				continue
			}

			newDiff.UsersLeaving = append(newDiff.UsersLeaving, user)
		}

		newDiff.UsersWorking = append(newDiff.UsersWorking, diff.UsersWorking...)

		newDiffs[location] = newDiff
	}

	return newDiffs
}

func (service *service) diffToMessage(diffs shiftDiffs) (string, string) {
	msg := strings.Builder{}
	msgHTML := strings.Builder{}

	defaultTZ, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		defaultTZ = time.Local
	}

	timeStr := diffs.ReferenceTime.
		Add(service.config.NotifyBeforeShiftStart).
		In(defaultTZ).
		Format("Mon, 15:04")

	// Title.
	msg.WriteString("TROLL CHANGES FOR ")
	msg.WriteString(timeStr)
	msg.WriteString("\n\n")

	msgHTML.WriteString("<h1>Troll Changes for ")
	msgHTML.WriteString(timeStr)
	msgHTML.WriteString("</h1><br>\n")

	for loc, diff := range diffs.DiffsInLocations {
		// Location.
		msg.WriteString("📍 ")
		msg.WriteString(loc)
		msg.WriteString("\n")

		msgHTML.WriteString("📍 <b>")
		msgHTML.WriteString(loc)
		msgHTML.WriteString("</b><br>\n")

		// Troll lists.
		msg.WriteString("Arriving Trolls 🔜:\n")
		msgHTML.WriteString("Arriving Trolls 🔜:<br>\n")
		usersToList(diff.UsersArriving, &msg, &msgHTML)

		msg.WriteString("Staying Trolls 🔄:\n")
		msgHTML.WriteString("Staying Trolls 🔄:<br>\n")
		usersToList(diff.UsersWorking, &msg, &msgHTML)

		msg.WriteString("Leaving Trolls 🔚:\n")
		msgHTML.WriteString("Leaving Trolls 🔚:<br>\n")
		usersToList(diff.UsersLeaving, &msg, &msgHTML)

		// Summary.
		msg.WriteString("\nExpecting ")
		msg.WriteString(strconv.Itoa(int(diff.ExpectedUsers)))
		msg.WriteString(" trolls total\n")
		msgHTML.WriteString("<br>\nExpecting ")
		msgHTML.WriteString(strconv.Itoa(int(diff.ExpectedUsers)))
		msgHTML.WriteString(" trolls total<br>\n")

		if len(diff.OpenUsers) > 0 {
			msg.WriteString("🚨 Open positions:\n")
			for shiftType, amount := range diff.OpenUsers {
				msg.WriteString("- ")
				msg.WriteString(strconv.Itoa(int(amount)))
				msg.WriteString("x ")
				msg.WriteString(shiftType)
				msg.WriteString("\n")
			}

			msgHTML.WriteString("🚨 Open positions:<br>\n")
			for shiftType, amount := range diff.OpenUsers {
				msgHTML.WriteString("- ")
				msgHTML.WriteString(strconv.Itoa(int(amount)))
				msgHTML.WriteString("x ")
				msgHTML.WriteString(shiftType)
				msgHTML.WriteString("<br>\n")
			}
		}

		msg.WriteString("\n")
		msgHTML.WriteString("<br>\n")
	}

	return msg.String(), msgHTML.String()
}

func usersToList(users []shiftUser, msg *strings.Builder, msgHTML *strings.Builder) {
	if len(users) == 0 {
		msg.WriteString("  _none_\n")
		msgHTML.WriteString("&nbsp;&nbsp;<i>none</i><br>\n")
		return
	}

	for _, user := range users {
		msg.WriteString("  - ")
		msg.WriteString(user.Nickname)
		msg.WriteString(" (")
		msg.WriteString(user.ShiftName)
		msg.WriteString(")\n")

		msgHTML.WriteString("&nbsp;&nbsp;- ")
		msgHTML.WriteString(user.Nickname)
		msgHTML.WriteString(" <i>(")
		msgHTML.WriteString(user.ShiftName)
		msgHTML.WriteString(")</i><br>\n")
	}
}
