package daemon

import (
	"errors"
	"fmt"
	"github.com/devgianlu/go-librespot/login5"
	login5pb "github.com/devgianlu/go-librespot/proto/spotify/login5/v3"
	"github.com/devgianlu/go-librespot/session"
	"testing"
)

func TestReauthorizeOnlyRejectedDeviceCredentials(t *testing.T) {
	rejected := fmt.Errorf("session: %w", &login5.LoginError{Code: login5pb.LoginError_INVALID_CREDENTIALS})
	if !retryDeviceAuthorization(session.DeviceAuthCredentials{}, rejected) {
		t.Fatal("rejected stored login must allow device pairing")
	}
	if retryDeviceAuthorization(session.DeviceAuthCredentials{}, errors.New("network unavailable")) {
		t.Fatal("network failure must not trigger reauthorization")
	}
	if retryDeviceAuthorization(session.SpotifyTokenCredentials{}, rejected) {
		t.Fatal("other authentication modes must be unchanged")
	}
}
