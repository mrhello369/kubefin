export class ClusterComputeCost {
    constructor(
      timestamp,
      totalCost,
      costFallbackBillingMode,
      costOnDemandBillingMode,
      costSpotBillingMode,
      cpuCoreCount,
      cpuCoreUsage,
      cpuCost,
      ramGiBCount,
      ramUsage,
      ramCost,
    ) {
      this.timestamp = timestamp;
      this.totalCost = totalCost;
      this.costFallbackBillingMode = costFallbackBillingMode;
      this.costOnDemandBillingMode = costOnDemandBillingMode;
      this.costSpotBillingMode = costSpotBillingMode;
      this.cpuCoreCount = cpuCoreCount;
      this.cpuCoreUsage = cpuCoreUsage;
      this.cpuCost = cpuCost;
      this.ramGiBCount = ramGiBCount;
      this.ramUsage = ramUsage;
      this.ramCost = ramCost;
    }
  }
