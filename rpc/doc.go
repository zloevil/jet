// Package rpc defines a request/response RPC protocol carried over Kafka.
//
// It contains the message types, the request pool that correlates responses to
// pending requests, and the Client and Server interfaces. The transport
// implementations live in the client and server sub-packages.
package rpc
