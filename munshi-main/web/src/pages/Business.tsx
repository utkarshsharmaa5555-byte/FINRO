import { useState } from 'react'
import { api, fileToDataURL, fmtDate, rupees, useApp, useData } from '../lib'
import { StockPanel } from './Stock'
import { BarChart, EntriesProposal, ErrorLine, Icon, MicButton, PageHead, Spinner, SummaryStats, type Entry, type Summary } from '../ui'

export function Khata() {
  const { t, lang, toast } = useApp()
  const [days, setDays] = useState(30)
  const rec = useData<{ summary: Summary; entries: Entry[] }>(`/records?days=${days}`)
  const [mode, setMode] = useState<'ai' | 'manual'>('ai')
  const [text, setText] = useState('')
  const [proposal, setProposal] = useState<{ entries: Entry[]; unclear: string } | null>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const todayISO = () => new Date().toISOString().slice(0, 10)
  const [manualDate, setManualDate] = useState(todayISO)
  const [manualKind, setManualKind] = useState<'sale' | 'expense' | 'udhaar_given' | 'udhaar_received'>('sale')
  const [manualAmount, setManualAmount] = useState('')
  const [manualNote, setManualNote] = useState('')
  const [manualParty, setManualParty] = useState('')
  const [manualBusy, setManualBusy] = useState(false)
  const [manualErr, setManualErr] = useState('')

  const parse = async (body: { text?: string; image?: string; source: string }) => {
    setBusy(true); setErr(''); setProposal(null)
    try { setProposal(await api('/records/parse', { body })) } catch (e) { setErr((e as Error).message) } finally { setBusy(false) }
  }
  const del = async (id: number) => {
    await api(`/records/${id}`, { method: 'DELETE' }).catch(() => toast(t('err_generic')))
    rec.reload()
  }

  const handleManualSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const amt = Math.round(parseFloat(manualAmount) * 100)
    if (!amt || amt <= 0) {
      setManualErr(t('err_amount'))
      return
    }
    if (!manualNote.trim()) {
      setManualErr(t('unclear'))
      return
    }
    setManualBusy(true)
    setManualErr('')
    try {
      await api('/records', {
        body: {
          entries: [{
            date: manualDate || todayISO(),
            kind: manualKind,
            amount_paise: amt,
            note: manualNote.trim(),
            party: manualParty.trim(),
            source: 'manual',
          }]
        }
      })
      toast(t('saved'))
      setManualAmount('')
      setManualNote('')
      setManualParty('')
      rec.reload()
    } catch (e) {
      setManualErr((e as Error).message)
    } finally {
      setManualBusy(false)
    }
  }

  const s = rec.data?.summary

  return (
    <>
      <PageHead kicker={t('khata_kicker')} title={t('khata_title')}>
        <div className="seg" role="group">
          {[7, 30, 90].map((d) => <button key={d} aria-pressed={days === d} onClick={() => setDays(d)}>{d}d</button>)}
        </div>
      </PageHead>

      <div className="sheet stack">
        <div className="row between wrap" style={{ alignItems: 'center', gap: 8 }}>
          <h3 style={{ margin: 0, display: 'flex', alignItems: 'center', gap: 6 }}>
            {mode === 'ai' ? (
              <>
                <Icon name="sparkles" size={18} />
                {t('ai_smart_entry')}
                <span className="chip green" style={{ fontSize: '0.72rem', padding: '2px 8px' }}>{t('ai_powered')}</span>
              </>
            ) : (
              <>
                <Icon name="pencil" size={18} />
                {t('add_manual_title')}
              </>
            )}
          </h3>
          <div className="seg" role="group">
            <button type="button" aria-pressed={mode === 'ai'} onClick={() => setMode('ai')}>
              <Icon name="sparkles" size={14} style={{ verticalAlign: 'middle', marginRight: 4 }} />
              {t('ai_smart_entry')}
            </button>
            <button type="button" aria-pressed={mode === 'manual'} onClick={() => setMode('manual')}>
              <Icon name="pencil" size={14} style={{ verticalAlign: 'middle', marginRight: 4 }} />
              {t('manual_entry')}
            </button>
          </div>
        </div>

        {mode === 'ai' ? (
          <form className="stack" onSubmit={(e) => { e.preventDefault(); parse({ text, source: 'text' }) }}>
            <p className="small muted" style={{ margin: 0 }}>
              {t('ai_smart_entry_desc')}
            </p>

            <div className="row wrap" style={{ gap: 6, alignItems: 'center' }}>
              <span className="tiny muted" style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                <Icon name="target" size={12} /> {t('insights')}:
              </span>
              {[t('ai_sample_1'), t('ai_sample_2'), t('ai_sample_3')].map((sample, idx) => (
                <button
                  key={idx}
                  type="button"
                  className="chip ghost sm"
                  style={{ cursor: 'pointer', textAlign: 'left' }}
                  onClick={() => {
                    setText(sample)
                    parse({ text: sample, source: 'text' })
                  }}
                  title="Click to test with AI"
                >
                  💡 {sample}
                </button>
              ))}
            </div>

            <div className="row" style={{ alignItems: 'stretch' }}>
              <label className="sr" htmlFor="khatain">{t('add_hint')}</label>
              <input id="khatain" className="input grow" value={text} onChange={(e) => setText(e.target.value)} placeholder={t('add_hint')} />
              <MicButton onText={(s) => { setText(s); parse({ text: s, source: 'voice' }) }} />
              <label className="btn icon" title={t('photo_bill')} aria-label={t('photo_bill')}>
                <Icon name="camera" />
                <input type="file" accept="image/*" capture="environment" hidden onChange={async (e) => { const f = e.target.files?.[0]; e.target.value = ''; if (f) parse({ image: await fileToDataURL(f, 1800), source: 'photo' }) }} />
              </label>
              <button className="btn primary" disabled={busy || !text.trim()}>
                {busy ? (
                  <>
                    <Spinner /> {t('ai_parsing')}
                  </>
                ) : (
                  <>
                    <Icon name="sparkles" size={16} /> {t('ai_parse_btn')}
                  </>
                )}
              </button>
            </div>
            <ErrorLine msg={err} />
            {proposal && <EntriesProposal key={JSON.stringify(proposal)} entries={proposal.entries} unclear={proposal.unclear} onSaved={() => { setText(''); rec.reload() }} />}
          </form>
        ) : (
          <form className="stack" onSubmit={handleManualSubmit}>
            <div className="grid2" style={{ alignItems: 'start' }}>
              <label className="stack sm">
                <span className="small muted">{t('date')}</span>
                <input className="input" type="date" value={manualDate} onChange={(e) => setManualDate(e.target.value)} required />
              </label>
              <label className="stack sm">
                <span className="small muted">{t('amount')} (₹)</span>
                <input className="input" type="number" step="0.01" min="0.01" value={manualAmount} onChange={(e) => setManualAmount(e.target.value)} placeholder="0.00" required />
              </label>
            </div>
            <div className="grid2" style={{ alignItems: 'start' }}>
              <label className="stack sm">
                <span className="small muted">{t('kind_sale')} / {t('kind_expense')}</span>
                <select className="input select" value={manualKind} onChange={(e) => setManualKind(e.target.value as any)}>
                  <option value="sale">{t('kind_sale')}</option>
                  <option value="expense">{t('kind_expense')}</option>
                  <option value="udhaar_given">{t('kind_udhaar_given')}</option>
                  <option value="udhaar_received">{t('kind_udhaar_received')}</option>
                </select>
              </label>
              <label className="stack sm">
                <span className="small muted">{t('note')}</span>
                <input className="input" value={manualNote} onChange={(e) => setManualNote(e.target.value)} placeholder="e.g. Samosa, Chai, Sugar 5kg" required />
              </label>
            </div>
            <div className="grid2" style={{ alignItems: 'start' }}>
              <label className="stack sm">
                <span className="small muted">{t('party_name')}</span>
                <input className="input" value={manualParty} onChange={(e) => setManualParty(e.target.value)} placeholder="Optional (e.g. Ramesh, Supplier)" />
              </label>
              <div style={{ alignSelf: 'end', display: 'flex', gap: 8 }}>
                <button className="btn primary" disabled={manualBusy || !manualAmount || !manualNote}>
                  {manualBusy ? <Spinner /> : <><Icon name="check" size={16} />{t('save')}</>}
                </button>
              </div>
            </div>
            <ErrorLine msg={manualErr} />
          </form>
        )}
      </div>

      <StockPanel key={rec.data?.entries.length ?? 0} />

      {rec.loading && !s && <div className="skeleton" style={{ height: 200, marginTop: '1rem' }} />}
      {s && (
        <div className="stack lg" style={{ marginTop: '1.2rem' }}>
          <SummaryStats s={s} />
          <div className="sheet">
            <div className="row between"><h3>{t('last_n_days', { n: days })}</h3><span className="tiny muted"><span style={{ color: 'var(--leaf)' }}>■</span> {t('sales')} <span style={{ color: 'var(--stamp)' }}>■</span> {t('expenses')}</span></div>
            {s.daily && <BarChart daily={s.daily} />}
            <p className="small muted" style={{ marginTop: 8 }}>{t('avg_day')}: <strong className="mono">{rupees(s.avg_daily_sale_paise)}</strong></p>
          </div>
          <div className="grid2">
            <div className="sheet">
              <h3>{t('top_expenses')}</h3>
              <table className="ledger-table"><tbody>
                {s.top_expenses.map((e) => <tr key={e.note}><td>{e.note}</td><td className="amt mono">{rupees(e.amount_paise)}</td></tr>)}
              </tbody></table>
            </div>
            <div className="sheet">
              <h3>{t('who_owes')}</h3>
              <table className="ledger-table"><tbody>
                {s.udhaar_parties.length === 0 && <tr><td className="muted">—</td></tr>}
                {s.udhaar_parties.map((p) => <tr key={p.party}><td>{p.party}<div className="tiny faint">{fmtDate(p.last_date, lang)}</div></td><td className={`amt mono`} style={{ color: p.balance_paise > 0 ? 'var(--stamp)' : 'var(--leaf)' }}>{rupees(Math.abs(p.balance_paise))}</td></tr>)}
              </tbody></table>
            </div>
          </div>
          <div className="sheet">
            <h3>{t('entries')}</h3>
            <div style={{ overflowX: 'auto' }}>
              <table className="ledger-table">
                <thead><tr><th>{t('date')}</th><th>{t('note')}</th><th className="amt">{t('amount')}</th><th /></tr></thead>
                <tbody>
                  {rec.data!.entries.map((e) => (
                    <tr key={e.id}>
                      <td className="mono small" style={{ whiteSpace: 'nowrap' }}>{fmtDate(e.date, lang)}</td>
                      <td>
                        <span className={`chip ${e.kind === 'expense' ? 'red' : e.kind === 'udhaar_given' ? 'yellow' : 'green'}`}>{t(`kind_${e.kind}` as 'kind_sale')}</span>{' '}
                        {e.note}{e.party ? ` · ${e.party}` : ''} {e.source !== 'text' && <span className="tiny faint">({e.source})</span>}
                      </td>
                      <td className="amt mono" style={{ color: e.kind === 'expense' ? 'var(--stamp)' : undefined }}>{e.kind === 'expense' ? '−' : ''}{rupees(e.amount_paise)}</td>
                      <td><button className="btn ghost sm icon" aria-label={t('delete')} onClick={() => del(e.id!)}><Icon name="trash" size={16} /></button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

const PROFILE_FIELDS: { key: string; label: string; type?: 'number' | 'select'; options?: string[] }[] = [
  { key: 'business_name', label: 'Business name' },
  { key: 'business_type', label: 'What you sell / do' },
  { key: 'business_domain', label: 'Domain', type: 'select', options: ['food & street vending', 'retail / kirana', 'agriculture & allied', 'manufacturing', 'handicraft & artisan', 'services', 'tech startup', 'textiles & tailoring', 'beauty & wellness', 'transport & logistics', 'other'] },
  { key: 'business_stage', label: 'Stage', type: 'select', options: ['idea', 'new (under 1 year)', 'running (1-5 years)', 'growing (5+ years)'] },
  { key: 'city', label: 'City / town' },
  { key: 'state', label: 'State', type: 'select', options: ['Andhra Pradesh', 'Telangana', 'Tamil Nadu', 'Karnataka', 'Kerala', 'Maharashtra', 'Uttar Pradesh', 'Bihar', 'West Bengal', 'Gujarat', 'Rajasthan', 'Madhya Pradesh', 'Delhi', 'Odisha', 'Punjab', 'Haryana', 'Assam', 'Other'] },
  { key: 'gender', label: 'Gender', type: 'select', options: ['female', 'male', 'transgender', 'prefer not to say'] },
  { key: 'age', label: 'Age', type: 'number' },
  { key: 'social_category', label: 'Social category', type: 'select', options: ['General', 'OBC', 'SC', 'ST', 'Minority', 'prefer not to say'] },
  { key: 'years_running', label: 'Years running', type: 'number' },
  { key: 'employees', label: 'People working', type: 'number' },
  { key: 'monthly_turnover_band', label: 'Monthly turnover' },
  { key: 'upi_vpa', label: 'UPI ID' },
  { key: 'goal', label: 'Goal for next year' },
]

export function Profile() {
  const { t, lang, toast } = useApp()
  const d = useData<{ profile: any; memories: { id: number; fact: string; created_at: string }[] }>('/profile')
  const [edit, setEdit] = useState(false)
  const [form, setForm] = useState<Record<string, any>>({})
  const [fact, setFact] = useState('')
  const [busy, setBusy] = useState(false)

  if (d.loading && !d.data) return <div className="skeleton" style={{ height: 300 }} />
  const p = d.data?.profile ?? {}

  const save = async () => {
    setBusy(true)
    const body: Record<string, any> = { ...form }
    for (const f of PROFILE_FIELDS) if (f.type === 'number' && body[f.key] !== undefined && body[f.key] !== '') body[f.key] = Number(body[f.key])
    if (typeof body.documents === 'string') body.documents = body.documents.split(',').map((s: string) => s.trim()).filter(Boolean)
    try { await api('/profile', { method: 'PUT', body }); setEdit(false); d.reload(); toast(t('updated')) } catch (e) { toast((e as Error).message) } finally { setBusy(false) }
  }
  const addFact = async () => {
    if (!fact.trim()) return
    await api('/memories', { body: { fact } }).catch((e) => toast(e.message))
    setFact(''); d.reload()
  }
  const delFact = async (id: number) => { await api(`/memories/${id}`, { method: 'DELETE' }); d.reload() }

  return (
    <>
      <PageHead kicker={t('profile_kicker')} title={t('profile_title')}>
        {!edit ? <button className="btn sm" onClick={() => { setForm({ ...p, documents: (p.documents ?? []).join(', ') }); setEdit(true) }}>{t('edit')}</button>
          : <><button className="btn ghost sm" onClick={() => setEdit(false)}>{t('cancel')}</button><button className="btn primary sm" onClick={save} disabled={busy}>{t('save')}</button></>}
      </PageHead>
      <div className="grid2" style={{ alignItems: 'start' }}>
        <div className="sheet">
          <h3 className="display" style={{ fontSize: '1.5rem' }}>{p.business_name || p.owner_name}</h3>
          <table className="ledger-table" style={{ marginTop: 8 }}>
            <tbody>
              {PROFILE_FIELDS.map((f) => (
                <tr key={f.key}>
                  <td className="muted small" style={{ width: '40%' }}>{f.label}</td>
                  <td>
                    {edit ? (
                      f.type === 'select'
                        ? <select className="select" value={form[f.key] ?? ''} onChange={(e) => setForm({ ...form, [f.key]: e.target.value })}><option value="">—</option>{f.options!.map((o) => <option key={o}>{o}</option>)}</select>
                        : <input className="input" type={f.type === 'number' ? 'number' : 'text'} value={form[f.key] ?? ''} onChange={(e) => setForm({ ...form, [f.key]: e.target.value })} />
                    ) : <span className={f.key === 'upi_vpa' ? 'mono' : ''}>{p[f.key] ?? <span className="faint">—</span>}</span>}
                  </td>
                </tr>
              ))}
              <tr>
                <td className="muted small">{t('documents')}</td>
                <td>{edit ? <input className="input" value={form.documents ?? ''} onChange={(e) => setForm({ ...form, documents: e.target.value })} placeholder="Aadhaar, PAN, Udyam" />
                  : <div className="row wrap" style={{ gap: 4 }}>{(p.documents ?? []).map((x: string) => <span key={x} className="chip green">✓ {x}</span>)}</div>}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div className="sheet ruled stack" style={{ lineHeight: '32px' }}>
          <h3>{t('memory')}</h3>
          <p className="small muted" style={{ lineHeight: 1.5 }}>{t('memory_hint')}</p>
          <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
            {d.data?.memories.map((m) => (
              <li key={m.id} className="row between" style={{ gap: 6 }}>
                <span>✎ {m.fact} <span className="tiny faint mono">{fmtDate(m.created_at, lang)}</span></span>
                <button className="btn ghost sm icon" aria-label={t('delete')} onClick={() => delFact(m.id)}><Icon name="x" size={16} /></button>
              </li>
            ))}
          </ul>
          <form className="row" onSubmit={(e) => { e.preventDefault(); addFact() }}>
            <input className="input grow" value={fact} onChange={(e) => setFact(e.target.value)} placeholder={t('add_memory')} aria-label={t('add_memory')} />
            <button className="btn sm">{t('save')}</button>
          </form>
        </div>
      </div>
    </>
  )
}
