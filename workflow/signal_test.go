package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemorySignalChannel_Send(t *testing.T) {
	tests := []struct {
		name       string
		workflowID string
		signal     Signal
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "send valid signal",
			workflowID: "workflow-1",
			signal: Signal{
				Name:     "approval",
				Payload:  "approved",
				SenderID: "user-1",
			},
			wantErr: false,
		},
		{
			name:       "send signal without sender",
			workflowID: "workflow-1",
			signal: Signal{
				Name:    "approval",
				Payload: "approved",
			},
			wantErr: false,
		},
		{
			name:       "empty workflow ID",
			workflowID: "",
			signal: Signal{
				Name:    "approval",
				Payload: "approved",
			},
			wantErr: true,
			errMsg:  "workflowID cannot be empty",
		},
		{
			name:       "empty signal name",
			workflowID: "workflow-1",
			signal: Signal{
				Name:    "",
				Payload: "approved",
			},
			wantErr: true,
			errMsg:  "signal name cannot be empty",
		},
		{
			name:       "nil payload",
			workflowID: "workflow-1",
			signal: Signal{
				Name:    "event",
				Payload: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewInMemorySignalChannel()
			err := sc.Send(context.Background(), tt.workflowID, tt.signal)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInMemorySignalChannel_Receive(t *testing.T) {
	tests := []struct {
		name       string
		workflowID string
		signalName string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "empty workflow ID",
			workflowID: "",
			signalName: "approval",
			wantErr:    true,
			errMsg:     "workflowID cannot be empty",
		},
		{
			name:       "empty signal name",
			workflowID: "workflow-1",
			signalName: "",
			wantErr:    true,
			errMsg:     "signalName cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewInMemorySignalChannel()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_, err := sc.Receive(ctx, tt.workflowID, tt.signalName)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestInMemorySignalChannel_SendAndReceive(t *testing.T) {
	t.Run("receive signal sent before receive call", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		workflowID := "workflow-1"
		signal := Signal{
			Name:     "approval",
			Payload:  "approved",
			SenderID: "user-1",
		}

		// Send signal first
		err := sc.Send(context.Background(), workflowID, signal)
		require.NoError(t, err)

		// Then receive it
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		received, err := sc.Receive(ctx, workflowID, "approval")
		require.NoError(t, err)
		require.NotNil(t, received)
		assert.Equal(t, "approval", received.Name)
		assert.Equal(t, "approved", received.Payload)
		assert.Equal(t, "user-1", received.SenderID)
		assert.False(t, received.SentAt.IsZero())
	})

	t.Run("receive signal sent after receive starts", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		workflowID := "workflow-1"
		signal := Signal{
			Name:     "approval",
			Payload:  "approved",
			SenderID: "user-1",
		}

		receiveCh := make(chan *Signal, 1)
		errCh := make(chan error, 1)

		// Start receive in a goroutine
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			sig, err := sc.Receive(ctx, workflowID, "approval")
			if err != nil {
				errCh <- err
				return
			}
			receiveCh <- sig
		}()

		// Give the goroutine time to start waiting
		time.Sleep(100 * time.Millisecond)

		// Send signal
		err := sc.Send(context.Background(), workflowID, signal)
		require.NoError(t, err)

		// Check that receive got the signal
		select {
		case received := <-receiveCh:
			require.NotNil(t, received)
			assert.Equal(t, "approval", received.Name)
			assert.Equal(t, "approved", received.Payload)
		case err := <-errCh:
			t.Fatalf("unexpected error: %v", err)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for signal")
		}
	})
}

func TestInMemorySignalChannel_ContextCancellation(t *testing.T) {
	t.Run("receive cancelled by context", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel immediately
		cancel()

		received, err := sc.Receive(ctx, "workflow-1", "approval")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Nil(t, received)
	})

	t.Run("receive deadline exceeded", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		received, err := sc.Receive(ctx, "workflow-1", "approval")
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Nil(t, received)
	})
}

func TestInMemorySignalChannel_MultipleSignals(t *testing.T) {
	t.Run("receive multiple signals in order", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		workflowID := "workflow-1"

		// Send multiple signals
		signals := []Signal{
			{Name: "approval", Payload: "approved-1", SenderID: "user-1"},
			{Name: "approval", Payload: "approved-2", SenderID: "user-2"},
			{Name: "approval", Payload: "approved-3", SenderID: "user-3"},
		}

		for _, sig := range signals {
			err := sc.Send(context.Background(), workflowID, sig)
			require.NoError(t, err)
		}

		// Receive them in order
		for i, expectedSig := range signals {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			received, err := sc.Receive(ctx, workflowID, "approval")
			cancel()

			require.NoError(t, err, "signal %d", i)
			require.NotNil(t, received, "signal %d", i)
			assert.Equal(t, expectedSig.Payload, received.Payload, "signal %d", i)
			assert.Equal(t, expectedSig.SenderID, received.SenderID, "signal %d", i)
		}

		// Fourth receive should timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		received, err := sc.Receive(ctx, workflowID, "approval")
		assert.Error(t, err)
		assert.Nil(t, received)
	})

	t.Run("different signal names are independent", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		workflowID := "workflow-1"

		// Send signals with different names
		err := sc.Send(context.Background(), workflowID, Signal{
			Name:    "approval",
			Payload: "approved",
		})
		require.NoError(t, err)

		err = sc.Send(context.Background(), workflowID, Signal{
			Name:    "cancellation",
			Payload: "cancelled",
		})
		require.NoError(t, err)

		// Receive approval signal
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		received, err := sc.Receive(ctx, workflowID, "approval")
		cancel()

		require.NoError(t, err)
		assert.Equal(t, "approved", received.Payload)

		// Receive cancellation signal
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		received, err = sc.Receive(ctx, workflowID, "cancellation")
		cancel()

		require.NoError(t, err)
		assert.Equal(t, "cancelled", received.Payload)
	})
}

func TestInMemorySignalChannel_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent sends to same workflow", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		workflowID := "workflow-1"
		numGoroutines := 10
		signalsPerGoroutine := 5

		done := make(chan struct{})
		errCh := make(chan error, numGoroutines*signalsPerGoroutine)

		// Spawn goroutines to send signals concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < signalsPerGoroutine; j++ {
					signal := Signal{
						Name:     "approval",
						Payload:  id*1000 + j,
						SenderID: "",
					}
					err := sc.Send(context.Background(), workflowID, signal)
					if err != nil {
						errCh <- err
					}
				}
				done <- struct{}{}
			}(i)
		}

		// Wait for all goroutines to finish
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Check for any errors
		close(errCh)
		for err := range errCh {
			require.NoError(t, err)
		}

		// Verify we can receive all signals
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		receivedCount := 0
		for {
			received, err := sc.Receive(ctx, workflowID, "approval")
			if err != nil {
				// Timeout is expected when all signals are consumed
				if err == context.DeadlineExceeded {
					break
				}
				t.Fatalf("unexpected error: %v", err)
			}
			require.NotNil(t, received)
			receivedCount++
		}

		assert.Equal(t, numGoroutines*signalsPerGoroutine, receivedCount)
	})

	t.Run("concurrent sends and receives", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		workflowID := "workflow-1"

		sendDone := make(chan struct{})
		receiveDone := make(chan struct{})
		errCh := make(chan error, 20)

		// Goroutine that sends signals
		go func() {
			for i := 0; i < 10; i++ {
				signal := Signal{
					Name:     "approval",
					Payload:  i,
					SenderID: "sender",
				}
				err := sc.Send(context.Background(), workflowID, signal)
				if err != nil {
					errCh <- err
				}
				time.Sleep(10 * time.Millisecond)
			}
			sendDone <- struct{}{}
		}()

		// Goroutine that receives signals
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			for i := 0; i < 10; i++ {
				received, err := sc.Receive(ctx, workflowID, "approval")
				if err != nil {
					errCh <- err
					break
				}
				if received == nil {
					errCh <- context.DeadlineExceeded
					break
				}
			}
			receiveDone <- struct{}{}
		}()

		// Wait for both goroutines
		<-sendDone
		<-receiveDone

		// Check for errors
		close(errCh)
		for err := range errCh {
			require.NoError(t, err)
		}
	})
}

func TestInMemorySignalChannel_SignalTimestamp(t *testing.T) {
	t.Run("signal gets timestamp on send", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		workflowID := "workflow-1"

		beforeSend := time.Now()
		signal := Signal{
			Name:    "approval",
			Payload: "approved",
		}

		err := sc.Send(context.Background(), workflowID, signal)
		require.NoError(t, err)
		afterSend := time.Now()

		// Receive the signal
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		received, err := sc.Receive(ctx, workflowID, "approval")
		require.NoError(t, err)

		// Verify timestamp is set and within the expected range
		assert.False(t, received.SentAt.IsZero())
		assert.True(t, received.SentAt.After(beforeSend) || received.SentAt.Equal(beforeSend))
		assert.True(t, received.SentAt.Before(afterSend) || received.SentAt.Equal(afterSend))
	})

	t.Run("preserves explicit timestamp", func(t *testing.T) {
		sc := NewInMemorySignalChannel()
		workflowID := "workflow-1"

		fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		signal := Signal{
			Name:     "approval",
			Payload:  "approved",
			SentAt:   fixedTime,
			SenderID: "user-1",
		}

		err := sc.Send(context.Background(), workflowID, signal)
		require.NoError(t, err)

		// Receive the signal
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		received, err := sc.Receive(ctx, workflowID, "approval")
		require.NoError(t, err)

		// Verify the explicit timestamp is preserved
		assert.Equal(t, fixedTime, received.SentAt)
	})
}

func TestInMemorySignalChannel_IsolationBetweenWorkflows(t *testing.T) {
	t.Run("signals isolated by workflow ID", func(t *testing.T) {
		sc := NewInMemorySignalChannel()

		// Send signals to different workflows
		err := sc.Send(context.Background(), "workflow-1", Signal{
			Name:    "approval",
			Payload: "approved-1",
		})
		require.NoError(t, err)

		err = sc.Send(context.Background(), "workflow-2", Signal{
			Name:    "approval",
			Payload: "approved-2",
		})
		require.NoError(t, err)

		// Receive from workflow-1
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		received1, err := sc.Receive(ctx, "workflow-1", "approval")
		cancel()

		require.NoError(t, err)
		assert.Equal(t, "approved-1", received1.Payload)

		// Receive from workflow-2
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		received2, err := sc.Receive(ctx, "workflow-2", "approval")
		cancel()

		require.NoError(t, err)
		assert.Equal(t, "approved-2", received2.Payload)
	})
}
