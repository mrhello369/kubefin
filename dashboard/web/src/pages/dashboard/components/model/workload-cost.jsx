export class WorkloadCost {
    constructor(
      namespace,
      workloadType,
      workloadName,
      totalCost,
      podCount,
      cpuCoreRequest,
      cpuCoreUsage,
      ramGiBRequest,
      ramGiBUsage,
    ) {
      this.namespace = namespace ? namespace : "-";
      this.workloadType = workloadType ? workloadType : "-";
      this.workloadName = workloadName ? workloadName : "-";
      this.totalCost = totalCost ? totalCost : 0;
      this.podCount = podCount ? podCount : 0;
      this.cpuCoreRequest = cpuCoreRequest ? cpuCoreRequest : 0;
      this.cpuCoreUsage = cpuCoreUsage ? cpuCoreUsage : 0;
      this.ramGiBRequest = ramGiBRequest ? ramGiBRequest : 0;
      this.ramGiBUsage = ramGiBUsage ? ramGiBUsage : 0;
    }
  }
