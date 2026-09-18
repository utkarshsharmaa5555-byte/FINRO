import { Fragment, useState } from 'react'
import type { Key } from '../i18n'
import { ago, api, Link, LANGS_SPEECH, rupees, useApp, useData } from '../lib'
import { BarChart, ErrorLine, Icon, PageHead, SpeakButton, Spinner, type Summary } from '../ui'

export type ItemStats = {
  id: number; name: string; kind: 'product' | 'dish' | 'ingredient'; category: string; unit: string
  qty: number; reorder_level: number; cost_paise: number; price_paise: number
  rate_7d: number; rate_30d: number; trend_pct: number; days_left: number | null
  revenue_30d_paise: number; margin_pct: number; status: string; reorder_qty: number; typical_buy: number; last_bought: string | null
}
type AdviceLine = { item: string; why: string; qty: number; unit: string; urgency?: string; item_id: number }
export type Advice = { headline: string; buy_more: AdviceLine[]; buy_less: AdviceLine[]; keep_steady: string[]; insights: string[]; money_tip: string; created_at: string }

const statusColor: Record<string, string> = { out: 'red', critical: 'red', low: 'yellow', ok: 'green', overstock: 'yellow', dead: 'red', fresh: '' }

export function StatusChip({ s }: { s: string }) {
  const { t } = useApp()
  return <span className={`chip ${statusColor[s] ?? ''}`}>{t(`st_${s}` as Key)}</span>
}

export function Trend({ pct }: { pct: number }) {
  if (!pct) return <span className="mono tiny faint">±0%</span>
  const up = pct > 0
  return <span className="mono tiny" style={{ color: up ? 'var(--leaf)' : 'var(--stamp)', fontWeight: 700 }}>{up ? '▲' : '▼'} {Math.abs(pct)}%</span>
}

const fmtQty = (n: number) => (Number.isInteger(n) ? String(n) : n.toFixed(1))

// ---------- AI buying advice ----------

export function AdviceCard({ advice, onRefresh, busy }: { advice: Advice | null; onRefresh?: () => void; busy?: boolean }) {
  const { t, lang } = useApp()
  if (!advice) {
    return (
      <div className="sheet ruled stack" style={{ lineHeight: '32px' }}>
        <div className="row between wrap"><h3>✦ {t('ai_advice')}</h3><span className="stamp flat">AI</span></div>
        <p className="small muted" style={{ lineHeight: 1.5 }}>{t('advice_hint')}</p>
        {onRefresh && <button className="btn primary" style={{ alignSelf: 'flex-start' }} onClick={onRefresh} disabled={busy}>{busy ? <Spinner /> : <Icon name="target" size={18} />}{t('analyze')}</button>}
      </div>
    )
  }
  const speakText = [advice.headline, ...advice.buy_more.map((b) => `${b.item}: ${b.why}`), ...advice.buy_less.map((b) => `${b.item}: ${b.why}`), advice.money_tip].join('. ')
  return (
    <article className="sheet stack">
      <div className="row between wrap">
        <div>
          <div className="caps muted">✦ {t('ai_advice')}</div>
          <h3 style={{ fontSize: '1.15rem', marginTop: 2 }}>{advice.headline}</h3>
        </div>
        <div className="row" style={{ gap: 4 }}>
          <SpeakButton text={speakText} />
          {onRefresh && <button className="btn sm" onClick={onRefresh} disabled={busy}>{busy ? <Spinner /> : <Icon name="refresh" size={15} />}{t('reanalyze')}</button>}
        </div>
      </div>
      <div className="grid2" style={{ gap: '0.8rem' }}>
        <div>
          <div className="caps" style={{ color: 'var(--leaf)' }}>▲ {t('buy_more')}</div>
          <ul className="advice-list">
            {advice.buy_more.map((b) => (
              <li key={b.item_id}>
                <div className="row between" style={{ gap: 6, alignItems: 'baseline' }}>
                  <strong>{b.item}</strong>
                  {b.qty > 0 ? <span className="chip green mono">{t('order_qty')} {fmtQty(b.qty)} {b.unit}</span> : <span className="chip green">{t('demand_signal')}</span>}
                </div>
                <div className="small muted">{b.why}</div>
                {b.urgency === 'today' && <span className="stamp red flat" style={{ fontSize: '0.62rem', marginTop: 4 }}>{t('urgency_today')}</span>}
              </li>
            ))}
          </ul>
        </div>
        <div>
          <div className="caps" style={{ color: 'var(--stamp)' }}>▼ {t('buy_less')}</div>
          <ul className="advice-list">
            {advice.buy_less.map((b) => (
              <li key={b.item_id}>
                <div className="row between" style={{ gap: 6, alignItems: 'baseline' }}>
                  <strong>{b.item}</strong>
                  {b.qty > 0 && <span className="chip yellow mono">{t('reduce_to')} {fmtQty(b.qty)} {b.unit}</span>}
                </div>
                <div className="small muted">{b.why}</div>
              </li>
            ))}
          </ul>
          {advice.keep_steady.length > 0 && (
            <div style={{ marginTop: 8 }}>
              <div className="caps muted">= {t('keep_steady')}</div>
              <div className="row wrap" style={{ gap: 4, marginTop: 4 }}>{advice.keep_steady.map((k) => <span key={k} className="chip">{k}</span>)}</div>
            </div>
          )}
        </div>
      </div>
      {advice.insights.length > 0 && (
        <div className="sheet plain ruled" style={{ lineHeight: '32px', boxShadow: 'none' }}>
          <div className="caps muted">{t('insights')}</div>
          <ul style={{ margin: 0, paddingLeft: '1.1rem' }}>{advice.insights.map((x, i) => <li key={i}>{x}</li>)}</ul>
          {advice.money_tip && <p><strong>₹</strong> {advice.money_tip}</p>}
        </div>
      )}
      <p className="tiny faint">✦ {t('analysed')} {ago(advice.created_at, lang)}</p>
    </article>
  )
}

export function useAdvice(initial: Advice | null | undefined, after?: () => void) {
  const { toast } = useApp()
  const [advice, setAdvice] = useState<Advice | null>(initial ?? null)
  const [busy, setBusy] = useState(false)
  const run = async () => {
    setBusy(true)
    try { setAdvice(await api<Advice>('/stock/advice', { method: 'POST' })); after?.() } catch (e) { toast((e as Error).message) } finally { setBusy(false) }
  }
  return { advice: advice ?? initial ?? null, busy, run }
}

// ---------- stock panel (Khata page) ----------

function MoveForm({ item, onDone }: { item: ItemStats; onDone: () => void }) {
  const { t, toast } = useApp()
  const [kind, setKind] = useState(item.status === 'out' || item.status === 'critical' || item.status === 'low' ? 'bought' : 'adjust')
  const [qty, setQty] = useState(item.status === 'out' || item.status === 'critical' || item.status === 'low' ? String(item.reorder_qty || '') : String(item.qty))
  const [busy, setBusy] = useState(false)
  const save = async () => {
    setBusy(true)
    try { await api(`/stock/${item.id}`, { method: 'PATCH', body: { kind, qty: Number(qty) } }); toast(t('saved')); onDone() } catch (e) { toast((e as Error).message) } finally { setBusy(false) }
  }
  return (
    <form className="row wrap" style={{ gap: 6, padding: '0.4rem 0' }} onSubmit={(e) => { e.preventDefault(); save() }}>
      <div className="seg" role="group">
        {(['bought', 'wasted', 'adjust'] as const).map((k) => (
          <button type="button" key={k} aria-pressed={kind === k} onClick={() => setKind(k)}>{t(k === 'bought' ? 'restock' : k === 'wasted' ? 'wasted' : 'set_count')}</button>
        ))}
      </div>
      <input className="input mono" style={{ width: 96, minHeight: 36 }} inputMode="decimal" value={qty} onChange={(e) => setQty(e.target.value.replace(/[^\d.]/g, ''))} aria-label={t('in_stock')} />
      <span className="small muted">{item.unit}</span>
      <button className="btn primary sm" disabled={busy || qty === ''}>{busy ? <Spinner /> : t('update')}</button>
    </form>
  )
}

function AddItemForm({ onDone }: { onDone: () => void }) {
  const { t, toast } = useApp()
  const [f, setF] = useState({ name: '', kind: 'product', unit: 'pcs', qty: '', reorder_level: '', price: '', cost: '' })
  const [err, setErr] = useState('')
  const save = async () => {
    setErr('')
    try {
      await api('/stock', { body: { name: f.name, kind: f.kind, unit: f.unit, qty: Number(f.qty || 0), reorder_level: Number(f.reorder_level || 0), price_paise: Math.round(Number(f.price || 0) * 100), cost_paise: Math.round(Number(f.cost || 0) * 100) } })
      toast(t('saved'))
      onDone()
    } catch (e) { setErr((e as Error).message) }
  }
  const num = (k: keyof typeof f, label: Key) => (
    <div className="field"><label htmlFor={`ai-${k}`}>{t(label)}</label><input id={`ai-${k}`} className="input mono" inputMode="decimal" value={f[k]} onChange={(e) => setF({ ...f, [k]: e.target.value.replace(/[^\d.]/g, '') })} /></div>
  )
  return (
    <form className="stack" onSubmit={(e) => { e.preventDefault(); save() }}>
      <div className="grid3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: '0.6rem' }}>
        <div className="field" style={{ gridColumn: 'span 2' }}><label htmlFor="ai-name">{t('item_name')}</label><input id="ai-name" className="input" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} maxLength={80} /></div>
        <div className="field"><label htmlFor="ai-kind">{t('kind_product')} / {t('kind_dish')}</label>
          <select id="ai-kind" className="select" value={f.kind} onChange={(e) => setF({ ...f, kind: e.target.value })}>
            {(['product', 'dish', 'ingredient'] as const).map((k) => <option key={k} value={k}>{t(`kind_${k}` as Key)}</option>)}
          </select>
        </div>
        <div className="field"><label htmlFor="ai-unit">{t('unit')}</label>
          <select id="ai-unit" className="select" value={f.unit} onChange={(e) => setF({ ...f, unit: e.target.value })}>
            {['pcs', 'pack', 'kg', 'litre', 'plate', 'bottle', 'packet', 'bar', 'cup', 'cylinder'].map((u) => <option key={u}>{u}</option>)}
          </select>
        </div>
        {f.kind !== 'dish' && num('qty', 'in_stock')}
        {f.kind !== 'dish' && num('reorder_level', 'reorder_level')}
        {f.kind !== 'ingredient' && num('price', 'price')}
        {num('cost', 'cost')}
      </div>
      <ErrorLine msg={err} />
      <button className="btn primary sm" style={{ alignSelf: 'flex-start' }} disabled={!f.name.trim()}><Icon name="plus" size={16} />{t('add_item')}</button>
    </form>
  )
}

export function StockPanel() {
  const { t, lang } = useApp()
  const d = useData<{ items: ItemStats[]; advice: Advice | null }>('/stock', [lang])
  const adv = useAdvice(d.data?.advice, () => d.reload())
  const [open, setOpen] = useState<number | null>(null)
  const [adding, setAdding] = useState(false)
  if (d.loading && !d.data) return <div className="skeleton" style={{ height: 240, marginTop: '1rem' }} />
  const items = d.data?.items ?? []
  const groups: ItemStats['kind'][] = ['product', 'dish', 'ingredient']

  return (
    <section className="stack lg" style={{ marginTop: '1.2rem' }}>
      <AdviceCard advice={adv.advice} onRefresh={adv.run} busy={adv.busy} />
      <div className="sheet">
        <div className="row between wrap">
          <h3>{t('stock_title')}</h3>
          <button className="btn sm" onClick={() => setAdding(!adding)} aria-expanded={adding}><Icon name={adding ? 'x' : 'plus'} size={15} />{t('add_item')}</button>
        </div>
        {adding && <div style={{ margin: '0.8rem 0' }}><AddItemForm onDone={() => { setAdding(false); d.reload() }} /></div>}
        <div style={{ overflowX: 'auto' }}>
          <table className="ledger-table">
            <thead><tr><th>{t('item_name')}</th><th className="amt">{t('in_stock')}</th><th className="amt">{t('run_rate')}</th><th className="amt">{t('trend')}</th><th /></tr></thead>
            {groups.map((g) => {
              const rows = items.filter((i) => i.kind === g)
              if (!rows.length) return null
              return (
                <tbody key={g}>
                  <tr><td colSpan={5} className="caps faint" style={{ paddingTop: '0.9rem' }}>{t(`kind_${g}` as Key)}</td></tr>
                  {rows.map((it) => (
                    <Fragment key={it.id}>
                      <tr>
                        <td>
                          <strong>{it.name}</strong>
                          <div className="row wrap" style={{ gap: 4, marginTop: 2 }}>
                            <StatusChip s={it.status} />
                            {it.days_left != null && it.kind !== 'dish' && <span className="tiny muted">{t('days_left', { n: fmtQty(it.days_left) })}</span>}
                          </div>
                        </td>
                        <td className="amt mono">{it.kind === 'dish' ? '—' : `${fmtQty(it.qty)} ${it.unit}`}</td>
                        <td className="amt mono">{fmtQty(it.rate_7d)}<span className="tiny faint"> {it.unit}{t('per_day')}</span></td>
                        <td className="amt"><Trend pct={it.trend_pct} /></td>
                        <td>{it.kind !== 'dish' && <button className="btn ghost sm icon" aria-label={t('update')} aria-expanded={open === it.id} onClick={() => setOpen(open === it.id ? null : it.id)}><Icon name={open === it.id ? 'x' : 'refresh'} size={15} /></button>}</td>
                      </tr>
                      {open === it.id && <tr><td colSpan={5}><MoveForm item={it} onDone={() => { setOpen(null); d.reload() }} /></td></tr>}
                    </Fragment>
                  ))}
                </tbody>
              )
            })}
          </table>
        </div>
      </div>
    </section>
  )
}

// ---------- Business overview ----------

type Overview = {
  today_sales_paise: number; yesterday_sales_paise: number; avg7_sales_paise: number; avg30_sales_paise: number; stock_value_paise: number
  records: Summary; weekdays: { dow: number; avg_paise: number }[]
  items: ItemStats[]; alerts: ItemStats[]; top_sellers: ItemStats[]; slow_movers: ItemStats[]; advice: Advice | null
}

function weekdayName(dow: number, lang: string, style: 'short' | 'long' = 'short') {
  return new Date(2024, 0, dow).toLocaleDateString(LANGS_SPEECH(lang), { weekday: style }) // 2024-01-01 is a Monday (ISO dow 1)
}

export function OverviewPage() {
  const { t, lang } = useApp()
  const d = useData<Overview>('/overview', [lang])
  const adv = useAdvice(d.data?.advice)
  if (d.loading && !d.data) return <div className="grid2">{[1, 2, 3, 4].map((i) => <div key={i} className="skeleton" style={{ height: 180 }} />)}</div>
  if (!d.data) return <ErrorLine msg={d.error} />
  const o = d.data
  const maxRate = Math.max(1, ...o.top_sellers.map((s) => s.revenue_30d_paise))
  const maxDow = Math.max(1, ...o.weekdays.map((w) => w.avg_paise))
  const best = [...o.weekdays].sort((a, b) => b.avg_paise - a.avg_paise)[0]
  const vs7 = o.avg30_sales_paise ? Math.round((o.avg7_sales_paise / o.avg30_sales_paise - 1) * 100) : 0

  return (
    <>
      <PageHead kicker={t('ov_kicker')} title={t('ov_title')}>
        <Link className="btn sm" to="/app/khata"><Icon name="book" size={15} />{t('nav_khata')}</Link>
        <Link className="btn sm" to="/app/blueprint"><Icon name="target" size={15} />{t('nav_blueprint')}</Link>
      </PageHead>

      <div className="grid3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))' }}>
        <div className="stat">
          <div className="label">{o.today_sales_paise ? t('today') : t('yesterday')}</div>
          <div className="value green">{rupees(o.today_sales_paise || o.yesterday_sales_paise)}</div>
          {!o.today_sales_paise && <div className="tiny faint">{t('no_sales_today')}</div>}
        </div>
        <div className="stat"><div className="label">{t('avg_7')}</div><div className="value">{rupees(o.avg7_sales_paise)}</div><Trend pct={vs7} /></div>
        <div className="stat"><div className="label">{t('profit')} · 30d</div><div className={`value ${o.records.profit_paise >= 0 ? 'green' : 'red'}`}>{rupees(o.records.profit_paise)}</div></div>
        <div className="stat"><div className="label">{t('udhaar')}</div><div className="value">{rupees(o.records.udhaar_outstanding_paise)}</div></div>
        <div className="stat"><div className="label">{t('stock_value')}</div><div className="value">{rupees(o.stock_value_paise)}</div></div>
      </div>

      <div className="grid2" style={{ marginTop: '1.2rem', alignItems: 'start' }}>
        <section className="sheet">
          <div className="row between"><h3>⚠ {t('running_out')}</h3><span className="chip red">{o.alerts.length}</span></div>
          {o.alerts.length === 0 && <p className="muted small" style={{ marginTop: 8 }}>✓ {t('no_alerts')}</p>}
          <ul className="alert-list">
            {o.alerts.map((a) => {
              const days = a.days_left ?? 0
              return (
                <li key={a.id}>
                  <div className="row between" style={{ gap: 6 }}>
                    <strong>{a.name}</strong>
                    <StatusChip s={a.status} />
                  </div>
                  <div className="days-bar" aria-hidden="true"><span style={{ width: `${Math.min(100, (days / 7) * 100)}%`, background: a.status === 'low' ? 'var(--turmeric)' : 'var(--stamp)' }} /></div>
                  <div className="row between tiny muted">
                    <span className="mono">{fmtQty(a.qty)} {a.unit} · {fmtQty(a.rate_7d)} {a.unit}{t('per_day')}</span>
                    <span>{a.status === 'out' ? t('st_out') : t('days_left', { n: fmtQty(days) })}</span>
                  </div>
                  {a.reorder_qty > 0 && <span className="chip green mono" style={{ marginTop: 4 }}>{t('order_qty')} {fmtQty(a.reorder_qty)} {a.unit}</span>}
                </li>
              )
            })}
          </ul>
        </section>

        <section className="sheet">
          <h3>{t('top_sellers')}</h3>
          <table className="ledger-table" style={{ marginTop: 6 }}>
            <tbody>
              {o.top_sellers.map((s) => (
                <tr key={s.id}>
                  <td>
                    <strong>{s.name}</strong>
                    <div className="mini-bar" aria-hidden="true"><span style={{ width: `${(s.revenue_30d_paise / maxRate) * 100}%` }} /></div>
                    <div className="tiny faint mono">{rupees(s.revenue_30d_paise)} · 30d{s.margin_pct ? ` · ${s.margin_pct}% margin` : ''}</div>
                  </td>
                  <td className="amt mono">{fmtQty(s.rate_7d)}<div className="tiny faint">{s.unit}{t('per_day')}</div></td>
                  <td className="amt"><Trend pct={s.trend_pct} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      </div>

      <div className="grid2" style={{ marginTop: '1.2rem', alignItems: 'start' }}>
        <section className="sheet">
          <div className="row between wrap">
            <h3>{t('weekday_sales')}</h3>
            {best && <span className="stamp green">{t('best_day')}: {weekdayName(best.dow, lang, 'long')}</span>}
          </div>
          <div className="dow-chart" role="img" aria-label={t('weekday_sales')}>
            {o.weekdays.map((w) => (
              <div key={w.dow} className="dow">
                <span className="tiny mono faint">{Math.round(w.avg_paise / 100000)}k</span>
                <div className="dow-bar" style={{ height: `${(w.avg_paise / maxDow) * 100}%`, background: w.dow === best?.dow ? 'var(--leaf)' : 'var(--ink-3)' }} />
                <span className="tiny">{weekdayName(w.dow, lang)}</span>
              </div>
            ))}
          </div>
        </section>
        <section className="sheet">
          <h3>{t('last_n_days', { n: 30 })}</h3>
          {o.records.daily && <BarChart daily={o.records.daily} />}
          {o.slow_movers.length > 0 && (
            <div style={{ marginTop: 10 }}>
              <div className="caps muted">▼ {t('slow_movers')}</div>
              <div className="row wrap" style={{ gap: 4, marginTop: 4 }}>
                {o.slow_movers.map((s) => <span key={s.id} className="chip yellow">{s.name} · {s.status === 'overstock' ? t('st_overstock') : `${s.trend_pct}%`}</span>)}
              </div>
            </div>
          )}
        </section>
      </div>

      <div style={{ marginTop: '1.2rem' }}>
        <AdviceCard advice={adv.advice} onRefresh={adv.run} busy={adv.busy} />
      </div>
    </>
  )
}
