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
    isDraft: false,
    additions: 0,
    deletions: 0,
    reviewPending: false,
    reviewState: "PENDING",
    branch: "main",
    avatarUrl: "",
    firstSeenAt: "2026-01-01T00:00:00Z",
    lastSeenAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("statusOf", () => {
  it.each([
    [{ state: "MERGED" }, "MERGED"],
    [{ state: "CLOSED" }, "CLOSED"],
    [{ state: "OPEN" }, "OPEN"],
    [{ state: "OPEN", isDraft: true }, "DRAFT"],
  ] as const)("%j → %s", (over, expected) => {
    expect(statusOf(makePR(over))).toBe(expected)
  })

  it("MERGED tem precedência sobre isDraft", () => {
    expect(statusOf(makePR({ state: "MERGED", isDraft: true }))).toBe("MERGED")
  })
})

describe("reviewStateOf", () => {
  it.each(["APPROVED", "CHANGES_REQUESTED", "COMMENTED"] as const)(
    "preserva %s",
    (s) => {
      expect(reviewStateOf(makePR({ reviewState: s }))).toBe(s)
    }
  )

  it("PENDING permanece PENDING", () => {
    expect(reviewStateOf(makePR({ reviewState: "PENDING" }))).toBe("PENDING")
  })

  it("estado desconhecido vira PENDING", () => {
    expect(reviewStateOf(makePR({ reviewState: "DISMISSED" }))).toBe("PENDING")
  })

  it("string vazia vira PENDING", () => {
    expect(reviewStateOf(makePR({ reviewState: "" }))).toBe("PENDING")
  })
})
