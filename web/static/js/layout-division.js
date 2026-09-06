(() => {
  const MAX_TABLES_PER_WAITER = 4;

  function suggestFloorWaiterDivision(input = {}) {
    const tables = Math.max(0, Number(input.tables) || 0);
    const waiters = Math.max(0, Number(input.waiters) || 0);
    const coordinators = Math.max(0, Number(input.coordinators) || 0);
    const leaders = Math.max(0, Number(input.leaders) || 0);
    const coleaders = Math.max(0, Number(input.coleaders) || 0);
    const includeCoLeader = Boolean(input.includeCoLeader);

    const shadowWaiters = waiters > 0 ? 1 : 0;
    const servingWaiters = Math.max(0, waiters - shadowWaiters);
    const effectiveCoLeaders = coleaders || 1;
    const offerCoLeader = tables > 0 && (servingWaiters === 0 || tables > servingWaiters * MAX_TABLES_PER_WAITER);
    const usedCoLeaders = includeCoLeader && offerCoLeader ? effectiveCoLeaders : 0;
    const servingPeople = servingWaiters + usedCoLeaders;
    const tablesPerPerson = servingPeople > 0 ? tables / servingPeople : 0;
    const shares = [];
    if (servingPeople > 0 && tables > 0) {
      const base = Math.floor(tables / servingPeople);
      const extra = tables % servingPeople;
      for (let index = 0; index < servingPeople; index += 1) {
        shares.push(base + (index < extra ? 1 : 0));
      }
    }

    return {
      tableCount: tables,
      waiterCount: waiters,
      coordinatorCount: coordinators,
      leaderCount: leaders,
      coLeaderCount: coleaders,
      shadowWaiters,
      servingWaiters,
      usedCoLeaders,
      servingPeople,
      tablesPerPerson,
      offerCoLeader,
      shares,
    };
  }

  function coleaderNames(count) {
    const total = Math.max(0, Number(count) || 0);
    if (total <= 0) return [];
    if (total === 1) return ["Colíder"];
    return Array.from({ length: total }, (_, index) => `Colíder ${index + 1}`);
  }

  window.emenysSuggestFloorWaiterDivision = suggestFloorWaiterDivision;
  window.emenysColeaderNames = coleaderNames;
})();
