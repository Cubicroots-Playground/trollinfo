package matrixmessenger

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/CubicrootXYZ/gologger"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type service struct {
	roomUserCache roomCache
	config        *Config
	client        MatrixClient
	logger        gologger.Logger
	state         *state
}

type Config struct {
	Username   string
	Password   string
	Homeserver string
	DeviceID   string
}

// ParseFromEnvironment parses the config from the environment.
func (c *Config) ParseFromEnvironment() {
	c.Username = os.Getenv("TROLLINFO_MATRIX_USERNAME")
	c.Password = os.Getenv("TROLLINFO_MATRIX_PASSWORD")
	c.Homeserver = os.Getenv("TROLLINFO_MATRIX_HOMESERVER")
	c.DeviceID = os.Getenv("TROLLINFO_MATRIX_DEVICE_ID")
}

type state struct {
	rateLimitedUntil      time.Time // If we run into a rate limit this will tell us to stop operation
	rateLimitedUntilMutex sync.Mutex
}

func NewMessenger(config *Config, logger gologger.Logger) (Messenger, error) {
	s := &service{
		roomUserCache: make(roomCache),
		config:        config,
		logger:        logger,
		state: &state{
			rateLimitedUntilMutex: sync.Mutex{},
		},
	}

	err := s.setupMautrixClient()

	return s, err
}

func (service *service) setupMautrixClient() error {
	service.logger.Debugf("setting up mautrix client ...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	client, err := mautrix.NewClient(service.config.Homeserver, "", "")
	if err != nil {
		return err
	}

	service.client = client

	_, err = client.Login(ctx, &mautrix.ReqLogin{
		Type:             "m.login.password",
		Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: service.config.Username},
		Password:         service.config.Password,
		DeviceID:         id.DeviceID(service.config.DeviceID),
		StoreCredentials: true,
	})

	service.logger.Debugf("matrix client setup finished")
	return err
}

// sendMessageEvent sends a message event to matrix, will take care of encryption if available
func (messenger *service) sendMessageEvent(ctx context.Context, messageEvent *messageEvent, roomID string, eventType event.Type) (*mautrix.RespSendEvent, error) {
	messenger.logger.Infof("Sending message to room %s", roomID)
	return messenger.client.SendMessageEvent(ctx, id.RoomID(roomID), eventType, &messageEvent)
}

// enrichCleartext adds parts of the encrypted event back into the cleartext event as specified by the matrix spec
func enrichCleartext(encryptedEvent *event.EncryptedEventContent, evt *messageEvent) {
	if evt.RelatesTo.EventID == "" && evt.RelatesTo.InReplyTo == nil {
		return
	}

	encryptedEvent.RelatesTo = &event.RelatesTo{}
	encryptedEvent.RelatesTo.EventID = id.EventID(evt.RelatesTo.EventID)
	encryptedEvent.RelatesTo.Key = evt.RelatesTo.Key
	encryptedEvent.RelatesTo.Type = event.RelationType(evt.RelatesTo.RelType)

	if evt.RelatesTo.InReplyTo != nil {
		encryptedEvent.RelatesTo.InReplyTo = &event.InReplyTo{
			EventID: id.EventID(evt.RelatesTo.InReplyTo.EventID),
		}
	}
}

func (messenger *service) getUserIDsInRoom(ctx context.Context, roomID id.RoomID) []id.UserID {
	// Check cache first
	if users := messenger.roomUserCache.GetUsers(roomID); users != nil {
		return users
	}

	userIDs := make([]id.UserID, 0)
	members, err := messenger.client.JoinedMembers(ctx, roomID)
	if err != nil {
		messenger.logger.Err(err)
		return userIDs
	}

	i := 0
	for userID := range members.Joined {
		userIDs = append(userIDs, userID)
		i++
	}

	messenger.roomUserCache.AddUsers(roomID, userIDs)
	return userIDs
}

func (messenger *service) encounteredRateLimit() {
	messenger.state.rateLimitedUntilMutex.Lock()
	messenger.state.rateLimitedUntil = time.Now().Add(time.Minute)
	messenger.state.rateLimitedUntilMutex.Unlock()
}
