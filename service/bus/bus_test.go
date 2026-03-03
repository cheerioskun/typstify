package bus

import (
	"context"
	"testing"
	"time"
)

func TestEmit(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), <-time.After(time.Second*3))
	defer cancel()

	eb := NewEventBus(ctx, false)

	var tag int
	eb.Subscribe(&tag, "testclient", "anytopic|anothertopic", func(topic string, data interface{}) {
		t.Log("executing client callback", topic, data)
	})

	eb.Emit("anytopic", "anydata")
	eb.Emit("anothertopic", "anydata2")

	// time.Sleep(time.Second * 5)
}
