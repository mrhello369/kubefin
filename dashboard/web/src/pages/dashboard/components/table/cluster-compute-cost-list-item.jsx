export class ClusterComputeCostListItem {
  constructor(
    timestamp,
    totalCost,
    costFallbackBillingMode,
    costOnDemandBillingMode,
    costSpotBillingMode,
    cpuCoreCount,
    cpuCoreCountIndex,
    cpuCoreUsage,
    cpuCost,
    ramGiBCount,
    ramGiBCountIndex,
    ramGiBUsage,
    ramCost,
  ) {
    this.timestamp = timestamp ? timestamp : 0;
    this.totalCost = totalCost ? totalCost : 0;
    this.costFallbackBillingMode = costFallbackBillingMode ? costFallbackBillingMode : 0;
    this.costOnDemandBillingMode = costOnDemandBillingMode ? costOnDemandBillingMode : 0;
    this.costSpotBillingMode = costSpotBillingMode ? costSpotBillingMode : 0;
    this.cpuCoreCount = cpuCoreCount ? cpuCoreCount : 0;
    this.cpuCoreCountIndex = cpuCoreCountIndex ? cpuCoreCountIndex : 1.0;
    this.cpuCoreUsage = cpuCoreUsage ? cpuCoreUsage : 0;
    this.cpuCost = cpuCost ? cpuCost : 0;
    this.ramGiBCount = ramGiBCount ? ramGiBCount : 0;
    this.ramGiBCountIndex = ramGiBCountIndex ? ramGiBCountIndex : 1.0;
    this.ramGiBUsage = ramGiBUsage ? ramGiBUsage : 0;
    this.ramCost = ramCost ? ramCost : 0;
  }
}
