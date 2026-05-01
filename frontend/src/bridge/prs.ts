import type { MergeMethod, PRFullDetails, PRRecord } from "@/lib/types"

import { requireBridge } from "./client"
import { toPRRecord } from "./mappers"
import type { PRFullDetailsWire, PRRecordWire } from "./wire"

export interface PRsBridge {
  ListPendingPRs(): Promise<PRRecordWire[]>
  ListHistoryPRs(): Promise<PRRecordWire[]>
  GetPRDetails(prID: string): Promise<PRFullDetailsWire>
  GetPRDiff(prID: string): Promise<string>
  MergePR(prID: string, method: MergeMethod): Promise<void>
  RefreshNow(): Promise<void>
  OpenPRInBrowser(url: string): Promise<void>
}

export async function listPendingPRs(): Promise<PRRecord[]> {
  return (await requireBridge("ListPendingPRs")()).map(toPRRecord)
}

export async function listHistoryPRs(): Promise<PRRecord[]> {
  return (await requireBridge("ListHistoryPRs")()).map(toPRRecord)
}

export const getPRDetails = (prID: string): Promise<PRFullDetails> =>
  requireBridge("GetPRDetails")(prID)

export const getPRDiff = (prID: string): Promise<string> =>
  requireBridge("GetPRDiff")(prID)

export const mergePR = (prID: string, method: MergeMethod): Promise<void> =>
  requireBridge("MergePR")(prID, method)

export const refreshNow = (): Promise<void> => requireBridge("RefreshNow")()

export const openPRInBrowser = (url: string): Promise<void> =>
  requireBridge("OpenPRInBrowser")(url)
