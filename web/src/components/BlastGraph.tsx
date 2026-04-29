import * as d3 from 'd3'
import { useEffect, useRef, useCallback } from 'react'
import { GraphNodePayload, GraphEdgePayload, SimNode, SimLink } from '../types'

interface BlastGraphProps {
  nodes: Map<string, GraphNodePayload>
  edges: GraphEdgePayload[]
  selectedId: string | null
  onSelectNode: (id: string | null) => void
  isRunning: boolean
}

const NODE_RADIUS = {
  origin: 12,
  critical: 9,
  affected: 7,
  default: 6,
}

const NODE_COLOR = {
  added: '#f59e0b',
  removed: '#dc2626',
  signature_changed: '#f59e0b',
  body_changed: '#f59e0b',
  renamed: '#fb923c',
  critical: '#dc2626',
  affected: '#3b82f6',
  default: '#404040',
}

function getNodeColor(node: GraphNodePayload): string {
  return NODE_COLOR[node.changeType as keyof typeof NODE_COLOR] ?? NODE_COLOR.default
}

function getNodeRadius(node: GraphNodePayload): number {
  const isOrigin = ['added', 'removed', 'signature_changed', 'body_changed', 'renamed'].includes(node.changeType)
  if (isOrigin) return NODE_RADIUS.origin
  if (node.changeType === 'critical') return NODE_RADIUS.critical
  if (node.changeType === 'affected') return NODE_RADIUS.affected
  return NODE_RADIUS.default
}

export default function BlastGraph({ nodes, edges, selectedId, onSelectNode, isRunning }: BlastGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null)
  const simRef = useRef<d3.Simulation<SimNode, SimLink> | null>(null)
  const nodesRef = useRef<Map<string, SimNode>>(new Map())
  const gRef = useRef<d3.Selection<SVGGElement, unknown, null, undefined> | null>(null)

  const initSvg = useCallback(() => {
    const svg = svgRef.current
    if (!svg) return

    d3.select(svg).selectAll('*').remove()

    const g = d3.select(svg).append('g')
    gRef.current = g

    // Arrow marker for edges
    d3.select(svg).append('defs').selectAll('marker')
      .data(['edge', 'edge-hot'])
      .join('marker')
      .attr('id', d => `arrow-${d}`)
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 22)
      .attr('refY', 0)
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', (_d, i) => i === 0 ? '#2d2d2d' : '#dc2626')

    // Zoom behavior
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', ev => {
        g.attr('transform', ev.transform.toString())
      })

    d3.select(svg)
      .call(zoom)
      .call(zoom.translateTo, svg.clientWidth / 2, svg.clientHeight / 2)

    // Background click to deselect
    d3.select(svg).on('click.deselect', (ev) => {
      if ((ev.target as SVGElement) === svg) {
        onSelectNode(null)
      }
    })
  }, [onSelectNode])

  // Initialize SVG on mount
  useEffect(() => {
    initSvg()
  }, [initSvg])

  // Update graph when nodes/edges change
  useEffect(() => {
    const svg = svgRef.current
    const g = gRef.current
    if (!svg || !g) return

    const nodeList = Array.from(nodes.values())
    const edgeList = edges

    // Update nodesRef, preserving positions
    nodeList.forEach(n => {
      if (!nodesRef.current.has(n.id)) {
        // New node — place near center with jitter
        const cx = svg.clientWidth / 2
        const cy = svg.clientHeight / 2
        nodesRef.current.set(n.id, {
          ...n,
          x: cx + (Math.random() - 0.5) * 120,
          y: cy + (Math.random() - 0.5) * 120,
        })
      } else {
        // Update data, preserve position
        const existing = nodesRef.current.get(n.id)!
        nodesRef.current.set(n.id, { ...existing, ...n })
      }
    })

    const simNodes = Array.from(nodesRef.current.values()).filter(n => nodes.has(n.id))
    const simLinks: SimLink[] = edgeList
      .filter(e => nodes.has(e.source) && nodes.has(e.target) && e.source !== e.target)
      .map(e => ({
        ...e,
        sourceNode: nodesRef.current.get(e.source),
        targetNode: nodesRef.current.get(e.target),
      }))

    // Stop previous simulation
    simRef.current?.stop()

    const sim = d3.forceSimulation<SimNode>(simNodes)
      .force('link', d3.forceLink<SimNode, SimLink>(simLinks)
        .id(d => d.id)
        .distance(90)
        .strength(0.4)
      )
      .force('charge', d3.forceManyBody().strength(-220).distanceMax(400))
      .force('collision', d3.forceCollide<SimNode>().radius(d => getNodeRadius(d) + 8))
      .force('x', d3.forceX(svg.clientWidth / 2).strength(0.03))
      .force('y', d3.forceY(svg.clientHeight / 2).strength(0.03))
      .alphaDecay(0.025)

    simRef.current = sim

    // Render edges
    const link = g.selectAll<SVGLineElement, SimLink>('.edge')
      .data(simLinks, d => d.source + '→' + d.target)
      .join(
        enter => enter.append('line')
          .attr('class', 'edge')
          .attr('stroke', d => d.isHot ? '#7f1d1d' : '#2a2a2a')
          .attr('stroke-width', d => Math.min(3, 1 + d.frequency * 0.3))
          .attr('stroke-opacity', 0.6)
          .attr('marker-end', d => `url(#arrow-${d.isHot ? 'edge-hot' : 'edge'})`)
          .style('opacity', 0)
          .call(e => e.transition().duration(300).style('opacity', 1)),
        update => update
          .attr('stroke', d => d.isHot ? '#7f1d1d' : '#2a2a2a'),
        exit => exit.transition().duration(150).style('opacity', 0).remove()
      )

    // Render nodes
    const node = g.selectAll<SVGGElement, SimNode>('.node')
      .data(simNodes, d => d.id)
      .join(
        enter => {
          const nodeG = enter.append('g')
            .attr('class', 'node')
            .style('cursor', 'pointer')
            .style('opacity', 0)
            .call(
              d3.drag<SVGGElement, SimNode>()
                .on('start', (ev, d) => {
                  if (!ev.active) sim.alphaTarget(0.3).restart()
                  d.fx = d.x
                  d.fy = d.y
                })
                .on('drag', (ev, d) => {
                  d.fx = ev.x
                  d.fy = ev.y
                })
                .on('end', (ev, d) => {
                  if (!ev.active) sim.alphaTarget(0)
                  d.fx = null
                  d.fy = null
                })
            )
            .on('click', (ev, d) => {
              ev.stopPropagation()
              onSelectNode(d.id)
            })

          // Glow ring for changed/critical nodes
          nodeG.filter(d => d.changeType !== 'affected' && d.changeType !== 'default')
            .append('circle')
            .attr('class', 'node-glow')
            .attr('r', d => getNodeRadius(d) + 5)
            .attr('fill', 'none')
            .attr('stroke', d => getNodeColor(d))
            .attr('stroke-opacity', 0.25)
            .attr('stroke-width', 4)

          // Main circle
          nodeG.append('circle')
            .attr('class', 'node-circle')
            .attr('r', d => getNodeRadius(d))
            .attr('fill', d => getNodeColor(d))
            .attr('fill-opacity', d => d.changeType === 'affected' ? 0.7 : 1)
            .attr('stroke', d => {
              const c = getNodeColor(d)
              return c
            })
            .attr('stroke-opacity', 0.4)
            .attr('stroke-width', 1)

          // Label
          nodeG.append('text')
            .attr('class', 'node-label')
            .attr('dy', d => getNodeRadius(d) + 10)
            .attr('text-anchor', 'middle')
            .attr('fill', '#a0a0a0')
            .attr('font-family', 'JetBrains Mono, monospace')
            .attr('font-size', '9px')
            .text(d => truncateLabel(d.label, 16))

          nodeG.transition().duration(250).delay((_, i) => i * 15).style('opacity', 1)

          return nodeG
        },
        update => {
          // Update circle fill in case changeType changed
          update.select('.node-circle')
            .attr('fill', d => getNodeColor(d))
          return update
        },
        exit => exit.transition().duration(150).style('opacity', 0).remove()
      )

    // Tick function
    sim.on('tick', () => {
      link
        .attr('x1', d => (d as unknown as { source: SimNode }).source.x ?? 0)
        .attr('y1', d => (d as unknown as { source: SimNode }).source.y ?? 0)
        .attr('x2', d => (d as unknown as { target: SimNode }).target.x ?? 0)
        .attr('y2', d => (d as unknown as { target: SimNode }).target.y ?? 0)

      node.attr('transform', d => `translate(${d.x ?? 0},${d.y ?? 0})`)
    })

    // Pulse animation for critical nodes
    const criticalNodes = g.selectAll<SVGGElement, SimNode>('.node')
      .filter(d => d.changeType === 'critical')

    criticalNodes.select('.node-glow')
      .each(function () {
        const el = d3.select(this)
        function pulse() {
          el.transition().duration(1200).ease(d3.easeSinInOut)
            .attr('stroke-opacity', 0.5)
            .attr('r', function (d) { return getNodeRadius(d as SimNode) + 9 })
            .transition().duration(1200).ease(d3.easeSinInOut)
            .attr('stroke-opacity', 0.1)
            .attr('r', function (d) { return getNodeRadius(d as SimNode) + 4 })
            .on('end', pulse)
        }
        pulse()
      })

  }, [nodes, edges, onSelectNode])

  // Highlight selected node
  useEffect(() => {
    const g = gRef.current
    if (!g) return

    g.selectAll<SVGGElement, SimNode>('.node')
      .select('.node-circle')
      .attr('stroke-width', d => d.id === selectedId ? 2.5 : 1)
      .attr('stroke-opacity', d => d.id === selectedId ? 1 : 0.4)
      .attr('filter', d => d.id === selectedId ? 'brightness(1.3)' : 'none')
  }, [selectedId])

  return (
    <div className="graph-wrap">
      {nodes.size === 0 && !isRunning && (
        <div className="graph-empty" aria-live="polite">
          <svg width="48" height="48" viewBox="0 0 48 48" fill="none" aria-hidden="true">
            <circle cx="12" cy="24" r="5" stroke="#2d2d2d" strokeWidth="1.5" />
            <circle cx="36" cy="12" r="5" stroke="#2d2d2d" strokeWidth="1.5" />
            <circle cx="36" cy="36" r="5" stroke="#2d2d2d" strokeWidth="1.5" />
            <line x1="17" y1="22" x2="31" y2="14" stroke="#2d2d2d" strokeWidth="1" strokeDasharray="3 2" />
            <line x1="17" y1="26" x2="31" y2="34" stroke="#2d2d2d" strokeWidth="1" strokeDasharray="3 2" />
          </svg>
          <p className="graph-empty-text">Blast radius will appear here</p>
        </div>
      )}

      {isRunning && nodes.size > 0 && (
        <div className="graph-live-badge" aria-live="polite">
          <span className="graph-live-dot" />
          Live
        </div>
      )}

      <div className="graph-legend" aria-label="Graph legend">
        <div className="legend-item">
          <span className="legend-dot" style={{ background: '#f59e0b' }} />
          <span>Changed</span>
        </div>
        <div className="legend-item">
          <span className="legend-dot" style={{ background: '#dc2626' }} />
          <span>Critical path</span>
        </div>
        <div className="legend-item">
          <span className="legend-dot" style={{ background: '#3b82f6' }} />
          <span>Affected</span>
        </div>
      </div>

      <svg
        ref={svgRef}
        className="graph-svg"
        aria-label="Call blast radius graph"
        role="img"
      />

      <style>{`
        .graph-wrap {
          position: relative;
          flex: 1;
          overflow: hidden;
          background: var(--bg);
        }
        .graph-svg {
          width: 100%;
          height: 100%;
          display: block;
        }
        .graph-empty {
          position: absolute;
          inset: 0;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: var(--space-4);
          pointer-events: none;
        }
        .graph-empty-text {
          font-size: var(--text-sm);
          color: var(--text-dim);
        }
        .graph-live-badge {
          position: absolute;
          top: var(--space-4);
          left: var(--space-4);
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 4px 10px;
          background: var(--surface);
          border: 1px solid var(--border);
          border-radius: 20px;
          font-size: var(--text-xs);
          color: var(--text-secondary);
          z-index: 10;
        }
        .graph-live-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          background: var(--amber);
          animation: livePulse 1.2s ease-in-out infinite;
        }
        @keyframes livePulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.4; transform: scale(0.8); }
        }
        .graph-legend {
          position: absolute;
          bottom: var(--space-4);
          left: var(--space-4);
          display: flex;
          gap: var(--space-4);
          padding: var(--space-2) var(--space-3);
          background: var(--surface);
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
          z-index: 10;
        }
        .legend-item {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          color: var(--text-secondary);
        }
        .legend-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
          flex-shrink: 0;
        }
      `}</style>
    </div>
  )
}

function truncateLabel(label: string, max: number): string {
  if (label.length <= max) return label
  return label.slice(0, max - 1) + '…'
}
