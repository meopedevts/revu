import type { MergeMethod, PRFullDetails, PRRecord } from "@/lib/types"

import { requireBridge } from "./client"

export interface PRsBridge {
  ListPendingPRs(): Promise<PRRecord[]>
  ListHistoryPRs(): Promise<PRRecord[]>
  GetPRDetails(prID: string): Promise<PRFullDetails>
  GetPRDiff(prID: string): Promise<string>
  MergePR(prID: string, method: MergeMethod): Promise<void>
  RefreshNow(): Promise<void>
  OpenPRInBrowser(url: string): Promise<void>
}

export const listPendingPRs = (): Promise<PRRecord[]> =>
  requireBridge("ListPendingPRs")()

export const listHistoryPRs = (): Promise<PRRecord[]> =>
  requireBridge("ListHistoryPRs")()

export const getPRDetails = (prID: string): Promise<PRFullDetails> =>
  requireBridge("GetPRDetails")(prID)

export const getPRDiff = (prID: string): Promise<string> =>
  requireBridge("GetPRDiff")(prID)

export const mergePR = (prID: string, method: MergeMethod): Promise<void> =>
  requireBridge("MergePR")(prID, method)

export const refreshNow = (): Promise<void> => requireBridge("RefreshNow")()

export const openPRInBrowser = (url: string): Promise<void> =>
  requireBridge("OpenPRInBrowser")(url)
