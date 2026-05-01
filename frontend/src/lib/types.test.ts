import { describe, expect, it } from "vitest"

import { type PRRecord, reviewStateOf, statusOf } from "./types"

function makePR(overrides: Partial<PRRecord> = {}): PRRecord {
  return {
    id: "pr-1",
    number: 1,
    repo: "owner/repo",
    title: "test",
    author: "alice",
    url: "https://github.com/owner/repo/pull/1",
    state: "OPEN",
    is_draft: false,
    additions: 0,
    deletions: 0,
    review_pending: false,
    review_state: "PENDING",
    first_seen_at: "2026-01-01T00:00:00Z",
    last_seen_at: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("statusOf", () => {
  it.each([
    [{ state: "MERGED" }, "MERGED"],
    [{ state: "CLOSED" }, "CLOSED"],
    [{ state: "OPEN" }, "OPEN"],
    [{ state: "OPEN", is_draft: true }, "DRAFT"],
  ] as const)("%j → %s", (over, expected) => {
    expect(statusOf(makePR(over))).toBe(expected)
  })

  it("MERGED tem precedência sobre is_draft", () => {
    expect(statusOf(makePR({ state: "MERGED", is_draft: true }))).toBe("MERGED")
  })
})

describe("reviewStateOf", () => {
  it.each(["APPROVED", "CHANGES_REQUESTED", "COMMENTED"] as const)(
    "preserva %s",
    (s) => {
      expect(reviewStateOf(makePR({ review_state: s }))).toBe(s)
    }
  )

  it("PENDING permanece PENDING", () => {
    expect(reviewStateOf(makePR({ review_state: "PENDING" }))).toBe("PENDING")
  })

  it("estado desconhecido vira PENDING", () => {
    expect(reviewStateOf(makePR({ review_state: "DISMISSED" }))).toBe("PENDING")
  })

  it("string vazia vira PENDING", () => {
    expect(reviewStateOf(makePR({ review_state: "" }))).toBe("PENDING")
  })
})
