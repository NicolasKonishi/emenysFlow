package services

import (
	"reflect"
	"testing"
)

func TestSuggestFloorWaiterDivision(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		tables          int
		waiters         int
		coordinators    int
		leaders         int
		coleaders       int
		includeCoLeader bool
		wantShadow      int
		wantServingW    int
		wantUsedCo      int
		wantServing     int
		wantOffer       bool
		wantShares      []int
		wantTablesPer   float64
	}{
		{
			name:   "four tables each does not offer coleader",
			tables: 20, waiters: 6, coordinators: 1, leaders: 1, coleaders: 1,
			wantShadow: 1, wantServingW: 5, wantServing: 5, wantShares: []int{4, 4, 4, 4, 4},
			wantTablesPer: 4,
		},
		{
			name:   "more than four tables each offers coleader",
			tables: 21, waiters: 6, coordinators: 1, leaders: 1, coleaders: 1,
			wantShadow: 1, wantServingW: 5, wantServing: 5, wantOffer: true, wantShares: []int{5, 4, 4, 4, 4},
			wantTablesPer: 4.2,
		},
		{
			name:   "accepted coleader joins the split",
			tables: 21, waiters: 6, coordinators: 1, leaders: 1, coleaders: 1, includeCoLeader: true,
			wantShadow: 1, wantServingW: 5, wantUsedCo: 1, wantServing: 6, wantOffer: true, wantShares: []int{4, 4, 4, 3, 3, 3},
			wantTablesPer: 3.5,
		},
		{
			name:   "coordination and leader never receive tables",
			tables: 12, waiters: 4, coordinators: 2, leaders: 1, coleaders: 1,
			wantShadow: 1, wantServingW: 3, wantServing: 3, wantShares: []int{4, 4, 4},
			wantTablesPer: 4,
		},
		{
			name:   "single waiter is reserved as shadow and coleader is offered",
			tables: 8, waiters: 1, coordinators: 1, leaders: 1, coleaders: 1,
			wantShadow: 1, wantServingW: 0, wantServing: 0, wantOffer: true,
		},
		{
			name:   "single waiter with coleader accepted",
			tables: 8, waiters: 1, coordinators: 1, leaders: 1, coleaders: 1, includeCoLeader: true,
			wantShadow: 1, wantServingW: 0, wantUsedCo: 1, wantServing: 1, wantOffer: true, wantShares: []int{8},
			wantTablesPer: 8,
		},
		{
			name:   "missing coleader count still offers one person",
			tables: 15, waiters: 3, includeCoLeader: true,
			wantShadow: 1, wantServingW: 2, wantUsedCo: 1, wantServing: 3, wantOffer: true, wantShares: []int{5, 5, 5},
			wantTablesPer: 5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SuggestFloorWaiterDivision(tc.tables, tc.waiters, tc.coordinators, tc.leaders, tc.coleaders, tc.includeCoLeader)
			if got.ShadowWaiters != tc.wantShadow || got.ServingWaiters != tc.wantServingW || got.UsedCoLeaders != tc.wantUsedCo || got.ServingPeople != tc.wantServing || got.OfferCoLeader != tc.wantOffer {
				t.Fatalf("counts %+v", got)
			}
			if tc.wantShares != nil && !reflect.DeepEqual(got.Shares, tc.wantShares) {
				t.Fatalf("shares %v want %v", got.Shares, tc.wantShares)
			}
			if tc.wantTablesPer != 0 && got.TablesPerPerson != tc.wantTablesPer {
				t.Fatalf("tables per person %v want %v", got.TablesPerPerson, tc.wantTablesPer)
			}
			if got.CoordinatorCount != tc.coordinators || got.LeaderCount != tc.leaders {
				t.Fatalf("staff roles should stay informational, got %+v", got)
			}
		})
	}
}
