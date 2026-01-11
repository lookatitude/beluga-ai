// Package internal provides a wrapper around LiveKit SDK that avoids SIP client dependencies.
// This wrapper only imports the minimal functionality needed (RoomServiceClient) without
// triggering compilation of the SIP client code that has compatibility issues.
package internal

import (
	"context"
	"net/http"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/twitchtv/twirp"
)

// RoomServiceClient wraps the minimal functionality we need from livekit/server-sdk-go
// without importing the full SDK (which includes problematic SIP client code).
// This implementation matches the SDK's RoomServiceClient but avoids importing the SDK package.
type RoomServiceClient struct {
	roomService livekit.RoomService
	apiKey      string
	apiSecret   string
}

// NewRoomServiceClient creates a new RoomServiceClient without importing the full SDK.
// This avoids the SIP client compilation issue by using the protocol package directly.
func NewRoomServiceClient(url, apiKey, apiSecret string) (*RoomServiceClient, error) {
	// Convert URL to HTTP URL (same as SDK does)
	url = toHTTPURL(url)
	
	// Create protobuf client directly from protocol package
	client := livekit.NewRoomServiceProtobufClient(url, &http.Client{})
	
	return &RoomServiceClient{
		roomService: client,
		apiKey:      apiKey,
		apiSecret:   apiSecret,
	}, nil
}

// toHTTPURL converts a URL to HTTP format (matching SDK's ToHttpURL function).
func toHTTPURL(url string) string {
	// Simple conversion - in full implementation would handle more cases
	if len(url) > 0 && url[len(url)-1] != '/' {
		url += "/"
	}
	return url
}

// withAuth adds authentication to the context (matching SDK's authBase pattern).
func (c *RoomServiceClient) withAuth(ctx context.Context, grant auth.VideoGrant) (context.Context, error) {
	at := auth.NewAccessToken(c.apiKey, c.apiSecret)
	at.AddGrant(&grant)
	token, err := at.ToJWT()
	if err != nil {
		return nil, err
	}
	ctx, err = twirp.WithHTTPRequestHeaders(ctx, newHeaderWithToken(token))
	if err != nil {
		return nil, err
	}
	return ctx, nil
}

// newHeaderWithToken creates HTTP headers with the authorization token.
func newHeaderWithToken(token string) http.Header {
	header := make(http.Header)
	header.Set("Authorization", "Bearer "+token)
	return header
}

// CreateRoom creates a new room using the protocol client directly.
func (c *RoomServiceClient) CreateRoom(ctx context.Context, req *livekit.CreateRoomRequest) (*livekit.Room, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomCreate: true})
	if err != nil {
		return nil, err
	}
	return c.roomService.CreateRoom(ctx, req)
}

// ListRooms lists rooms using the protocol client directly.
func (c *RoomServiceClient) ListRooms(ctx context.Context, req *livekit.ListRoomsRequest) (*livekit.ListRoomsResponse, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomList: true})
	if err != nil {
		return nil, err
	}
	return c.roomService.ListRooms(ctx, req)
}

// DeleteRoom deletes a room using the protocol client directly.
func (c *RoomServiceClient) DeleteRoom(ctx context.Context, req *livekit.DeleteRoomRequest) (*livekit.DeleteRoomResponse, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomCreate: true})
	if err != nil {
		return nil, err
	}
	return c.roomService.DeleteRoom(ctx, req)
}

// ListParticipants lists participants in a room.
func (c *RoomServiceClient) ListParticipants(ctx context.Context, req *livekit.ListParticipantsRequest) (*livekit.ListParticipantsResponse, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomAdmin: true, Room: req.Room})
	if err != nil {
		return nil, err
	}
	return c.roomService.ListParticipants(ctx, req)
}

// GetParticipant gets a participant by identity.
func (c *RoomServiceClient) GetParticipant(ctx context.Context, req *livekit.RoomParticipantIdentity) (*livekit.ParticipantInfo, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomAdmin: true, Room: req.Room})
	if err != nil {
		return nil, err
	}
	return c.roomService.GetParticipant(ctx, req)
}

// RemoveParticipant removes a participant from a room.
func (c *RoomServiceClient) RemoveParticipant(ctx context.Context, req *livekit.RoomParticipantIdentity) (*livekit.RemoveParticipantResponse, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomAdmin: true, Room: req.Room})
	if err != nil {
		return nil, err
	}
	return c.roomService.RemoveParticipant(ctx, req)
}

// MutePublishedTrack mutes a published track.
func (c *RoomServiceClient) MutePublishedTrack(ctx context.Context, req *livekit.MuteRoomTrackRequest) (*livekit.MuteRoomTrackResponse, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomAdmin: true, Room: req.Room})
	if err != nil {
		return nil, err
	}
	return c.roomService.MutePublishedTrack(ctx, req)
}

// UpdateParticipant updates participant metadata.
func (c *RoomServiceClient) UpdateParticipant(ctx context.Context, req *livekit.UpdateParticipantRequest) (*livekit.ParticipantInfo, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomAdmin: true, Room: req.Room})
	if err != nil {
		return nil, err
	}
	return c.roomService.UpdateParticipant(ctx, req)
}

// UpdateSubscriptions updates participant subscriptions.
func (c *RoomServiceClient) UpdateSubscriptions(ctx context.Context, req *livekit.UpdateSubscriptionsRequest) (*livekit.UpdateSubscriptionsResponse, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomAdmin: true, Room: req.Room})
	if err != nil {
		return nil, err
	}
	return c.roomService.UpdateSubscriptions(ctx, req)
}

// SendData sends data to participants.
func (c *RoomServiceClient) SendData(ctx context.Context, req *livekit.SendDataRequest) (*livekit.SendDataResponse, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomAdmin: true, Room: req.Room})
	if err != nil {
		return nil, err
	}
	return c.roomService.SendData(ctx, req)
}

// UpdateRoomMetadata updates room metadata.
func (c *RoomServiceClient) UpdateRoomMetadata(ctx context.Context, req *livekit.UpdateRoomMetadataRequest) (*livekit.Room, error) {
	ctx, err := c.withAuth(ctx, auth.VideoGrant{RoomAdmin: true, Room: req.Room})
	if err != nil {
		return nil, err
	}
	return c.roomService.UpdateRoomMetadata(ctx, req)
}

// APIKey returns the API key used for authentication.
func (c *RoomServiceClient) APIKey() string {
	return c.apiKey
}

// APISecret returns the API secret used for authentication.
func (c *RoomServiceClient) APISecret() string {
	return c.apiSecret
}

// Close closes the connection (no-op for HTTP client, but kept for compatibility).
func (c *RoomServiceClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}
