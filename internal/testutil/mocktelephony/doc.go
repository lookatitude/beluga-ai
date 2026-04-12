// Package mocktelephony provides an in-memory SIPEndpoint implementation for
// testing code that depends on the voice/telephony package.
//
// It is an internal test utility that records all calls for assertion and
// supports configurable error injection. Used by the voice subsystem tests
// in the Beluga framework.
//
// This package is internal and must not be imported outside of this module.
package mocktelephony
