// Unified-diff parser just good enough to feed react-diff-viewer-continued
// one entry per changed file. We reconstruct approximate "old" and "new"
// file contents by replaying hunks in order — context + removed lines into
// old, context + added lines into new. It's not byte-for-byte what GitHub
// shows, but it renders a faithful side-by-side/unified diff for each file.

export interface FileDiff {
  path: string
  language: string
  oldContent: string
  newContent: string
}

export function parseDiff(raw: string): FileDiff[] {
  if (!raw) return []
  const out: FileDiff[] = []
  const lines = raw.split('\n')

  let current: {
    path: string
    oldLines: string[]
    newLines: string[]
  } | null = null
  let inHunk = false

  const flush = () => {
    if (current) {
      out.push({
        path: current.path,
        language: detectLanguage(current.path),
        oldContent: current.oldLines.join('\n'),
        newContent: current.newLines.join('\n'),
      })
    }
    current = null
    inHunk = false
  }

  for (const line of lines) {
    if (line.startsWith('diff --git ')) {
      flush()
      const m = line.match(/^diff --git a\/(.+?) b\/(.+)$/)
      const path = m ? m[2] : 'unknown'
      current = { path, oldLines: [], newLines: [] }
      continue
    }
    if (!current) continue
    if (
      line.startsWith('index ') ||
      line.startsWith('--- ') ||
      line.startsWith('+++ ') ||
      line.startsWith('new file mode') ||
      line.startsWith('deleted file mode') ||
      line.startsWith('rename from') ||
      line.startsWith('rename to') ||
      line.startsWith('similarity index') ||
      line.startsWith('Binary files ')
    ) {
      continue
    }
    if (line.startsWith('@@')) {
      inHunk = true
      continue
    }
    if (!inHunk) continue
    if (line.startsWith('+')) {
      current.newLines.push(line.slice(1))
    } else if (line.startsWith('-')) {
      current.oldLines.push(line.slice(1))
    } else if (line.startsWith(' ')) {
      const body = line.slice(1)
      current.oldLines.push(body)
      current.newLines.push(body)
    } else if (line === '' || line.startsWith('\\')) {
      // blank line inside a hunk is a legitimate diff line; backslash
      // lines like "\ No newline at end of file" are metadata we drop.
      if (line === '') {
        current.oldLines.push('')
        current.newLines.push('')
      }
    }
  }
  flush()
  return out
}

// detectLanguage maps a path to a prism-compatible language id. Unknown
// extensions fall back to 'text' so the highlighter just prints the lines
// without crashing.
export function detectLanguage(path: string): string {
  const lower = path.toLowerCase()
  const dot = lower.lastIndexOf('.')
  const ext = dot >= 0 ? lower.slice(dot + 1) : ''
  const map: Record<string, string> = {
    ts: 'typescript',
    tsx: 'tsx',
    js: 'javascript',
    jsx: 'jsx',
    go: 'go',
    py: 'python',
    rs: 'rust',
    rb: 'ruby',
    java: 'java',
    kt: 'kotlin',
    swift: 'swift',
    c: 'c',
    h: 'c',
    cpp: 'cpp',
    cc: 'cpp',
    hpp: 'cpp',
    cs: 'csharp',
    php: 'php',
    sh: 'bash',
    bash: 'bash',
    zsh: 'bash',
    yml: 'yaml',
    yaml: 'yaml',
    json: 'json',
    toml: 'toml',
    md: 'markdown',
    sql: 'sql',
    css: 'css',
    scss: 'scss',
    html: 'html',
    xml: 'xml',
    dockerfile: 'docker',
  }
  return map[ext] ?? 'text'
}
