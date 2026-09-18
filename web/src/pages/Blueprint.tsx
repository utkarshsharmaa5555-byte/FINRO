import { useEffect, useState, type ReactNode } from 'react'
import { api, rupeesInr, useApp, useData } from '../lib'
import { BlueprintView, Empty, ErrorLine, Icon, PageHead, SpeakButton, Spinner, type Blueprint } from '../ui'

type Slot = 'today' | 'investment' | 'vision'
type ImageMeta = { slot: Slot; model: string; prompt: string; v: number }
type SlotResult = { slot: Slot; ok: boolean; model?: string; prompt?: string; error?: string }

const FUND_COLORS = ['var(--ink)', 'var(--stamp)', 'var(--leaf)', 'var(--turmeric)', 'var(--ink-3)', 'var(--margin)']

// Drawn stand-in scenes in the ledger style, used until (or unless) AI images exist.
function Scene({ slot }: { slot: Slot }) {
  const ink = 'var(--ink)'
  const common = { fill: 'none', stroke: ink, strokeWidth: 2.2, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const }
  return (
    <svg viewBox="0 0 240 160" role="img" aria-label={slot} className="scene">
      <rect x="0" y="0" width="240" height="160" fill="var(--paper-2)" />
      {[40, 72, 104, 136].map((y) => <line key={y} x1="0" x2="240" y1={y} y2={y} stroke="var(--rule)" />)}
      <line x1="0" x2="240" y1="128" y2="128" stroke={ink} strokeWidth="1.5" />
      {slot === 'today' && (
        <g>
          <circle cx="198" cy="34" r="14" fill="var(--turmeric)" opacity=".55" />
          <path d="M60 88h96l-8 30H68z" {...common} fill="var(--card)" />
          <path d="M56 88l20-26h64l20 26" {...common} fill="var(--stamp)" fillOpacity=".25" />
          <circle cx="78" cy="126" r="9" {...common} fill="var(--card)" />
          <circle cx="142" cy="126" r="9" {...common} fill="var(--card)" />
          <path d="M84 84c4-8 12-8 16 0M112 84c4-8 12-8 16 0" {...common} />
          <path d="M186 128v-30m0 0c-8 0-12-10-6-16s18-2 16 8" {...common} />
        </g>
      )}
      {slot === 'investment' && (
        <g>
          <rect x="40" y="70" width="70" height="56" rx="4" {...common} fill="var(--card)" />
          <path d="M40 84h70M75 70v56" {...common} />
          <path d="M140 126V92h60v34" {...common} fill="var(--card)" />
          <path d="M136 92l34-22 34 22" {...common} fill="var(--leaf)" fillOpacity=".2" />
          {[0, 1, 2].map((i) => <ellipse key={i} cx={170} cy={58 - i * 9} rx="16" ry="5" {...common} fill="var(--turmeric)" fillOpacity=".5" />)}
          <path d="M116 100h18m-6-6l6 6-6 6" {...common} stroke="var(--stamp)" />
        </g>
      )}
      {slot === 'vision' && (
        <g>
          <path d="M0 128c40-30 80-40 120-40s80 10 120 40" fill="var(--turmeric)" fillOpacity=".25" stroke="none" />
          <circle cx="120" cy="84" r="24" fill="var(--turmeric)" opacity=".6" />
          {[0, 1, 2, 3, 4].map((i) => <line key={i} x1={120 + Math.cos((Math.PI * (i + 1)) / 6) * 32} y1={84 - Math.sin((Math.PI * (i + 1)) / 6) * 32} x2={120 + Math.cos((Math.PI * (i + 1)) / 6) * 44} y2={84 - Math.sin((Math.PI * (i + 1)) / 6) * 44} {...common} stroke="var(--turmeric)" />)}
          {[30, 170].map((x) => (
            <g key={x}>
              <path d={`M${x} 104h44l-4 16h-36z`} {...common} fill="var(--card)" />
              <path d={`M${x - 2} 104l10-14h32l10 14`} {...common} fill="var(--stamp)" fillOpacity=".25" />
              <circle cx={x + 10} cy="124" r="5" {...common} fill="var(--card)" />
              <circle cx={x + 34} cy="124" r="5" {...common} fill="var(--card)" />
            </g>
          ))}
          <path d="M96 40l8-10 8 10M104 30v18" {...common} stroke="var(--leaf)" />
        </g>
      )}
    </svg>
  )
}

function Picture({ bpId, slot, meta, painting }: { bpId: number; slot: Slot; meta?: ImageMeta; painting: boolean }) {
  const { t } = useApp()
  const [showPrompt, setShowPrompt] = useState(false)
  return (
    <figure className={`flow-img ${painting ? 'painting' : ''}`}>
      {meta ? <img src={`/api/blueprint/${bpId}/image/${slot}?v=${meta.v}`} alt={meta.prompt} loading="lazy" /> : <Scene slot={slot} />}
      {meta && (
        <figcaption>
          <button className="chip" onClick={() => setShowPrompt(!showPrompt)} aria-expanded={showPrompt}>✦ AI · {meta.model}</button>
          {showPrompt && <p className="tiny muted" style={{ marginTop: 4 }}><strong>{t('ai_prompt')}:</strong> {meta.prompt}</p>}
        </figcaption>
      )}
    </figure>
  )
}

function Node({ n, title, children, picture, tone }: { n: number; title: string; children: ReactNode; picture?: ReactNode; tone?: string }) {
  return (
    <>
      <section className="flow-node sheet" style={tone ? { borderColor: tone } : undefined}>
        <div className="flow-step" style={tone ? { background: tone } : undefined}>{n}</div>
        <div className={picture ? 'flow-body with-pic' : 'flow-body'}>
          <div className="stack" style={{ gap: '0.55rem' }}>
            <h3 className="display" style={{ fontSize: '1.35rem' }}>{title}</h3>
            {children}
          </div>
          {picture}
        </div>
      </section>
    </>
  )
}

const Arrow = () => (
  <div className="flow-arrow" aria-hidden="true">
    <svg width="24" height="40" viewBox="0 0 24 40"><path d="M12 0v30" stroke="var(--ink)" strokeWidth="2" strokeDasharray="4 4" /><path d="M5 27l7 10 7-10" fill="none" stroke="var(--ink)" strokeWidth="2" strokeLinejoin="round" /></svg>
  </div>
)

function Flowchart({ b, images, painting }: { b: Blueprint; images: ImageMeta[]; painting: boolean }) {
  const { t } = useApp()
  const d = b.data
  const fin = d.financials ?? {}
  const img = (slot: Slot) => images.find((i) => i.slot === slot)
  const funds: { item: string; amount_inr: number }[] = d.funding_need?.use_of_funds ?? []
  const total = funds.reduce((s, f) => s + (f.amount_inr || 0), 0) || d.funding_need?.amount_inr || 0
  const milestones: { month: number; goal: string }[] = [...(d.milestones ?? [])].sort((a, b) => a.month - b.month)
  const risks: { risk: string; mitigation: string }[] = d.risks ?? []
  const speak = [d.business_name, d.one_liner, d.problem, d.funding_need?.purpose, d.vision].filter(Boolean).join('. ')

  return (
    <div className="flow">
      <header className="flow-hero sheet">
        <div className="stack" style={{ gap: '0.5rem' }}>
          <div className="row wrap" style={{ gap: 6 }}><span className="stamp green">{d.stage}</span><span className="caps faint">{t('blueprint')} · {b.created_at}</span></div>
          <h2 className="display" style={{ fontSize: 'clamp(1.8rem, 4vw, 2.6rem)' }}>{d.business_name}</h2>
          <p style={{ fontSize: '1.1rem' }}>{d.one_liner}</p>
          <div className="row"><SpeakButton text={speak} /></div>
        </div>
        <Picture bpId={b.id} slot="vision" meta={img('vision')} painting={painting} />
      </header>
      <Arrow />

      <Node n={1} title={t('fl_today')} picture={<Picture bpId={b.id} slot="today" meta={img('today')} painting={painting} />}>
        {d.offering && <p><strong>→</strong> {d.offering}</p>}
        {d.customers && <p className="small muted">{d.customers}</p>}
        <div className="grid3" style={{ gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '0.5rem' }}>
          <div className="stat"><div className="label">{t('sales')} 30d</div><div className="value green" style={{ fontSize: '1.1rem' }}>{rupeesInr(fin.sales_last_30d_inr ?? 0)}</div></div>
          <div className="stat"><div className="label">{t('profit')} 30d</div><div className="value" style={{ fontSize: '1.1rem' }}>{rupeesInr(fin.profit_last_30d_inr ?? 0)}</div></div>
          <div className="stat"><div className="label">{t('avg_day')}</div><div className="value" style={{ fontSize: '1.1rem' }}>{rupeesInr(fin.avg_daily_sale_inr ?? 0)}</div></div>
        </div>
        <p className="tiny faint">✓ {t('financials')}</p>
      </Node>
      <Arrow />

      <Node n={2} title={t('fl_opportunity')} tone="var(--turmeric)">
        {d.problem && <p>{d.problem}</p>}
        {d.growth_drivers?.length > 0 && (
          <div><div className="caps muted">▲ {t('growth_drivers')}</div><ul className="do-list small">{d.growth_drivers.map((g: string, i: number) => <li key={i}>{g}</li>)}</ul></div>
        )}
        {d.strengths?.length > 0 && <div className="row wrap" style={{ gap: 4 }}>{d.strengths.map((s: string, i: number) => <span key={i} className="chip green">✓ {s}</span>)}</div>}
      </Node>
      <Arrow />

      <Node n={3} title={t('fl_investment')} tone="var(--stamp)" picture={<Picture bpId={b.id} slot="investment" meta={img('investment')} painting={painting} />}>
        <div className="row wrap" style={{ alignItems: 'baseline', gap: 8 }}>
          <span className="mono" style={{ fontSize: '2rem', fontWeight: 700, color: 'var(--stamp)' }}>{rupeesInr(d.funding_need?.amount_inr ?? total)}</span>
          {d.funding_need?.purpose && <span className="muted">{d.funding_need.purpose}</span>}
        </div>
        {funds.length > 0 && total > 0 && (
          <>
            <div className="fund-bar" role="img" aria-label={t('use_of_funds')}>
              {funds.map((f, i) => <span key={i} style={{ width: `${(f.amount_inr / total) * 100}%`, background: FUND_COLORS[i % FUND_COLORS.length] }} title={`${f.item}: ${rupeesInr(f.amount_inr)}`} />)}
            </div>
            <ul className="fund-legend">
              {funds.map((f, i) => (
                <li key={i}><span className="dot" style={{ background: FUND_COLORS[i % FUND_COLORS.length] }} />{f.item}<span className="mono">{rupeesInr(f.amount_inr)} · {Math.round((f.amount_inr / total) * 100)}%</span></li>
              ))}
            </ul>
          </>
        )}
      </Node>
      <Arrow />

      {milestones.length > 0 && (
        <>
          <Node n={4} title={t('fl_milestones')} tone="var(--leaf)">
            <ol className="timeline">
              {milestones.map((m, i) => (
                <li key={i}>
                  <span className="tl-month mono">{t('month')} {m.month}</span>
                  <span className="tl-dot" />
                  <span className="tl-goal">{m.goal}</span>
                </li>
              ))}
            </ol>
          </Node>
          <Arrow />
        </>
      )}

      {risks.length > 0 && (
        <>
          <Node n={5} title={t('fl_risks')}>
            <div className="stack" style={{ gap: '0.6rem' }}>
              {risks.map((r, i) => (
                <div key={i} className="risk-row">
                  <div className="diamond"><span>?</span></div>
                  <div className="risk-box">{r.risk}</div>
                  <div className="risk-arrow" aria-hidden="true">→</div>
                  <div className="risk-box ok">{r.mitigation}</div>
                </div>
              ))}
            </div>
          </Node>
          <Arrow />
        </>
      )}

      <Node n={milestones.length && risks.length ? 6 : 4} title={t('fl_vision')} tone="var(--leaf)">
        <p style={{ fontSize: '1.15rem' }}>{d.vision || d.revenue_model}</p>
        {d.vision && d.revenue_model && <p className="small muted">{d.revenue_model}</p>}
      </Node>

      {d.missing_info?.length > 0 && (
        <div className="sticky-row">
          <div className="caps muted" style={{ width: '100%' }}>{t('missing_info')}</div>
          {d.missing_info.map((m: string, i: number) => <div key={i} className="sticky" style={{ transform: `rotate(${(i % 3) - 1.2}deg)` }}>{m}</div>)}
        </div>
      )}
    </div>
  )
}

export function BlueprintPage() {
  const { t, toast } = useApp()
  const bp = useData<Blueprint | null>('/blueprint')
  const [tab, setTab] = useState<'flow' | 'doc'>('flow')
  const [images, setImages] = useState<ImageMeta[]>([])
  const [building, setBuilding] = useState(false)
  const [painting, setPainting] = useState(false)
  const [unavailable, setUnavailable] = useState(false)
  const [err, setErr] = useState('')
  const b = bp.data

  const loadImages = async (id: number) => setImages(await api<ImageMeta[]>(`/blueprint/${id}/images`).catch(() => []))
  useEffect(() => { if (b?.id) loadImages(b.id) }, [b?.id])

  const build = async () => {
    setBuilding(true); setErr('')
    try { bp.setData(await api<Blueprint>('/blueprint', { body: {} })); setImages([]); setUnavailable(false) } catch (e) { setErr((e as Error).message) } finally { setBuilding(false) }
  }
  const paint = async (force = false) => {
    if (!b) return
    setPainting(true); setErr('')
    try {
      const r = await api<{ slots: SlotResult[] }>(`/blueprint/${b.id}/illustrate${force ? '?force=1' : ''}`, { method: 'POST' })
      const failed = r.slots.filter((s) => !s.ok)
      setUnavailable(failed.length === r.slots.length)
      if (failed.length && failed.length < r.slots.length) toast(`${failed.length} / ${r.slots.length}: ${failed[0].error}`)
      await loadImages(b.id)
    } catch (e) { setErr((e as Error).message) } finally { setPainting(false) }
  }

  return (
    <>
      <PageHead kicker={t('matching_kicker')} title={t('blueprint')}>
        <button className="btn sm" onClick={build} disabled={building || painting}>{building ? <Spinner /> : <Icon name="book" size={15} />}{b ? t('rebuild') : t('build_blueprint')}</button>
        {b && <button className="btn primary sm" onClick={() => paint(images.length === 3)} disabled={building || painting}>{painting ? <Spinner /> : <span aria-hidden="true">✦</span>}{images.length === 3 ? t('reillustrate') : t('illustrate')}</button>}
      </PageHead>
      <ErrorLine msg={err || bp.error} />
      {painting && <div className="sheet plain row" style={{ marginBottom: '1rem' }}><Spinner /><span className="small">{t('illustrating')}</span></div>}
      {unavailable && <p className="chip yellow" style={{ marginBottom: '1rem' }}>{t('images_unavailable')}</p>}
      {bp.loading && !b && <div className="skeleton" style={{ height: 420 }} />}
      {!bp.loading && !b && (
        <Empty title={t('no_blueprint')} hint={t('no_blueprint_hint')}>
          <button className="btn primary" onClick={build} disabled={building}>{building ? <Spinner /> : <Icon name="book" size={18} />}{t('build_blueprint')}</button>
        </Empty>
      )}
      {b && (
        <>
          <div className="seg" role="tablist" style={{ marginBottom: '1rem' }}>
            <button role="tab" aria-selected={tab === 'flow'} aria-pressed={tab === 'flow'} onClick={() => setTab('flow')}>{t('flowchart')}</button>
            <button role="tab" aria-selected={tab === 'doc'} aria-pressed={tab === 'doc'} onClick={() => setTab('doc')}>{t('document')}</button>
          </div>
          {tab === 'flow' ? <Flowchart b={b} images={images} painting={painting} /> : <BlueprintView b={b} />}
        </>
      )}
    </>
  )
}
