import { describe, expect, it } from "vitest"

import type { PRFullDetails, Review, StatusCheck } from "@/lib/types"

import {
  derivePRState,
  deriveReviewState,
  mergeableNow,
  mergeBlockedReason,
} from "./derive-state"

function makeDetails(overrides: Partial<PRFullDetails> = {}): PRFullDetails {
  return {
    number: 1,
    title: "test",
    body: "",
    url: "https://github.com/owner/repo/pull/1",
    state: "OPEN",
    isDraft: false,
    author: "alice",
    additions: 0,
    deletions: 0,
    changedFiles: 0,
    labels: [],
    reviews: [],
    statusChecks: [],
    files: [],
    mergeable: "MERGEABLE",
    baseRefName: "main",
    headRefName: "feat/x",
    createdAt: "",
    updatedAt: "",
    mergedAt: null,
    ...overrides,
  }
}

function review(state: string): Review {
  return { author: "bob", state, submittedAt: "" }
}

function check(conclusion: string): StatusCheck {
  return { name: "ci", status: "COMPLETED", conclusion, url: "" }
}

describe("derivePRState", () => {
  it("retorna OPEN quando details é null", () => {
    expect(derivePRState(null)).toBe("OPEN")
  })

  it.each([
    ["MERGED", "MERGED"],
    ["CLOSED", "CLOSED"],
    ["OPEN", "OPEN"],
  ] as const)("estado %s mapeia pra %s", (state, expected) => {
    expect(derivePRState(makeDetails({ state }))).toBe(expected)
  })

  it("retorna DRAFT quando isDraft + state OPEN", () => {
    expect(derivePRState(makeDetails({ isDraft: true }))).toBe("DRAFT")
  })

  it("MERGED tem precedência sobre isDraft", () => {
    expect(derivePRState(makeDetails({ state: "MERGED", isDraft: true }))).toBe(
      "MERGED"
    )
  })
})

describe("deriveReviewState", () => {
  it("retorna PENDING quando details é null", () => {
    expect(deriveReviewState(null)).toBe("PENDING")
  })

  it("retorna PENDING quando reviews vazio", () => {
    expect(deriveReviewState(makeDetails())).toBe("PENDING")
  })

  it("CHANGES_REQUESTED tem precedência sobre APPROVED", () => {
    expect(
      deriveReviewState(
        makeDetails({
          reviews: [review("APPROVED"), review("CHANGES_REQUESTED")],
        })
      )
    ).toBe("CHANGES_REQUESTED")
  })

  it("APPROVED tem precedência sobre COMMENTED", () => {
    expect(
      deriveReviewState(
        makeDetails({ reviews: [review("COMMENTED"), review("APPROVED")] })
      )
    ).toBe("APPROVED")
  })

  it("COMMENTED quando só há comentários", () => {
    expect(
      deriveReviewState(makeDetails({ reviews: [review("COMMENTED")] }))
    ).toBe("COMMENTED")
  })

  it("PENDING quando reviews têm estado desconhecido", () => {
    expect(
      deriveReviewState(makeDetails({ reviews: [review("DISMISSED")] }))
    ).toBe("PENDING")
  })
})

describe("mergeableNow", () => {
  it("false quando details é null", () => {
    expect(mergeableNow(null)).toBe(false)
  })

  it("true no caminho feliz: OPEN + !draft + MERGEABLE + sem checks falhando", () => {
    expect(mergeableNow(makeDetails())).toBe(true)
  })

  it("false quando state ≠ OPEN", () => {
    expect(mergeableNow(makeDetails({ state: "CLOSED" }))).toBe(false)
  })

  it("false quando isDraft", () => {
    expect(mergeableNow(makeDetails({ isDraft: true }))).toBe(false)
  })

  it.each(["CONFLICTING", "UNKNOWN"])("false quando mergeable=%s", (m) => {
    expect(mergeableNow(makeDetails({ mergeable: m }))).toBe(false)
  })

  it.each(["FAILURE", "TIMED_OUT", "CANCELLED", "failure"])(
    "false quando algum check tem conclusion=%s",
    (c) => {
      expect(mergeableNow(makeDetails({ statusChecks: [check(c)] }))).toBe(
        false
      )
    }
  )

  it("true quando checks são SUCCESS", () => {
    expect(
      mergeableNow(makeDetails({ statusChecks: [check("SUCCESS")] }))
    ).toBe(true)
  })
})

describe("mergeBlockedReason", () => {
  it("null quando details é null", () => {
    expect(mergeBlockedReason(null)).toBeNull()
  })

  it("null no caminho feliz", () => {
    expect(mergeBlockedReason(makeDetails())).toBeNull()
  })

  it.each([
    [{ state: "MERGED" }, "PR já foi merged"],
    [{ state: "CLOSED" }, "PR fechado"],
    [{ isDraft: true }, "PR está em draft"],
    [{ mergeable: "CONFLICTING" }, "conflitos — resolva pelo GitHub"],
    [{ mergeable: "UNKNOWN" }, "GitHub ainda está checando se pode merge"],
  ])("%j → %s", (over, expected) => {
    expect(mergeBlockedReason(makeDetails(over))).toBe(expected)
  })

  it("retorna 'algum check falhou' quando há check failing", () => {
    expect(
      mergeBlockedReason(makeDetails({ statusChecks: [check("FAILURE")] }))
    ).toBe("algum check falhou")
  })

  it("MERGED tem precedência sobre check failing", () => {
    expect(
      mergeBlockedReason(
        makeDetails({ state: "MERGED", statusChecks: [check("FAILURE")] })
      )
    ).toBe("PR já foi merged")
  })
})
