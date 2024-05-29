package shiftnotifier

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Cubicroots-Playground/trollinfo/internal/angelapi"
	"github.com/Cubicroots-Playground/trollinfo/internal/matrixmessenger"
	"github.com/go-co-op/gocron/v2"

	_ "time/tzdata"
)

type service struct {
	angelAPI  angelapi.Service
	messenger matrixmessenger.Messenger
	config    *Config
	scheduler gocron.Scheduler
	wg        *sync.WaitGroup

	latestDiffs *shiftDiffs
}

// Config holds the configuration for the shift notifier.
type Config struct {
	LocationNames          []string
	NotifyBeforeShiftStart time.Duration
	MatrixRoomID           string

	ListenAddr string
	Token      string
}

// ParseFromEnvironment parses the config from the environment.
func (c *Config) ParseFromEnvironment() {
	c.LocationNames = strings.Split(os.Getenv("TROLLINFO_LOCATIONS"), ",")
	c.MatrixRoomID = os.Getenv("TROLLINFO_MATRIX_ROOM_ID")
	c.NotifyBeforeShiftStart = time.Minute * 15
	c.ListenAddr = os.Getenv("TROLLINFO_HTTP_LISTEN_ADDR")
	c.Token = os.Getenv("TROLLINFO_HTTP_TOKEN")
}

// New assembles a new shift notifier.
func New(config *Config, angelAPI angelapi.Service, messenger matrixmessenger.Messenger) Service {
	s := &service{
		angelAPI:  angelAPI,
		messenger: messenger,
		config:    config,
		wg:        &sync.WaitGroup{},
	}

	http.HandleFunc("/data", s.serveJSONData)
	http.HandleFunc("/", s.serveHumanPortrait)
	http.HandleFunc("/landscape", s.serveHumanLandscape)

	return s
}

func (service *service) Start() error {
	go func(config Config) {
		err := http.ListenAndServe(config.ListenAddr, nil)
		if err != nil {
			slog.Error("failed serving HTTP server", "error", err)
		}
	}(*service.config)

	s, err := gocron.NewScheduler()
	if err != nil {
		return err
	}

	service.scheduler = s

	_, err = service.scheduler.NewJob(
		// Always 14 Minutes before full hour.
		gocron.CronJob("46 * * * *", false),
		gocron.NewTask(service.notifyShifts),
	)
	if err != nil {
		return err
	}

	// If we are between XX:46 and XX:59 get the diffs now!
	if time.Now().Minute() > 46 {
		_ = service.getNextShifts()
	}

	// Start the scheduler.
	service.wg.Add(1)
	service.scheduler.Start()
	slog.Info("started notifier", "jobs", len(service.scheduler.Jobs()))

	service.wg.Wait()
	return nil
}

func (service *service) Stop() error {
	err := service.scheduler.Shutdown()
	service.wg.Done()

	return err
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

func (service *service) notifyShifts() {
	deadline := time.Now().Add(time.Minute * 4)
	var err error
	for time.Until(deadline) > 0 {
		err = service.getNextShifts()
		if err == nil {
			break
		}
	}
	if err != nil {
		slog.Error("failed to run notifier", "error", err.Error())
	}
}

func (service *service) getNextShifts() error {
	slog.Info("checking shifts now")

	locations, err := service.getLocationIDs()
	if err != nil {
		slog.Error("failed to list locations", "error", err.Error())
		return err
	}

	diffs := map[string]shiftDiff{}

	// Use time.Date(2024, 5, 31, 19, 54, 0, 0, time.UTC) for testing.
	refTime := time.Now()

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

	service.latestDiffs = &shiftDiffs{
		DiffsInLocations: service.cleanUpDiffs(diffs),
		ReferenceTime:    refTime,
	}

	// Only send message if we have any content.
	hasContent := false
	for _, diff := range service.latestDiffs.DiffsInLocations {
		if len(diff.UsersArriving) > 0 || len(diff.UsersLeaving) > 0 {
			hasContent = true
			break
		}
	}

	if !hasContent {
		return nil
	}

	msg, msgFormatted := service.diffToMessage(service.latestDiffs)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = service.messenger.SendMessage(ctx, matrixmessenger.HTMLMessage(
		msg, msgFormatted, service.config.MatrixRoomID,
	))
	if err != nil {
		slog.Error("failed to send matrix message", "error", err.Error())
		return err
	}

	return nil
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
