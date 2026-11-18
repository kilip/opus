export const NODE_ERRORS: Record<string, string> = {
  NODE_NOT_FOUND: "The requested node was not found.",
  MASTER_DELETE_FORBIDDEN: "Master node cannot be deleted.",
  NODE_NAME_CONFLICT: "A node with this name already exists.",
  INVALID_NODE_NAME:
    "Node name must be 1–63 characters using letters, numbers, dots, underscores, or hyphens.",
  INVALID_IP_ADDRESS: "Invalid IPv4 or IPv6 address.",
  INVALID_MAC_ADDRESS: "Invalid MAC address (e.g. aa:bb:cc:dd:ee:ff).",
  NODE_MAC_NOT_CONFIGURED:
    "This node does not have a valid MAC address configured.",
  JOIN_TOKEN_EXPIRED: "Join token has expired.",
  JOIN_TOKEN_ALREADY_USED: "Join token has already been used.",
};
