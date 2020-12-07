package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestListIPPrefix(t *testing.T) {
	setup(UsingAccount("foo"))
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
			"result": [
				{
					"id": "f68579455bd947efb65ffa1bcf33b52c",
					"created_at": "2020-04-24T21:25:55.643771Z",
					"modified_at": "2020-04-24T21:25:55.643771Z",
					"cidr": "10.1.2.3/24",
					"account_id": "foo",
					"description": "Sample Prefix",
					"approved": "V",
					"on_demand_enabled": true,
					"advertised": true,
					"advertised_modified_at": "2020-04-24T21:25:55.643771Z"
				}
			],
			"success": true,
			"errors": [],
			"messages": []
		}`)
	}

	mux.HandleFunc("/accounts/foo/addressing/prefixes", handler)

	createdAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")
	modifiedAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")
	advertisedModifiedAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")

	want := []IPPrefix{
		{
			ID:                   "f68579455bd947efb65ffa1bcf33b52c",
			CreatedAt:            &createdAt,
			ModifiedAt:           &modifiedAt,
			CIDR:                 "10.1.2.3/24",
			AccountID:            "foo",
			Description:          "Sample Prefix",
			Approved:             "V",
			OnDemandEnabled:      true,
			Advertised:           true,
			AdvertisedModifiedAt: &advertisedModifiedAt,
		},
	}

	actual, err := client.ListPrefixes(context.Background())
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestGetIPPrefix(t *testing.T) {
	setup(UsingAccount("foo"))
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
			"result": {
				"id": "f68579455bd947efb65ffa1bcf33b52c",
				"created_at": "2020-04-24T21:25:55.643771Z",
				"modified_at": "2020-04-24T21:25:55.643771Z",
				"cidr": "10.1.2.3/24",
				"account_id": "foo",
				"description": "Sample Prefix",
				"approved": "V",
				"on_demand_enabled": true,
				"advertised": true,
				"advertised_modified_at": "2020-04-24T21:25:55.643771Z"
			},
			"success": true,
			"errors": [],
			"messages": []
		}`)
	}

	mux.HandleFunc("/accounts/foo/addressing/prefixes/f68579455bd947efb65ffa1bcf33b52c", handler)

	createdAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")
	modifiedAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")
	advertisedModifiedAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")

	want := IPPrefix{
		ID:                   "f68579455bd947efb65ffa1bcf33b52c",
		CreatedAt:            &createdAt,
		ModifiedAt:           &modifiedAt,
		CIDR:                 "10.1.2.3/24",
		AccountID:            "foo",
		Description:          "Sample Prefix",
		Approved:             "V",
		OnDemandEnabled:      true,
		Advertised:           true,
		AdvertisedModifiedAt: &advertisedModifiedAt,
	}

	actual, err := client.GetPrefix(context.Background(), "f68579455bd947efb65ffa1bcf33b52c")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdatePrefixDescription(t *testing.T) {
	setup(UsingAccount("foo"))
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PATCH", "Expected method 'PATCH', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
			"result": {
				"id": "f68579455bd947efb65ffa1bcf33b52c",
				"created_at": "2020-04-24T21:25:55.643771Z",
				"modified_at": "2020-04-24T21:25:55.643771Z",
				"cidr": "10.1.2.3/24",
				"account_id": "foo",
				"description": "My IP Prefix",
				"approved": "V",
				"on_demand_enabled": true,
				"advertised": true,
				"advertised_modified_at": "2020-04-24T21:25:55.643771Z"
			},
			"success": true,
			"errors": [],
			"messages": []
		}`)
	}

	mux.HandleFunc("/accounts/foo/addressing/prefixes/f68579455bd947efb65ffa1bcf33b52c", handler)

	createdAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")
	modifiedAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")
	advertisedModifiedAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")

	want := IPPrefix{
		ID:                   "f68579455bd947efb65ffa1bcf33b52c",
		CreatedAt:            &createdAt,
		ModifiedAt:           &modifiedAt,
		CIDR:                 "10.1.2.3/24",
		AccountID:            "foo",
		Description:          "My IP Prefix",
		Approved:             "V",
		OnDemandEnabled:      true,
		Advertised:           true,
		AdvertisedModifiedAt: &advertisedModifiedAt,
	}

	actual, err := client.UpdatePrefixDescription(context.Background(), "f68579455bd947efb65ffa1bcf33b52c", "My IP Prefix")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestGetAdvertisementStatus(t *testing.T) {
	setup(UsingAccount("foo"))
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
			"result": {
				"advertised": true,
				"advertised_modified_at": "2020-04-24T21:25:55.643771Z"
			},
			"success": true,
			"errors": [],
			"messages": []
		}`)
	}

	mux.HandleFunc("/accounts/foo/addressing/prefixes/f68579455bd947efb65ffa1bcf33b52c/bgp/status", handler)

	advertisedModifiedAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")

	want := AdvertisementStatus{
		Advertised:           true,
		AdvertisedModifiedAt: &advertisedModifiedAt,
	}

	actual, err := client.GetAdvertisementStatus(context.Background(), "f68579455bd947efb65ffa1bcf33b52c")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdateAdvertisementStatus(t *testing.T) {
	setup(UsingAccount("foo"))
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PATCH", "Expected method 'PATCH', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
			"result": {
				"advertised": false,
				"advertised_modified_at": "2020-04-24T21:25:55.643771Z"
			},
			"success": true,
			"errors": [],
			"messages": []
		}`)
	}

	mux.HandleFunc("/accounts/foo/addressing/prefixes/f68579455bd947efb65ffa1bcf33b52c/bgp/status", handler)

	advertisedModifiedAt, _ := time.Parse(time.RFC3339, "2020-04-24T21:25:55.643771Z")

	want := AdvertisementStatus{
		Advertised:           false,
		AdvertisedModifiedAt: &advertisedModifiedAt,
	}

	actual, err := client.UpdateAdvertisementStatus(context.Background(), "f68579455bd947efb65ffa1bcf33b52c", false)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}
