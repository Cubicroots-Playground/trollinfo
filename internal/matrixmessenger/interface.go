package matrixmessenger

import (
	"context"
	"errors"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

//go:generate mockgen -destination=service_mock.go -package=matrixmessenger . Messenger

// Messenger provides an interface for sending and handling matrix events
// The Queue... methods work asynchronous but do not provide any information about the sent message
// The Send... methods work snchronous and provide more detailed feedback, they can be used asynchronous
type Messenger interface {
	SendMessageAsync(ctx context.Context, message *Message) error
	SendMessage(ctx context.Context, message *Message) (*MessageResponse, error)
	CreateChannel(ctx context.Context, userID string) (*ChannelResponse, error)
	// TODO future improvement: make room member cache flushable throug this interface and flush it on room member updates
}

// Errors returned by the messenger
var (
	ErrRetriesExceeded = errors.New("amount of retries exceeded")
)

// MatrixClient defines an interface to wrap the matrix API
type MatrixClient interface {
	SendMessageEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, contentJSON interface{}, extra ...mautrix.ReqSendEvent) (resp *mautrix.RespSendEvent, err error)
	RedactEvent(ctx context.Context, roomID id.RoomID, eventID id.EventID, extra ...mautrix.ReqRedact) (resp *mautrix.RespSendEvent, err error)
	JoinedMembers(ctx context.Context, roomID id.RoomID) (resp *mautrix.RespJoinedMembers, err error)
	CreateRoom(ctx context.Context, req *mautrix.ReqCreateRoom) (resp *mautrix.RespCreateRoom, err error)
}
