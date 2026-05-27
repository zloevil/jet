// Package notification builds notification receivers from permission and
// resource policies.
//
// Receivers are resolved at send time from a pluggable policy Resolver (for
// example an RBAC service), then filtered by channel type, producing a typed
// Notification ready to be dispatched.
package notification
