export class NamespaceCostInfo {
    constructor(
      namespace,
      podCount,
      cpuRequest,
      ramGiBRequest,
      totalCost
    ) {
      this.namespace = namespace;
      this.podCount = podCount;
      this.cpuRequest = cpuRequest;
      this.ramGiBRequest = ramGiBRequest;
      this.totalCost=totalCost;
  }
}
