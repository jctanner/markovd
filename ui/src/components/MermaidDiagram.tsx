import { useEffect, useRef, useState } from 'react';
import mermaid from 'mermaid';

let mermaidInitialized = false;

function initMermaid(theme: 'dark' | 'default') {
  mermaid.initialize({
    startOnLoad: false,
    theme,
    securityLevel: 'loose',
    flowchart: { curve: 'basis', padding: 16 },
    fontFamily: 'var(--font-mono)',
  });
  mermaidInitialized = true;
}

interface Props {
  code: string;
}

export default function MermaidDiagram({ code }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [svg, setSvg] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    const isDark = document.documentElement.getAttribute('data-theme') !== 'light';
    const theme = isDark ? 'dark' : 'default';

    if (!mermaidInitialized) {
      initMermaid(theme);
    } else {
      mermaid.initialize({ theme });
    }

    const id = `mermaid-${Date.now()}`;
    mermaid.render(id, code)
      .then(({ svg: rendered }) => {
        setSvg(rendered);
        setError('');
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to render diagram');
        setSvg('');
      });
  }, [code]);

  if (error) {
    return <div className="msg-error" style={{ fontSize: 12 }}>{error}</div>;
  }

  return (
    <div
      ref={containerRef}
      className="mermaid-diagram"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  );
}
