import { useState, type ReactNode } from 'react'
import { LANGS, type Lang } from './i18n'
import { ago, api, fmtDate, Link, Md, rupees, rupeesInr, speakText, stopSpeaking, useApp, useVoice } from './lib'

// ---------- icons (stroke, hand-inked feel) ----------
const paths: Record<string, string> = {
  chat: 'M4 5h16v11H9l-5 4V5z M8 9h8 M8 12h5',
  shield: 'M12 3l7 3v6c0 4.5-3 7.5-7 9-4-1.5-7-4.5-7-9V6l7-3z M9 12l2 2 4-4',
  qr: 'M4 4h6v6H4z M14 4h6v6h-6z M4 14h6v6H4z M14 14h2v2h-2z M18 14h2 M14 18h2v2 M18 18h2v2h-2',
  book: 'M5 4h11a3 3 0 013 3v13H8a3 3 0 01-3-3V4z M5 17a3 3 0 013-3h11 M9 8h6',
  coins: 'M12 7c4 0 7-1.3 7-3s-3-3-7-3-7 1.3-7 3 3 3 7 3z M5 4v5c0 1.7 3 3 7 3s7-1.3 7-3V4 M5 9v5c0 1.7 3 3 7 3s7-1.3 7-3V9 M5 14v5c0 1.7 3 3 7 3s7-1.3 7-3v-5',
  target: 'M12 21a9 9 0 100-18 9 9 0 000 18z M12 16a4 4 0 100-8 4 4 0 000 8z M12 12h.01',
  file: 'M7 3h7l5 5v13H7z M14 3v5h5 M10 13h6 M10 17h6',
  user: 'M12 12a4 4 0 100-8 4 4 0 000 8z M4 21c1.5-4 4.5-6 8-6s6.5 2 8 6',
  help: 'M12 21a9 9 0 100-18 9 9 0 000 18z M9.5 9a2.5 2.5 0 015 .5c0 1.5-2.5 2-2.5 3.5 M12 17h.01',
  gear: 'M12 15a3 3 0 100-6 3 3 0 000 6z M19 12l2-1-1-3-2 .2-1.3-1.3.2-2-3-1-1 2h-1.8l-1-2-3 1 .2 2L6 8.2 4 8 3 11l2 1v1.8L3 15l1 3 2-.2 1.3 1.3-.2 2 3 1 1-2h1.8l1 2 3-1-.2-2 1.3-1.3 2 .2 1-3-2-1z',
  radar: 'M3 12h4l3-8 4 16 3-8h4',
  mic: 'M12 15a3 3 0 003-3V6a3 3 0 10-6 0v6a3 3 0 003 3z M5 11a7 7 0 0014 0 M12 18v3',
  speaker: 'M4 9v6h4l5 4V5L8 9H4z M16 9a4 4 0 010 6 M18.5 6.5a8 8 0 010 11',
  stopsq: 'M7 7h10v10H7z',
  send: 'M4 12l16-8-6 16-2-6-8-2z',
  clip: 'M16 7l-7.5 7.5a2 2 0 002.8 2.8L19 9.6a4 4 0 00-5.7-5.7L5.6 11.6a6 6 0 008.5 8.5L20 14',
  sun: 'M12 17a5 5 0 100-10 5 5 0 000 10z M12 1v2 M12 21v2 M4.2 4.2l1.4 1.4 M18.4 18.4l1.4 1.4 M1 12h2 M21 12h2 M4.2 19.8l1.4-1.4 M18.4 5.6l1.4-1.4',
  moon: 'M20 14.5A8 8 0 019.5 4 8 8 0 1020 14.5z',
  plus: 'M12 5v14 M5 12h14',
  x: 'M6 6l12 12 M18 6L6 18',
  menu: 'M4 7h16 M4 12h16 M4 17h16',
  ext: 'M14 4h6v6 M20 4l-9 9 M18 14v6H4V6h6',
  refresh: 'M20 11a8 8 0 10-2.3 5.7 M20 5v6h-6',
  camera: 'M4 8h3l2-3h6l2 3h3v11H4z M12 16a3 3 0 100-6 3 3 0 000 6z',
  download: 'M12 4v11 M7 10l5 5 5-5 M5 20h14',
  printer: 'M7 9V3h10v6 M7 17H4v-7h16v7h-3 M7 14h10v7H7z',
  trash: 'M4 7h16 M9 7V4h6v3 M6 7l1 13h10l1-13',
  check: 'M5 12l5 5 9-10',
  dash: 'M4 4h7v9H4z M13 4h7v5h-7z M13 11h7v9h-7z M4 15h7v5H4z',
  flow: 'M4 4h7v5H4z M13 15h7v5h-7z M7.5 9v3.5h9V15',
  sparkles: 'M12 3l1.9 4.5L18.5 9.5l-4.6 2-1.9 4.5-1.9-4.5-4.6-2 4.6-2L12 3z M19 15l1 2.2 2.2 1-2.2 1-1 2.2-1-2.2-2.2-1 2.2-1 1-2.2z',
  pencil: 'M12 20h9 M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z',
}
export function Icon({ name, size = 20, title, style, className }: { name: string; size?: number; title?: string; style?: React.CSSProperties; className?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden={!title} role={title ? 'img' : undefined} style={style} className={className}>
      {title && <title>{title}</title>}
      <path d={paths[name]} />
    </svg>
  )
}

export function Spinner() {
  return <span className="typing" aria-label="loading"><span /><span /><span /></span>
}

export function LangSelect({ compact }: { compact?: boolean }) {
  const { lang, setLang, t } = useApp()
  return (
    <label className="row" style={{ gap: 6 }}>
      <span className={compact ? 'sr' : 'small muted'}>{t('lang')}</span>
      <select className="select" style={{ minHeight: 38, width: 'auto', padding: '0.3rem 0.5rem' }} value={lang} onChange={(e) => setLang(e.target.value as Lang)} aria-label={t('lang')}>
        {LANGS.map((l) => <option key={l.code} value={l.code}>{l.label}</option>)}
      </select>
    </label>
  )
}

export function ThemeToggle() {
  const { theme, toggleTheme, t } = useApp()
  return (
    <button className="btn ghost icon sm" onClick={toggleTheme} aria-label={theme === 'dark' ? t('theme_light') : t('theme_dark')} title={theme === 'dark' ? t('theme_light') : t('theme_dark')}>
      <Icon name={theme === 'dark' ? 'sun' : 'moon'} />
    </button>
  )
}

export function SpeakButton({ text }: { text: string }) {
  const { lang, t } = useApp()
  const [on, setOn] = useState(false)
  return (
    <button className="btn ghost sm icon" aria-label={on ? t('stop') : t('read_aloud')} title={on ? t('stop') : t('read_aloud')}
      onClick={() => { if (on) { stopSpeaking(); setOn(false) } else { setOn(true); speakText(text, lang, () => setOn(false)) } }}>
      <Icon name={on ? 'stopsq' : 'speaker'} size={18} />
    </button>
  )
}

export function MicButton({ onText, className = 'btn icon' }: { onText: (s: string) => void; className?: string }) {
  const { t } = useApp()
  const v = useVoice(onText)
  return (
    <button type="button" className={`${className} ${v.recording ? 'rec' : ''}`} onClick={v.toggle} disabled={v.busy}
      aria-label={v.recording ? t('listening') : t('speak')} title={v.recording ? t('listening') : t('speak')} aria-pressed={v.recording}>
      {v.busy ? <Spinner /> : <Icon name={v.recording ? 'stopsq' : 'mic'} />}
    </button>
  )
}

export function PageHead({ kicker, title, children }: { kicker: string; title: string; children?: ReactNode }) {
  return (
    <header className="page-head">
      <div>
        <div className="caps kicker">{kicker}</div>
        <h1 className="display">{title}</h1>
      </div>
      {children && <div className="row wrap">{children}</div>}
    </header>
  )
}

export function Empty({ title, hint, children }: { title: string; hint?: string; children?: ReactNode }) {
  return (
    <div className="sheet empty">
      <div className="display">{title}</div>
      {hint && <p>{hint}</p>}
      {children && <div style={{ marginTop: '1rem' }}>{children}</div>}
    </div>
  )
}

export function ErrorLine({ msg }: { msg: string }) {
  return msg ? <p className="error-text" role="alert">{msg}</p> : null
}

// ---------- domain cards (used in chat and pages) ----------

export type Verdict = { verdict: 'scam' | 'suspicious' | 'looks_safe'; confidence: number; headline: string; matched_patterns: string[]; red_flags_found: string[]; what_to_do: string[]; explanation: string }

export function VerdictCard({ v }: { v: Verdict }) {
  const { t } = useApp()
  const color = v.verdict === 'scam' ? 'red' : v.verdict === 'suspicious' ? 'yellow' : 'green'
  const label = t(`verdict_${v.verdict}` as 'verdict_scam')
  return (
    <div className={`sheet card-verdict ${v.verdict}`}>
      <div className="row between wrap">
        <span className={`stamp big thump ${color}`}>{label}</span>
        <span className="tiny faint mono">{v.confidence}% {t('confidence')}</span>
      </div>
      <h3>{v.headline}</h3>
      <p className="muted">{v.explanation}</p>
      {v.red_flags_found.length > 0 && (
        <div><div className="caps muted">{t('red_flags')}</div><ul className="flag-list">{v.red_flags_found.map((f, i) => <li key={i}>{f}</li>)}</ul></div>
      )}
      <div><div className="caps muted">{t('what_to_do')}</div><ol className="do-list">{v.what_to_do.map((f, i) => <li key={i}>{f}</li>)}</ol></div>
      <div className="row between wrap">
        {v.verdict !== 'looks_safe' ? <a className="chip red" href="tel:1930">☎ {t('helpline')}</a> : <span />}
        <SpeakButton text={[label, v.headline, v.explanation, ...v.what_to_do].join('. ')} />
      </div>
    </div>
  )
}

export type QR = { id: number; vpa: string; payee: string; amount_paise: number | null; upi_link: string }
export function QRCard({ q }: { q: QR }) {
  const { t } = useApp()
  return (
    <div className="sheet row wrap" style={{ alignItems: 'flex-start', gap: '1rem' }}>
      <div className="qr-frame"><img src={`/api/qr/${q.id}/png?size=360`} alt={`UPI QR for ${q.payee}`} /></div>
      <div className="stack grow" style={{ minWidth: 180 }}>
        <span className="stamp green flat" style={{ width: 'fit-content' }}>UPI · {q.vpa}</span>
        <h3 className="display" style={{ fontSize: '1.4rem' }}>{q.payee}</h3>
        {q.amount_paise ? <div className="amt">{rupees(q.amount_paise)}</div> : null}
        <p className="small muted">{t('qr_tip')}</p>
        <div className="row wrap">
          <Link className="btn sm primary" to={`/app/qr/${q.id}/standee`}><Icon name="printer" size={16} />{t('print_standee')}</Link>
          <a className="btn sm" href={`/api/qr/${q.id}/png?size=1024`} download={`finro-qr-${q.vpa}.png`}><Icon name="download" size={16} />PNG</a>
        </div>
      </div>
    </div>
  )
}

export type Entry = { id?: number; date: string; kind: 'sale' | 'expense' | 'udhaar_given' | 'udhaar_received'; amount_paise: number; note: string; party: string; source?: string }
const kindChip: Record<string, string> = { sale: 'green', expense: 'red', udhaar_given: 'yellow', udhaar_received: 'green' }

export function EntriesProposal({ entries, unclear, onSaved }: { entries: Entry[]; unclear?: string; onSaved?: () => void }) {
  const { t, lang, toast } = useApp()
  const [rows, setRows] = useState(entries)
  const [saved, setSaved] = useState(false)
  const [busy, setBusy] = useState(false)
  if (!rows.length) return <div className="sheet small muted">{t('unclear')}{unclear ? `: ${unclear}` : ''}</div>
  const save = async () => {
    setBusy(true)
    try {
      await api('/records', { body: { entries: rows } })
      setSaved(true)
      toast(t('saved'))
      onSaved?.()
    } catch (e) { toast((e as Error).message) } finally { setBusy(false) }
  }

  const salesTotal = rows.filter(r => r.kind === 'sale').reduce((acc, r) => acc + r.amount_paise, 0)
  const expTotal = rows.filter(r => r.kind === 'expense').reduce((acc, r) => acc + r.amount_paise, 0)
  const udhTotal = rows.filter(r => r.kind.startsWith('udhaar')).reduce((acc, r) => acc + r.amount_paise, 0)

  return (
    <div className="sheet stack" style={{ border: '1.5px solid var(--leaf)', background: 'var(--paper-2)' }}>
      <div className="row between wrap" style={{ alignItems: 'center', gap: '0.5rem' }}>
        <span className="chip green" style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <Icon name="sparkles" size={15} />
          <strong>{t('ai_extracted_title', { n: rows.length })}</strong>
        </span>
        <span className="tiny muted">{t('ai_extracted_hint')}</span>
      </div>
      {rows.length > 1 && (
        <div className="row wrap tiny muted" style={{ gap: 12, paddingBottom: 4 }}>
          {salesTotal > 0 && <span>{t('sales')}: <strong className="mono green">{rupees(salesTotal)}</strong></span>}
          {expTotal > 0 && <span>{t('expenses')}: <strong className="mono red">{rupees(expTotal)}</strong></span>}
          {udhTotal > 0 && <span>{t('udhaar')}: <strong className="mono yellow">{rupees(udhTotal)}</strong></span>}
        </div>
      )}
      <table className="ledger-table">
        <thead><tr><th>{t('date')}</th><th>{t('note')}</th><th className="amt">{t('amount')}</th><th /></tr></thead>
        <tbody>
          {rows.map((e, i) => (
            <tr key={i}>
              <td className="mono small">{fmtDate(e.date, lang)}</td>
              <td>
                <span className={`chip ${kindChip[e.kind]}`}>{t(`kind_${e.kind}` as 'kind_sale')}</span> {e.note}{e.party ? ` · ${e.party}` : ''}
                {e.source && <span className="tiny faint" style={{ marginLeft: 6 }}>({e.source})</span>}
              </td>
              <td className="amt mono">{rupees(e.amount_paise)}</td>
              <td>{!saved && <button className="btn ghost sm icon" aria-label={t('delete')} onClick={() => setRows(rows.filter((_, j) => j !== i))}><Icon name="x" size={16} /></button>}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {unclear && <p className="tiny muted">{t('unclear')}: {unclear}</p>}
      <div className="row between">
        {saved ? <span className="stamp green thump">{t('saved')}</span> : <button className="btn primary sm" onClick={save} disabled={busy}><Icon name="check" size={16} />{busy ? <Spinner /> : t('save_all')}</button>}
      </div>
    </div>
  )
}

export type Summary = { days: number; sales_paise: number; expense_paise: number; profit_paise: number; avg_daily_sale_paise: number; udhaar_outstanding_paise: number; daily: { date: string; sale: number; expense: number }[] | null; top_expenses: { note: string; amount_paise: number }[]; udhaar_parties: { party: string; balance_paise: number; last_date: string }[] }

export function SummaryStats({ s }: { s: Summary }) {
  const { t } = useApp()
  return (
    <div className="grid3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))' }}>
      <div className="stat"><div className="label">{t('sales')}</div><div className="value green">{rupees(s.sales_paise)}</div></div>
      <div className="stat"><div className="label">{t('expenses')}</div><div className="value red">{rupees(s.expense_paise)}</div></div>
      <div className="stat"><div className="label">{t('profit')}</div><div className={`value ${s.profit_paise >= 0 ? 'green' : 'red'}`}>{rupees(s.profit_paise)}</div></div>
      <div className="stat"><div className="label">{t('udhaar')}</div><div className="value">{rupees(s.udhaar_outstanding_paise)}</div></div>
    </div>
  )
}

export function BarChart({ daily }: { daily: { date: string; sale: number; expense: number }[] }) {
  const { lang } = useApp()
  const max = Math.max(1, ...daily.map((d) => Math.max(d.sale, d.expense)))
  return (
    <div>
      <div className="bar-chart" role="img" aria-label="Daily sales and expenses">
        {daily.map((d) => (
          <div className="pair" key={d.date} title={`${fmtDate(d.date, lang)} · ${rupees(d.sale)} / ${rupees(d.expense)}`}>
            <div className="bar" style={{ height: `${(d.sale / max) * 100}%` }} />
            <div className="bar exp" style={{ height: `${(d.expense / max) * 100}%` }} />
          </div>
        ))}
      </div>
      <div className="row between tiny faint mono" style={{ marginTop: 4 }}>
        <span>{fmtDate(daily[0]?.date, lang)}</span><span>{fmtDate(daily[daily.length - 1]?.date, lang)}</span>
      </div>
    </div>
  )
}

export type Grant = { id: number; source: string; source_url: string; title: string; summary: string; amount_text: string; deadline: string | null; deadline_raw: string; eligibility: string; tags: string[]; fetched_at: string; status: 'open' | 'closed' | 'rolling'; link_ok?: boolean | null }

export function GrantCard({ g, children }: { g: Grant; children?: ReactNode }) {
  const { t, lang } = useApp()
  const st = g.status === 'open' ? 'green' : g.status === 'closed' ? 'red' : 'yellow'
  return (
    <article className="sheet stack" style={{ gap: '0.55rem', opacity: g.status === 'closed' ? 0.72 : 1 }}>
      <div className="row between wrap" style={{ alignItems: 'flex-start' }}>
        <span className="caps faint">{g.source}</span>
        <span className={`stamp ${st}`}>{t(`status_${g.status}` as 'status_open')}</span>
      </div>
      <h3 style={{ fontSize: '1.08rem' }}>{g.title}</h3>
      {g.summary && <p className="small muted">{g.summary}</p>}
      <hr className="double" />
      <dl className="small" style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: '0.25rem 0.8rem', margin: 0 }}>
        {g.amount_text && <><dt className="muted">{t('amount')}</dt><dd className="mono" style={{ margin: 0 }}>{g.amount_text}</dd></>}
        <dt className="muted">{t('deadline')}</dt>
        <dd style={{ margin: 0 }}>{g.deadline ? <span className="mono">{fmtDate(g.deadline, lang)}</span> : <span className="faint">{t('no_deadline')}</span>}</dd>
        {g.eligibility && <><dt className="muted">✓</dt><dd style={{ margin: 0 }}>{g.eligibility}</dd></>}
      </dl>
      <div className="row between wrap">
        <span className="tiny faint">● {t('fetched', { t: ago(g.fetched_at, lang) })}</span>
        <a className="btn ghost sm" href={g.source_url} target="_blank" rel="noopener noreferrer">{t('view_source')} <Icon name="ext" size={14} /></a>
      </div>
      {children}
    </article>
  )
}

export type Criterion = { rule: string; status: 'pass' | 'fail' | 'unknown'; evidence: string; source_quote: string }
export type Fit = { grant_id: number; score: number; verdict: 'strong' | 'possible' | 'not_a_fit'; why: string; criteria: Criterion[]; next_steps: string[]; grant?: Grant }

export function FitCard({ f, onDraft, drafting }: { f: Fit; onDraft?: (id: number) => void; drafting?: boolean }) {
  const { t } = useApp()
  const color = f.verdict === 'strong' ? 'var(--leaf)' : f.verdict === 'possible' ? 'var(--turmeric)' : 'var(--stamp)'
  const [open, setOpen] = useState(f.verdict !== 'not_a_fit')
  return (
    <article className="sheet stack" style={{ gap: '0.6rem' }}>
      <div className="row" style={{ alignItems: 'flex-start' }}>
        <div className="score" style={{ color }}>{f.score}</div>
        <div className="grow">
          <span className="caps faint">{f.grant?.source}</span>
          <h3 style={{ fontSize: '1.05rem' }}>{f.grant?.title ?? `#${f.grant_id}`}</h3>
          <p className="small muted">{f.why}</p>
        </div>
        <span className={`stamp ${f.verdict === 'strong' ? 'green' : f.verdict === 'possible' ? 'yellow' : 'red'}`}>{t(f.verdict)}</span>
      </div>
      <button className="btn ghost sm" style={{ alignSelf: 'flex-start' }} onClick={() => setOpen(!open)} aria-expanded={open}>{open ? '−' : '+'} {f.criteria.length} rules</button>
      {open && (
        <div>
          {f.criteria.map((c, i) => (
            <div className="crit" key={i}>
              <span className={`ico ${c.status}`}>{c.status === 'pass' ? '✓' : c.status === 'fail' ? '✗' : '?'}</span>
              <div>
                <div>{c.rule}</div>
                <div className="tiny muted">{c.evidence}</div>
                {c.source_quote && <div className="quote">“{c.source_quote}”</div>}
              </div>
            </div>
          ))}
          {f.next_steps.length > 0 && <ul className="small" style={{ margin: '0.5rem 0 0', paddingLeft: '1.2rem' }}>{f.next_steps.map((s, i) => <li key={i}>{s}</li>)}</ul>}
        </div>
      )}
      <div className="row between wrap">
        {f.grant && <a className="btn ghost sm" href={f.grant.source_url} target="_blank" rel="noopener noreferrer">{t('view_source')} <Icon name="ext" size={14} /></a>}
        {onDraft && f.verdict !== 'not_a_fit' && <button className="btn primary sm" disabled={drafting} onClick={() => onDraft(f.grant_id)}>{drafting ? <Spinner /> : <Icon name="file" size={16} />}{t('draft_app')}</button>}
      </div>
    </article>
  )
}

export type Blueprint = { id: number; created_at: string; data: any }

export function BlueprintView({ b, compact }: { b: Blueprint; compact?: boolean }) {
  const { t } = useApp()
  const d = b.data
  const fin = d.financials || {}
  return (
    <article className="sheet stack">
      <div className="row between wrap">
        <span className="caps faint">{t('blueprint')} · {b.created_at}</span>
        <span className="stamp flat">{d.stage}</span>
      </div>
      <h3 className="display" style={{ fontSize: '1.5rem' }}>{d.business_name}</h3>
      <p style={{ fontSize: '1.05rem' }}>{d.one_liner}</p>
      <div className="grid3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))' }}>
        <div className="stat"><div className="label">{t('sales')} 30d</div><div className="value green" style={{ fontSize: '1.2rem' }}>{rupeesInr(fin.sales_last_30d_inr ?? 0)}</div></div>
        <div className="stat"><div className="label">{t('profit')} 30d</div><div className="value" style={{ fontSize: '1.2rem' }}>{rupeesInr(fin.profit_last_30d_inr ?? 0)}</div></div>
        <div className="stat"><div className="label">{t('use_of_funds')}</div><div className="value red" style={{ fontSize: '1.2rem' }}>{rupeesInr(d.funding_need?.amount_inr ?? 0)}</div></div>
      </div>
      <p className="tiny faint">✓ {t('financials')}</p>
      {!compact && (
        <>
          <Section title="Problem">{d.problem}</Section>
          <Section title="Offering">{d.offering}</Section>
          <Section title="Customers">{d.customers}</Section>
          <Section title="Revenue model">{d.revenue_model}</Section>
          {d.funding_need?.use_of_funds?.length > 0 && (
            <div>
              <div className="caps muted">{t('use_of_funds')} — {d.funding_need.purpose}</div>
              <table className="ledger-table"><tbody>
                {d.funding_need.use_of_funds.map((u: any, i: number) => <tr key={i}><td>{u.item}</td><td className="amt mono">{rupeesInr(u.amount_inr)}</td></tr>)}
              </tbody></table>
            </div>
          )}
          {d.milestones?.length > 0 && (
            <div>
              <div className="caps muted">{t('milestones')}</div>
              <table className="ledger-table"><tbody>
                {d.milestones.map((m: any, i: number) => <tr key={i}><td className="mono" style={{ width: 70 }}>{t('month')} {m.month}</td><td>{m.goal}</td></tr>)}
              </tbody></table>
            </div>
          )}
          {d.risks?.length > 0 && (
            <div>
              <div className="caps muted">{t('risks')}</div>
              <ul className="small" style={{ margin: 0, paddingLeft: '1.2rem' }}>{d.risks.map((r: any, i: number) => <li key={i}><strong>{r.risk}</strong> — {r.mitigation}</li>)}</ul>
            </div>
          )}
        </>
      )}
      {d.missing_info?.length > 0 && (
        <div className="row wrap" style={{ gap: 6 }}>
          <span className="caps muted">{t('missing_info')}:</span>
          {d.missing_info.map((m: string, i: number) => <span key={i} className="chip yellow">{m}</span>)}
        </div>
      )}
    </article>
  )
}
function Section({ title, children }: { title: string; children: ReactNode }) {
  if (!children) return null
  return <div><div className="caps muted">{title}</div><p>{children}</p></div>
}

export type Pattern = { id: number; slug: string; kind: string; title: string; description: string; red_flags: string[]; actions: string[]; source_url: string; pattern_slug: string; published_at: string | null }

export function PatternCard({ p }: { p: Pattern }) {
  const { t } = useApp()
  return (
    <article className="sheet stack" style={{ gap: '0.5rem' }}>
      <div className="row between"><h3>{p.title}</h3><SpeakButton text={[p.title, p.description, ...p.actions].join('. ')} /></div>
      <p className="small muted">{p.description}</p>
      <ul className="flag-list small">{p.red_flags.map((f, i) => <li key={i}>{f}</li>)}</ul>
      <ol className="do-list small">{p.actions.map((f, i) => <li key={i}>{f}</li>)}</ol>
      {p.source_url && <a className="tiny faint" href={p.source_url} target="_blank" rel="noopener noreferrer">{t('source')} ↗</a>}
    </article>
  )
}

export function DraftSummary({ d }: { d: any }) {
  const { t } = useApp()
  return (
    <div className="sheet stack" style={{ gap: '0.5rem' }}>
      <div className="row between wrap"><span className="stamp yellow">DRAFT</span><span className="chip yellow">{t('gaps', { n: d.needed_count ?? 0 })}</span></div>
      <h3>{d.title}</h3>
      <p className="small muted">{d.grant?.title}</p>
      <Link className="btn sm primary" style={{ alignSelf: 'flex-start' }} to={`/app/apply/${d.id}`}><Icon name="file" size={16} />{t('edit')}</Link>
    </div>
  )
}

export { Md }
