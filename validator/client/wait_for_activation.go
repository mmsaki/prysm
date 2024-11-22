package client

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/math"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	octrace "go.opentelemetry.io/otel/trace"
)

// WaitForActivation checks whether the validator pubkey is in the active
// validator set. If not, this operation will block until an activation message is
// received. This method also monitors the keymanager for updates while waiting for an activation
// from the gRPC server.
//
// If the channel parameter is nil, WaitForActivation creates and manages its own channel.
func (v *validator) WaitForActivation(ctx context.Context, accountsChangedChan chan [][fieldparams.BLSPubkeyLength]byte) error {
	// Monitor the key manager for updates.
	if accountsChangedChan == nil {
		accountsChangedChan = make(chan [][fieldparams.BLSPubkeyLength]byte, 1)
		km, err := v.Keymanager()
		if err != nil {
			return err
		}
		// subscribe to the channel if it's the first time
		sub := km.SubscribeAccountChanges(accountsChangedChan)
		defer func() {
			sub.Unsubscribe()
			close(accountsChangedChan)
		}()
	}
	return v.internalWaitForActivation(ctx, accountsChangedChan)
}

// internalWaitForActivation performs the following:
// 1) While the key manager is empty, subscribe to keymanager changes until some validator keys exist.
// 2) Open a server side stream for activation events against the given keys.
// 3) In another go routine, the key manager is monitored for updates and emits an update event on
// the accountsChangedChan. When an event signal is received, restart the internalWaitForActivation routine.
// 4) If the stream is reset in error, restart the routine.
// 5) If the stream returns a response indicating one or more validators are active, exit the routine.
func (v *validator) internalWaitForActivation(ctx context.Context, accountsChangedChan <-chan [][fieldparams.BLSPubkeyLength]byte) error {
	ctx, span := trace.StartSpan(ctx, "validator.WaitForActivation")
	defer span.End()
	validatingKeys, err := v.km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, msgCouldNotFetchKeys)
	}
	// if there are no validating keys, wait for some
	if len(validatingKeys) == 0 {
		log.Warn(msgNoKeysFetched)
		return v.waitForAccountsChange(ctx, accountsChangedChan)
	}

	stream, err := v.validatorClient.WaitForActivation(ctx, &ethpb.ValidatorActivationRequest{
		PublicKeys: bytesutil.FromBytes48Array(validatingKeys),
	})
	if err != nil {
		return v.retryWaitForActivation(ctx, span, err, "Connection broken while waiting for activation. Reconnecting...", accountsChangedChan)
	}

	someAreActive := false
	for !someAreActive {
		select {
		case <-ctx.Done():
			log.Debug("Context closed, exiting fetching validating keys")
			return ctx.Err()
		case <-accountsChangedChan:
			// Accounts (keys) changed, restart the process.
			return v.internalWaitForActivation(ctx, accountsChangedChan)
		default:
			res, err := (stream).Recv() // retrieve from stream one loop at a time
			// If the stream is closed, we stop the loop.
			if errors.Is(err, io.EOF) {
				break
			}
			// If context is canceled we return from the function.
			if errors.Is(ctx.Err(), context.Canceled) {
				return errors.Wrap(ctx.Err(), "context has been canceled so shutting down the loop")
			}
			if err != nil {
				return v.retryWaitForActivation(ctx, span, err, "Connection broken while waiting for activation. Reconnecting...", accountsChangedChan)
			}

			for _, s := range res.Statuses {
				v.pubkeyToStatus[bytesutil.ToBytes48(s.PublicKey)] = &validatorStatus{
					publicKey: s.PublicKey,
					status:    s.Status,
					index:     s.Index,
				}
			}

			someAreActive = v.checkAndLogValidatorStatus()
		}
	}

	return nil
}

func (v *validator) retryWaitForActivation(ctx context.Context, span octrace.Span, err error, message string, accountsChangedChan <-chan [][fieldparams.BLSPubkeyLength]byte) error {
	tracing.AnnotateError(span, err)
	attempts := activationAttempts(ctx)
	log.WithError(err).WithField("attempts", attempts).Error(message)
	// Reconnection attempt backoff, up to 60s.
	time.Sleep(time.Second * time.Duration(math.Min(uint64(attempts), 60)))
	// TODO: refactor this to use the health tracker instead for reattempt
	return v.internalWaitForActivation(incrementRetries(ctx), accountsChangedChan)
}

func (v *validator) waitForAccountsChange(ctx context.Context, accountsChangedChan <-chan [][fieldparams.BLSPubkeyLength]byte) error {
	select {
	case <-ctx.Done():
		log.Debug("Context closed, exiting waitForAccountsChange")
		return ctx.Err()
	case <-accountsChangedChan:
		// If the accounts changed, try again.
		return v.internalWaitForActivation(ctx, accountsChangedChan)
	}
}

// Preferred way to use context keys is with a non built-in type. See: RVV-B0003
type waitForActivationContextKey string

const waitForActivationAttemptsContextKey = waitForActivationContextKey("WaitForActivation-attempts")

func activationAttempts(ctx context.Context) int {
	attempts, ok := ctx.Value(waitForActivationAttemptsContextKey).(int)
	if !ok {
		return 1
	}
	return attempts
}

func incrementRetries(ctx context.Context) context.Context {
	attempts := activationAttempts(ctx)
	return context.WithValue(ctx, waitForActivationAttemptsContextKey, attempts+1)
}
