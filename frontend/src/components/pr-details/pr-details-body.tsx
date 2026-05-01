import ReactMarkdown from "react-markdown"
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter"
import { oneDark } from "react-syntax-highlighter/dist/esm/styles/prism"
import remarkGfm from "remark-gfm"

import { openPRInBrowser } from "@/bridge"

interface PRDetailsBodyProps {
  body: string
}

export function PRDetailsBody({ body }: PRDetailsBodyProps) {
  if (!body.trim()) {
    return (
      <div className="text-xs text-muted-foreground italic">sem descrição</div>
    )
  }
  return (
    <div className="prose prose-sm prose-invert max-w-none text-sm leading-relaxed">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          code(props) {
            const { className, children, ...rest } = props
            const match = /language-(\w+)/.exec(className ?? "")
            const inline = !className
            if (inline) {
              return (
                <code
                  className="rounded bg-muted px-1 py-0.5 font-mono text-[0.85em]"
                  {...rest}
                >
                  {children}
                </code>
              )
            }
            return (
              <SyntaxHighlighter
                language={match?.[1] ?? "text"}
                style={oneDark}
                PreTag="div"
                customStyle={{
                  margin: 0,
                  fontSize: "0.8rem",
                  borderRadius: "0.5rem",
                }}
              >
                {(typeof children === "string" ? children : "").replace(
                  /\n$/,
                  ""
                )}
              </SyntaxHighlighter>
            )
          },
          a(props) {
            const href = props.href ?? ""
            const { children: anchorChildren, ...rest } = props
            return (
              <a
                {...rest}
                href={href}
                onClick={(e) => {
                  if (href.startsWith("http")) {
                    e.preventDefault()
                    void openPRInBrowser(href)
                  }
                }}
                className="text-primary underline"
              >
                {anchorChildren}
              </a>
            )
          },
        }}
      >
        {body}
      </ReactMarkdown>
    </div>
  )
}
