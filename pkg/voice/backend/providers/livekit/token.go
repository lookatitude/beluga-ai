package livekit

import (
	"context"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	lksdkwrapper "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit/internal"
)

// GenerateAccessToken generates a JWT access token for LiveKit authentication.
func GenerateAccessToken(config *LiveKitConfig, roomName, participantIdentity string, grant *auth.VideoGrant) (string, error) {
	at := auth.NewAccessToken(config.APIKey, config.APISecret)
	at.AddGrant(grant).
		SetIdentity(participantIdentity).
		SetName(participantIdentity).
		SetValidFor(24 * time.Hour)

	if roomName != "" {
		at.SetMetadata("room:" + roomName)
	}

	return at.ToJWT()
}

// GenerateAccessTokenWithHook generates a JWT access token with custom auth hook support.
func GenerateAccessTokenWithHook(ctx context.Context, config *LiveKitConfig, roomName, participantIdentity string, authHook iface.AuthHook) (string, error) {
	// If auth hook is provided, use it for authentication
	if authHook != nil && config.AuthHook != nil {
		// Get token from session config or generate default
		token := "" // TODO: Get from session config

		// Authenticate using hook
		authResult, err := authHook.Authenticate(ctx, token, map[string]any{
			"room":     roomName,
			"identity": participantIdentity,
		})
		if err != nil {
			return "", backend.NewBackendError("GenerateAccessTokenWithHook", backend.ErrCodeAuthenticationFailed, err)
		}

		if !authResult.Authorized {
			return "", backend.NewBackendError("GenerateAccessTokenWithHook", backend.ErrCodeAuthorizationFailed,
				nil)
		}

		// Use authenticated user ID
		participantIdentity = authResult.UserID
	}

	// Generate token with standard grant
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}

	return GenerateAccessToken(config, roomName, participantIdentity, grant)
}

// CreateTokenFromRoomService creates a token using the room service client configuration.
// Note: The wrapper doesn't have CreateToken method, so we generate it directly.
func CreateTokenFromRoomService(roomService *lksdkwrapper.RoomServiceClient, roomName, participantIdentity string) *auth.AccessToken {
	// Generate token directly using auth package since wrapper doesn't expose CreateToken
	at := auth.NewAccessToken(roomService.APIKey(), roomService.APISecret())
	return at.
		SetIdentity(participantIdentity).
		SetName(participantIdentity).
		AddGrant(&auth.VideoGrant{
			RoomJoin: true,
			Room:     roomName,
		})
}
