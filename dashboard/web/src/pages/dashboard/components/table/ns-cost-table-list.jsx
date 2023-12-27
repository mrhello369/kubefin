// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
import { useColumnWidths } from "../../../commons/use-column-widths";
import {
  NS_COST_COLUMN_DEFINITIONS,
  NS_COST_FILTERING_PROPERTIES,
} from "./table-property-filter-config";
import { NSCostListItem } from "./ns-cost-list-item";
import "../../../../styles/base.scss";
import { NSPropertyFilterTable } from "./ns-property-filter-table";
import { keepThreeDecimal, keepTwoDecimal } from "../components-common";

export default function NSCostTableList(props) {
  const [columnDefinitions, saveWidths] = useColumnWidths(
    "React-TableServerSide-Widths",
    NS_COST_COLUMN_DEFINITIONS
  );
  const namespaceCostMap = props.namespaceCostMap;
  if (!(namespaceCostMap instanceof Map) || namespaceCostMap.size === 0) {
    return <div>The namespace Cost is empty or not a valid Map.</div>;
  }
  // console.log("namespaceCostMap@ClusterNsCostTableList = ", namespaceCostMap);

  const nsCostList = Array.from(namespaceCostMap).map(
    ([key, value]) => {
      return new NSCostListItem(
        key,
        keepTwoDecimal(value.podCount),
        keepTwoDecimal(value.cpuRequest),
        keepTwoDecimal(value.ramGiBRequest),
        keepThreeDecimal(value.totalCost)
      );
    }
  );

  return (
    <NSPropertyFilterTable
      data={nsCostList}
      columnDefinitions={columnDefinitions}
      saveWidths={saveWidths}
      filteringProperties={NS_COST_FILTERING_PROPERTIES}
    />
  );
}
