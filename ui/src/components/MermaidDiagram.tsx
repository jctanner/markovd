import { useEffect, useRef, useState } from 'react';
import mermaid from 'mermaid';

function configureMermaid(isDark: boolean) {
  mermaid.initialize({
    startOnLoad: false,
    theme: 'base',
    securityLevel: 'loose',
    flowchart: { curve: 'basis', padding: 16 },
    fontFamily: 'var(--font-mono)',
    themeVariables: isDark
      ? {
          primaryColor: '#1e293b',
          primaryTextColor: '#f1f5f9',
          primaryBorderColor: '#334155',
          lineColor: '#475569',
          secondaryColor: '#162032',
          tertiaryColor: '#0f1525',
          background: '#0f1525',
          mainBkg: '#1e293b',
          nodeBorder: '#334155',
          clusterBkg: '#0f1525',
          clusterBorder: '#1e293b',
          titleColor: '#f1f5f9',
          edgeLabelBackground: '#162032',
        }
      : {
          primaryColor: '#e2e8f0',
          primaryTextColor: '#0f172a',
          primaryBorderColor: '#cbd5e1',
          lineColor: '#64748b',
          secondaryColor: '#f1f5f9',
          tertiaryColor: '#f8fafc',
          background: '#ffffff',
          mainBkg: '#e2e8f0',
          nodeBorder: '#cbd5e1',
          clusterBkg: '#f8fafc',
          clusterBorder: '#e2e8f0',
          titleColor: '#0f172a',
          edgeLabelBackground: '#ffffff',
        },
  });
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
    configureMermaid(isDark);

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
