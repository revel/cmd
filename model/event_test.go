package model_test

import (
	"testing"

	"github.com/revel/revel"
	"github.com/stretchr/testify/assert"
)

// Test that the event handler can be attached and it dispatches the event received.
func TestEventHandler(t *testing.T) {
	counter := 0
	newListener := func(typeOf revel.Event, value interface{}) (responseOf revel.EventResponse) {
		if typeOf == revel.ENGINE_SHUTDOWN_REQUEST {
			counter++
		}
		return
	}
	// Attach the same handler twice so we expect to see the response twice as well
	revel.AddInitEventHandler(newListener)
	revel.AddInitEventHandler(newListener)
	revel.StopServer(1)
	assert.Equal(t, counter, 2, "Expected event handler to have been called")
}
