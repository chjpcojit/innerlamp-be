package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	deliveryhttp "chjpcoj.com.innerlamp/services/appService/delivery/http"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type fakeAnnouncementUsecase struct{ calls int }

func (f *fakeAnnouncementUsecase) ListForUser(context.Context, int64, int, int) (domain.UserAnnouncements, error) {
	f.calls++
	return domain.UserAnnouncements{Receive: true}, nil
}
func (f *fakeAnnouncementUsecase) GetReceive(context.Context, int64) (bool, error) {
	f.calls++
	return true, nil
}
func (f *fakeAnnouncementUsecase) SetReceive(_ context.Context, _ int64, r bool) (bool, error) {
	f.calls++
	return r, nil
}

var announcementRoutes = []struct{ method, path, body string }{
	{http.MethodGet, "/v1/users/me/temple-announcements", ""},
	{http.MethodGet, "/v1/users/me/temple-announcement-settings", ""},
	{http.MethodPut, "/v1/users/me/temple-announcement-settings", `{"receive":true}`},
}

func newAnnouncementRouteMux(t *testing.T, max int64) (*http.ServeMux, *fakeAnnouncementUsecase) {
	t.Helper()
	limit, err := deliveryhttp.NewUserRateLimit(max, time.Hour)
	if err != nil {
		t.Fatalf("NewUserRateLimit: %v", err)
	}
	uc := &fakeAnnouncementUsecase{}
	mux := http.NewServeMux()
	registerTempleAnnouncementRoutes(mux, fakeAuth, limit, deliveryhttp.NewTempleAnnouncementHandler(uc))
	return mux, uc
}

func TestRegisterTempleAnnouncementRoutes_EveryRouteRequiresAuth(t *testing.T) {
	for _, rt := range announcementRoutes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			mux, uc := newAnnouncementRouteMux(t, 100)
			if rr := do(mux, rt.method, rt.path, rt.body, false); rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rr.Code)
			}
			if uc.calls != 0 {
				t.Errorf("use case ran for an unauthenticated request")
			}
		})
	}
}

func TestRegisterTempleAnnouncementRoutes_RateLimitSharedAcrossGroupAfterAuth(t *testing.T) {
	mux, uc := newAnnouncementRouteMux(t, 2)
	for i := 0; i < 5; i++ {
		if rr := do(mux, http.MethodGet, "/v1/users/me/temple-announcements", "", false); rr.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous %d: %d", i, rr.Code)
		}
	}
	for _, rt := range announcementRoutes[:2] {
		if rr := do(mux, rt.method, rt.path, rt.body, true); rr.Code != http.StatusOK {
			t.Fatalf("%s %s: %d", rt.method, rt.path, rr.Code)
		}
	}
	before := uc.calls
	rr := do(mux, http.MethodPut, announcementRoutes[2].path, announcementRoutes[2].body, true)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request: %d, want 429", rr.Code)
	}
	if uc.calls != before {
		t.Error("a limited request must not reach the use case")
	}
}
