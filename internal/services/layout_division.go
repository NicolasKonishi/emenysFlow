package services

const MaxTablesPerWaiter = 4

// FloorWaiterDivision is the suggested split of tables among floor staff.
// Coordination and the leader never receive tables. One waiter is always
// reserved as the couple's shadow. If the remaining waiters would exceed
// MaxTablesPerWaiter tables each, the co-leader can join the split.
type FloorWaiterDivision struct {
	TableCount       int
	WaiterCount      int
	CoordinatorCount int
	LeaderCount      int
	CoLeaderCount    int
	ShadowWaiters    int
	ServingWaiters   int
	UsedCoLeaders    int
	ServingPeople    int
	TablesPerPerson  float64
	OfferCoLeader    bool
	Shares           []int
}

func SuggestFloorWaiterDivision(tables, waiters, coordinators, leaders, coleaders int, includeCoLeader bool) FloorWaiterDivision {
	if tables < 0 {
		tables = 0
	}
	if waiters < 0 {
		waiters = 0
	}
	if coordinators < 0 {
		coordinators = 0
	}
	if leaders < 0 {
		leaders = 0
	}
	if coleaders < 0 {
		coleaders = 0
	}

	shadow := 0
	if waiters > 0 {
		shadow = 1
	}
	servingWaiters := waiters - shadow
	if servingWaiters < 0 {
		servingWaiters = 0
	}

	effectiveCoLeaders := coleaders
	if effectiveCoLeaders == 0 {
		effectiveCoLeaders = 1
	}
	offer := tables > 0 && (servingWaiters == 0 || tables > servingWaiters*MaxTablesPerWaiter)

	usedCo := 0
	if includeCoLeader && offer {
		usedCo = effectiveCoLeaders
	}

	serving := servingWaiters + usedCo
	per := 0.0
	if serving > 0 {
		per = float64(tables) / float64(serving)
	}

	shares := make([]int, 0, serving)
	if serving > 0 && tables > 0 {
		base := tables / serving
		extra := tables % serving
		for i := 0; i < serving; i++ {
			n := base
			if i < extra {
				n++
			}
			shares = append(shares, n)
		}
	}

	return FloorWaiterDivision{
		TableCount:       tables,
		WaiterCount:      waiters,
		CoordinatorCount: coordinators,
		LeaderCount:      leaders,
		CoLeaderCount:    coleaders,
		ShadowWaiters:    shadow,
		ServingWaiters:   servingWaiters,
		UsedCoLeaders:    usedCo,
		ServingPeople:    serving,
		TablesPerPerson:  per,
		OfferCoLeader:    offer,
		Shares:           shares,
	}
}
