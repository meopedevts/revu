import { useId } from "react"

interface LogoProps {
  className?: string
  /** Decorativo quando acompanhado de texto "revu" (evita leitura dupla). */
  decorative?: boolean
}

export function Logo({ className, decorative = false }: LogoProps) {
  const maskId = useId()

  if (decorative) {
    return (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 64 64"
        className={className}
        aria-hidden="true"
        focusable="false"
      >
        <mask id={maskId}>
          <rect width="64" height="64" fill="#fff" />
          <text
            x="32"
            y="46"
            fontFamily="Inter, 'Inter Variable', system-ui, sans-serif"
            fontSize="40"
            fontWeight="600"
            letterSpacing="-2"
            textAnchor="middle"
            fill="#000"
          >
            r
          </text>
        </mask>
        <rect
          width="64"
          height="64"
          fill="currentColor"
          mask={`url(#${maskId})`}
          rx="14"
          ry="14"
        />
      </svg>
    )
  }

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 64 64"
      className={className}
      role="img"
      aria-label="revu"
    >
      <title>revu</title>
      <mask id={maskId}>
        <rect width="64" height="64" fill="#fff" />
        <text
          x="32"
          y="46"
          fontFamily="Inter, 'Inter Variable', system-ui, sans-serif"
          fontSize="40"
          fontWeight="600"
          letterSpacing="-2"
          textAnchor="middle"
          fill="#000"
        >
          r
        </text>
      </mask>
      <rect
        width="64"
        height="64"
        fill="currentColor"
        mask={`url(#${maskId})`}
        rx="14"
        ry="14"
      />
    </svg>
  )
}
