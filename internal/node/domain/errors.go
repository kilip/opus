package domain

import "errors"

var (
	// ErrTokenNotFound indicates the join token does not exist.
	ErrTokenNotFound = errors.New("JOIN_TOKEN_NOT_FOUND")
	// ErrTokenExpired indicates the join token has expired.
	ErrTokenExpired = errors.New("JOIN_TOKEN_EXPIRED")
	// ErrTokenAlreadyUsed indicates the join token has already been consumed.
	ErrTokenAlreadyUsed = errors.New("JOIN_TOKEN_ALREADY_USED")
	// ErrNodeNameConflict indicates the node name is already registered.
	ErrNodeNameConflict = errors.New("NODE_NAME_CONFLICT")
	// ErrInvalidNodeName indicates the node name is empty or does not meet criteria.
	ErrInvalidNodeName = errors.New("INVALID_NODE_NAME")
	// ErrInvalidIPAddress indicates the IP address is not a valid IPv4 or IPv6 address.
	ErrInvalidIPAddress = errors.New("INVALID_IP_ADDRESS")
	// ErrInvalidMACAddress indicates the MAC address is not a valid hardware address.
	ErrInvalidMACAddress = errors.New("INVALID_MAC_ADDRESS")
	// ErrNodeMACNotConfigured indicates the node has no valid MAC address configured for Wake-on-LAN.
	ErrNodeMACNotConfigured = errors.New("NODE_MAC_NOT_CONFIGURED")
	// ErrRateLimited indicates too many join requests were sent.
	ErrRateLimited = errors.New("RATE_LIMITED")
	// ErrNodeNotConnected indicates the node does not have an active connection.
	ErrNodeNotConnected = errors.New("NODE_NOT_CONNECTED")
	// ErrNodeNotFound indicates the node does not exist.
	ErrNodeNotFound = errors.New("NODE_NOT_FOUND")
	// ErrMasterConnectForbidden indicates the master node cannot connect via worker control channel.
	ErrMasterConnectForbidden = errors.New("MASTER_CONNECT_FORBIDDEN")
	// ErrMasterDeleteForbidden indicates the master node cannot be deleted.
	ErrMasterDeleteForbidden = errors.New("MASTER_DELETE_FORBIDDEN")
	// ErrUnauthorized indicates missing or invalid credentials.
	ErrUnauthorized = errors.New("UNAUTHORIZED")
	// ErrUnknownEnvelopeType indicates an unrecognised envelope type.
	ErrUnknownEnvelopeType = errors.New("UNKNOWN_ENVELOPE_TYPE")
)
