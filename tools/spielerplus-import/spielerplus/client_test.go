package spielerplus

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func eventRowHTML(id, dateDDMM string) string {
	return fmt.Sprintf(`<div data-event-id="%s" data-event-type="training"><span class="title">Training</span><span class="date">%s</span></div>`, id, dateDDMM)
}

// newTestClient points a Client at an httptest server instead of the real
// spielerplus.de, so FetchEvents/FetchAttendance can be exercised end to
// end (request construction, header, and response parsing) without network
// access.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient("sid=test")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	c.baseURL = srv.URL
	return c
}

func TestClient_FetchEvents_PaginatesUntilEmpty(t *testing.T) {
	var ajaxCalls []string
	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, eventRowHTML("1", "01.08."))
	})
	mux.HandleFunc("/events/ajaxgetevents", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Requested-With"); got != "XMLHttpRequest" {
			t.Errorf("X-Requested-With = %q, want XMLHttpRequest", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		offset := r.FormValue("offset")
		old := r.FormValue("old")
		ajaxCalls = append(ajaxCalls, fmt.Sprintf("offset=%s old=%s", offset, old))

		switch offset {
		case "0":
			fmt.Fprint(w, eventRowHTML("2", "25.07."))
		case "5":
			fmt.Fprint(w, eventRowHTML("3", "18.07."))
		default:
			// end of history: no rows.
			fmt.Fprint(w, `<html><body></body></html>`)
		}
	})

	c := newTestClient(t, mux)
	now := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	events, err := c.FetchEvents(now)
	if err != nil {
		t.Fatalf("FetchEvents() error = %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %+v", len(events), events)
	}
	gotIDs := []string{events[0].ID, events[1].ID, events[2].ID}
	wantIDs := []string{"1", "2", "3"}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Errorf("events[%d].ID = %q, want %q", i, gotIDs[i], wantIDs[i])
		}
	}
	// Event 3 (18.07.) resolved relative to event 2 (25.07.2026) must stay
	// in 2026, not jump forward/back a year.
	if events[2].Start.Year() != 2026 {
		t.Errorf("events[2].Start = %v, want year 2026", events[2].Start)
	}

	wantCalls := []string{"offset=0 old=true", "offset=5 old=true", "offset=10 old=true"}
	if len(ajaxCalls) != len(wantCalls) {
		t.Fatalf("ajax calls = %v, want %v", ajaxCalls, wantCalls)
	}
	for i := range wantCalls {
		if ajaxCalls[i] != wantCalls[i] {
			t.Errorf("ajax call %d = %q, want %q", i, ajaxCalls[i], wantCalls[i])
		}
	}
}

func TestClient_FetchEvents_NoHistoryPages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, eventRowHTML("1", "01.08."))
	})
	mux.HandleFunc("/events/ajaxgetevents", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body></body></html>`)
	})

	c := newTestClient(t, mux)
	events, err := c.FetchEvents(time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FetchEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].ID != "1" {
		t.Fatalf("events = %+v, want only the initial page's event", events)
	}
}

func TestClient_FetchAttendance_SendsEventIDAndType(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/events/ajaxgetparticipation", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.FormValue("eventid"); got != "42" {
			t.Errorf("eventid = %q, want 42", got)
		}
		if got := r.FormValue("eventtype"); got != "training" {
			t.Errorf("eventtype = %q, want training", got)
		}
		if got := r.Header.Get("X-Requested-With"); got != "XMLHttpRequest" {
			t.Errorf("X-Requested-With = %q, want XMLHttpRequest", got)
		}
		fmt.Fprint(w, `<div data-user-id="7"><span class="selected" title="Zugesagt">x</span></div>`)
	})

	c := newTestClient(t, mux)
	records, err := c.FetchAttendance("42", EventTraining)
	if err != nil {
		t.Fatalf("FetchAttendance() error = %v", err)
	}
	if len(records) != 1 || records[0].MemberID != "7" || records[0].Status != ParticipationAccepted {
		t.Fatalf("records = %+v", records)
	}
}

func TestClient_Get_NotAuthenticated(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/site/login", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body>login please</body></html>`)
	})
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/site/login", http.StatusFound)
	})

	c := newTestClient(t, mux)
	_, err := c.FetchEvents(time.Now())
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("FetchEvents() error = %v, want ErrNotAuthenticated", err)
	}
}

func TestNewClient_EmptyCookie(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected an error for an empty session cookie")
	}
	if _, err := NewClient("   "); err == nil {
		t.Fatal("expected an error for a whitespace-only session cookie")
	}
}
