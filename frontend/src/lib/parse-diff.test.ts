import { describe, expect, it } from "vitest"

import { detectLanguage, parseDiff } from "./parse-diff"

describe("parseDiff", () => {
  it("retorna array vazio quando input é vazio", () => {
    expect(parseDiff("")).toEqual([])
  })

  it("parseia hunk simples com adds/removes/context", () => {
    const raw = [
      "diff --git a/foo.ts b/foo.ts",
      "index abc..def 100644",
      "--- a/foo.ts",
      "+++ b/foo.ts",
      "@@ -1,3 +1,3 @@",
      " context",
      "-removed",
      "+added",
      " trailing",
    ].join("\n")

    const result = parseDiff(raw)
    expect(result).toHaveLength(1)
    expect(result[0].path).toBe("foo.ts")
    expect(result[0].language).toBe("typescript")
    expect(result[0].oldContent).toBe("context\nremoved\ntrailing")
    expect(result[0].newContent).toBe("context\nadded\ntrailing")
  })

  it("parseia múltiplos arquivos", () => {
    const raw = [
      "diff --git a/a.ts b/a.ts",
      "@@ -1 +1 @@",
      "-old-a",
      "+new-a",
      "diff --git a/b.go b/b.go",
      "@@ -1 +1 @@",
      "-old-b",
      "+new-b",
    ].join("\n")

    const result = parseDiff(raw)
    expect(result).toHaveLength(2)
    expect(result[0].path).toBe("a.ts")
    expect(result[0].oldContent).toBe("old-a")
    expect(result[1].path).toBe("b.go")
    expect(result[1].language).toBe("go")
  })

  it("parseia múltiplos hunks no mesmo arquivo", () => {
    const raw = [
      "diff --git a/x.ts b/x.ts",
      "@@ -1,1 +1,1 @@",
      "-a",
      "+a2",
      "@@ -10,1 +10,1 @@",
      "-b",
      "+b2",
    ].join("\n")

    const result = parseDiff(raw)
    expect(result).toHaveLength(1)
    expect(result[0].oldContent).toBe("a\nb")
    expect(result[0].newContent).toBe("a2\nb2")
  })

  it("ignora cabeçalhos rename/binary/index", () => {
    const raw = [
      "diff --git a/x.ts b/y.ts",
      "similarity index 90%",
      "rename from x.ts",
      "rename to y.ts",
      "index abc..def",
      "Binary files differ",
      "@@ -1 +1 @@",
      "-old",
      "+new",
    ].join("\n")

    const result = parseDiff(raw)
    expect(result[0].path).toBe("y.ts")
    expect(result[0].oldContent).toBe("old")
    expect(result[0].newContent).toBe("new")
  })

  it("preserva linhas em branco dentro de hunks", () => {
    const raw = [
      "diff --git a/x.ts b/x.ts",
      "@@ -1,3 +1,3 @@",
      " a",
      "",
      " b",
    ].join("\n")

    const result = parseDiff(raw)
    expect(result[0].oldContent).toBe("a\n\nb")
    expect(result[0].newContent).toBe("a\n\nb")
  })

  it("descarta '\\ No newline at end of file'", () => {
    const raw = [
      "diff --git a/x.ts b/x.ts",
      "@@ -1 +1 @@",
      "-old",
      "+new",
      "\\ No newline at end of file",
    ].join("\n")

    const result = parseDiff(raw)
    expect(result[0].oldContent).toBe("old")
    expect(result[0].newContent).toBe("new")
  })

  it("usa 'unknown' quando regex de path falha", () => {
    const raw = ["diff --git malformed", "@@ -1 +1 @@", "-x", "+y"].join("\n")
    const result = parseDiff(raw)
    expect(result[0].path).toBe("unknown")
  })
})

describe("detectLanguage", () => {
  it.each([
    ["foo.ts", "typescript"],
    ["foo.tsx", "tsx"],
    ["foo.js", "javascript"],
    ["foo.jsx", "jsx"],
    ["main.go", "go"],
    ["script.py", "python"],
    ["lib.rs", "rust"],
    ["app.rb", "ruby"],
    ["Main.java", "java"],
    ["App.kt", "kotlin"],
    ["app.swift", "swift"],
    ["main.c", "c"],
    ["util.h", "c"],
    ["main.cpp", "cpp"],
    ["main.cc", "cpp"],
    ["util.hpp", "cpp"],
    ["Program.cs", "csharp"],
    ["index.php", "php"],
    ["run.sh", "bash"],
    ["run.bash", "bash"],
    ["run.zsh", "bash"],
    ["config.yml", "yaml"],
    ["config.yaml", "yaml"],
    ["data.json", "json"],
    ["Cargo.toml", "toml"],
    ["README.md", "markdown"],
    ["query.sql", "sql"],
    ["style.css", "css"],
    ["style.scss", "scss"],
    ["index.html", "html"],
    ["data.xml", "xml"],
  ])("mapeia %s → %s", (path, expected) => {
    expect(detectLanguage(path)).toBe(expected)
  })

  it("fallback 'text' pra extensão desconhecida", () => {
    expect(detectLanguage("foo.xyz")).toBe("text")
  })

  it("fallback 'text' quando não há extensão", () => {
    expect(detectLanguage("Makefile")).toBe("text")
  })

  it("é case-insensitive", () => {
    expect(detectLanguage("FOO.TS")).toBe("typescript")
  })
})
